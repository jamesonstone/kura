package worktree

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSweepConfigRejectsInvalidAndSymlinkFiles(t *testing.T) {
	t.Run("invalid YAML", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "git-wt.yaml")
		if err := os.WriteFile(path, []byte("version: 1\nsweep: [invalid\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readSweepConfigDocument(path); err == nil {
			t.Fatal("expected invalid YAML error")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "target.yaml")
		link := filepath.Join(root, "git-wt.yaml")
		if err := os.WriteFile(target, []byte("version: 1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := readSweepConfigDocument(link); err == nil || !strings.Contains(err.Error(), "non-symlink") {
			t.Fatalf("symlink error = %v", err)
		}
	})
}

func TestRenderSweepConfigPreservesSupportedNonRootSettings(t *testing.T) {
	contents := []byte(`version: 1
sweep:
  include_builtin_roots: true
  roots: []
  process_check: disabled # retained
  jobs: 8
  github_timeout: 4s
  sizes:
    enabled: false
    jobs: 2
`)
	path := filepath.Join(t.TempDir(), "git-wt.yaml")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := readSweepConfigDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	document.raw.Sweep.Roots = []string{"~/new"}
	rendered, err := renderSweepConfig(document, document.raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"process_check: disabled # retained", "jobs: 8", "github_timeout: 4s", "enabled: false", "~/new"} {
		if !bytes.Contains(rendered, []byte(expected)) {
			t.Fatalf("rendered missing %q:\n%s", expected, rendered)
		}
	}
}

func TestSweepLineDiffShowsAddedAndRemovedLines(t *testing.T) {
	diff := strings.Join(sweepLineDiff("one\ntwo\n", "one\nthree\n"), "\n")
	if !strings.Contains(diff, "  one") || !strings.Contains(diff, "- two") || !strings.Contains(diff, "+ three") {
		t.Fatalf("diff = %q", diff)
	}
}

func TestAtomicSweepConfigWriteLeavesNoTemporaryFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "git-wt.yaml")
	if err := atomicWriteSweepFile(path, []byte("version: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "git-wt.yaml" {
		t.Fatalf("entries = %#v", entries)
	}
}
