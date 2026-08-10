// kura installs embedded host commands and dispatches installed executable aliases.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/jamesonstone/kura/internal/catalog"
	"github.com/jamesonstone/kura/internal/cli"
	"github.com/jamesonstone/kura/internal/dispatch"
	"github.com/jamesonstone/kura/internal/install"
	"github.com/jamesonstone/kura/internal/selector"
	"github.com/jamesonstone/kura/internal/worktree"
)

var version = "dev"

func main() {
	ctx := context.Background()
	if dispatch.Alias(os.Args[0]) == dispatch.GitWorktree {
		runGitWorktree(ctx)
		return
	}
	toolCatalog, err := catalog.Default()
	if err != nil {
		fatal("kura: load catalog", err)
	}
	executable, err := os.Executable()
	if err != nil {
		fatal("kura: locate executable", err)
	}
	app := &cli.App{
		Catalog:   toolCatalog,
		Installer: install.NewService(toolCatalog, executable, runtime.GOOS),
		Selector:  selector.Terminal{Input: os.Stdin, Output: os.Stdout},
		Out:       os.Stdout,
		ErrOut:    os.Stderr,
		Version:   resolvedVersion(),
	}
	if err := app.Run(ctx, os.Args[1:]); err != nil {
		fatal("", err)
	}
}

func resolvedVersion() string {
	if version != "" && version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func runGitWorktree(ctx context.Context) {
	cwd, err := os.Getwd()
	if err != nil {
		fatal("git wt: determine current directory", err)
	}
	app := worktree.NewApp(os.Stdout, os.Stderr)
	if err := app.Run(ctx, cwd, os.Args[1:]); err != nil {
		fatal("git wt", err)
	}
}

func fatal(prefix string, err error) {
	if prefix == "" {
		fmt.Fprintf(os.Stderr, "kura: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "%s: %v\n", prefix, err)
	}
	os.Exit(1)
}
