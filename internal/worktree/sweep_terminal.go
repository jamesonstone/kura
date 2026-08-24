package worktree

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func (a *App) runSweepTerminal(
	ctx context.Context,
	cwd string,
	config SweepConfig,
	options SweepOptions,
	report *SweepReport,
) error {
	if !a.isTerminal() {
		return fmt.Errorf("interactive sweep requires terminal input and output")
	}
	if err := writeSweepHuman(a.out, *report, options, sweepUseColor(a, options)); err != nil {
		return err
	}
	reader := bufio.NewReader(a.stdin)
	var selected []SweepCandidate
	var err error
	if options.Interactive {
		selected, err = a.selectSweepTerminal(ctx, report.Candidates, options)
	} else {
		selected, err = a.chooseSweepMenu(ctx, reader, report.Candidates, options)
	}
	if err != nil || len(selected) == 0 {
		return err
	}
	if err := writeSweepReview(a.out, selected, sweepUseColor(a, options)); err != nil {
		return err
	}
	if _, err := fmt.Fprint(a.out, "Press Enter to confirm these exact removals; any other input cancels: "); err != nil {
		return err
	}
	confirmation, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	if strings.TrimSpace(confirmation) != "" {
		_, _ = fmt.Fprintln(a.out, "Sweep cancelled; no worktrees were removed.")
		return nil
	}
	if err := persistSweepReview(config.StateRoot, *report); err != nil {
		return fmt.Errorf("persist confirmed sweep snapshot: %w", err)
	}
	if err := a.applySweepCandidates(ctx, cwd, config, options, report, selected); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.out, "Applied %d sweep action(s).\n", len(report.Actions))
	return err
}

func (a *App) chooseSweepMenu(
	ctx context.Context,
	reader *bufio.Reader,
	candidates []SweepCandidate,
	options SweepOptions,
) ([]SweepCandidate, error) {
	if _, err := fmt.Fprint(a.out, "[r] remove ready  [d] ready + dirty  [m] review merged  [i] selector  [q] quit: "); err != nil {
		return nil, err
	}
	choice, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "r":
		return selectSweepStates(candidates, SweepRemoveReady, SweepStaleMetadata), nil
	case "d":
		return selectSweepStates(candidates, SweepRemoveReady, SweepMergedLocalFiles, SweepStaleMetadata), nil
	case "m", "i":
		return a.selectSweepTerminal(ctx, candidates, options)
	case "", "q":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown sweep menu choice %q", strings.TrimSpace(choice))
	}
}

func (a *App) selectSweepTerminal(
	ctx context.Context,
	candidates []SweepCandidate,
	options SweepOptions,
) (selected []SweepCandidate, err error) {
	input, inputOK := a.stdin.(*os.File)
	output, outputOK := sweepOutputFile(a.out)
	if !inputOK || !outputOK {
		return nil, fmt.Errorf("interactive sweep requires file-backed terminal streams")
	}
	all := filterSweepCandidates(candidates, options.Only, "")
	if len(all) == 0 {
		return nil, nil
	}
	state, err := term.MakeRaw(int(input.Fd()))
	if err != nil {
		return nil, err
	}
	defer func() {
		if restoreErr := term.Restore(int(input.Fd()), state); err == nil && restoreErr != nil {
			err = restoreErr
		}
	}()
	if _, err := fmt.Fprint(output, hideCursor); err != nil {
		return nil, err
	}
	defer func() {
		if _, cursorErr := fmt.Fprint(output, showCursor); err == nil && cursorErr != nil {
			err = cursorErr
		}
	}()
	selection := make(map[string]bool)
	filter, filtering, explain, sortBy, current := "", false, false, options.Sort, 0
	visible := append([]SweepCandidate(nil), all...)
	selectorInput := io.Reader(input)
	if ctx.Done() != nil {
		var restoreInput func() error
		selectorInput, restoreInput, err = newContextTerminalReader(ctx, input)
		if err != nil {
			return nil, err
		}
		defer func() {
			if restoreErr := restoreInput(); err == nil && restoreErr != nil {
				err = restoreErr
			}
		}()
	}
	lineCount, err := renderSweepSelector(output, visible, current, selection, filter, sortBy, explain, sweepUseColor(a, options))
	if err != nil {
		return nil, err
	}
	defer func() { _ = clearWorktreeSelector(output, lineCount) }()
	for {
		key, readErr := readSweepKey(selectorInput)
		if readErr != nil {
			return nil, readErr
		}
		if filtering {
			filter, filtering = applySweepFilterKey(filter, key)
		} else {
			choose, cancel := updateSweepSelection(key, visible, &current, selection, &sortBy, &explain)
			if cancel {
				return nil, nil
			}
			if key.kind == sweepKeyFilter {
				filtering = true
			}
			if choose && len(selection) != 0 {
				return selectedSweepCandidates(all, selection), nil
			}
		}
		visible = filterSweepCandidates(all, options.Only, filter)
		sortSweepCandidates(visible, sortBy)
		if len(visible) == 0 {
			current = 0
		} else if current >= len(visible) {
			current = len(visible) - 1
		}
		if err := clearWorktreeSelector(output, lineCount); err != nil {
			return nil, err
		}
		lineCount, err = renderSweepSelector(output, visible, current, selection, filter, sortBy, explain, sweepUseColor(a, options))
		if err != nil {
			return nil, err
		}
	}
}
