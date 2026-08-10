package install

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallForceReplacesSymlinkWithoutChangingItsTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires platform-specific privileges on Windows")
	}
	service, options, self := newTestService(t, "linux")
	if err := os.MkdirAll(options.BinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "external")
	if err := os.WriteFile(external, []byte("external content"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(options.BinDir, "git-wt")
	if err := os.Symlink(external, target); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err == nil {
		t.Fatal("symlink destination was replaced without --force")
	}
	options.Force = true
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err != nil {
		t.Fatal(err)
	}
	assertFile(t, target, self, 0o755)
	assertFile(t, external, []byte("external content"), 0o644)
}

func TestInstallNeverReplacesDirectory(t *testing.T) {
	service, options, _ := newTestService(t, "linux")
	target := filepath.Join(options.BinDir, "git-wt")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	options.Force = true
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err == nil {
		t.Fatal("directory destination was replaced with --force")
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatalf("directory destination changed: info=%v err=%v", info, err)
	}
}

func TestInstallRejectsSymlinkedState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires platform-specific privileges on Windows")
	}
	service, options, _ := newTestService(t, "linux")
	if err := os.MkdirAll(options.StateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "state.json")
	content := []byte(`{"schema_version":1,"files":{}}`)
	if err := os.WriteFile(external, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(options.StateDir, "installations.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err == nil {
		t.Fatal("symlinked Kura state was accepted")
	}
	assertFile(t, external, content, 0o600)
	if _, err := os.Stat(filepath.Join(options.BinDir, "git-wt")); !os.IsNotExist(err) {
		t.Fatalf("failed state preflight installed a command: %v", err)
	}
}

func TestInstallTreatsModeChangeAsLocalModification(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode modification is not meaningful on Windows")
	}
	service, options, _ := newTestService(t, "linux")
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(options.BinDir, "git-wt")
	if err := os.Chmod(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Install(context.Background(), []string{"git-wt"}, options); err == nil {
		t.Fatal("locally changed mode was treated as unchanged")
	}
	assertFileMode(t, target, 0o700)
}
