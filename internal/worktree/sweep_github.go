package worktree

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	sweepGitHubBatchSize = 8
	sweepGitHubPageLimit = 10
)

type sweepEvidenceRequest struct {
	identity string
	owner    string
	name     string
	cwd      string
	branches []string
}

type sweepResolvedEvidence struct {
	identity      string
	defaultBranch string
	pullRequests  map[string][]SyncPullRequest
	err           error
}

type sweepEvidenceResolverFunc func(
	context.Context,
	[]sweepEvidenceRequest,
	time.Duration,
	*sweepProgress,
) map[string]sweepResolvedEvidence

type sweepGitHubPage struct {
	DefaultBranchRef struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
	PullRequests struct {
		Nodes    []SyncPullRequest `json:"nodes"`
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
	} `json:"pullRequests"`
}

func (a *App) collectSweepEvidence(
	ctx context.Context,
	config SweepConfig,
	repositories []sweepRepository,
	progress *sweepProgress,
) map[string]sweepResolvedEvidence {
	byCommonDir := make(map[string]sweepResolvedEvidence, len(repositories))
	requests := make(map[string]*sweepEvidenceRequest)
	for _, target := range repositories {
		owner, name, err := a.projectIdentity(ctx, target.primary)
		identity := strings.ToLower(owner + "/" + name)
		if err != nil {
			byCommonDir[target.commonDir] = sweepResolvedEvidence{identity: identity, err: err}
			continue
		}
		request := requests[identity]
		if request == nil {
			request = &sweepEvidenceRequest{identity: identity, owner: owner, name: name, cwd: target.primary}
			requests[identity] = request
		}
		for _, entry := range target.entries {
			if entry.branch != "" && !entry.primary && !entry.prunable {
				request.branches = append(request.branches, entry.branch)
			}
		}
		byCommonDir[target.commonDir] = sweepResolvedEvidence{identity: identity}
	}
	ordered := make([]sweepEvidenceRequest, 0, len(requests))
	for _, request := range requests {
		request.branches = uniqueStrings(request.branches)
		sort.Strings(request.branches)
		ordered = append(ordered, *request)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].identity < ordered[j].identity })
	resolved := a.resolveSweepBatch(ctx, ordered, config.Timeout, progress)
	for commonDir, evidence := range byCommonDir {
		if evidence.err != nil {
			continue
		}
		result, ok := resolved[evidence.identity]
		if !ok {
			result = sweepResolvedEvidence{identity: evidence.identity, err: fmt.Errorf("GitHub evidence missing from batch")}
		}
		byCommonDir[commonDir] = result
	}
	return byCommonDir
}

func (a *App) resolveSweepEvidenceBatch(
	ctx context.Context,
	requests []sweepEvidenceRequest,
	timeout time.Duration,
	progress *sweepProgress,
) map[string]sweepResolvedEvidence {
	results := make(map[string]sweepResolvedEvidence, len(requests))
	batchCount := (len(requests) + sweepGitHubBatchSize - 1) / sweepGitHubBatchSize
	for start := 0; start < len(requests); start += sweepGitHubBatchSize {
		end := min(start+sweepGitHubBatchSize, len(requests))
		progress.update("Querying GitHub batch %d/%d (%d repositories)", start/sweepGitHubBatchSize+1, batchCount, end-start)
		for identity, result := range a.resolveSweepEvidenceChunk(ctx, requests[start:end], timeout) {
			results[identity] = result
		}
	}
	return results
}

func (a *App) resolveSweepEvidenceChunk(
	ctx context.Context,
	requests []sweepEvidenceRequest,
	timeout time.Duration,
) map[string]sweepResolvedEvidence {
	results := make(map[string]sweepResolvedEvidence, len(requests))
	active := append([]sweepEvidenceRequest(nil), requests...)
	cursors := make(map[string]string)
	for pageNumber := 0; pageNumber < sweepGitHubPageLimit && len(active) != 0; pageNumber++ {
		pages, err := a.querySweepGitHubPages(ctx, active, cursors, timeout)
		if err != nil && sweepGitHubGloballyUnavailable(err) {
			for _, request := range active {
				results[request.identity] = sweepResolvedEvidence{identity: request.identity, err: err}
			}
			active = nil
			break
		} else if err != nil && len(active) > 1 {
			pages = make(map[string]sweepGitHubPage)
			for _, request := range active {
				single, singleErr := a.querySweepGitHubPages(ctx, []sweepEvidenceRequest{request}, cursors, timeout)
				if singleErr != nil {
					results[request.identity] = sweepResolvedEvidence{identity: request.identity, err: singleErr}
					continue
				}
				pages[request.identity] = single[request.identity]
			}
		} else if err != nil {
			results[active[0].identity] = sweepResolvedEvidence{identity: active[0].identity, err: err}
			active = nil
			break
		}
		next := make([]sweepEvidenceRequest, 0)
		for _, request := range active {
			if results[request.identity].err != nil {
				continue
			}
			page, ok := pages[request.identity]
			if !ok {
				results[request.identity] = sweepResolvedEvidence{identity: request.identity, err: fmt.Errorf("GitHub omitted repository from batch")}
				continue
			}
			result := results[request.identity]
			result.identity = request.identity
			if result.defaultBranch == "" {
				result.defaultBranch = page.DefaultBranchRef.Name
			}
			result.pullRequests = appendSweepPullRequests(result.pullRequests, request.branches, page.PullRequests.Nodes)
			results[request.identity] = result
			if page.PullRequests.PageInfo.HasNextPage && len(request.branches) != 0 {
				cursors[request.identity] = page.PullRequests.PageInfo.EndCursor
				next = append(next, request)
			}
		}
		active = next
	}
	for _, request := range active {
		results[request.identity] = sweepResolvedEvidence{identity: request.identity, err: fmt.Errorf("GitHub evidence exceeds %d pull requests", sweepGitHubPageLimit*100)}
	}
	for _, request := range requests {
		result := results[request.identity]
		if result.err == nil && result.defaultBranch == "" {
			result.err = fmt.Errorf("GitHub did not report a default branch")
		}
		for branch := range result.pullRequests {
			sort.Slice(result.pullRequests[branch], func(i, j int) bool {
				return result.pullRequests[branch][i].Number < result.pullRequests[branch][j].Number
			})
		}
		results[request.identity] = result
	}
	return results
}

func appendSweepPullRequests(
	current map[string][]SyncPullRequest,
	branches []string,
	pullRequests []SyncPullRequest,
) map[string][]SyncPullRequest {
	if current == nil {
		current = make(map[string][]SyncPullRequest, len(branches))
		for _, branch := range branches {
			current[branch] = []SyncPullRequest{}
		}
	}
	for _, pullRequest := range pullRequests {
		if _, wanted := current[pullRequest.HeadRefName]; wanted {
			current[pullRequest.HeadRefName] = append(current[pullRequest.HeadRefName], pullRequest)
		}
	}
	return current
}

func sweepGitHubGloballyUnavailable(err error) bool {
	message := strings.ToLower(err.Error())
	for _, marker := range []string{
		"rate limit", "secondary rate", "bad credentials", "authentication required", "not logged into",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
