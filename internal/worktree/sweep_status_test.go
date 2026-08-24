package worktree

import "testing"

func TestParseSweepStatusCategoriesAndFingerprint(t *testing.T) {
	status := parseSweepStatus("A  staged.txt\n M tracked.txt\n?? untracked.txt\n!! ignored.txt")
	if status.Staged != 1 || status.Tracked != 2 || status.Untracked != 1 || status.Ignored != 1 {
		t.Fatalf("status = %#v", status)
	}
	if status.Fingerprint == "" || !status.Dirty() {
		t.Fatalf("status lacks dirty fingerprint: %#v", status)
	}
	second := parseSweepStatus("!! ignored.txt\n?? untracked.txt\n M tracked.txt\nA  staged.txt")
	if second.Fingerprint != status.Fingerprint {
		t.Fatalf("fingerprint is not order-independent: %s != %s", second.Fingerprint, status.Fingerprint)
	}
}

func TestParseSweepStatusClean(t *testing.T) {
	status := parseSweepStatus("")
	if status.Dirty() || status.Fingerprint != "" {
		t.Fatalf("clean status = %#v", status)
	}
}
