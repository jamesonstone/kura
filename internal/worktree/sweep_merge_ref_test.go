package worktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSweepRemovesSquashMergedLaneTrackingDefaultBranch(t *testing.T) {
	fixture := newGitFixture(t)
	path, headOID := createSquashMergedLane(t, fixture, "GH-132")
	runGit(t, fixture.primary, "config", "branch.GH-132.merge", "refs/heads/main")
	runGit(t, fixture.primary, "config", "branch.GH-132.remote", "origin")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"GH-132": {mergedSyncPR(133, "GH-132", "main", headOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Auto: true, Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	candidate := findSweepCandidate(t, report, "GH-132")
	if candidate.State != SweepRemoveReady {
		t.Fatalf("state = %s, want %s", candidate.State, SweepRemoveReady)
	}
	if err := fixture.app.applySweepCandidates(
		context.Background(),
		fixture.primary,
		config,
		options,
		&report,
		[]SweepCandidate{candidate},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree remains: %v", err)
	}
	if gitText(t, fixture.primary, "show-ref", "--verify", "refs/heads/GH-132") != "" {
		t.Fatal("local branch remains")
	}
}

func TestSweepForceRemovesDirtyLaneTrackingDefaultBranch(t *testing.T) {
	fixture := newGitFixture(t)
	path, headOID := createSquashMergedLane(t, fixture, "GH-140")
	runGit(t, fixture.primary, "config", "branch.GH-140.merge", "refs/heads/main")
	runGit(t, fixture.primary, "config", "branch.GH-140.remote", "origin")
	if err := os.WriteFile(filepath.Join(path, "local.txt"), []byte("discard\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"GH-140": {mergedSyncPR(141, "GH-140", "main", headOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	candidate := findSweepCandidate(t, report, "GH-140")
	if candidate.State != SweepMergedLocalFiles || !candidate.ForceWorktree {
		t.Fatalf("candidate state=%s forceWorktree=%v", candidate.State, candidate.ForceWorktree)
	}
	if err := fixture.app.applySweepCandidates(
		context.Background(),
		fixture.primary,
		config,
		options,
		&report,
		[]SweepCandidate{candidate},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dirty worktree remains: %v", err)
	}
	if gitText(t, fixture.primary, "show-ref", "--verify", "refs/heads/GH-140") != "" {
		t.Fatal("local branch remains")
	}
}
