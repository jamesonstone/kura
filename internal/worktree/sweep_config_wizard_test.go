package worktree

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSweepConfigWizardCreatesDefaultsAndTypedPaths(t *testing.T) {
	app, output, home, configPath := newSweepWizardTestApp(t, "\n\ny\n~/future\np\nn\n\n")
	saved, err := app.runSweepConfigWizard(home, configPath)
	if err != nil || !saved {
		t.Fatalf("saved=%v err=%v\n%s", saved, err, output.String())
	}
	document, err := readSweepConfigDocument(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if document.raw.Sweep.IncludeBuiltinRoots == nil || !*document.raw.Sweep.IncludeBuiltinRoots {
		t.Fatal("built-in roots were not enabled")
	}
	if len(document.raw.Sweep.ProjectRoots) != 1 || document.raw.Sweep.ProjectRoots[0] != "~/future" {
		t.Fatalf("project roots = %#v", document.raw.Sweep.ProjectRoots)
	}
	assertSweepConfigModes(t, configPath)
	for _, expected := range []string{"Create one now?", "Include the default", "Would you like to add more paths?", "Warning:", "Configuration changes:", "Saved sweep configuration"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, output.String())
		}
	}
}

func TestSweepConfigWizardRecordsEmptyChoice(t *testing.T) {
	app, _, home, configPath := newSweepWizardTestApp(t, "\nn\nn\n\n")
	if saved, err := app.runSweepConfigWizard(home, configPath); err != nil || !saved {
		t.Fatalf("saved=%v err=%v", saved, err)
	}
	document, err := readSweepConfigDocument(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if document.raw.Sweep.IncludeBuiltinRoots == nil || *document.raw.Sweep.IncludeBuiltinRoots ||
		len(document.raw.Sweep.Roots)+len(document.raw.Sweep.ProjectRoots)+len(document.raw.Sweep.ExcludeRoots) != 0 {
		t.Fatalf("empty choice = %#v", document.raw.Sweep)
	}
}

func TestSweepConfigWizardDeclineWritesNothing(t *testing.T) {
	app, output, home, configPath := newSweepWizardTestApp(t, "n\n")
	if saved, err := app.runSweepConfigWizard(home, configPath); err != nil || saved {
		t.Fatalf("saved=%v err=%v", saved, err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("declined config exists: %v", err)
	}
	if !strings.Contains(output.String(), "Using in-memory defaults") {
		t.Fatalf("output = %s", output.String())
	}
}

func TestSweepConfigEditorPreservesCommentsAndCreatesBackup(t *testing.T) {
	app, output, home, configPath := newSweepWizardTestApp(t, "a\n~/two\nw\nv\n\n")
	original := []byte(`# heading
version: 1
sweep:
  include_builtin_roots: true # keep include
  roots:
    - ~/one # keep item
  project_roots: []
  exclude_roots: []
  process_check: disabled # keep setting
`)
	writeWizardFixture(t, configPath, original)
	saved, err := app.runSweepConfigWizard(home, configPath)
	if err != nil || !saved {
		t.Fatalf("saved=%v err=%v\n%s", saved, err, output.String())
	}
	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# heading", "# keep include", "# keep item", "# keep setting", "~/one", "~/two", "process_check: disabled"} {
		if !bytes.Contains(updated, []byte(expected)) {
			t.Fatalf("updated config missing %q:\n%s", expected, updated)
		}
	}
	backup, err := os.ReadFile(configPath + ".bak")
	if err != nil || !bytes.Equal(backup, original) {
		t.Fatalf("backup=%q err=%v", backup, err)
	}
	assertSweepConfigModes(t, configPath)
	info, err := os.Stat(configPath + ".bak")
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("backup mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestSweepConfigEditorToggleRemoveAndCancel(t *testing.T) {
	t.Run("toggle and remove", func(t *testing.T) {
		app, _, home, configPath := newSweepWizardTestApp(t, "d\nr\n1\nv\n\n")
		writeWizardFixture(t, configPath, []byte("version: 1\nsweep:\n  include_builtin_roots: true\n  roots: [~/one]\n  project_roots: []\n  exclude_roots: []\n"))
		if saved, err := app.runSweepConfigWizard(home, configPath); err != nil || !saved {
			t.Fatalf("saved=%v err=%v", saved, err)
		}
		document, err := readSweepConfigDocument(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if *document.raw.Sweep.IncludeBuiltinRoots || len(document.raw.Sweep.Roots) != 0 {
			t.Fatalf("updated sweep = %#v", document.raw.Sweep)
		}
	})
	t.Run("cancel", func(t *testing.T) {
		app, _, home, configPath := newSweepWizardTestApp(t, "q\n")
		original := []byte("version: 1\nsweep:\n  include_builtin_roots: true\n")
		writeWizardFixture(t, configPath, original)
		if saved, err := app.runSweepConfigWizard(home, configPath); err != nil || saved {
			t.Fatalf("saved=%v err=%v", saved, err)
		}
		current, _ := os.ReadFile(configPath)
		if !bytes.Equal(current, original) {
			t.Fatalf("cancel changed config: %q", current)
		}
		if _, err := os.Stat(configPath + ".bak"); !os.IsNotExist(err) {
			t.Fatalf("cancel created backup: %v", err)
		}
	})
}

func TestPromptSweepAddPathTypesAndDeduplication(t *testing.T) {
	for name, kind := range map[string]string{"worktree": "w", "project": "p", "exclude": "e"} {
		t.Run(name, func(t *testing.T) {
			app, _, home, _ := newSweepWizardTestApp(t, "~/future\n"+kind+"\n")
			raw := newSweepYAMLDefaults()
			if err := app.promptSweepAddPath(home, &raw); err != nil {
				t.Fatal(err)
			}
			app.stdin = strings.NewReader("~/future\n" + kind + "\n")
			if err := app.promptSweepAddPath(home, &raw); err != nil {
				t.Fatal(err)
			}
			var values []string
			switch kind {
			case "w":
				values = raw.Sweep.Roots
			case "p":
				values = raw.Sweep.ProjectRoots
			default:
				values = raw.Sweep.ExcludeRoots
			}
			if len(values) != 1 || values[0] != "~/future" {
				t.Fatalf("values = %#v", values)
			}
		})
	}
}

func TestSweepConfigPromptsRecoverFromInvalidInput(t *testing.T) {
	app, output, home, _ := newSweepWizardTestApp(t, "\nrelative\n~/future\nx\nw\n")
	raw := newSweepYAMLDefaults()
	if err := app.promptSweepAddPath(home, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw.Sweep.Roots) != 1 || !strings.Contains(output.String(), "Choose w, p, or e") {
		t.Fatalf("roots=%#v output=%s", raw.Sweep.Roots, output.String())
	}
	app.stdin = strings.NewReader("99\n1\n")
	if err := app.promptSweepRemovePath(&raw); err != nil {
		t.Fatal(err)
	}
	if len(raw.Sweep.Roots) != 0 || !strings.Contains(output.String(), "Choose a path number") {
		t.Fatalf("roots=%#v output=%s", raw.Sweep.Roots, output.String())
	}
}

func newSweepWizardTestApp(t *testing.T, input string) (*App, *bytes.Buffer, string, string) {
	t.Helper()
	home := t.TempDir()
	configPath := filepath.Join(home, ".config", "kura", "git-wt.yaml")
	output := &bytes.Buffer{}
	app := NewApp(output, output)
	app.stdin = strings.NewReader(input)
	app.isTerminal = func() bool { return true }
	app.homeDir = func() (string, error) { return home, nil }
	return app, output, home, configPath
}

func writeWizardFixture(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertSweepConfigModes(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode=%v err=%v", info.Mode().Perm(), err)
	}
	directory, err := os.Stat(filepath.Dir(path))
	if err != nil || directory.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode=%v err=%v", directory.Mode().Perm(), err)
	}
}
