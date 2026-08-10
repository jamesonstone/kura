package install

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathsUsesExactDiscoverableExecutable(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "bin", "kura")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("kura"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := &Service{ExecutablePath: executable, GOOS: "linux", environment: pathEnvironment{
		getenv: func(name string) string {
			switch name {
			case "XDG_DATA_HOME":
				return filepath.Join(root, "data")
			case "XDG_STATE_HOME":
				return filepath.Join(root, "state")
			default:
				return ""
			}
		},
		homeDir:  func() (string, error) { return root, nil },
		lookPath: func(string) (string, error) { return executable, nil },
		stat:     os.Stat,
	}}
	paths, err := service.resolvePaths(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if paths.bin != filepath.Dir(executable) ||
		paths.man != filepath.Join(root, "data", "man", "man1") ||
		paths.state != filepath.Join(root, "state", "kura") {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestResolvePathsUsesDiscoverableSymlinkDirectory(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "releases", "kura")
	found := filepath.Join(root, "bin", "kura")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(found), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("kura"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(executable, found); err != nil {
		t.Fatal(err)
	}
	service := &Service{ExecutablePath: executable, GOOS: "linux", environment: pathEnvironment{
		getenv:   func(string) string { return "" },
		homeDir:  func() (string, error) { return root, nil },
		lookPath: func(string) (string, error) { return found, nil },
		stat:     os.Stat,
	}}
	paths, err := service.resolvePaths(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if paths.bin != filepath.Dir(found) {
		t.Fatalf("bin = %q, want discoverable directory %q", paths.bin, filepath.Dir(found))
	}
}

func TestResolvePathsFallsBackToUserDirectories(t *testing.T) {
	root := t.TempDir()
	service := &Service{ExecutablePath: filepath.Join(root, "download", "kura"), GOOS: "linux", environment: pathEnvironment{
		getenv:   func(string) string { return "" },
		homeDir:  func() (string, error) { return root, nil },
		lookPath: func(string) (string, error) { return "", errors.New("missing") },
		stat:     os.Stat,
	}}
	paths, err := service.resolvePaths(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if paths.bin != filepath.Join(root, ".local", "bin") ||
		paths.man != filepath.Join(root, ".local", "share", "man", "man1") ||
		paths.state != filepath.Join(root, ".local", "state", "kura") {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestResolvePathsUsesWindowsLocalAppData(t *testing.T) {
	root := t.TempDir()
	localAppData := filepath.Join(root, "LocalAppData")
	service := &Service{GOOS: "windows", environment: pathEnvironment{
		getenv: func(name string) string {
			if name == "LOCALAPPDATA" {
				return localAppData
			}
			return ""
		},
		homeDir:  func() (string, error) { return root, nil },
		lookPath: func(string) (string, error) { return "", errors.New("missing") },
		stat:     os.Stat,
	}}
	paths, err := service.resolvePaths(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if paths.bin != filepath.Join(localAppData, "Kura", "bin") ||
		paths.state != filepath.Join(localAppData, "Kura", "state") {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestResolvePathsHonorsOptionsBeforeEnvironment(t *testing.T) {
	root := t.TempDir()
	service := &Service{GOOS: "linux", environment: pathEnvironment{
		getenv:   func(name string) string { return filepath.Join(root, "environment", name) },
		homeDir:  func() (string, error) { return root, nil },
		lookPath: func(string) (string, error) { return "", errors.New("missing") },
		stat:     os.Stat,
	}}
	options := Options{BinDir: filepath.Join(root, "explicit-bin"), ManDir: filepath.Join(root, "explicit-man")}
	paths, err := service.resolvePaths(options)
	if err != nil {
		t.Fatal(err)
	}
	if paths.bin != options.BinDir || paths.man != options.ManDir ||
		paths.state != filepath.Join(root, "environment", "KURA_STATE_DIR") {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestPathContainsUsesWholeCleanEntries(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "bin")
	pathValue := filepath.Join(root, "other") + string(os.PathListSeparator) + target
	if !PathContains(target, pathValue) {
		t.Fatal("PATH entry was not found")
	}
	if PathContains(filepath.Join(root, "bi"), pathValue) {
		t.Fatal("PATH substring was treated as an entry")
	}
}
