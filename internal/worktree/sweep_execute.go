package worktree

import (
	"context"
	"errors"
	"fmt"
)

func (a *App) applySweepCandidates(
	ctx context.Context,
	cwd string,
	config SweepConfig,
	options SweepOptions,
	report *SweepReport,
	selected []SweepCandidate,
) error {
	return a.applySweepCandidatesWithProgress(ctx, cwd, config, options, report, selected, nil)
}

func (a *App) applySweepCandidatesWithProgress(
	ctx context.Context,
	cwd string,
	config SweepConfig,
	options SweepOptions,
	report *SweepReport,
	selected []SweepCandidate,
	progress *sweepProgress,
) error {
	if len(selected) != 0 {
		progress.start(fmt.Sprintf("Revalidating 1/%d: %s", len(selected), selected[0].Path))
		defer progress.stopLine()
	}
	var failures []error
	pruned := make(map[string]bool)
	for index, candidate := range selected {
		progress.update("Revalidating %d/%d: %s", index+1, len(selected), candidate.Path)
		if options.Only != "" && candidate.State != options.Only {
			continue
		}
		if candidate.State == SweepStaleMetadata && pruned[candidate.CommonDir] {
			continue
		}
		current, err := a.refreshSweepCandidate(ctx, cwd, config, options, candidate.ID)
		if err != nil {
			actionErr := fmt.Errorf("revalidate %s: %w", candidate.Path, err)
			recordSweepAction(report, candidate, "preserved", actionErr)
			report.addFailure("revalidate-candidate", candidate.Repository, candidate.Path, actionErr)
			failures = append(failures, actionErr)
			continue
		}
		if current.Snapshot != candidate.Snapshot || current.State != candidate.State {
			actionErr := fmt.Errorf("candidate changed after review; refresh required")
			recordSweepAction(report, candidate, "preserved", actionErr)
			report.addFailure("candidate-drift", candidate.Repository, candidate.Path, actionErr)
			failures = append(failures, actionErr)
			continue
		}
		if err := validateSweepAuthority(current, options.Auto); err != nil {
			recordSweepAction(report, current, "preserved", err)
			report.addFailure("candidate-authority", current.Repository, current.Path, err)
			failures = append(failures, err)
			continue
		}
		if current.State == SweepStaleMetadata {
			progress.update("Pruning metadata %d/%d: %s", index+1, len(selected), current.Repository)
			err = a.pruneSweepMetadata(ctx, current)
			pruned[current.CommonDir] = err == nil
		} else {
			progress.update("Removing worktree %d/%d: %s", index+1, len(selected), current.Path)
			err = a.removeSweepWorktree(ctx, current)
		}
		if err != nil {
			recordSweepAction(report, current, "failed", err)
			report.addFailure("remove-candidate", current.Repository, current.Path, err)
			failures = append(failures, err)
			continue
		}
		recordSweepAction(report, current, "removed", nil)
	}
	return errors.Join(failures...)
}

func (a *App) refreshSweepCandidate(
	ctx context.Context,
	cwd string,
	config SweepConfig,
	options SweepOptions,
	id string,
) (SweepCandidate, error) {
	refreshConfig := config
	refreshConfig.Sizes = false
	refreshOptions := options
	refreshOptions.NoSizes = true
	refreshOptions.Auto = false
	refreshOptions.Interactive = false
	report := a.buildSweepReport(ctx, cwd, refreshConfig, refreshOptions)
	for _, candidate := range report.Candidates {
		if candidate.ID == id {
			return candidate, nil
		}
	}
	return SweepCandidate{}, fmt.Errorf("candidate is no longer present")
}

func validateSweepAuthority(candidate SweepCandidate, automatic bool) error {
	if !candidate.Selectable {
		return fmt.Errorf("candidate is not selectable: %s", candidate.Reason)
	}
	if automatic && !candidate.AutoRemovable {
		return fmt.Errorf("candidate requires interactive confirmation")
	}
	if candidate.State != SweepRemoveReady && candidate.State != SweepMergedLocalFiles &&
		candidate.State != SweepMergedLocalCommits && candidate.State != SweepStaleMetadata {
		return fmt.Errorf("state %s cannot be removed", candidate.State)
	}
	return nil
}

func (a *App) pruneSweepMetadata(ctx context.Context, candidate SweepCandidate) error {
	output, err := a.git(ctx, candidate.PrimaryPath, "worktree", "prune", "--verbose")
	if err != nil {
		return err
	}
	_ = output
	return nil
}

func (a *App) removeSweepWorktree(ctx context.Context, candidate SweepCandidate) error {
	repo, err := a.repository(ctx, candidate.PrimaryPath)
	if err != nil {
		return err
	}
	entry, err := a.registeredWorktree(ctx, repo.top, candidate.Path)
	if err != nil {
		return err
	}
	if entry.branch != candidate.Branch || entry.head != candidate.HeadOID {
		return fmt.Errorf("registered branch or head changed after confirmation")
	}
	if candidate.ForceBranch {
		return a.removeSweepDivergentBranch(ctx, repo, *entry, candidate)
	}
	return a.removeSweepMergedHead(ctx, repo, *entry, candidate)
}

func (a *App) removeSweepMergedHead(
	ctx context.Context,
	repo repository,
	entry worktreeEntry,
	candidate SweepCandidate,
) error {
	proofRef, err := a.prepareMergedLocalBranchDeletion(ctx, repo, entry.branch, entry.head)
	if err != nil {
		return err
	}
	cleanup := func() error {
		return a.cleanupMergedLocalBranchDeletion(context.WithoutCancel(ctx), repo, proofRef, entry.head)
	}
	if candidate.ForceWorktree {
		_, err = a.git(ctx, repo.top, "worktree", "remove", "--force", entry.path)
	} else {
		var removal worktreeRemoval
		removal, err = a.inspectWorktreeRemoval(ctx, repo, entry, preserveIgnoredBuildOutput)
		if err == nil {
			err = a.executeWorktreeRemoval(ctx, repo, removal)
		}
	}
	if err != nil {
		return errors.Join(err, cleanup())
	}
	deleteErr := a.deleteMergedLocalBranch(ctx, repo, entry.branch, entry.head, proofRef)
	return errors.Join(deleteErr, cleanup())
}

func (a *App) removeSweepDivergentBranch(
	ctx context.Context,
	repo repository,
	entry worktreeEntry,
	candidate SweepCandidate,
) error {
	if candidate.ForceWorktree {
		if _, err := a.git(ctx, repo.top, "worktree", "remove", "--force", entry.path); err != nil {
			return err
		}
	} else {
		removal, err := a.inspectWorktreeRemoval(ctx, repo, entry, preserveIgnoredBuildOutput)
		if err != nil {
			return err
		}
		if err := a.executeWorktreeRemoval(ctx, repo, removal); err != nil {
			return err
		}
	}
	actual, err := a.gitText(ctx, repo.top, "rev-parse", "--verify", "refs/heads/"+entry.branch)
	if err != nil {
		return err
	}
	if actual != entry.head {
		return fmt.Errorf("local branch moved from confirmed head %s to %s", entry.head, actual)
	}
	_, err = a.git(ctx, repo.top, "branch", "-D", "--", entry.branch)
	return err
}

func recordSweepAction(report *SweepReport, candidate SweepCandidate, status string, err error) {
	action := SweepAction{CandidateID: candidate.ID, Path: candidate.Path, Action: "remove", Status: status}
	if err != nil {
		action.Detail = err.Error()
	}
	report.Actions = append(report.Actions, action)
}

func summarizeSweepActions(report SweepReport) (removed, pruned, preserved int) {
	metadata := make(map[string]bool)
	for _, candidate := range report.Candidates {
		if candidate.State == SweepStaleMetadata {
			metadata[candidate.ID] = true
		}
	}
	for _, action := range report.Actions {
		if action.Status != "removed" {
			preserved++
			continue
		}
		if metadata[action.CandidateID] {
			pruned++
		} else {
			removed++
		}
	}
	return removed, pruned, preserved
}
