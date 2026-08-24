package worktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSweepStaleUnprovenRetirementPreservesLocalBranch(t *testing.T) {
	fixture := newGitFixture(t)
	path, _ := createPublishedLane(t, fixture, "topic/stale-unproven")
	headOID := ageSweepTestLane(t, path)
	if err := os.WriteFile(filepath.Join(path, "local.txt"), []byte("discard after review\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	candidate := findSweepCandidate(t, report, "topic/stale-unproven")
	if !candidate.Stale || !candidate.StaleRetirable || !candidate.Selectable || !candidate.ForceWorktree {
		t.Fatalf("candidate = %#v", candidate)
	}
	if candidate.State != SweepUnproven || candidate.AutoRemovable || candidate.Status.Fingerprint == "" {
		t.Fatalf("authority = %#v", candidate)
	}
	if err := fixture.app.applySweepCandidates(context.Background(), fixture.primary, config, options, &report, []SweepCandidate{candidate}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("retired path remains: %v", err)
	}
	if actual := gitText(t, fixture.primary, "rev-parse", "refs/heads/topic/stale-unproven"); actual != headOID {
		t.Fatalf("preserved branch = %q, want %q", actual, headOID)
	}
}

func TestSweepStaleUnprovenRetirementIsInteractiveOnly(t *testing.T) {
	candidate := SweepCandidate{
		State: SweepUnproven, Reason: "pull-request-missing", Stale: true,
		StaleRetirable: true, Selectable: true,
	}
	if err := validateSweepAuthority(candidate, false); err != nil {
		t.Fatalf("interactive authority rejected: %v", err)
	}
	if err := validateSweepAuthority(candidate, true); err == nil {
		t.Fatal("automatic STALE unproven retirement was accepted")
	}
}

func TestPrepareStaleRetirementBlocksActiveProcess(t *testing.T) {
	fixture := newGitFixture(t)
	path, headOID := createPublishedLane(t, fixture, "topic/stale-active")
	candidate := SweepCandidate{
		ID: "stale-active", PrimaryPath: fixture.primary, Path: path,
		Branch: "topic/stale-active", HeadOID: headOID, State: SweepUnproven,
		Reason: "pull-request-missing", Stale: true,
	}
	target := sweepRepository{primary: fixture.primary}
	config := SweepConfig{ProcessCheck: "best_effort", processPaths: []string{filepath.Join(path, "nested")}}
	fixture.app.prepareStaleRetirableCandidate(context.Background(), config, target, &candidate)
	if candidate.State != SweepProtectedActive || candidate.Selectable || candidate.StaleRetirable {
		t.Fatalf("active candidate = %#v", candidate)
	}
}

func TestSweepStaleRetirementReviewShowsPreservedBranch(t *testing.T) {
	candidate := SweepCandidate{
		Repository: "example/project", Path: "/tmp/GH-1", Branch: "GH-1",
		HeadOID: "abc", State: SweepUnproven, Stale: true, StaleRetirable: true,
		Selectable: true, ForceWorktree: true,
	}
	var output strings.Builder
	if err := writeSweepReview(&output, []SweepCandidate{candidate}, false); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"STALE interactive override", "merge remains unproven", "preserved as recovery ref",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("review missing %q:\n%s", expected, output.String())
		}
	}
}

func ageSweepTestLane(t *testing.T, path string) string {
	t.Helper()
	command := gitCommand(path, "commit", "--amend", "--no-edit", "--date=2025-01-01T00:00:00Z")
	command.Env = append(command.Env,
		"GIT_AUTHOR_DATE=2025-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2025-01-01T00:00:00Z",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("age test lane: %v\n%s", err, output)
	}
	return gitText(t, path, "rev-parse", "HEAD")
}
