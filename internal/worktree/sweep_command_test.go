package worktree

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSweepCommandEmitsVersionedJSON(t *testing.T) {
	fixture := newGitFixture(t)
	_, headOID := createMergedLane(t, fixture, "topic/command")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/command": {mergedSyncPR(12, "topic/command", "main", headOID)},
	})
	configPath := writeSweepTestConfig(t, fixture)
	fixture.out.Reset()
	err := fixture.app.Run(context.Background(), fixture.primary, []string{
		"sweep", "--dry-run", "--json", "--config", configPath, "--no-sizes",
	})
	if err != nil {
		t.Fatal(err)
	}
	var report SweepReport
	if err := json.Unmarshal(fixture.out.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, fixture.out.String())
	}
	if report.SchemaVersion != 1 || findSweepCandidate(t, report, "topic/command").State != SweepRemoveReady {
		t.Fatalf("report = %#v", report)
	}
}

func TestSweepCommandAutoIsImmediatelyIdempotent(t *testing.T) {
	fixture := newGitFixture(t)
	path, headOID := createMergedLane(t, fixture, "topic/command-auto")
	configureSweepEvidence(fixture, map[string][]SyncPullRequest{
		"topic/command-auto": {mergedSyncPR(13, "topic/command-auto", "main", headOID)},
	})
	configPath := writeSweepTestConfig(t, fixture)
	args := []string{"sweep", "--auto", "--json", "--config", configPath, "--no-sizes"}
	if err := fixture.app.Run(context.Background(), fixture.primary, args); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("auto path remains: %v", err)
	}
	fixture.out.Reset()
	if err := fixture.app.Run(context.Background(), fixture.primary, args); err != nil {
		t.Fatal(err)
	}
	var report SweepReport
	if err := json.Unmarshal(fixture.out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Actions) != 0 {
		t.Fatalf("second run actions = %#v", report.Actions)
	}
}

func writeSweepTestConfig(t *testing.T, fixture gitFixture) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "git-wt.yaml")
	contents := "version: 1\nsweep:\n  include_builtin_roots: false\n  roots:\n    - " + fixture.worktreeRoot + "\n  process_check: disabled\n  sizes:\n    enabled: false\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	stateRoot := t.TempDir()
	fixture.app.getenv = func(key string) string {
		if key == "XDG_STATE_HOME" {
			return stateRoot
		}
		return ""
	}
	return path
}
