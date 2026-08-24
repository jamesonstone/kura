package worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSweepConfigBuiltinsAndYAML(t *testing.T) {
	home := t.TempDir()
	configHome := filepath.Join(home, "config")
	configPath := filepath.Join(configHome, "kura", "git-wt.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	contents := []byte(`version: 1
sweep:
  roots: [~/extra]
  project_roots: [~/src]
  exclude_roots: [~/extra/skip]
  process_check: disabled
  jobs: 2
  github_timeout: 4s
  sizes:
    enabled: true
    jobs: 3
`)
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewApp(os.Stdout, os.Stderr)
	app.homeDir = func() (string, error) { return home, nil }
	app.getenv = func(key string) string {
		if key == "XDG_CONFIG_HOME" {
			return configHome
		}
		return ""
	}
	config, err := app.loadSweepConfig(SweepOptions{Jobs: 4, Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Roots) != 5 || !samePath(config.ProjectRoots[0], filepath.Join(home, "src")) ||
		config.ProcessCheck != "disabled" || config.Jobs != 2 || config.SizeJobs != 3 || config.Timeout != 4*time.Second {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestLoadSweepConfigCanDisableBuiltins(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, "sweep.yaml")
	if err := os.WriteFile(configPath, []byte("version: 1\nsweep:\n  include_builtin_roots: false\n  roots: [~/only]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewApp(os.Stdout, os.Stderr)
	app.homeDir = func() (string, error) { return home, nil }
	config, err := app.loadSweepConfig(SweepOptions{ConfigPath: configPath, Jobs: 4, Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Roots) != 1 || !samePath(config.Roots[0], filepath.Join(home, "only")) {
		t.Fatalf("roots = %#v", config.Roots)
	}
}

func TestLoadSweepConfigRejectsMissingVersionAndUnknownFields(t *testing.T) {
	for name, contents := range map[string]string{
		"missing-version": "sweep:\n  include_builtin_roots: false\n",
		"unknown-field":   "version: 1\nsweep:\n  surprise: true\n",
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			configPath := filepath.Join(home, "git-wt.yaml")
			if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
			app := NewApp(os.Stdout, os.Stderr)
			app.homeDir = func() (string, error) { return home, nil }
			if _, err := app.loadSweepConfig(SweepOptions{ConfigPath: configPath, Jobs: 4, Timeout: 10 * time.Second}); err == nil {
				t.Fatal("expected invalid config to fail")
			}
		})
	}
}
