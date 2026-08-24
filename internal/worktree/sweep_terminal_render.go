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
	selected map[string]bool,
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
	selected map[string]bool,
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
		fmt.Sprintf("Sweep (%d/%d, selected %d)  j/k/arrows move  Space toggle  / filter  s sort  e explain  Enter review  q cancel", current+1, len(candidates), countSweepSelection(selected)),
		width,
	)
	header := truncateTerminalLine("    STATE                    SIZE       REPOSITORY                 BRANCH             PR     PATH", width)
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
		if selected[candidate.ID] {
			mark = "[x]"
		} else if !candidate.Selectable {
			mark = "[-]"
		}
		pr := "-"
		if candidate.PullRequest != nil {
			pr = fmt.Sprintf("#%d", candidate.PullRequest.Number)
		}
		line := fmt.Sprintf(
			"%s%s %-24s %-10s %-26s %-18s %-6s %s",
			pointer,
			mark,
			sweepStateLabel(candidate.State),
			humanSweepBytes(candidate.SizeBytes),
			candidate.Repository,
			candidate.Branch,
			pr,
			candidate.Path,
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
		detail := fmt.Sprintf("%s: %s %s", candidate.Reason, candidate.Detail, candidate.ProcessEvidence.Detail)
		for _, displayLine := range strings.Split(truncateTerminalLine(detail, width), "\n") {
			if _, err := fmt.Fprintf(output, "%s\r\n", displayLine); err != nil {
				return 0, err
			}
			lines += selectorDisplayRows(displayLine, width)
		}
	}
	return lines, nil
}

func countSweepSelection(selected map[string]bool) int {
	count := 0
	for _, active := range selected {
		if active {
			count++
		}
	}
	return count
}
