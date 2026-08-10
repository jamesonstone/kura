package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertStatuses(t *testing.T, report Report, want map[string]Status) {
	t.Helper()
	if len(report.Results) != len(want) {
		t.Fatalf("result count = %d, want %d: %#v", len(report.Results), len(want), report.Results)
	}
	for _, result := range report.Results {
		name := filepath.Base(result.Path)
		if wantStatus, ok := want[name]; !ok || result.Status != wantStatus {
			t.Fatalf("result for %q = %s, want %s (known=%t)", name, result.Status, wantStatus, ok)
		}
	}
}

func assertFile(t *testing.T, path string, want []byte, mode os.FileMode) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(want) {
		t.Fatalf("%s content = %q, want %q", path, content, want)
	}
	assertFileMode(t, path, mode)
}

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != want {
		t.Fatalf("%s mode = %#o, want %#o", path, info.Mode().Perm(), want)
	}
}

func assertNoTemporaryFiles(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".kura-stage-") || strings.HasPrefix(entry.Name(), ".kura-backup-") {
			t.Fatalf("temporary installation file remains: %s", filepath.Join(directory, entry.Name()))
		}
	}
}
