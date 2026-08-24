package worktree

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type sweepGraphQLError struct {
	Message string `json:"message"`
}

func (a *App) querySweepGitHubPages(
	ctx context.Context,
	requests []sweepEvidenceRequest,
	cursors map[string]string,
	timeout time.Duration,
) (map[string]sweepGitHubPage, error) {
	if len(requests) == 0 {
		return map[string]sweepGitHubPage{}, nil
	}
	query, args := sweepGitHubQuery(requests, cursors)
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	commandArgs := append([]string{"api", "graphql", "-f", "query=" + query}, args...)
	output, err := a.command(queryCtx, requests[0].cwd, "gh", commandArgs...)
	if err != nil {
		return nil, fmt.Errorf("batch GitHub evidence lookup: %w", err)
	}
	var payload struct {
		Data   map[string]json.RawMessage `json:"data"`
		Errors []sweepGraphQLError        `json:"errors"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("decode batched GitHub evidence: %w", err)
	}
	if len(payload.Errors) != 0 {
		messages := make([]string, 0, len(payload.Errors))
		for _, graphErr := range payload.Errors {
			messages = append(messages, graphErr.Message)
		}
		return nil, fmt.Errorf("GitHub GraphQL: %s", strings.Join(messages, "; "))
	}
	pages := make(map[string]sweepGitHubPage, len(requests))
	for index, request := range requests {
		raw, ok := payload.Data[fmt.Sprintf("r%d", index)]
		if !ok || string(raw) == "null" {
			return nil, fmt.Errorf("GitHub did not return %s", request.identity)
		}
		var page sweepGitHubPage
		if err := json.Unmarshal(raw, &page); err != nil {
			return nil, fmt.Errorf("decode GitHub evidence for %s: %w", request.identity, err)
		}
		pages[request.identity] = page
	}
	return pages, nil
}

func sweepGitHubQuery(requests []sweepEvidenceRequest, cursors map[string]string) (string, []string) {
	variables := make([]string, 0, len(requests)*3)
	fields := make([]string, 0, len(requests))
	args := make([]string, 0, len(requests)*6)
	for index, request := range requests {
		variables = append(variables,
			fmt.Sprintf("$owner%d:String!", index),
			fmt.Sprintf("$name%d:String!", index),
			fmt.Sprintf("$cursor%d:String", index),
		)
		fields = append(fields, fmt.Sprintf(`r%d:repository(owner:$owner%d,name:$name%d){defaultBranchRef{name}pullRequests(first:100,after:$cursor%d,states:[OPEN,CLOSED,MERGED],orderBy:{field:CREATED_AT,direction:DESC}){nodes{number state mergedAt baseRefName headRefName headRefOid isCrossRepository url}pageInfo{hasNextPage endCursor}}}`, index, index, index, index))
		args = append(args, "-F", fmt.Sprintf("owner%d=%s", index, request.owner), "-F", fmt.Sprintf("name%d=%s", index, request.name))
		if cursor := cursors[request.identity]; cursor != "" {
			args = append(args, "-F", fmt.Sprintf("cursor%d=%s", index, cursor))
		}
	}
	return "query(" + strings.Join(variables, ",") + "){" + strings.Join(fields, "") + "}", args
}
