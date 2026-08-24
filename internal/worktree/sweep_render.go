package worktree

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func writeSweepJSON(writer io.Writer, report SweepReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeSweepHuman(writer io.Writer, report SweepReport, options SweepOptions, color bool) error {
	if _, err := fmt.Fprintln(writer, sweepColorize("WORKTREE SWEEP", colorBold, color)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Run: %s  Result: %s\n\n", report.RunID, report.Result); err != nil {
		return err
	}
	for _, state := range sweepStateOrder {
		candidates := sweepCandidatesForState(report.Candidates, state, options.Only)
		if len(candidates) == 0 {
			continue
		}
		var bytes int64
		for _, candidate := range candidates {
			bytes += candidate.SizeBytes
		}
		label := sweepColorize(sweepStateLabel(state), sweepStateColor(state), color)
		if _, err := fmt.Fprintf(writer, "%s  %d worktree(s)  %s\n", label, len(candidates), humanSweepBytes(bytes)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer, "  SIZE      LAST UPDATED     BRANCH       PR       REPOSITORY               PATH"); err != nil {
			return err
		}
		for _, candidate := range candidates {
			pr := "-"
			if candidate.PullRequest != nil {
				pr = fmt.Sprintf("#%d", candidate.PullRequest.Number)
			}
			if _, err := fmt.Fprintf(
				writer,
				"  %-9s %-16s %-12s %-8s %-24s %s\n",
				humanSweepBytes(candidate.SizeBytes),
				sweepUpdatedLabel(candidate),
				sanitizeTerminalField(candidate.Branch),
				pr,
				sanitizeTerminalField(candidate.Repository),
				sanitizeTerminalField(candidate.Path),
			); err != nil {
				return err
			}
			if files := sweepLocalFileSummary(candidate.Status, 48); files != "" {
				if _, err := fmt.Fprintf(writer, "    local files: %s\n", files); err != nil {
					return err
				}
			}
			if options.Verbose {
				if _, err := fmt.Fprintf(writer, "    %s: %s\n", sanitizeTerminalField(candidate.Reason), sanitizeTerminalField(candidate.Detail)); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}
	if len(report.Failures) != 0 {
		return writeSweepFailureBlock(writer, report.Failures, color)
	}
	return nil
}

func writeSweepExplanation(writer io.Writer, report SweepReport, target string, color bool) error {
	for _, candidate := range report.Candidates {
		if candidate.ID != target && candidate.Path != target {
			continue
		}
		_, err := fmt.Fprintf(
			writer,
			"%s\nID: %s\nRepository: %s\nBranch: %s\nPath: %s\nSize: %s\nLast updated: %s\nLocal files: %s\nReason: %s\nDetail: %s\nProcess: %s %s\n",
			sweepColorize(sweepStateLabel(candidate.State), sweepStateColor(candidate.State), color),
			candidate.ID,
			sanitizeTerminalField(candidate.Repository),
			sanitizeTerminalField(candidate.Branch),
			sanitizeTerminalField(candidate.Path),
			humanSweepBytes(candidate.SizeBytes),
			sweepUpdatedLabel(candidate),
			sweepLocalFileSummary(candidate.Status, 72),
			sanitizeTerminalField(candidate.Reason),
			sanitizeTerminalField(candidate.Detail),
			candidate.ProcessEvidence.State,
			sanitizeTerminalField(candidate.ProcessEvidence.Detail),
		)
		return err
	}
	return fmt.Errorf("no sweep candidate matches %q", target)
}

func sweepCandidatesForState(candidates []SweepCandidate, state SweepState, only SweepState) []SweepCandidate {
	if only != "" && only != state {
		return nil
	}
	result := make([]SweepCandidate, 0)
	for _, candidate := range candidates {
		if candidate.State == state {
			result = append(result, candidate)
		}
	}
	return result
}

func sweepUseColor(app *App, options SweepOptions) bool {
	if app.getenv("NO_COLOR") != "" || options.Color == "never" {
		return false
	}
	return options.Color == "always" || options.Color == "auto" && app.isTerminal()
}

func sweepStateColor(state SweepState) string {
	switch state {
	case SweepRemoveReady:
		return colorGreen
	case SweepMergedLocalFiles:
		return colorYellow
	case SweepMergedLocalCommits:
		return colorBrightMagenta
	case SweepProtectedActive:
		return colorBrightCyan
	case SweepStaleMetadata:
		return "\x1b[2m"
	default:
		return colorRed
	}
}

func sweepColorize(value, colorCode string, enabled bool) string {
	if !enabled {
		return value
	}
	return colorCode + value + colorReset
}

func sortedSweepStatusLines(lines []string) []string {
	result := append([]string(nil), lines...)
	sort.Strings(result)
	for index := range result {
		result[index] = strings.TrimSpace(result[index])
	}
	return result
}

func sweepOutputFile(writer io.Writer) (*os.File, bool) {
	file, ok := writer.(*os.File)
	return file, ok
}
