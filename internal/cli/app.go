package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/jamesonstone/kura/internal/catalog"
	"github.com/jamesonstone/kura/internal/install"
	"github.com/jamesonstone/kura/internal/selector"
)

const usage = `Usage: kura [command]

Run without a command to choose tools interactively.

Commands:
  list                         List available tools
  install [flags] <tool...>    Install named tools
  version                      Print the Kura version
  help                         Show this help

Install flags:
  --all                        Install every available tool
  --bin-dir <path>             Override the executable destination
  --man-dir <path>             Override the manpage destination
  --state-dir <path>           Override Kura ownership state
  --force                      Replace an unowned or modified file

Environment overrides:
  KURA_BIN_DIR, KURA_MAN_DIR, KURA_STATE_DIR`

type App struct {
	Catalog   *catalog.Catalog
	Installer install.Installer
	Selector  selector.Selector
	Out       io.Writer
	ErrOut    io.Writer
	Version   string
}

func (app *App) Run(ctx context.Context, args []string) error {
	if app.Catalog == nil || app.Installer == nil {
		return fmt.Errorf("kura is not initialized")
	}
	if app.Out == nil {
		app.Out = io.Discard
	}
	if app.ErrOut == nil {
		app.ErrOut = io.Discard
	}
	if len(args) == 0 {
		return app.runInteractive(ctx)
	}
	switch args[0] {
	case "help", "-h", "--help":
		if len(args) != 1 {
			return fmt.Errorf("help accepts no arguments")
		}
		_, err := fmt.Fprintln(app.Out, usage)
		return err
	case "version", "-v", "--version":
		if len(args) != 1 {
			return fmt.Errorf("version accepts no arguments")
		}
		_, err := fmt.Fprintf(app.Out, "kura %s\n", app.Version)
		return err
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("list accepts no arguments")
		}
		return app.list()
	case "install":
		return app.runInstall(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}
}

func (app *App) runInteractive(ctx context.Context) error {
	if app.Selector == nil {
		return fmt.Errorf("interactive selector is unavailable")
	}
	items := make([]selector.Item, 0, len(app.Catalog.Tools))
	for _, tool := range app.Catalog.Tools {
		items = append(items, selector.Item{ID: tool.ID, Name: tool.Name, Description: tool.Description})
	}
	selected, err := app.Selector.Select(ctx, items)
	if errors.Is(err, selector.ErrCanceled) {
		_, writeErr := fmt.Fprintln(app.Out, "Installation canceled; no files were changed.")
		return writeErr
	}
	if errors.Is(err, selector.ErrNotTerminal) {
		return fmt.Errorf("interactive selection requires a terminal; use 'kura list' and 'kura install <tool>'")
	}
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		_, err := fmt.Fprintln(app.Out, "No tools selected; no files were changed.")
		return err
	}
	report, err := app.Installer.Install(ctx, selected, install.Options{})
	if err != nil {
		return err
	}
	return app.printReport(report)
}

func (app *App) list() error {
	writer := tabwriter.NewWriter(app.Out, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "ID\tNAME\tDESCRIPTION"); err != nil {
		return err
	}
	for _, tool := range app.Catalog.Tools {
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\n", tool.ID, tool.Name, tool.Description); err != nil {
			return err
		}
	}
	return writer.Flush()
}
