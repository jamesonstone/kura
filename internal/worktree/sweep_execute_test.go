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

func TestSweepApplyUsesOneFleetRevalidationForManyTargets(t *testing.T) {
	fixture := newGitFixture(t)
	firstPath, firstOID := createMergedLane(t, fixture, "topic/batch-first")
	secondPath, secondOID := createMergedLane(t, fixture, "topic/batch-second")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/batch-first":  {mergedSyncPR(17, "topic/batch-first", "main", firstOID)},
		"topic/batch-second": {mergedSyncPR(18, "topic/batch-second", "main", secondOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	selected := []SweepCandidate{
		findSweepCandidate(t, report, "topic/batch-first"),
		findSweepCandidate(t, report, "topic/batch-second"),
	}
	resolver := fixture.app.resolveSweepBatch
	calls := 0
	fixture.app.resolveSweepBatch = func(ctx context.Context, requests []sweepEvidenceRequest, timeout time.Duration, progress *sweepProgress) map[string]sweepResolvedEvidence {
		calls++
		return resolver(ctx, requests, timeout, progress)
	}
	if err := fixture.app.applySweepCandidates(context.Background(), fixture.primary, config, options, &report, selected); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("GitHub evidence resolutions during apply = %d, want 1", calls)
	}
	for _, path := range []string{firstPath, secondPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("revalidated path remains %s: %v", path, err)
		}
	}
}

func TestSweepMissingTargetDoesNotBlockIndependentRemoval(t *testing.T) {
	fixture := newGitFixture(t)
	missingPath, missingOID := createMergedLane(t, fixture, "topic/missing-before-apply")
	readyPath, readyOID := createMergedLane(t, fixture, "topic/independent-ready")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/missing-before-apply": {mergedSyncPR(19, "topic/missing-before-apply", "main", missingOID)},
		"topic/independent-ready":    {mergedSyncPR(20, "topic/independent-ready", "main", readyOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	missing := findSweepCandidate(t, report, "topic/missing-before-apply")
	ready := findSweepCandidate(t, report, "topic/independent-ready")
	runGit(t, fixture.primary, "worktree", "remove", missingPath)
	err := fixture.app.applySweepCandidates(context.Background(), fixture.primary, config, options, &report, []SweepCandidate{missing, ready})
	if err == nil {
		t.Fatal("expected missing-target failure")
	}
	if _, statErr := os.Stat(readyPath); !os.IsNotExist(statErr) {
		t.Fatalf("independent ready path remains: %v", statErr)
	}
	if len(report.Actions) != 2 || report.Actions[0].Status != "preserved" || report.Actions[1].Status != "removed" {
		t.Fatalf("actions = %#v", report.Actions)
	}
}

func TestSweepRemovalRechecksLocalStatusFingerprint(t *testing.T) {
	fixture := newGitFixture(t)
	path, headOID := createMergedLane(t, fixture, "topic/local-drift")
	if err := os.WriteFile(filepath.Join(path, "reviewed.txt"), []byte("reviewed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/local-drift": {mergedSyncPR(21, "topic/local-drift", "main", headOID)},
	})
	candidate := findSweepCandidate(t, buildSweepTestReport(t, fixture, sweepTestConfig(t, fixture)), "topic/local-drift")
	if err := os.WriteFile(filepath.Join(path, "added-after-refresh.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := fixture.app.removeSweepWorktree(context.Background(), candidate)
	if err == nil || !strings.Contains(err.Error(), "local status changed after fleet revalidation") {
		t.Fatalf("removal error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("locally drifted worktree was removed: %v", err)
	}
}

func TestSweepApplyCatchesStatusDriftAfterFleetRefresh(t *testing.T) {
	fixture := newGitFixture(t)
	path, headOID := createMergedLane(t, fixture, "topic/apply-local-drift")
	if err := os.WriteFile(filepath.Join(path, "reviewed.txt"), []byte("reviewed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/apply-local-drift": {mergedSyncPR(22, "topic/apply-local-drift", "main", headOID)},
	})
	config := sweepTestConfig(t, fixture)
	config.ProcessCheck = "best_effort"
	fixture.app.lookPath = func(string) (string, error) { return "/usr/sbin/lsof", nil }
	baseRun := fixture.app.run
	lsofCalls := 0
	fixture.app.run = func(ctx context.Context, cwd, name string, args ...string) ([]byte, error) {
		if name != "lsof" {
			return baseRun(ctx, cwd, name, args...)
		}
		lsofCalls++
		if lsofCalls == 3 {
			if err := os.WriteFile(filepath.Join(path, "after-fleet-refresh.txt"), []byte("new\n"), 0o644); err != nil {
				return nil, err
			}
		}
		return []byte{}, nil
	}
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	candidate := findSweepCandidate(t, report, "topic/apply-local-drift")
	err := fixture.app.applySweepCandidates(context.Background(), fixture.primary, config, options, &report, []SweepCandidate{candidate})
	if err == nil || !strings.Contains(err.Error(), "local status changed after fleet revalidation") {
		t.Fatalf("apply error = %v", err)
	}
	if lsofCalls != 3 {
		t.Fatalf("process snapshots = %d, want 3 including initial report", lsofCalls)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("worktree drifted after fleet refresh was removed: %v", err)
	}
}
