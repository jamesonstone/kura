package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/jamesonstone/kura/internal/install"
)

func (app *App) runInstall(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("kura install", flag.ContinueOnError)
	flags.SetOutput(app.ErrOut)
	all := flags.Bool("all", false, "install every available tool")
	binDir := flags.String("bin-dir", "", "override executable destination")
	manDir := flags.String("man-dir", "", "override manpage destination")
	stateDir := flags.String("state-dir", "", "override Kura ownership state")
	force := flags.Bool("force", false, "replace an unowned or modified file")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	toolIDs := flags.Args()
	if *all {
		if len(toolIDs) != 0 {
			return fmt.Errorf("--all cannot be combined with named tools")
		}
		for _, tool := range app.Catalog.Tools {
			toolIDs = append(toolIDs, tool.ID)
		}
	}
	if len(toolIDs) == 0 {
		return fmt.Errorf("install requires at least one tool or --all")
	}
	report, err := app.Installer.Install(ctx, toolIDs, install.Options{
		BinDir:   *binDir,
		ManDir:   *manDir,
		StateDir: *stateDir,
		Force:    *force,
	})
	if err != nil {
		return err
	}
	return app.printReport(report)
}

func (app *App) printReport(report install.Report) error {
	for _, result := range report.Results {
		if _, err := fmt.Fprintf(app.Out, "%s\t%s\t%s\n", result.Status, result.ToolID, result.Path); err != nil {
			return err
		}
	}
	for _, warning := range report.Warnings {
		if _, err := fmt.Fprintf(app.ErrOut, "warning: %s\n", warning); err != nil {
			return err
		}
	}
	return nil
}
