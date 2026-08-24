package worktree

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSweepGitHubEvidenceBatchesRepositories(t *testing.T) {
	app := NewApp(os.Stdout, os.Stderr)
	calls := 0
	app.run = func(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
		calls++
		joined := strings.Join(args, " ")
		if name != "gh" || !strings.Contains(joined, "api graphql") || !strings.Contains(joined, "r1:repository") {
			t.Fatalf("command = %s %v", name, args)
		}
		return []byte(`{"data":{"r0":{"defaultBranchRef":{"name":"main"},"pullRequests":{"nodes":[{"number":2,"state":"MERGED","baseRefName":"main","headRefName":"GH-1","headRefOid":"abc","isCrossRepository":false,"url":"https://example/2"}],"pageInfo":{"hasNextPage":false,"endCursor":""}}},"r1":{"defaultBranchRef":{"name":"trunk"},"pullRequests":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`), nil
	}
	requests := []sweepEvidenceRequest{
		{identity: "example/one", owner: "example", name: "one", cwd: t.TempDir(), branches: []string{"GH-1"}},
		{identity: "example/two", owner: "example", name: "two", cwd: t.TempDir(), branches: []string{"GH-2"}},
	}
	result := app.resolveSweepEvidenceBatch(context.Background(), requests, time.Second, nil)
	if calls != 1 || result["example/one"].defaultBranch != "main" || result["example/two"].defaultBranch != "trunk" {
		t.Fatalf("calls=%d result=%#v", calls, result)
	}
	if len(result["example/one"].pullRequests["GH-1"]) != 1 || len(result["example/two"].pullRequests["GH-2"]) != 0 {
		t.Fatalf("pull requests = %#v", result)
	}
}

func TestSweepGitHubEvidencePaginatesBatches(t *testing.T) {
	app := NewApp(os.Stdout, os.Stderr)
	calls := 0
	app.run = func(_ context.Context, _ string, _ string, args ...string) ([]byte, error) {
		calls++
		joined := strings.Join(args, " ")
		if calls == 1 {
			return sweepGraphQLPayload(1, true, "next"), nil
		}
		if !strings.Contains(joined, "cursor0=next") {
			t.Fatalf("second command lacks cursor: %v", args)
		}
		return sweepGraphQLPayload(2, false, ""), nil
	}
	request := sweepEvidenceRequest{identity: "example/one", owner: "example", name: "one", cwd: t.TempDir(), branches: []string{"GH-1"}}
	result := app.resolveSweepEvidenceBatch(context.Background(), []sweepEvidenceRequest{request}, time.Second, nil)
	if calls != 2 || len(result[request.identity].pullRequests["GH-1"]) != 2 {
		t.Fatalf("calls=%d result=%#v", calls, result)
	}
}

func TestSweepGitHubBatchFailureFallsBackAndFailsClosed(t *testing.T) {
	app := NewApp(os.Stdout, os.Stderr)
	calls := 0
	app.run = func(_ context.Context, _ string, _ string, args ...string) ([]byte, error) {
		calls++
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "r1:repository"):
			return []byte(`{"errors":[{"message":"batch failed"}]}`), nil
		case strings.Contains(joined, "owner0=good"):
			return sweepGraphQLPayload(1, false, ""), nil
		default:
			return []byte(`{"errors":[{"message":"repository unavailable"}]}`), nil
		}
	}
	requests := []sweepEvidenceRequest{
		{identity: "good/one", owner: "good", name: "one", cwd: t.TempDir(), branches: []string{"GH-1"}},
		{identity: "bad/two", owner: "bad", name: "two", cwd: t.TempDir(), branches: []string{"GH-2"}},
	}
	result := app.resolveSweepEvidenceBatch(context.Background(), requests, time.Second, nil)
	if calls != 3 || result["good/one"].err != nil || result["bad/two"].err == nil {
		t.Fatalf("calls=%d result=%#v", calls, result)
	}
}

func TestSweepGitHubRateLimitFailureDoesNotFanOut(t *testing.T) {
	app := NewApp(os.Stdout, os.Stderr)
	calls := 0
	app.run = func(_ context.Context, _ string, _ string, _ ...string) ([]byte, error) {
		calls++
		return nil, fmt.Errorf("API rate limit exceeded")
	}
	requests := []sweepEvidenceRequest{
		{identity: "one/a", owner: "one", name: "a", cwd: t.TempDir()},
		{identity: "two/b", owner: "two", name: "b", cwd: t.TempDir()},
	}
	result := app.resolveSweepEvidenceBatch(context.Background(), requests, time.Second, nil)
	if calls != 1 || result["one/a"].err == nil || result["two/b"].err == nil {
		t.Fatalf("calls=%d result=%#v", calls, result)
	}
}

func TestCollectSweepEvidenceDeduplicatesRepositoryIdentity(t *testing.T) {
	fixture := newGitFixture(t)
	requestsSeen := 0
	fixture.app.resolveSweepBatch = func(_ context.Context, requests []sweepEvidenceRequest, _ time.Duration, _ *sweepProgress) map[string]sweepResolvedEvidence {
		requestsSeen = len(requests)
		return map[string]sweepResolvedEvidence{requests[0].identity: {identity: requests[0].identity, defaultBranch: "main", pullRequests: map[string][]SyncPullRequest{}}}
	}
	repositories := []sweepRepository{
		{commonDir: "one", primary: fixture.primary, entries: []worktreeEntry{{branch: "GH-1"}}},
		{commonDir: "two", primary: fixture.primary, entries: []worktreeEntry{{branch: "GH-2"}}},
	}
	result := fixture.app.collectSweepEvidence(context.Background(), SweepConfig{Timeout: time.Second}, repositories, nil)
	if requestsSeen != 1 || result["one"].identity != result["two"].identity {
		t.Fatalf("requests=%d result=%#v", requestsSeen, result)
	}
}

func sweepGraphQLPayload(number int, next bool, cursor string) []byte {
	return []byte(fmt.Sprintf(`{"data":{"r0":{"defaultBranchRef":{"name":"main"},"pullRequests":{"nodes":[{"number":%d,"state":"MERGED","baseRefName":"main","headRefName":"GH-1","headRefOid":"abc","isCrossRepository":false,"url":"https://example/%d"}],"pageInfo":{"hasNextPage":%t,"endCursor":"%s"}}}}}`, number, number, next, cursor))
}
