package worktree

import "testing"

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

func TestInspectSweepProcessReportsUnavailable(t *testing.T) {
	evidence := inspectSweepProcess("/tmp/worktree", SweepConfig{
		ProcessCheck: "best_effort",
		processError: "lsof unavailable",
	})
	if evidence.State != "unavailable" || evidence.Detail != "lsof unavailable" {
		t.Fatalf("evidence = %#v", evidence)
	}
}
