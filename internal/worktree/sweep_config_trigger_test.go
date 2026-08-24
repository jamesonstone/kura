package worktree

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBareSweepFirstRunCreatesConfigAndContinues(t *testing.T) {
	home := t.TempDir()
	app, output := newSweepTriggerApp(t, home, "\n\nn\n\nq\n", true)
	if err := app.Run(context.Background(), home, []string{"sweep"}); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, ".config", "kura", "git-wt.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Create one now?", "Saved sweep configuration", "WORKTREE SWEEP", "[r] remove ready"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, output.String())
		}
	}
}

func TestBareSweepDeclineContinuesWithoutConfig(t *testing.T) {
	home := t.TempDir()
	app, output := newSweepTriggerApp(t, home, "n\nq\n", true)
	if err := app.Run(context.Background(), home, []string{"sweep"}); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, ".config", "kura", "git-wt.yaml")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("declined config exists: %v", err)
	}
	if !strings.Contains(output.String(), "Using in-memory defaults") || !strings.Contains(output.String(), "WORKTREE SWEEP") {
		t.Fatalf("output = %s", output.String())
	}
}

func TestNonInteractiveSweepNeverPromptsOrCreatesConfig(t *testing.T) {
	home := t.TempDir()
	app, output := newSweepTriggerApp(t, home, "", true)
	if err := app.Run(context.Background(), home, []string{"sweep", "--dry-run", "--no-sizes"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "Create one now?") {
		t.Fatalf("dry run prompted:\n%s", output.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "kura", "git-wt.yaml")); !os.IsNotExist(err) {
		t.Fatalf("dry run created config: %v", err)
	}
}

func TestSweepConfigCommandManagesExplicitPath(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, "custom", "sweep.yaml")
	app, _ := newSweepTriggerApp(t, home, "\nn\nn\n\n", true)
	if err := app.Run(context.Background(), home, []string{"sweep", "config", "--config", configPath}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestSweepConfigCommandRequiresTTY(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, "custom.yaml")
	app, _ := newSweepTriggerApp(t, home, "", false)
	err := app.Run(context.Background(), home, []string{"sweep", "config", "--config", configPath})
	if err == nil || !strings.Contains(err.Error(), "requires terminal") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(configPath); !os.IsNotExist(statErr) {
		t.Fatalf("non-TTY command created config: %v", statErr)
	}
}

func TestSweepConfigCommandRejectsEmptyOrDuplicateConfig(t *testing.T) {
	home := t.TempDir()
	for _, args := range [][]string{
		{"sweep", "config", "--config="},
		{"sweep", "config", "--config", "/one", "--config", "/two"},
	} {
		app, _ := newSweepTriggerApp(t, home, "", true)
		if err := app.Run(context.Background(), home, args); err == nil {
			t.Fatalf("expected %v to fail", args)
		}
	}
}

func TestOperationalExplicitMissingConfigRemainsError(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, "missing.yaml")
	app, _ := newSweepTriggerApp(t, home, "", true)
	err := app.Run(context.Background(), home, []string{"sweep", "--dry-run", "--config", configPath})
	if err == nil || !strings.Contains(err.Error(), "file does not exist") {
		t.Fatalf("error = %v", err)
	}
}

func newSweepTriggerApp(t *testing.T, home, input string, terminal bool) (*App, *bytes.Buffer) {
	t.Helper()
	output := &bytes.Buffer{}
	app := NewApp(output, output)
	app.stdin = strings.NewReader(input)
	app.isTerminal = func() bool { return terminal }
	app.homeDir = func() (string, error) { return home, nil }
	app.lookPath = func(string) (string, error) { return "", errors.New("unavailable in test") }
	app.getenv = func(key string) string {
		if key == "XDG_STATE_HOME" {
			return filepath.Join(home, "state")
		}
		return ""
	}
	return app, output
}
