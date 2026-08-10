package selector

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

var (
	ErrCanceled    = errors.New("selection canceled")
	ErrNotTerminal = errors.New("interactive selection requires a terminal")
)

type Selector interface {
	Select(context.Context, []Item) ([]string, error)
}

type Terminal struct {
	Input  *os.File
	Output *os.File
}

func (terminal Terminal) Select(ctx context.Context, items []Item) ([]string, error) {
	if terminal.Input == nil || terminal.Output == nil ||
		!term.IsTerminal(int(terminal.Input.Fd())) || !term.IsTerminal(int(terminal.Output.Fd())) {
		return nil, ErrNotTerminal
	}
	state, err := term.MakeRaw(int(terminal.Input.Fd()))
	if err != nil {
		return nil, fmt.Errorf("enable terminal selection: %w", err)
	}
	defer func() { _ = term.Restore(int(terminal.Input.Fd()), state) }()
	if _, err := fmt.Fprint(terminal.Output, "\x1b[?1049h\x1b[?25l"); err != nil {
		return nil, fmt.Errorf("open terminal selector: %w", err)
	}
	defer func() { _, _ = fmt.Fprint(terminal.Output, "\x1b[?25h\x1b[?1049l") }()

	model := NewModel(items)
	reader := bufio.NewReader(terminal.Input)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, err := fmt.Fprintf(terminal.Output, "\x1b[H\x1b[2J%s", Render(model)); err != nil {
			return nil, fmt.Errorf("render terminal selector: %w", err)
		}
		pressed, err := readKey(reader)
		if err != nil {
			return nil, fmt.Errorf("read terminal selection: %w", err)
		}
		switch pressed {
		case keyUp:
			model.Move(-1)
		case keyDown:
			model.Move(1)
		case keyToggle:
			model.Toggle()
		case keyConfirm:
			return model.Selected(), nil
		case keyCancel:
			return nil, ErrCanceled
		}
	}
}
