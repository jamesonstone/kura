package worktree

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSweepTerminalShowsExactApplyFailure(t *testing.T) {
	fixture := newGitFixture(t)
	path, headOID := createMergedLane(t, fixture, "topic/terminal-failure")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/terminal-failure": {mergedSyncPR(23, "topic/terminal-failure", "main", headOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	report.Failures = append(report.Failures, SweepFailure{
		Operation: "github-evidence", Repository: "example/initial", Path: "/tmp/initial",
		Error: "initial failure already shown",
	})
	if err := os.WriteFile(filepath.Join(path, "changed.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prepareSweepTerminalInput(fixture, "r\n\n")
	err := fixture.app.runSweepTerminal(context.Background(), fixture.primary, config, options, &report)
	if err == nil {
		t.Fatal("expected terminal apply failure")
	}
	output := fixture.out.String()
	for _, expected := range []string{
		"SWEEP COMPLETION",
		"Removed 0 worktree(s)",
		"preserved/failed 1 target(s)",
		"FAILURES (1)",
		"candidate-drift",
		path,
		"candidate changed after review; refresh required",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("terminal output missing %q:\n%s", expected, output)
		}
	}
	if strings.Count(output, "initial failure already shown") != 1 || strings.Count(output, "candidate changed after review") != 1 {
		t.Fatalf("initial or apply failure repeated:\n%s", output)
	}
}

func TestSweepTerminalShowsPartialSuccessAndFailure(t *testing.T) {
	fixture := newGitFixture(t)
	driftedPath, driftedOID := createMergedLane(t, fixture, "topic/terminal-partial-drift")
	removedPath, removedOID := createMergedLane(t, fixture, "topic/terminal-partial-ready")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/terminal-partial-drift": {mergedSyncPR(24, "topic/terminal-partial-drift", "main", driftedOID)},
		"topic/terminal-partial-ready": {mergedSyncPR(25, "topic/terminal-partial-ready", "main", removedOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	if err := os.WriteFile(filepath.Join(driftedPath, "changed.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prepareSweepTerminalInput(fixture, "r\n\n")
	err := fixture.app.runSweepTerminal(context.Background(), fixture.primary, config, options, &report)
	if err == nil {
		t.Fatal("expected partial apply failure")
	}
	output := fixture.out.String()
	for _, expected := range []string{"Removed 1 worktree(s)", "preserved/failed 1 target(s)", "FAILURES (1)", driftedPath} {
		if !strings.Contains(output, expected) {
			t.Fatalf("partial output missing %q:\n%s", expected, output)
		}
	}
	if _, statErr := os.Stat(removedPath); !os.IsNotExist(statErr) {
		t.Fatalf("independent worktree remains: %v", statErr)
	}
}

func TestSweepTerminalShowsSuccessfulCompletionWithoutFailures(t *testing.T) {
	fixture := newGitFixture(t)
	path, headOID := createMergedLane(t, fixture, "topic/terminal-success")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/terminal-success": {mergedSyncPR(26, "topic/terminal-success", "main", headOID)},
	})
	config := sweepTestConfig(t, fixture)
	options := SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	prepareSweepTerminalInput(fixture, "r\n\n")
	if err := fixture.app.runSweepTerminal(context.Background(), fixture.primary, config, options, &report); err != nil {
		t.Fatal(err)
	}
	output := fixture.out.String()
	if !strings.Contains(output, "Removed 1 worktree(s)") || strings.Contains(output, "FAILURES (") {
		t.Fatalf("success output:\n%s", output)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("successful worktree remains: %v", err)
	}
}

func TestSweepCompletionSanitizesFailureFields(t *testing.T) {
	report := SweepReport{
		Candidates: []SweepCandidate{{ID: "one"}},
		Actions:    []SweepAction{{CandidateID: "one", Status: "preserved"}},
	}
	failures := []SweepFailure{{
		Operation: "remove\x1b[31m", Repository: "example/project\x1b[2J",
		Path: "/tmp/lane\nnext", Error: "failed\x1b[1m",
	}}
	var output bytes.Buffer
	if err := writeSweepCompletion(&output, report, failures, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "\x1b[") || !strings.Contains(output.String(), "example/project") {
		t.Fatalf("completion output = %q", output.String())
	}
}

func prepareSweepTerminalInput(fixture gitFixture, input string) {
	fixture.out.Reset()
	fixture.app.isTerminal = func() bool { return true }
	fixture.app.stdin = strings.NewReader(input)
}
