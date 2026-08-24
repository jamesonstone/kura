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
	targets := make([]SweepCandidate, 0, len(selected))
	for _, candidate := range selected {
		if options.Only == "" || candidate.State == options.Only {
			targets = append(targets, candidate)
		}
	}
	if len(targets) == 0 {
		return nil
	}
	progress.start(fmt.Sprintf("Revalidating %d selected target(s) in one fleet refresh", len(targets)))
	defer progress.stopLine()
	var failures []error
	refreshed := a.refreshSweepCandidates(ctx, cwd, config, options)
	approved := make([]SweepCandidate, 0, len(targets))
	for _, candidate := range targets {
		current, ok := refreshed[candidate.ID]
		if !ok {
			err := fmt.Errorf("revalidate %s: candidate is no longer present", candidate.Path)
			recordSweepPreservation(report, &failures, candidate, "revalidate-candidate", err)
			continue
		}
		if current.Snapshot != candidate.Snapshot || current.State != candidate.State {
			err := fmt.Errorf("candidate changed after review; refresh required")
			recordSweepPreservation(report, &failures, candidate, "candidate-drift", err)
			continue
		}
		if err := validateSweepAuthority(current, options.Auto); err != nil {
			recordSweepPreservation(report, &failures, current, "candidate-authority", err)
			continue
		}
		approved = append(approved, current)
	}
	progress.update("Refreshing process evidence for %d unchanged target(s)", len(approved))
	processFailures := a.revalidateSweepProcesses(ctx, config, approved)
	ready := make([]SweepCandidate, 0, len(approved))
	for _, candidate := range approved {
		if err := processFailures[candidate.ID]; err != nil {
			recordSweepPreservation(report, &failures, candidate, "candidate-process-drift", err)
			continue
		}
		ready = append(ready, candidate)
	}
	pruned := make(map[string]bool)
	for index, current := range ready {
		if current.State == SweepStaleMetadata && pruned[current.CommonDir] {
			continue
		}
		var err error
		if current.State == SweepStaleMetadata {
			progress.update("Pruning metadata %d/%d: %s", index+1, len(ready), current.Repository)
			err = a.pruneSweepMetadata(ctx, current)
			pruned[current.CommonDir] = err == nil
		} else {
			progress.update("Removing worktree %d/%d: %s", index+1, len(ready), current.Path)
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

func validateSweepAuthority(candidate SweepCandidate, automatic bool) error {
	if !candidate.Selectable {
		return fmt.Errorf("candidate is not selectable: %s", candidate.Reason)
	}
	if automatic && !candidate.AutoRemovable {
		return fmt.Errorf("candidate requires interactive confirmation")
	}
	if candidate.StaleRetirable {
		if !candidate.Stale || candidate.State != SweepUnproven {
			return fmt.Errorf("invalid STALE retirement authority")
		}
		return nil
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
	status, err := a.inspectSweepStatus(ctx, candidate.PrimaryPath, candidate.Path)
	if err != nil {
		return fmt.Errorf("reinspect local status before removal: %w", err)
	}
	if status.Fingerprint != candidate.Status.Fingerprint {
		return fmt.Errorf("local status changed after fleet revalidation; refresh required")
	}
	if candidate.StaleRetirable {
		if candidate.Branch != "" {
			branchHead, branchErr := a.gitText(ctx, repo.top, "rev-parse", "--verify", "refs/heads/"+candidate.Branch)
			if branchErr != nil || branchHead != candidate.HeadOID {
				return fmt.Errorf("local recovery branch changed or is unavailable; refresh required")
			}
		}
		return a.removeSweepStaleUnproven(ctx, repo, *entry, candidate)
	}
	if candidate.ForceBranch {
		return a.removeSweepDivergentBranch(ctx, repo, *entry, candidate)
	}
	return a.removeSweepMergedHead(ctx, repo, *entry, candidate)
}

func (a *App) removeSweepStaleUnproven(
	ctx context.Context,
	repo repository,
	entry worktreeEntry,
	candidate SweepCandidate,
) error {
	if candidate.ForceWorktree {
		_, err := a.git(ctx, repo.top, "worktree", "remove", "--force", entry.path)
		return err
	}
	removal, err := a.inspectWorktreeRemoval(ctx, repo, entry, preserveIgnoredBuildOutput)
	if err != nil {
		return err
	}
	return a.executeWorktreeRemoval(ctx, repo, removal)
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
