package worktree

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

func (a *App) resolveSweepPullRequests(
	ctx context.Context,
	cwd string,
	repository string,
	branches []string,
) (map[string][]SyncPullRequest, error) {
	args := []string{
		"pr", "list", "--repo", repository, "--state", "all", "--limit", "1000",
		"--json", "number,state,mergedAt,baseRefName,headRefName,headRefOid,isCrossRepository,url",
	}
	output, err := a.command(ctx, cwd, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("batch pull-request lookup: %w", err)
	}
	var pullRequests []SyncPullRequest
	if err := json.Unmarshal(output, &pullRequests); err != nil {
		return nil, fmt.Errorf("decode GitHub pull requests: %w", err)
	}
	wanted := make(map[string]bool, len(branches))
	result := make(map[string][]SyncPullRequest, len(branches))
	for _, branch := range branches {
		wanted[branch] = true
		result[branch] = []SyncPullRequest{}
	}
	for _, pullRequest := range pullRequests {
		if wanted[pullRequest.HeadRefName] {
			result[pullRequest.HeadRefName] = append(result[pullRequest.HeadRefName], pullRequest)
		}
	}
	for branch := range result {
		sort.Slice(result[branch], func(i, j int) bool {
			return result[branch][i].Number < result[branch][j].Number
		})
	}
	return result, nil
}
