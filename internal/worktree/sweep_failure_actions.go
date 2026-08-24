package worktree

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func (a *App) addressSweepFailures(
	reader *bufio.Reader,
	config SweepConfig,
	report SweepReport,
) (bool, error) {
	failures := report.Failures
	if len(failures) == 0 {
		_, err := fmt.Fprintln(a.out, "No sweep failures require attention.")
		return false, err
	}
	if _, err := fmt.Fprintln(a.out, "\nFAILURE ACTIONS"); err != nil {
		return false, err
	}
	for index, failure := range failures {
		if _, err := fmt.Fprintf(
			a.out,
			"  %d. %s  %s\n     %s\n     hint: %s\n",
			index+1,
			sanitizeTerminalField(failure.Operation),
			sweepFailureTarget(failure),
			sanitizeTerminalField(failure.Error),
			sweepFailureHint(failure),
		); err != nil {
			return false, err
		}
	}
	for {
		if _, err := fmt.Fprint(a.out, "[r] retry now  [e] exclude exact failure paths  [q] back: "); err != nil {
			return false, err
		}
		choice, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "r":
			return true, nil
		case "e":
			return a.excludeSweepFailurePaths(reader, config, report)
		case "", "q":
			return false, nil
		default:
			_, _ = fmt.Fprintln(a.out, "Choose r, e, or q.")
		}
	}
}

func (a *App) excludeSweepFailurePaths(
	reader *bufio.Reader,
	config SweepConfig,
	report SweepReport,
) (bool, error) {
	if _, err := fmt.Fprint(a.out, "Failure numbers to exclude (comma-separated, a for all, blank cancels): "); err != nil {
		return false, err
	}
	value, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	paths, err := selectedSweepFailurePaths(value, report.Failures, report.Candidates)
	if err != nil {
		_, _ = fmt.Fprintln(a.out, sanitizeTerminalField(err.Error()))
		return false, nil
	}
	if len(paths) == 0 {
		return false, nil
	}
	home, err := a.homeDir()
	if err != nil {
		return false, err
	}
	document, err := readSweepConfigDocument(config.ConfigPath)
	if err != nil {
		return false, err
	}
	raw := document.raw
	for _, path := range paths {
		path = filepath.Clean(path)
		if !filepath.IsAbs(path) || path == string(filepath.Separator) || samePath(path, home) {
			return false, fmt.Errorf("refusing unsafe failure exclusion path %q", path)
		}
		raw.Sweep.ExcludeRoots = appendUniqueSweepPath(raw.Sweep.ExcludeRoots, compactSweepPath(home, path))
	}
	contents, err := renderSweepConfig(document, raw)
	if err != nil {
		return false, err
	}
	if document.exists && string(contents) == string(document.contents) {
		_, err = fmt.Fprintln(a.out, "Selected failure paths are already excluded.")
		return err == nil, err
	}
	if err := writeSweepConfigDiff(a.out, document.contents, contents); err != nil {
		return false, err
	}
	confirmed, err := promptSweepYesNo(reader, a.out, "Write these exact exclusions?", false)
	if err != nil || !confirmed {
		return false, err
	}
	if err := writeSweepConfig(config.ConfigPath, document, contents); err != nil {
		return false, err
	}
	_, err = fmt.Fprintf(a.out, "Saved %d failure exclusion(s); refreshing sweep.\n", len(paths))
	return err == nil, err
}

func selectedSweepFailurePaths(
	value string,
	failures []SweepFailure,
	candidates []SweepCandidate,
) ([]string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return nil, nil
	}
	selected := make(map[int]bool)
	if value == "a" {
		for index := range failures {
			selected[index] = true
		}
	} else {
		for _, part := range strings.Split(value, ",") {
			number, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || number < 1 || number > len(failures) {
				return nil, fmt.Errorf("choose failure numbers from 1 through %d", len(failures))
			}
			selected[number-1] = true
		}
	}
	paths := make([]string, 0, len(selected))
	seen := make(map[string]bool)
	for index := range failures {
		path := strings.TrimSpace(failures[index].Path)
		if selected[index] && path != "" && !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
		if !selected[index] || failures[index].Operation != "github-evidence" {
			continue
		}
		for _, candidate := range candidates {
			candidatePath := strings.TrimSpace(candidate.Path)
			if filepath.Clean(candidate.PrimaryPath) == filepath.Clean(path) && candidatePath != "" && !seen[candidatePath] {
				seen[candidatePath] = true
				paths = append(paths, candidatePath)
			}
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("selected failures do not have addressable paths")
	}
	return paths, nil
}

func sweepFailureHint(failure SweepFailure) string {
	switch failure.Operation {
	case "common-directory":
		return "exclude an intentionally abandoned marker, or repair its .git pointer outside sweep"
	case "github-evidence":
		return "retry after fixing GitHub access/origin identity, or exclude a retired repository"
	default:
		return "retry after correcting the reported path or repository state"
	}
}
