package worktree

import (
	"context"
	"fmt"
)

func (a *App) prepareStaleRetirableCandidate(
	ctx context.Context,
	config SweepConfig,
	target sweepRepository,
	candidate *SweepCandidate,
) {
	if !candidate.Stale || candidate.State != SweepUnproven {
		return
	}
	status, err := a.inspectSweepStatus(ctx, target.primary, candidate.Path)
	if err != nil {
		candidate.Detail = appendSweepDetail(candidate.Detail, "stale retirement status unavailable: "+err.Error())
		candidate.Snapshot = sweepCandidateSnapshot(*candidate)
		return
	}
	candidate.Status = status
	candidate.ProcessEvidence = inspectSweepProcess(candidate.Path, config)
	if candidate.ProcessEvidence.State == "active" {
		candidate.State = SweepProtectedActive
		candidate.Reason = "live-process"
		candidate.Detail = candidate.ProcessEvidence.Detail
		candidate.Snapshot = sweepCandidateSnapshot(*candidate)
		return
	}
	candidate.StaleRetirable = true
	candidate.Selectable = true
	candidate.AutoRemovable = false
	candidate.ForceWorktree = status.Dirty()
	candidate.ForceBranch = false
	candidate.Detail = appendSweepDetail(
		candidate.Detail,
		fmt.Sprintf("STALE interactive retirement; local branch preserved; %s", sweepStatusDetail(status)),
	)
	candidate.Snapshot = sweepCandidateSnapshot(*candidate)
}

func appendSweepDetail(existing, addition string) string {
	if existing == "" {
		return addition
	}
	return existing + "; " + addition
}
