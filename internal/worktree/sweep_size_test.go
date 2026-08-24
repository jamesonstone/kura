package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSweepDirectorySizeDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "local.bin"), make([]byte, 1024), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(external, "large.bin"), make([]byte, 8192), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "external")); err != nil {
		t.Fatal(err)
	}
	size, err := sweepDirectorySize(root)
	if err != nil {
		t.Fatal(err)
	}
	if size != 1024 {
		t.Fatalf("size = %d, want 1024", size)
	}
}

func TestHumanSweepBytes(t *testing.T) {
	for value, want := range map[int64]string{0: "0B", 1024: "1.0KB", 1024 * 1024: "1.0MB"} {
		if got := humanSweepBytes(value); got != want {
			t.Fatalf("humanSweepBytes(%d) = %q, want %q", value, got, want)
		}
	}
}
