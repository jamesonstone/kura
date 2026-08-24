package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistSweepReportAndConfirmedReview(t *testing.T) {
	root := t.TempDir()
	report := SweepReport{SchemaVersion: 1, RunID: "run-1", Result: "report", Candidates: []SweepCandidate{}}
	if err := persistSweepReview(root, report); err != nil {
		t.Fatal(err)
	}
	if err := persistSweepReport(root, report); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"run-1.review.json", "run-1.json"} {
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %o", name, info.Mode().Perm())
		}
		contents, err := os.ReadFile(path)
		if err != nil || len(contents) == 0 {
			t.Fatalf("%s contents=%q err=%v", name, contents, err)
		}
	}
}
