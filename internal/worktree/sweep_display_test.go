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
	selected, retry, err := app.chooseSweepMenu(context.Background(), bufio.NewReader(strings.NewReader("r\n")), &report, SweepConfig{}, SweepOptions{})
	if err != nil || retry || len(selected) != 3 {
		t.Fatalf("selected=%#v retry=%t err=%v", selected, retry, err)
	}
	if selected[0].ID != "ready" || selected[1].ID != "files" || selected[2].ID != "metadata" {
		t.Fatalf("combined selection = %#v", selected)
	}
	prompt := output.String()
	if strings.Contains(strings.ToLower(prompt), "dirty") ||
		!strings.Contains(prompt, "[r] remove ready + Merged + Local Files + STALE (2 WT + 1 metadata)") ||
		strings.Contains(prompt, "[l] ") || strings.Contains(prompt, "[s] ") ||
		strings.Contains(prompt, "[b] ") || strings.Contains(prompt, "[m] ") {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestSweepMenuRejectsRetiredKeysAndEmptyRemoval(t *testing.T) {
	candidates := []SweepCandidate{
		{ID: "commits", State: SweepMergedLocalCommits, Selectable: true},
	}
	var output bytes.Buffer
	app := NewApp(&output, &output)
	report := SweepReport{Candidates: candidates}
	selected, retry, err := app.chooseSweepMenu(
		context.Background(),
		bufio.NewReader(strings.NewReader("l\ns\nb\nm\nr\nq\n")),
		&report,
		SweepConfig{},
		SweepOptions{},
	)
	if err != nil || retry || len(selected) != 0 {
		t.Fatalf("selected=%#v retry=%t err=%v", selected, retry, err)
	}
	prompt := output.String()
	if strings.Count(prompt, "Choose r, i, f, or q.") != 4 {
		t.Fatalf("retired-key help count mismatch:\n%s", prompt)
	}
	if !strings.Contains(prompt, "No eligible worktrees to remove.") {
		t.Fatalf("empty [r] did not continue:\n%s", prompt)
	}
}

func TestDefaultSweepRemovalCandidatesHonorOnly(t *testing.T) {
	candidates := []SweepCandidate{
		{ID: "ready", State: SweepRemoveReady, Selectable: true},
		{ID: "files", State: SweepMergedLocalFiles, Selectable: true},
		{ID: "unproven", State: SweepUnproven, Stale: true, StaleRetirable: true, Selectable: true},
		{ID: "commits", State: SweepMergedLocalCommits, Stale: true, Selectable: true},
	}
	got := defaultSweepRemovalCandidates(candidates, SweepMergedLocalFiles)
	if len(got) != 1 || got[0].ID != "files" {
		t.Fatalf("only merged-local-files = %#v", got)
	}
	got = defaultSweepRemovalCandidates(candidates, SweepUnproven)
	if len(got) != 1 || got[0].ID != "unproven" {
		t.Fatalf("only unproven stale = %#v", got)
	}
	got = defaultSweepRemovalCandidates(candidates, "")
	if len(got) != 3 || got[0].ID != "ready" || got[1].ID != "files" || got[2].ID != "unproven" {
		t.Fatalf("default removal leaked merged-local-commits: %#v", got)
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
