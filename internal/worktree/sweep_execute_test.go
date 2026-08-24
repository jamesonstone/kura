package worktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSweepAutoRemovesOnlyReadyLane(t *testing.T) {
	fixture := newGitFixture(t)
	readyPath, readyOID := createMergedLane(t, fixture, "topic/auto-ready")
	dirtyPath, dirtyOID := createMergedLane(t, fixture, "topic/auto-dirty")
	if err := os.WriteFile(filepath.Join(dirtyPath, "local.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/auto-ready": {mergedSyncPR(7, "topic/auto-ready", "main", readyOID)},
		"topic/auto-dirty": {mergedSyncPR(8, "topic/auto-dirty", "main", dirtyOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Auto: true, Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	if err := fixture.app.applySweepCandidates(context.Background(), fixture.primary, config, options, &report, automaticSweepCandidates(report.Candidates)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(readyPath); !os.IsNotExist(err) {
		t.Fatalf("ready path remains: %v", err)
	}
	if _, err := os.Stat(dirtyPath); err != nil {
		t.Fatalf("dirty path removed: %v", err)
	}
	if gitText(t, fixture.primary, "show-ref", "--verify", "refs/heads/topic/auto-ready") != "" {
		t.Fatal("ready branch remains")
	}
	if gitText(t, fixture.primary, "show-ref", "--verify", "refs/heads/topic/auto-dirty") == "" {
		t.Fatal("dirty branch was deleted")
	}
}

func TestSweepConfirmedDirtyRemoval(t *testing.T) {
	fixture := newGitFixture(t)
	path, headOID := createMergedLane(t, fixture, "topic/confirmed-dirty")
	if err := os.WriteFile(filepath.Join(path, "local.txt"), []byte("discard\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/confirmed-dirty": {mergedSyncPR(9, "topic/confirmed-dirty", "main", headOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	candidate := findSweepCandidate(t, report, "topic/confirmed-dirty")
	if err := fixture.app.applySweepCandidates(context.Background(), fixture.primary, config, options, &report, []SweepCandidate{candidate}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dirty path remains: %v", err)
	}
}

func TestSweepConfirmedLocalCommitRemoval(t *testing.T) {
	fixture := newGitFixture(t)
	path, mergedOID := createMergedLane(t, fixture, "topic/confirmed-history")
	if err := os.WriteFile(filepath.Join(path, "later.txt"), []byte("discard\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, path, "add", "later.txt")
	runGit(t, path, "commit", "-m", "unpublished local commit")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/confirmed-history": {mergedSyncPR(10, "topic/confirmed-history", "main", mergedOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	candidate := findSweepCandidate(t, report, "topic/confirmed-history")
	if err := fixture.app.applySweepCandidates(context.Background(), fixture.primary, config, options, &report, []SweepCandidate{candidate}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("history path remains: %v", err)
	}
	if gitText(t, fixture.primary, "show-ref", "--verify", "refs/heads/topic/confirmed-history") != "" {
		t.Fatal("divergent local branch remains")
	}
}

func TestSweepSnapshotDriftPreservesCandidate(t *testing.T) {
	fixture := newGitFixture(t)
	path, headOID := createMergedLane(t, fixture, "topic/drift")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/drift": {mergedSyncPR(11, "topic/drift", "main", headOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	candidate := findSweepCandidate(t, report, "topic/drift")
	if err := os.WriteFile(filepath.Join(path, "changed.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fixture.app.applySweepCandidates(context.Background(), fixture.primary, config, options, &report, []SweepCandidate{candidate}); err == nil {
		t.Fatal("expected drift error")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("drifted path was removed: %v", err)
	}
}

func TestSweepAutoPrunesNativeStaleMetadata(t *testing.T) {
	fixture := newGitFixture(t)
	stalePath, _ := createPublishedLane(t, fixture, "topic/stale")
	_, keepOID := createPublishedLane(t, fixture, "topic/keep-seed")
	if err := os.RemoveAll(stalePath); err != nil {
		t.Fatal(err)
	}
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/keep-seed": {mergedSyncPR(16, "topic/keep-seed", "main", keepOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Auto: true, Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	stale := findSweepCandidate(t, report, "topic/stale")
	if stale.State != SweepStaleMetadata || !stale.AutoRemovable {
		t.Fatalf("stale = %#v", stale)
	}
	if err := fixture.app.applySweepCandidates(context.Background(), fixture.primary, config, options, &report, []SweepCandidate{stale}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(gitText(t, fixture.primary, "worktree", "list", "--porcelain"), stalePath) {
		t.Fatal("stale metadata remains")
	}
}
