package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jamesonstone/kura/internal/catalog"
	"github.com/jamesonstone/kura/internal/install"
	"github.com/jamesonstone/kura/internal/selector"
)

func TestInteractiveSelectionInstallsChosenTools(t *testing.T) {
	app, installer := newTestApp(t)
	app.Selector = &fakeSelector{selected: []string{"git-wt"}}
	if err := app.Run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if strings.Join(installer.ids, ",") != "git-wt" || installer.options != (install.Options{}) {
		t.Fatalf("install call = %v %#v", installer.ids, installer.options)
	}
	if !strings.Contains(app.Out.(*bytes.Buffer).String(), "installed\tgit-wt\t/tmp/bin/git-wt") {
		t.Fatalf("output = %q", app.Out.(*bytes.Buffer))
	}
}

func TestInteractiveCancellationAndEmptySelectionDoNotInstall(t *testing.T) {
	tests := map[string]struct {
		selected []string
		err      error
	}{
		"cancel": {err: selector.ErrCanceled},
		"empty":  {selected: []string{}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			app, installer := newTestApp(t)
			app.Selector = &fakeSelector{selected: test.selected, err: test.err}
			if err := app.Run(context.Background(), nil); err != nil {
				t.Fatal(err)
			}
			if installer.calls != 0 {
				t.Fatalf("installer called %d times", installer.calls)
			}
			if !strings.Contains(app.Out.(*bytes.Buffer).String(), "no files were changed") {
				t.Fatalf("output = %q", app.Out.(*bytes.Buffer))
			}
		})
	}
}

func TestInteractiveNonTerminalProvidesNonInteractiveGuidance(t *testing.T) {
	app, _ := newTestApp(t)
	app.Selector = &fakeSelector{err: selector.ErrNotTerminal}
	err := app.Run(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "kura install <tool>") {
		t.Fatalf("error = %v", err)
	}
}

func TestListVersionAndHelp(t *testing.T) {
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"list"}, "git-wt  git wt"},
		{[]string{"version"}, "kura test-version"},
		{[]string{"--version"}, "kura test-version"},
		{[]string{"help"}, "Run without a command"},
		{[]string{"--help"}, "install [flags]"},
	} {
		app, _ := newTestApp(t)
		if err := app.Run(context.Background(), test.args); err != nil {
			t.Fatalf("Run(%v): %v", test.args, err)
		}
		if !strings.Contains(app.Out.(*bytes.Buffer).String(), test.want) {
			t.Fatalf("Run(%v) output = %q, want %q", test.args, app.Out.(*bytes.Buffer), test.want)
		}
	}
}

func TestInstallParsesNamedAndAllOptions(t *testing.T) {
	tests := []struct {
		args    []string
		options install.Options
	}{
		{[]string{"install", "git-wt"}, install.Options{}},
		{[]string{"install", "--all"}, install.Options{}},
		{[]string{"install", "--force", "--bin-dir", "/bin", "--man-dir", "/man", "--state-dir", "/state", "git-wt"}, install.Options{
			BinDir: "/bin", ManDir: "/man", StateDir: "/state", Force: true,
		}},
	}
	for _, test := range tests {
		app, installer := newTestApp(t)
		if err := app.Run(context.Background(), test.args); err != nil {
			t.Fatalf("Run(%v): %v", test.args, err)
		}
		if strings.Join(installer.ids, ",") != "git-wt" || installer.options != test.options {
			t.Fatalf("Run(%v) call = %v %#v", test.args, installer.ids, installer.options)
		}
	}
}

func TestInvalidCommandsDoNotInstall(t *testing.T) {
	for _, args := range [][]string{
		{"missing"}, {"list", "extra"}, {"version", "extra"}, {"install"},
		{"install", "--all", "git-wt"}, {"install", "--unknown"},
	} {
		app, installer := newTestApp(t)
		if err := app.Run(context.Background(), args); err == nil {
			t.Fatalf("Run(%v) unexpectedly passed", args)
		}
		if installer.calls != 0 {
			t.Fatalf("Run(%v) called installer", args)
		}
	}
}

func TestInstallHelpDoesNotInstall(t *testing.T) {
	for _, argument := range []string{"--help", "-h"} {
		app, installer := newTestApp(t)
		if err := app.Run(context.Background(), []string{"install", argument}); err != nil {
			t.Fatalf("install %s: %v", argument, err)
		}
		if installer.calls != 0 {
			t.Fatalf("install %s called installer", argument)
		}
		if !strings.Contains(app.ErrOut.(*bytes.Buffer).String(), "Usage of kura install") {
			t.Fatalf("install %s output = %q", argument, app.ErrOut.(*bytes.Buffer))
		}
	}
}

type fakeSelector struct {
	selected []string
	err      error
}

func (fake *fakeSelector) Select(context.Context, []selector.Item) ([]string, error) {
	return fake.selected, fake.err
}

type fakeInstaller struct {
	ids     []string
	options install.Options
	calls   int
}

func (installer *fakeInstaller) Install(_ context.Context, ids []string, options install.Options) (install.Report, error) {
	installer.calls++
	installer.ids = append([]string(nil), ids...)
	installer.options = options
	return install.Report{Results: []install.Result{{ToolID: "git-wt", Path: "/tmp/bin/git-wt", Status: install.StatusInstalled}}}, nil
}

func newTestApp(t *testing.T) (*App, *fakeInstaller) {
	t.Helper()
	toolCatalog, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	installer := &fakeInstaller{}
	return &App{
		Catalog: toolCatalog, Installer: installer, Selector: &fakeSelector{},
		Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}, Version: "test-version",
	}, installer
}

func TestInstallerErrorIsReturned(t *testing.T) {
	app, _ := newTestApp(t)
	app.Installer = errorInstaller{}
	if err := app.Run(context.Background(), []string{"install", "git-wt"}); !errors.Is(err, errInstall) {
		t.Fatalf("error = %v", err)
	}
}

var errInstall = errors.New("install failed")

type errorInstaller struct{}

func (errorInstaller) Install(context.Context, []string, install.Options) (install.Report, error) {
	return install.Report{}, errInstall
}
