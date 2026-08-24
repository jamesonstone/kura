package worktree

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func sweepUpdatedLabel(candidate SweepCandidate) string {
	if candidate.LastUpdated == "" {
		return "-"
	}
	updated, err := time.Parse(time.RFC3339, candidate.LastUpdated)
	if err != nil {
		return "-"
	}
	label := updated.Format("2006-01-02")
	if candidate.Stale {
		label += " STALE"
	}
	return label
}

func sweepLocalFileSummary(status SweepStatus, width int) string {
	if width <= 0 || len(status.Lines) == 0 {
		return ""
	}
	paths := make([]string, 0, len(status.Lines))
	for _, line := range status.Lines {
		if path := sweepStatusPath(line); path != "" {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return ""
	}
	parts := make([]string, 0, 3)
	for index, path := range paths {
		remaining := len(paths) - index - 1
		suffix := ""
		if remaining > 0 {
			suffix = fmt.Sprintf(" +%d", remaining)
		}
		available := width - len(strings.Join(parts, ", ")) - len(suffix)
		if len(parts) != 0 {
			available -= 2
		}
		if available < 5 {
			break
		}
		part := compactSweepFilePath(path, min(available, 24))
		parts = append(parts, part)
		joined := strings.Join(parts, ", ") + suffix
		if len(joined) >= width || len(parts) == 2 {
			break
		}
	}
	remaining := len(paths) - len(parts)
	result := strings.Join(parts, ", ")
	if remaining > 0 {
		result += fmt.Sprintf(" +%d", remaining)
	}
	return truncateTerminalLine(result, width)
}

func sweepStatusPath(line string) string {
	if len(line) < 4 {
		return ""
	}
	path := strings.TrimSpace(line[3:])
	if arrow := strings.LastIndex(path, " -> "); arrow >= 0 {
		path = path[arrow+4:]
	}
	if strings.HasPrefix(path, `"`) {
		if unquoted, err := strconv.Unquote(path); err == nil {
			path = unquoted
		}
	}
	return sanitizeTerminalField(path)
}

func compactSweepFilePath(path string, width int) string {
	path = filepath.ToSlash(path)
	if len(path) <= width {
		return path
	}
	base := filepath.Base(path)
	if len(base)+4 <= width {
		return ".../" + base
	}
	return truncateTerminalLine(base, width)
}

func sweepMenuCounts(candidates []SweepCandidate, states ...SweepState) (worktrees, metadata int) {
	for _, candidate := range selectSweepStates(candidates, states...) {
		if candidate.State == SweepStaleMetadata {
			metadata++
		} else {
			worktrees++
		}
	}
	return worktrees, metadata
}

func sweepMenuCountLabel(worktrees, metadata int) string {
	if metadata == 0 {
		return fmt.Sprintf("%d WT", worktrees)
	}
	return fmt.Sprintf("%d WT + %d metadata", worktrees, metadata)
}
