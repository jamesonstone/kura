package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesonstone/kura/internal/catalog"
)

func TestInstallAndUnchangedReinstall(t *testing.T) {
	service, options, self := newTestService(t, "linux")
	report, err := service.Install(context.Background(), []string{"git-wt"}, options)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, report, map[string]Status{"git-wt": StatusInstalled, "git-wt.1": StatusInstalled})
	assertFile(t, filepath.Join(options.BinDir, "git-wt"), self, 0o755)
	manpage, err := service.Catalog.Content(service.Catalog.Tools[0].Artifacts[1])
	if err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(options.ManDir, "git-wt.1"), manpage, 0o644)
	assertFileMode(t, filepath.Join(options.StateDir, "installations.json"), 0o600)

	report, err = service.Install(context.Background(), []string{"git-wt"}, options)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, report, map[string]Status{"git-wt": StatusUnchanged, "git-wt.1": StatusUnchanged})
}

func TestInstallUpdatesOwnedArtifactsAndProtectsModifications(t *testing.T) {
	service, options, original := newTestService(t, "linux")
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err != nil {
		t.Fatal(err)
	}
	updated := []byte("updated kura executable")
	if err := os.WriteFile(service.ExecutablePath, updated, 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := service.Install(context.Background(), []string{"git-wt"}, options)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, report, map[string]Status{"git-wt": StatusUpdated, "git-wt.1": StatusUnchanged})
	assertFile(t, filepath.Join(options.BinDir, "git-wt"), updated, 0o755)

	target := filepath.Join(options.BinDir, "git-wt")
	if err := os.WriteFile(target, []byte("user modification"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err == nil {
		t.Fatal("modified managed command was overwritten without --force")
	}
	assertFile(t, target, []byte("user modification"), 0o755)
	if string(original) == string(updated) {
		t.Fatal("test fixture did not change executable content")
	}
}

func TestInstallRefusesCollisionUnlessForced(t *testing.T) {
	service, options, self := newTestService(t, "linux")
	if err := os.MkdirAll(options.BinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(options.BinDir, "git-wt")
	if err := os.WriteFile(target, []byte("unowned"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err == nil {
		t.Fatal("unowned command was overwritten without --force")
	}
	if _, err := os.Stat(filepath.Join(options.ManDir, "git-wt.1")); !os.IsNotExist(err) {
		t.Fatalf("manpage changed during failed preflight: %v", err)
	}
	options.Force = true
	report, err := service.Install(context.Background(), []string{"git-wt"}, options)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, report, map[string]Status{"git-wt": StatusReplaced, "git-wt.1": StatusInstalled})
	assertFile(t, target, self, 0o755)
}

func TestInstallRollsBackWhenStateCommitFails(t *testing.T) {
	service, options, original := newTestService(t, "linux")
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(options.BinDir, "git-wt")
	statePath := filepath.Join(options.StateDir, "installations.json")
	originalState, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(service.ExecutablePath, []byte("new executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	rename := service.files.rename
	failed := false
	service.files.rename = func(oldPath, newPath string) error {
		if !failed && newPath == statePath && strings.Contains(filepath.Base(oldPath), ".kura-stage-") {
			failed = true
			return errors.New("injected state commit failure")
		}
		return rename(oldPath, newPath)
	}
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err == nil {
		t.Fatal("injected state failure did not fail installation")
	}
	assertFile(t, target, original, 0o755)
	assertFile(t, statePath, originalState, 0o600)
	assertNoTemporaryFiles(t, options.BinDir)
	assertNoTemporaryFiles(t, options.StateDir)
}

func TestInstallWindowsFiltersManpageAndAddsExecutableSuffix(t *testing.T) {
	service, options, self := newTestService(t, "windows")
	report, err := service.Install(context.Background(), []string{"git-wt"}, options)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, report, map[string]Status{"git-wt.exe": StatusInstalled})
	assertFile(t, filepath.Join(options.BinDir, "git-wt.exe"), self, 0o755)
	if _, err := os.Stat(options.ManDir); !os.IsNotExist(err) {
		t.Fatalf("Windows install created man directory: %v", err)
	}
}

func TestInstallRejectsUnknownAndDuplicateTools(t *testing.T) {
	service, options, _ := newTestService(t, "linux")
	for _, ids := range [][]string{{"missing"}, {"git-wt", "git-wt"}, nil} {
		if _, err := service.Install(context.Background(), ids, options); err == nil {
			t.Fatalf("Install(%v) unexpectedly passed", ids)
		}
	}
}

func TestInstallHonorsCanceledContextBeforeMutation(t *testing.T) {
	service, options, _ := newTestService(t, "linux")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Install(ctx, []string{"git-wt"}, options); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if _, err := os.Stat(options.BinDir); !os.IsNotExist(err) {
		t.Fatalf("canceled install created bin directory: %v", err)
	}
}

func newTestService(t *testing.T, goos string) (*Service, Options, []byte) {
	t.Helper()
	toolCatalog, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	self := []byte("test kura executable")
	selfPath := filepath.Join(root, "kura")
	if err := os.WriteFile(selfPath, self, 0o755); err != nil {
		t.Fatal(err)
	}
	service := NewService(toolCatalog, selfPath, goos)
	service.environment.getenv = func(name string) string {
		if name == "PATH" {
			return filepath.Join(root, "bin")
		}
		return ""
	}
	service.environment.homeDir = func() (string, error) { return root, nil }
	options := Options{
		BinDir:   filepath.Join(root, "bin"),
		ManDir:   filepath.Join(root, "man", "man1"),
		StateDir: filepath.Join(root, "state"),
	}
	return service, options, self
}
