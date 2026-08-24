package worktree

import (
	"bufio"
	"context"
	"errors"
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
		for {
			var retry bool
			selected, retry, err = a.chooseSweepMenu(ctx, reader, report, config, options)
			if err != nil || !retry {
				break
			}
			config, err = a.loadSweepConfig(options)
			if err != nil {
				break
			}
			progress := newSweepProgress(a.errOut, true)
			*report = a.buildSweepReportWithProgress(ctx, cwd, config, options, progress)
			if err = writeSweepHuman(a.out, *report, options, sweepUseColor(a, options)); err != nil {
				break
			}
		}
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
	failureStart := len(report.Failures)
	progress := newSweepProgress(a.errOut, true)
	applyErr := a.applySweepCandidatesWithProgress(ctx, cwd, config, options, report, selected, progress)
	completionErr := writeSweepCompletion(
		a.out,
		*report,
		report.Failures[failureStart:],
		sweepUseColor(a, options),
	)
	if completionErr != nil {
		report.addFailure("write-interactive-completion", "", "", completionErr)
	}
	return errors.Join(applyErr, completionErr)
}

func (a *App) chooseSweepMenu(
	ctx context.Context,
	reader *bufio.Reader,
	report *SweepReport,
	config SweepConfig,
	options SweepOptions,
) ([]SweepCandidate, bool, error) {
	candidates := report.Candidates
	readyWT, readyMetadata := sweepMenuCounts(candidates, SweepRemoveReady, SweepStaleMetadata)
	localWT, localMetadata := sweepMenuCounts(candidates, SweepRemoveReady, SweepMergedLocalFiles, SweepStaleMetadata)
	mergedWT, _ := sweepMenuCounts(candidates, SweepRemoveReady, SweepMergedLocalFiles, SweepMergedLocalCommits)
	selectableWT, selectableMetadata := sweepMenuCounts(candidates, SweepRemoveReady, SweepMergedLocalFiles, SweepMergedLocalCommits, SweepUnproven, SweepStaleMetadata)
	stale := staleSweepCandidates(candidates, options.Only)
	staleSelectable := countSelectableSweepCandidates(stale)
	bulkStale := bulkStaleSweepCandidates(candidates, options.Only)
	for {
		prompt := fmt.Sprintf(
			"Actions:\n  [r] remove ready (%s)\n  [l] remove ready + Merged + Local Files (%s)\n  [s] review STALE (%d total, %d selectable)\n  [b] bulk-delete STALE (%d; exact review required)\n  [m] review merged (%d WT)\n  [i] selector (%s)\n  [f] address failures (%d)\n  [q] quit\nChoice: ",
			sweepMenuCountLabel(readyWT, readyMetadata),
			sweepMenuCountLabel(localWT, localMetadata),
			len(stale),
			staleSelectable,
			len(bulkStale),
			mergedWT,
			sweepMenuCountLabel(selectableWT, selectableMetadata),
			len(report.Failures),
		)
		if _, err := fmt.Fprint(a.out, prompt); err != nil {
			return nil, false, err
		}
		choice, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, false, err
		}
		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "r":
			return selectSweepStates(candidates, SweepRemoveReady, SweepStaleMetadata), false, nil
		case "d", "l":
			return selectSweepStates(candidates, SweepRemoveReady, SweepMergedLocalFiles, SweepStaleMetadata), false, nil
		case "s":
			if len(stale) == 0 {
				_, _ = fmt.Fprintln(a.out, "No STALE worktrees match the current filter.")
				continue
			}
			selected, err := a.selectSweepTerminal(ctx, stale, options)
			return selected, false, err
		case "b":
			if len(bulkStale) == 0 {
				_, _ = fmt.Fprintln(a.out, "No selectable STALE worktrees match the current filter.")
				continue
			}
			return bulkStale, false, nil
		case "m", "i":
			selected, err := a.selectSweepTerminal(ctx, candidates, options)
			return selected, false, err
		case "f":
			retry, err := a.addressSweepFailures(reader, config, *report)
			if err != nil || retry {
				return nil, retry, err
			}
		case "", "q":
			return nil, false, nil
		default:
			_, _ = fmt.Fprintln(a.out, "Choose r, l, s, b, m, i, f, or q.")
		}
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
	selection := make(sweepSelection)
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
