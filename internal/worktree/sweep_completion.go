package worktree

import (
	"fmt"
	"io"
	"strings"
)

func writeSweepCompletion(
	writer io.Writer,
	report SweepReport,
	failures []SweepFailure,
	color bool,
) error {
	if _, err := fmt.Fprintln(writer, sweepColorize("\nSWEEP COMPLETION", colorBold, color)); err != nil {
		return err
	}
	removed, pruned, preserved := summarizeSweepActions(report)
	if _, err := fmt.Fprintf(
		writer,
		"Removed %d worktree(s); pruned %d metadata record(s); preserved/failed %d target(s).\n",
		removed,
		pruned,
		preserved,
	); err != nil {
		return err
	}
	return writeSweepFailureBlock(writer, failures, color)
}

func writeSweepFailureBlock(writer io.Writer, failures []SweepFailure, color bool) error {
	if len(failures) == 0 {
		return nil
	}
	heading := sweepColorize(fmt.Sprintf("FAILURES (%d)", len(failures)), colorRed, color)
	if _, err := fmt.Fprintln(writer, heading); err != nil {
		return err
	}
	for _, failure := range failures {
		target := sweepFailureTarget(failure)
		if target == "" {
			if _, err := fmt.Fprintf(writer, "  %s\n", sanitizeTerminalField(failure.Operation)); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintf(
			writer,
			"  %s  %s\n",
			sanitizeTerminalField(failure.Operation),
			target,
		); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "    %s\n", sanitizeTerminalField(failure.Error)); err != nil {
			return err
		}
	}
	return nil
}

func sweepFailureTarget(failure SweepFailure) string {
	parts := make([]string, 0, 2)
	if repository := sanitizeTerminalField(failure.Repository); repository != "" {
		parts = append(parts, repository)
	}
	if path := sanitizeTerminalField(failure.Path); path != "" {
		parts = append(parts, path)
	}
	return strings.Join(parts, "  ")
}
