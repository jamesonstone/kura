package worktree

import (
	"context"
	"fmt"
)

func (a *App) refreshSweepCandidates(
	ctx context.Context,
	cwd string,
	config SweepConfig,
	options SweepOptions,
) map[string]SweepCandidate {
	refreshConfig := config
	refreshConfig.Sizes = false
	refreshOptions := options
	refreshOptions.NoSizes = true
	refreshOptions.Auto = false
	refreshOptions.Interactive = false
	refreshed := a.buildSweepReport(ctx, cwd, refreshConfig, refreshOptions)
	byID := make(map[string]SweepCandidate, len(refreshed.Candidates))
	for _, candidate := range refreshed.Candidates {
		byID[candidate.ID] = candidate
	}
	return byID
}

func (a *App) revalidateSweepProcesses(
	ctx context.Context,
	config SweepConfig,
	candidates []SweepCandidate,
) map[string]error {
	result := make(map[string]error)
	worktrees := make([]SweepCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.State != SweepStaleMetadata {
			worktrees = append(worktrees, candidate)
		}
	}
	if len(worktrees) == 0 {
		return result
	}
	currentConfig := config
	currentConfig.processPaths = nil
	currentConfig.processError = ""
	a.populateSweepProcessSnapshot(ctx, &currentConfig)
	for _, candidate := range worktrees {
		current := inspectSweepProcess(candidate.Path, currentConfig)
		if err := validateSweepProcessEvidence(candidate.ProcessEvidence, current); err != nil {
			result[candidate.ID] = err
		}
	}
	return result
}

func validateSweepProcessEvidence(reviewed, current SweepProcessEvidence) error {
	if current.State == "active" {
		return fmt.Errorf("worktree became active after fleet revalidation: %s", current.Detail)
	}
	if current.State != reviewed.State {
		return fmt.Errorf("process evidence changed from %s to %s; refresh required", reviewed.State, current.State)
	}
	return nil
}

func recordSweepPreservation(
	report *SweepReport,
	failures *[]error,
	candidate SweepCandidate,
	operation string,
	err error,
) {
	recordSweepAction(report, candidate, "preserved", err)
	report.addFailure(operation, candidate.Repository, candidate.Path, err)
	*failures = append(*failures, err)
}
