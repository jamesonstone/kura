package worktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSweepDefaultBranchUsesPositionalRepository(t *testing.T) {
	app := NewApp(os.Stdout, os.Stderr)
	app.run = func(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
		if name != "gh" || strings.Join(args, " ") != "repo view example/project --json defaultBranchRef" {
			t.Fatalf("command = %s %v", name, args)
		}
		return []byte(`{"defaultBranchRef":{"name":"main"}}`), nil
	}
	branch, err := app.sweepDefaultBranch(context.Background(), t.TempDir(), "example/project")
	if err != nil || branch != "main" {
		t.Fatalf("branch=%q err=%v", branch, err)
	}
}

func TestSweepPullRequestsUseOneFleetBatch(t *testing.T) {
	app := NewApp(os.Stdout, os.Stderr)
	calls := 0
	app.run = func(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
		calls++
		joined := strings.Join(args, " ")
		if name != "gh" || !strings.Contains(joined, "pr list --repo example/project --state all --limit 1000") {
			t.Fatalf("command = %s %v", name, args)
		}
		return []byte(`[{"number":2,"state":"MERGED","baseRefName":"main","headRefName":"GH-1","headRefOid":"abc","isCrossRepository":false,"url":"https://example/2"}]`), nil
	}
	result, err := app.resolveSweepPullRequests(context.Background(), t.TempDir(), "example/project", []string{"GH-1", "GH-2"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(result["GH-1"]) != 1 || len(result["GH-2"]) != 0 {
		t.Fatalf("calls=%d result=%#v", calls, result)
	}
}

func TestSweepClassifiesRemoveReadyAndLocalFiles(t *testing.T) {
	fixture := newGitFixture(t)
	readyPath, readyOID := createMergedLane(t, fixture, "topic/ready")
	dirtyPath, dirtyOID := createMergedLane(t, fixture, "topic/dirty")
	if err := os.WriteFile(filepath.Join(dirtyPath, "local.txt"), []byte("preserve\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/ready": {mergedSyncPR(2, "topic/ready", "main", readyOID)},
		"topic/dirty": {mergedSyncPR(3, "topic/dirty", "main", dirtyOID)},
	})
	report := buildSweepTestReport(t, fixture, sweepTestConfig(t, fixture))
	ready := findSweepCandidate(t, report, "topic/ready")
	dirty := findSweepCandidate(t, report, "topic/dirty")
	if ready.State != SweepRemoveReady || !ready.AutoRemovable || !samePath(ready.Path, readyPath) {
		t.Fatalf("ready = %#v", ready)
	}
	if dirty.State != SweepMergedLocalFiles || dirty.AutoRemovable || !dirty.ForceWorktree {
		t.Fatalf("dirty = %#v", dirty)
	}
}

func TestSweepClassifiesMergedLocalCommits(t *testing.T) {
	fixture := newGitFixture(t)
	path, mergedOID := createMergedLane(t, fixture, "topic/diverged")
	if err := os.WriteFile(filepath.Join(path, "later.txt"), []byte("later\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, path, "add", "later.txt")
	runGit(t, path, "commit", "-m", "local follow-up")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/diverged": {mergedSyncPR(4, "topic/diverged", "main", mergedOID)},
	})
	candidate := findSweepCandidate(t, buildSweepTestReport(t, fixture, sweepTestConfig(t, fixture)), "topic/diverged")
	if candidate.State != SweepMergedLocalCommits || !candidate.ForceBranch || len(candidate.ExtraCommits) != 1 {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestSweepPreservesUnprovenAndForkPullRequests(t *testing.T) {
	fixture := newGitFixture(t)
	_, openOID := createPublishedLane(t, fixture, "topic/open")
	_, forkOID := createPublishedLane(t, fixture, "topic/fork")
	openPR := mergedSyncPR(5, "topic/open", "main", openOID)
	openPR.State, openPR.MergedAt = "OPEN", nil
	forkPR := mergedSyncPR(6, "topic/fork", "main", forkOID)
	forkPR.IsCrossRepository = true
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/open": {openPR}, "topic/fork": {forkPR},
	})
	report := buildSweepTestReport(t, fixture, sweepTestConfig(t, fixture))
	if candidate := findSweepCandidate(t, report, "topic/open"); candidate.State != SweepUnproven || candidate.Selectable {
		t.Fatalf("open = %#v", candidate)
	}
	if candidate := findSweepCandidate(t, report, "topic/fork"); candidate.Reason != "pull-request-from-fork" {
		t.Fatalf("fork = %#v", candidate)
	}
}

func TestSweepProtectsLockedAndCurrentWorktrees(t *testing.T) {
	fixture := newGitFixture(t)
	lockedPath, lockedOID := createPublishedLane(t, fixture, "topic/locked")
	currentPath, currentOID := createPublishedLane(t, fixture, "topic/current")
	runGit(t, fixture.primary, "worktree", "lock", "--reason", "active session", lockedPath)
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/locked":  {mergedSyncPR(14, "topic/locked", "main", lockedOID)},
		"topic/current": {mergedSyncPR(15, "topic/current", "main", currentOID)},
	})
	config := sweepTestConfig(t, fixture)
	report := fixture.app.buildSweepReport(
		context.Background(), currentPath, config,
		SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true},
	)
	locked := findSweepCandidate(t, report, "topic/locked")
	current := findSweepCandidate(t, report, "topic/current")
	if locked.State != SweepProtectedActive || locked.Reason != "locked-worktree" || locked.Selectable {
		t.Fatalf("locked = %#v", locked)
	}
	if current.State != SweepProtectedActive || current.Reason != "current-worktree" || current.Selectable {
		t.Fatalf("current = %#v", current)
	}
}
