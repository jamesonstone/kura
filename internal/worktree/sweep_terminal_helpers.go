package worktree

import (
	"fmt"
	"io"
	"strings"
)

func updateSweepSelection(
	key sweepKey,
	visible []SweepCandidate,
	current *int,
	selected map[string]bool,
	sortBy *string,
	explain *bool,
) (bool, bool) {
	if len(visible) == 0 && key.kind != sweepKeyCancel {
		return false, false
	}
	switch key.kind {
	case sweepKeyNext:
		*current = (*current + 1) % len(visible)
	case sweepKeyPrevious:
		*current = (*current - 1 + len(visible)) % len(visible)
	case sweepKeyToggle:
		candidate := visible[*current]
		if candidate.Selectable {
			selected[candidate.ID] = !selected[candidate.ID]
		}
	case sweepKeyAll:
		for _, candidate := range visible {
			if candidate.Selectable {
				selected[candidate.ID] = true
			}
		}
	case sweepKeyClear:
		clear(selected)
	case sweepKeySort:
		*sortBy = nextSweepSort(*sortBy)
		sortSweepCandidates(visible, *sortBy)
	case sweepKeyExplain:
		*explain = !*explain
	case sweepKeyChoose:
		return true, false
	case sweepKeyCancel:
		return false, true
	}
	return false, false
}

func applySweepFilterKey(filter string, key sweepKey) (string, bool) {
	switch key.kind {
	case sweepKeyChoose:
		return filter, false
	case sweepKeyCancel:
		return "", false
	case sweepKeyBackspace:
		if len(filter) != 0 {
			filter = filter[:len(filter)-1]
		}
	case sweepKeyRune:
		filter += string(key.char)
	}
	return filter, true
}

func filterSweepCandidates(candidates []SweepCandidate, only SweepState, filter string) []SweepCandidate {
	filter = strings.ToLower(filter)
	result := make([]SweepCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if only != "" && candidate.State != only {
			continue
		}
		haystack := strings.ToLower(candidate.Repository + " " + candidate.Branch + " " + candidate.Path + " " + candidate.Reason)
		if filter == "" || strings.Contains(haystack, filter) {
			result = append(result, candidate)
		}
	}
	return result
}

func selectSweepStates(candidates []SweepCandidate, states ...SweepState) []SweepCandidate {
	allowed := make(map[SweepState]bool)
	for _, state := range states {
		allowed[state] = true
	}
	result := make([]SweepCandidate, 0)
	for _, candidate := range candidates {
		if candidate.Selectable && allowed[candidate.State] {
			result = append(result, candidate)
		}
	}
	return result
}

func selectedSweepCandidates(candidates []SweepCandidate, selected map[string]bool) []SweepCandidate {
	result := make([]SweepCandidate, 0, len(selected))
	for _, candidate := range candidates {
		if selected[candidate.ID] {
			result = append(result, candidate)
		}
	}
	return result
}

func nextSweepSort(current string) string {
	order := []string{"state", "size", "updated", "repository", "path"}
	for index, value := range order {
		if value == current {
			return order[(index+1)%len(order)]
		}
	}
	return order[0]
}

func writeSweepReview(writer io.Writer, candidates []SweepCandidate, color bool) error {
	if _, err := fmt.Fprintln(writer, sweepColorize("\nEXACT REMOVAL REVIEW", colorBold, color)); err != nil {
		return err
	}
	var totalBytes int64
	for _, candidate := range candidates {
		totalBytes += candidate.SizeBytes
	}
	if _, err := fmt.Fprintf(writer, "Targets: %d  Approximate reclaimable space: %s\nRemote branches: untouched\n", len(candidates), humanSweepBytes(totalBytes)); err != nil {
		return err
	}
	for _, candidate := range candidates {
		if _, err := fmt.Fprintf(writer, "\n%s  %s\n  path: %s\n  branch: %s @ %s\n  size: %s\n  PR: %s\n", sweepColorize(sweepStateLabel(candidate.State), sweepStateColor(candidate.State), color), sanitizeTerminalField(candidate.Repository), sanitizeTerminalField(candidate.Path), sanitizeTerminalField(candidate.Branch), sanitizeTerminalField(candidate.HeadOID), humanSweepBytes(candidate.SizeBytes), sanitizeTerminalField(sweepPRSummary(candidate))); err != nil {
			return err
		}
		for _, line := range sortedSweepStatusLines(candidate.Status.Lines) {
			if _, err := fmt.Fprintf(writer, "  local file: %s\n", sanitizeTerminalField(line)); err != nil {
				return err
			}
		}
		for _, commit := range candidate.ExtraCommits {
			if _, err := fmt.Fprintf(writer, "  local commit: %s\n", sanitizeTerminalField(commit)); err != nil {
				return err
			}
		}
		worktreeMode, branchMode := "ordinary", "git branch -d"
		if candidate.ForceWorktree {
			worktreeMode = "git worktree remove --force"
		}
		if candidate.ForceBranch {
			branchMode = "git branch -D"
		}
		if _, err := fmt.Fprintf(writer, "  worktree removal: %s\n  local branch removal: %s\n", worktreeMode, branchMode); err != nil {
			return err
		}
		if candidate.ForceWorktree || candidate.ForceBranch {
			if _, err := fmt.Fprintln(writer, "  recovery: selected local files or commits will lose their supported recovery path"); err != nil {
				return err
			}
		}
	}
	return nil
}

func sweepPRSummary(candidate SweepCandidate) string {
	if candidate.PullRequest == nil {
		return "none"
	}
	return fmt.Sprintf("#%d %s", candidate.PullRequest.Number, candidate.PullRequest.URL)
}
