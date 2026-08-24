package worktree

import (
	"bytes"
	"strings"
	"testing"
)

func TestSweepHumanAndJSONUseSameReport(t *testing.T) {
	report := SweepReport{
		SchemaVersion: 1,
		RunID:         "run-1",
		Result:        "report",
		Candidates: []SweepCandidate{
			{ID: "one", Repository: "example/project", Path: "/tmp/one", Branch: "GH-1", State: SweepRemoveReady, SizeBytes: 2048},
			{ID: "two", Repository: "example/project", Path: "/tmp/two", Branch: "GH-2", State: SweepMergedLocalFiles, SizeBytes: 1024, LastUpdated: "2026-05-01T12:00:00Z", Stale: true, Status: SweepStatus{Lines: []string{"?? local.txt"}}},
		},
	}
	var human, machine bytes.Buffer
	if err := writeSweepHuman(&human, report, SweepOptions{}, false); err != nil {
		t.Fatal(err)
	}
	if err := writeSweepJSON(&machine, report); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"REMOVE READY", "MERGED + LOCAL FILES", "LAST UPDATED", "2026-05-01 STALE", "local files: local.txt", "/tmp/one", "/tmp/two"} {
		if !strings.Contains(human.String(), value) {
			t.Fatalf("human output missing %q:\n%s", value, human.String())
		}
	}
	for _, value := range []string{`"schema_version": 1`, `"id": "one"`, `"id": "two"`} {
		if !strings.Contains(machine.String(), value) {
			t.Fatalf("JSON missing %q:\n%s", value, machine.String())
		}
	}
}

func TestRenderSweepSelectorShowsSelectionAndBlockedRows(t *testing.T) {
	candidates := []SweepCandidate{
		{ID: "one", Repository: "example/project", Path: "/tmp/one", Branch: "GH-1", State: SweepRemoveReady, Selectable: true, SizeBytes: 1024, LastUpdated: "2026-05-01T12:00:00Z", Stale: true, Status: SweepStatus{Lines: []string{"?? local.txt"}}},
		{ID: "two", Repository: "example/project", Path: "/tmp/two", Branch: "main", State: SweepProtectedActive},
	}
	var output bytes.Buffer
	selected := make(sweepSelection)
	selected.add(candidates[0])
	lines, err := renderSweepSelectorAtSize(&output, candidates, 0, selected, "", "state", true, false, 120, 20)
	if err != nil {
		t.Fatal(err)
	}
	if lines < 5 || !strings.Contains(output.String(), "[x]") || !strings.Contains(output.String(), "[-]") || !strings.Contains(output.String(), "REMOVE READY") || !strings.Contains(output.String(), "2026-05-01 STALE") || !strings.Contains(output.String(), "local.txt") {
		t.Fatalf("selector output:\n%s", output.String())
	}
	if strings.Contains(output.String(), "\x1b[") {
		t.Fatalf("no-color selector emitted ANSI: %q", output.String())
	}
}

func TestSweepKeyBindings(t *testing.T) {
	for input, want := range map[string]sweepKeyKind{
		"j": sweepKeyNext, "k": sweepKeyPrevious, " ": sweepKeyToggle,
		"a": sweepKeyAll, "u": sweepKeyClear, "s": sweepKeySort,
		"e": sweepKeyExplain, "/": sweepKeyFilter, "\n": sweepKeyChoose,
	} {
		key, err := readSweepKey(strings.NewReader(input))
		if err != nil || key.kind != want {
			t.Fatalf("key %q = %#v, %v; want %v", input, key, err, want)
		}
	}
}

func TestUpdateSweepSelectionNeverSelectsBlockedCandidate(t *testing.T) {
	visible := []SweepCandidate{
		{ID: "blocked", State: SweepProtectedActive},
		{ID: "ready", State: SweepRemoveReady, Selectable: true},
	}
	selected := make(sweepSelection)
	current, sortBy, explain := 0, "state", false
	updateSweepSelection(sweepKey{kind: sweepKeyAll}, visible, &current, selected, &sortBy, &explain)
	if selected.contains(visible[0]) || !selected.contains(visible[1]) {
		t.Fatalf("selection = %#v", selected)
	}
}

func TestSpaceTogglePreservesMultiplePathQualifiedSelections(t *testing.T) {
	visible := []SweepCandidate{
		{ID: "shared", Path: "/tmp/one", Selectable: true},
		{ID: "shared", Path: "/tmp/two", Selectable: true},
		{ID: "shared", Path: "/tmp/three", Selectable: true},
	}
	selected := make(sweepSelection)
	current, sortBy, explain := 0, "state", false
	for index := range visible {
		current = index
		updateSweepSelection(sweepKey{kind: sweepKeyToggle}, visible, &current, selected, &sortBy, &explain)
	}
	if countSweepSelection(selected) != 3 || len(selectedSweepCandidates(visible, selected)) != 3 {
		t.Fatalf("selection = %#v", selected)
	}
	current = 1
	updateSweepSelection(sweepKey{kind: sweepKeyToggle}, visible, &current, selected, &sortBy, &explain)
	if countSweepSelection(selected) != 2 || selected.contains(visible[1]) {
		t.Fatalf("selection after deselect = %#v", selected)
	}
}

func TestSweepHumanSanitizesControlCharacters(t *testing.T) {
	report := SweepReport{RunID: "run", Result: "report", Candidates: []SweepCandidate{{
		Repository: "example/project\x1b[31m", Path: "/tmp/lane\x1b[2J", Branch: "GH-1",
		State: SweepRemoveReady,
	}}}
	var output bytes.Buffer
	if err := writeSweepHuman(&output, report, SweepOptions{}, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "\x1b[") {
		t.Fatalf("human output retained control characters: %q", output.String())
	}
}

func TestSweepOutputReportHonorsOnlyFilter(t *testing.T) {
	report := SweepReport{Candidates: []SweepCandidate{
		{ID: "ready", State: SweepRemoveReady},
		{ID: "dirty", State: SweepMergedLocalFiles},
	}}
	projected := sweepOutputReport(report, SweepOptions{Only: SweepRemoveReady})
	if len(projected.Candidates) != 1 || projected.Candidates[0].ID != "ready" {
		t.Fatalf("projected = %#v", projected.Candidates)
	}
	if len(report.Candidates) != 2 {
		t.Fatal("projection mutated the source report")
	}
}
