package worktree

import (
	"context"
	"io"
	"testing"
)

func TestInspectSweepProcessUsesFleetSnapshot(t *testing.T) {
	config := SweepConfig{
		ProcessCheck: "best_effort",
		processPaths: []string{"/tmp/worktrees/project/GH-1/subdir"},
	}
	active := inspectSweepProcess("/tmp/worktrees/project/GH-1", config)
	clear := inspectSweepProcess("/tmp/worktrees/project/GH-2", config)
	if active.State != "active" || clear.State != "clear" {
		t.Fatalf("active=%#v clear=%#v", active, clear)
	}
}

func TestValidateSweepProcessEvidenceFailsClosedOnChange(t *testing.T) {
	reviewed := SweepProcessEvidence{State: "clear"}
	current := SweepProcessEvidence{State: "active", Detail: "new process"}
	if err := validateSweepProcessEvidence(reviewed, current); err == nil {
		t.Fatal("new active process was accepted")
	}
	if err := validateSweepProcessEvidence(reviewed, reviewed); err != nil {
		t.Fatalf("unchanged process evidence rejected: %v", err)
	}
}

func TestRevalidateSweepProcessesUsesOneFleetSnapshot(t *testing.T) {
	app := NewApp(io.Discard, io.Discard)
	app.lookPath = func(string) (string, error) { return "/usr/sbin/lsof", nil }
	calls := 0
	app.run = func(context.Context, string, string, ...string) ([]byte, error) {
		calls++
		return []byte{}, nil
	}
	candidates := []SweepCandidate{
		{ID: "one", Path: "/tmp/one", ProcessEvidence: SweepProcessEvidence{State: "clear"}},
		{ID: "two", Path: "/tmp/two", ProcessEvidence: SweepProcessEvidence{State: "clear"}},
	}
	failures := app.revalidateSweepProcesses(context.Background(), SweepConfig{ProcessCheck: "best_effort"}, candidates)
	if calls != 1 || len(failures) != 0 {
		t.Fatalf("calls=%d failures=%#v", calls, failures)
	}
}

func TestInspectSweepProcessReportsUnavailable(t *testing.T) {
	evidence := inspectSweepProcess("/tmp/worktree", SweepConfig{
		ProcessCheck: "best_effort",
		processError: "lsof unavailable",
	})
	if evidence.State != "unavailable" || evidence.Detail != "lsof unavailable" {
		t.Fatalf("evidence = %#v", evidence)
	}
}
