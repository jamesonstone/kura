package worktree

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAddressSweepFailuresCanRetry(t *testing.T) {
	var output bytes.Buffer
	app := NewApp(&output, &output)
	failures := []SweepFailure{{
		Operation: "github-evidence", Repository: "example/project",
		Path: "/tmp/project", Error: "repository unavailable",
	}}
	retry, err := app.addressSweepFailures(
		bufio.NewReader(strings.NewReader("r\n")), SweepConfig{}, SweepReport{Failures: failures},
	)
	if err != nil || !retry {
		t.Fatalf("retry=%t err=%v", retry, err)
	}
	for _, expected := range []string{"FAILURE ACTIONS", "repository unavailable", "retry after fixing GitHub access"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("failure actions missing %q:\n%s", expected, output.String())
		}
	}
}

func TestAddressSweepFailuresWritesConfirmedExclusions(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".config", "kura", "git-wt.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	contents := []byte("version: 1\nsweep:\n  include_builtin_roots: false\n  roots: []\n  project_roots: []\n  exclude_roots: []\n")
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(home, "worktrees", "broken")
	second := filepath.Join(home, "src", "retired")
	linked := filepath.Join(home, "worktrees", "retired", "GH-1")
	failures := []SweepFailure{
		{Operation: "common-directory", Path: first, Error: "invalid marker"},
		{Operation: "github-evidence", Path: second, Error: "not found"},
	}
	report := SweepReport{
		Failures: failures,
		Candidates: []SweepCandidate{{
			PrimaryPath: second, Path: linked, Reason: "github-unavailable",
		}},
	}
	var output bytes.Buffer
	app := NewApp(&output, &output)
	app.homeDir = func() (string, error) { return home, nil }
	input := bufio.NewReader(strings.NewReader("e\na\ny\n"))
	retry, err := app.addressSweepFailures(input, SweepConfig{ConfigPath: configPath}, report)
	if err != nil || !retry {
		t.Fatalf("retry=%t err=%v\n%s", retry, err, output.String())
	}
	document, err := readSweepConfigDocument(configPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"~/src/retired", "~/worktrees/broken", "~/worktrees/retired/GH-1"}
	for _, path := range want {
		if !containsSweepPath(document.raw.Sweep.ExcludeRoots, path) {
			t.Fatalf("exclusions=%#v; missing %s", document.raw.Sweep.ExcludeRoots, path)
		}
	}
	if !strings.Contains(output.String(), "Saved 3 failure exclusion(s); refreshing sweep.") {
		t.Fatalf("output:\n%s", output.String())
	}
}

func TestSweepMenuShowsFailureAndStaleActions(t *testing.T) {
	candidates := []SweepCandidate{
		{ID: "stale-ready", State: SweepRemoveReady, Stale: true, Selectable: true},
		{ID: "stale-blocked", State: SweepProtectedActive, Stale: true},
		{ID: "recent", State: SweepRemoveReady, Selectable: true},
	}
	report := SweepReport{
		Candidates: candidates,
		Failures:   []SweepFailure{{Operation: "github-evidence", Path: "/tmp/project", Error: "missing"}},
	}
	var output bytes.Buffer
	app := NewApp(&output, &output)
	selected, retry, err := app.chooseSweepMenu(
		context.Background(), bufio.NewReader(strings.NewReader("q\n")),
		&report, SweepConfig{}, SweepOptions{},
	)
	if err != nil || retry || len(selected) != 0 {
		t.Fatalf("selected=%#v retry=%t err=%v", selected, retry, err)
	}
	for _, expected := range []string{"[s] review STALE (2 total, 1 selectable)", "[f] address failures (1)"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("menu missing %q:\n%s", expected, output.String())
		}
	}
	stale := staleSweepCandidates(candidates, "")
	if len(stale) != 2 || countSelectableSweepCandidates(stale) != 1 {
		t.Fatalf("stale=%#v", stale)
	}
}

func TestFailureExclusionRefusesBroadHomePath(t *testing.T) {
	home := t.TempDir()
	var output bytes.Buffer
	app := NewApp(&output, &output)
	app.homeDir = func() (string, error) { return home, nil }
	failures := []SweepFailure{{Operation: "root-discovery", Path: home, Error: "failed"}}
	_, err := app.addressSweepFailures(
		bufio.NewReader(strings.NewReader("e\na\ny\n")),
		SweepConfig{ConfigPath: filepath.Join(home, "config.yaml")}, SweepReport{Failures: failures},
	)
	if err == nil || !strings.Contains(err.Error(), "unsafe failure exclusion") {
		t.Fatalf("error = %v", err)
	}
}

func TestSweepTerminalRetriesFailuresAndReturnsToActions(t *testing.T) {
	fixture := newGitFixture(t)
	_, headOID := createMergedLane(t, fixture, "topic/failure-retry")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/failure-retry": {mergedSyncPR(27, "topic/failure-retry", "main", headOID)},
	})
	configPath := writeSweepTestConfig(t, fixture)
	options := SweepOptions{ConfigPath: configPath, Sort: "state", Jobs: 1, NoSizes: true}
	config, err := fixture.app.loadSweepConfig(options)
	if err != nil {
		t.Fatal(err)
	}
	report := fixture.app.buildSweepReport(context.Background(), fixture.primary, config, options)
	report.Failures = append(report.Failures, SweepFailure{
		Operation: "github-evidence", Path: fixture.primary, Error: "temporary failure",
	})
	resolver := fixture.app.resolveSweepBatch
	calls := 0
	fixture.app.resolveSweepBatch = func(ctx context.Context, requests []sweepEvidenceRequest, timeout time.Duration, progress *sweepProgress) map[string]sweepResolvedEvidence {
		calls++
		return resolver(ctx, requests, timeout, progress)
	}
	fixture.out.Reset()
	fixture.app.isTerminal = func() bool { return true }
	fixture.app.stdin = strings.NewReader("f\nr\nq\n")
	if err := fixture.app.runSweepTerminal(context.Background(), fixture.primary, config, options, &report); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(report.Failures) != 0 {
		t.Fatalf("calls=%d failures=%#v", calls, report.Failures)
	}
	if strings.Count(fixture.out.String(), "WORKTREE SWEEP") != 2 {
		t.Fatalf("retry did not render refreshed report:\n%s", fixture.out.String())
	}
}

func containsSweepPath(paths []string, target string) bool {
	for _, path := range paths {
		if path == target {
			return true
		}
	}
	return false
}
