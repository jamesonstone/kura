package worktree

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func renderSweepSelector(
	output *os.File,
	candidates []SweepCandidate,
	current int,
	selected sweepSelection,
	filter string,
	sortBy string,
	explain bool,
	color bool,
) (int, error) {
	width, height := selectorDefaultWidth, len(candidates)+5
	if terminalWidth, terminalHeight, err := term.GetSize(int(output.Fd())); err == nil {
		if terminalWidth > 0 {
			width = terminalWidth
		}
		if terminalHeight > 5 {
			height = terminalHeight
		}
	}
	return renderSweepSelectorAtSize(output, candidates, current, selected, filter, sortBy, explain, color, width, height)
}

func renderSweepSelectorAtSize(
	output io.Writer,
	candidates []SweepCandidate,
	current int,
	selected sweepSelection,
	filter string,
	sortBy string,
	explain bool,
	color bool,
	width int,
	height int,
) (int, error) {
	visibleCount := min(len(candidates), max(height-5, 1))
	start := max(current-visibleCount+1, 0)
	if start+visibleCount > len(candidates) {
		start = max(len(candidates)-visibleCount, 0)
	}
	end := min(start+visibleCount, len(candidates))
	title := truncateTerminalLine(
		fmt.Sprintf("Sweep (%d/%d, selected %d)  j/k/arrows move  Space toggle  a all  u clear  / filter  s sort  e explain  Enter review  q cancel", current+1, len(candidates), countSweepSelection(selected)),
		width,
	)
	header := truncateTerminalLine("    STATE                  SIZE     LAST UPDATED     REPOSITORY           BRANCH         PR    LOCAL FILES / PATH", width)
	if _, err := fmt.Fprintf(output, "%s\r\n%s\r\n", sweepColorize(title, colorBold, color), sweepColorize(header, colorBold, color)); err != nil {
		return 0, err
	}
	lines := selectorDisplayRows(title, width) + selectorDisplayRows(header, width)
	for index := start; index < end; index++ {
		candidate := candidates[index]
		pointer, mark := " ", "[ ]"
		if index == current {
			pointer = ">"
		}
		if selected.contains(candidate) {
			mark = "[x]"
		} else if !candidate.Selectable {
			mark = "[-]"
		}
		pr := "-"
		if candidate.PullRequest != nil {
			pr = fmt.Sprintf("#%d", candidate.PullRequest.Number)
		}
		pathOrFiles := candidate.Path
		if files := sweepLocalFileSummary(candidate.Status, 28); files != "" {
			pathOrFiles = files + " | " + candidate.Path
		}
		line := fmt.Sprintf(
			"%s%s %-22s %-8s %-16s %-20s %-14s %-5s %s",
			pointer,
			mark,
			sweepStateLabel(candidate.State),
			humanSweepBytes(candidate.SizeBytes),
			sweepUpdatedLabel(candidate),
			candidate.Repository,
			candidate.Branch,
			pr,
			pathOrFiles,
		)
		line = truncateTerminalLine(sanitizeTerminalField(line), width)
		if _, err := fmt.Fprintf(output, "%s\r\n", sweepColorize(line, sweepStateColor(candidate.State), color)); err != nil {
			return 0, err
		}
		lines += selectorDisplayRows(line, width)
	}
	footer := fmt.Sprintf("filter=%q sort=%s", filter, sortBy)
	if _, err := fmt.Fprintf(output, "%s\r\n", truncateTerminalLine(footer, width)); err != nil {
		return 0, err
	}
	lines += selectorDisplayRows(footer, width)
	if explain && len(candidates) != 0 {
		candidate := candidates[current]
		detail := fmt.Sprintf("%s: %s %s; updated=%s; local=%s", candidate.Reason, candidate.Detail, candidate.ProcessEvidence.Detail, sweepUpdatedLabel(candidate), sweepLocalFileSummary(candidate.Status, 40))
		for _, displayLine := range strings.Split(truncateTerminalLine(detail, width), "\n") {
			if _, err := fmt.Fprintf(output, "%s\r\n", displayLine); err != nil {
				return 0, err
			}
			lines += selectorDisplayRows(displayLine, width)
		}
	}
	return lines, nil
}

func countSweepSelection(selected sweepSelection) int {
	return len(selected)
}
