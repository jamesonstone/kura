package worktree

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestSweepStatusWorkInProgressIgnoresIgnoredOnly(t *testing.T) {
	if (SweepStatus{Ignored: 1, Lines: []string{"!! bin/out"}}).WorkInProgress() {
		t.Fatal("ignored-only status is not work in progress")
	}
	if !(SweepStatus{Untracked: 1, Lines: []string{"?? local.txt"}}).WorkInProgress() {
		t.Fatal("untracked status is work in progress")
	}
}

func TestSweepStaleBoundaryUsesFourteenDays(t *testing.T) {
	now := time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC)
	if sweepWorktreeIsStale(now.AddDate(0, 0, -sweepStaleAgeDays), now) {
		t.Fatal("exactly 14 days old must not be stale")
	}
	if !sweepWorktreeIsStale(now.AddDate(0, 0, -sweepStaleAgeDays).Add(-time.Second), now) {
		t.Fatal("older than 14 days must be stale")
	}
}

func TestSweepLocalFileSummaryIsCompactAndSanitized(t *testing.T) {
	status := SweepStatus{Lines: []string{
		"?? internal/worktree/very-long-generated-name.go",
		" M README.md",
		"R  old.txt -> docs/new-name.md",
		"!! bin/output",
	}}
	summary := sweepLocalFileSummary(status, 32)
	if len(summary) > 32 || !strings.Contains(summary, "+3") || strings.Contains(summary, "old.txt") {
		t.Fatalf("summary = %q", summary)
	}
	if strings.Contains(sweepLocalFileSummary(SweepStatus{Lines: []string{"?? bad\x1b[31m.txt"}}, 32), "\x1b") {
		t.Fatal("summary retained terminal control characters")
	}
}

func TestSweepMenuUsesStateTerminologyAndCounts(t *testing.T) {
	candidates := []SweepCandidate{
		{ID: "ready", State: SweepRemoveReady, Selectable: true},
		{ID: "files", State: SweepMergedLocalFiles, Selectable: true},
		{ID: "commits", State: SweepMergedLocalCommits, Selectable: true},
		{ID: "metadata", State: SweepStaleMetadata, Selectable: true},
	}
	var output bytes.Buffer
	app := NewApp(&output, &output)
	report := SweepReport{Candidates: candidates}
	selected, retry, err := app.chooseSweepMenu(context.Background(), bufio.NewReader(strings.NewReader("l\n")), &report, SweepConfig{}, SweepOptions{})
	if err != nil || retry || len(selected) != 3 {
		t.Fatalf("selected=%#v retry=%t err=%v", selected, retry, err)
	}
	prompt := output.String()
	if strings.Contains(strings.ToLower(prompt), "dirty") || !strings.Contains(prompt, "Merged + Local Files") || !strings.Contains(prompt, "2 WT + 1 metadata") {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestSummarizeSweepActionsSeparatesWorktreesAndMetadata(t *testing.T) {
	report := SweepReport{
		Candidates: []SweepCandidate{
			{ID: "worktree", SizeBytes: 2 * 1024 * 1024},
			{ID: "metadata", State: SweepStaleMetadata, SizeBytes: 999},
			{ID: "failed", SizeBytes: 8 * 1024 * 1024},
		},
		Actions: []SweepAction{
			{CandidateID: "worktree", Action: "remove", Status: "removed"},
			{CandidateID: "metadata", Action: "remove", Status: "removed"},
			{CandidateID: "failed", Action: "remove", Status: "preserved"},
		},
	}
	removed, pruned, preserved, reclaimed := summarizeSweepActions(report)
	if removed != 1 || pruned != 1 || preserved != 1 || reclaimed != 2*1024*1024 {
		t.Fatalf("summary = %d %d %d %d", removed, pruned, preserved, reclaimed)
	}
}
