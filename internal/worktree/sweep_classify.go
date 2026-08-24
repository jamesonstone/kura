package worktree

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (a *App) classifySweepRepositories(
	ctx context.Context,
	cwd string,
	config SweepConfig,
	repositories []sweepRepository,
	report *SweepReport,
) {
	for _, target := range repositories {
		identity, defaultBranch, prs, err := a.sweepRepositoryEvidence(ctx, config, target)
		if err != nil {
			report.addFailure("github-evidence", identity, target.primary, err)
			for _, entry := range target.entries {
				report.Candidates = append(report.Candidates, newUnprovenSweepCandidate(target, entry, identity, "github-unavailable", err.Error()))
			}
			continue
		}
		for _, entry := range target.entries {
			candidate := a.classifySweepEntry(ctx, cwd, config, target, entry, identity, defaultBranch, prs[entry.branch])
			report.Candidates = append(report.Candidates, candidate)
		}
	}
}

func (a *App) sweepRepositoryEvidence(
	ctx context.Context,
	config SweepConfig,
	target sweepRepository,
) (string, string, map[string][]SyncPullRequest, error) {
	owner, name, err := a.projectIdentity(ctx, target.primary)
	identity := strings.ToLower(owner + "/" + name)
	if err != nil {
		return identity, "", nil, err
	}
	evidenceCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	defaultBranch, err := a.resolveSweepDefault(evidenceCtx, target.primary, identity)
	if err != nil {
		return identity, "", nil, err
	}
	branches := make([]string, 0, len(target.entries))
	for _, entry := range target.entries {
		if entry.branch != "" && entry.branch != defaultBranch && !entry.primary && !entry.prunable {
			branches = append(branches, entry.branch)
		}
	}
	branches = uniqueStrings(branches)
	prs := make(map[string][]SyncPullRequest)
	if len(branches) != 0 {
		prs, err = a.resolveSweepPRs(evidenceCtx, target.primary, identity, branches)
	}
	return identity, defaultBranch, prs, err
}

func (a *App) sweepDefaultBranch(ctx context.Context, cwd, repository string) (string, error) {
	output, err := a.command(ctx, cwd, "gh", "repo", "view", repository, "--json", "defaultBranchRef")
	if err != nil {
		return "", fmt.Errorf("resolve GitHub default branch: %w", err)
	}
	var payload struct {
		DefaultBranchRef struct {
			Name string `json:"name"`
		} `json:"defaultBranchRef"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return "", fmt.Errorf("decode GitHub default branch: %w", err)
	}
	if payload.DefaultBranchRef.Name == "" {
		return "", fmt.Errorf("GitHub did not report a default branch")
	}
	return payload.DefaultBranchRef.Name, nil
}

func (a *App) classifySweepEntry(
	ctx context.Context,
	cwd string,
	config SweepConfig,
	target sweepRepository,
	entry worktreeEntry,
	identity string,
	defaultBranch string,
	prs []SyncPullRequest,
) SweepCandidate {
	candidate := baseSweepCandidate(target, entry, identity, defaultBranch)
	switch {
	case entry.prunable:
		candidate.State, candidate.Reason = SweepStaleMetadata, "native-prunable-metadata"
		candidate.Selectable, candidate.AutoRemovable = true, true
	case entry.primary:
		candidate.State, candidate.Reason = SweepProtectedActive, "primary-worktree"
	case pathContains(cwd, entry.path):
		candidate.State, candidate.Reason = SweepProtectedActive, "current-worktree"
	case entry.locked:
		candidate.State, candidate.Reason, candidate.Detail = SweepProtectedActive, "locked-worktree", entry.lockReason
	case entry.branch == "":
		candidate.State, candidate.Reason = SweepUnproven, "detached-worktree"
	case entry.branch == defaultBranch:
		candidate.State, candidate.Reason = SweepProtectedActive, "default-branch-worktree"
	case len(prs) == 0:
		candidate.State, candidate.Reason = SweepUnproven, "pull-request-missing"
	case len(prs) != 1:
		candidate.State, candidate.Reason = SweepUnproven, "pull-request-ambiguous"
		candidate.Detail = fmt.Sprintf("%d pull requests use this branch", len(prs))
	default:
		candidate = a.classifyMergedSweepEntry(ctx, config, target, entry, candidate, prs[0])
	}
	populateSweepUpdated(ctx, a, target.primary, &candidate)
	candidate.ID = sweepCandidateID(candidate)
	candidate.Snapshot = sweepCandidateSnapshot(candidate)
	return candidate
}

func (a *App) classifyMergedSweepEntry(
	ctx context.Context,
	config SweepConfig,
	target sweepRepository,
	entry worktreeEntry,
	candidate SweepCandidate,
	pr SyncPullRequest,
) SweepCandidate {
	candidate.PullRequest = &pr
	if refusal := sweepPRRefusal(pr, entry, candidate.DefaultBranch); refusal != "" {
		candidate.State, candidate.Reason = SweepUnproven, refusal
		return candidate
	}
	status, err := a.inspectSweepStatus(ctx, target.primary, entry.path)
	if err != nil {
		candidate.State, candidate.Reason, candidate.Detail = SweepUnproven, "status-unavailable", err.Error()
		return candidate
	}
	candidate.Status = status
	candidate.ProcessEvidence = inspectSweepProcess(entry.path, config)
	if candidate.ProcessEvidence.State == "active" {
		candidate.State, candidate.Reason = SweepProtectedActive, "live-process"
		candidate.Detail = candidate.ProcessEvidence.Detail
		return candidate
	}
	if entry.head != pr.HeadRefOID {
		extra, extraErr := a.sweepExtraCommits(ctx, target.primary, pr.HeadRefOID, entry.head)
		if extraErr != nil {
			candidate.State, candidate.Reason, candidate.Detail = SweepUnproven, "local-history-unavailable", extraErr.Error()
			return candidate
		}
		candidate.State, candidate.Reason = SweepMergedLocalCommits, "head-oid-mismatch"
		candidate.Detail = fmt.Sprintf("local %s; merged PR head %s", entry.head, pr.HeadRefOID)
		candidate.ExtraCommits, candidate.Selectable, candidate.ForceBranch = extra, true, true
		candidate.ForceWorktree = status.Dirty()
		return candidate
	}
	if status.Dirty() {
		candidate.State, candidate.Reason = SweepMergedLocalFiles, "merged-with-local-files"
		candidate.Detail, candidate.Selectable, candidate.ForceWorktree = sweepStatusDetail(status), true, true
		return candidate
	}
	candidate.State, candidate.Reason = SweepRemoveReady, "proven-safe-merged-lane"
	candidate.Selectable, candidate.AutoRemovable = true, true
	return candidate
}

func sweepPRRefusal(pr SyncPullRequest, entry worktreeEntry, defaultBranch string) string {
	switch {
	case pr.HeadRefName != entry.branch:
		return "pull-request-head-mismatch"
	case pr.IsCrossRepository:
		return "pull-request-from-fork"
	case !strings.EqualFold(pr.State, "MERGED"):
		if strings.EqualFold(pr.State, "OPEN") {
			return "pull-request-open"
		}
		return "pull-request-not-merged"
	case pr.MergedAt == nil:
		return "pull-request-missing-merge-time"
	case pr.BaseRefName != defaultBranch:
		return "pull-request-wrong-base"
	case pr.HeadRefOID == "":
		return "pull-request-missing-head-oid"
	default:
		return ""
	}
}

func baseSweepCandidate(target sweepRepository, entry worktreeEntry, identity, defaultBranch string) SweepCandidate {
	return SweepCandidate{
		Repository: identity, CommonDir: target.commonDir, PrimaryPath: target.primary,
		Path: entry.path, Branch: entry.branch, HeadOID: entry.head, DefaultBranch: defaultBranch,
		State: SweepUnproven, Reason: "unclassified", ProcessEvidence: SweepProcessEvidence{State: "not-checked"},
	}
}

func newUnprovenSweepCandidate(target sweepRepository, entry worktreeEntry, identity, reason, detail string) SweepCandidate {
	candidate := baseSweepCandidate(target, entry, identity, "")
	candidate.Reason, candidate.Detail = reason, detail
	candidate.ID = sweepCandidateID(candidate)
	candidate.Snapshot = sweepCandidateSnapshot(candidate)
	return candidate
}

func populateSweepUpdated(ctx context.Context, app *App, cwd string, candidate *SweepCandidate) {
	if candidate.HeadOID == "" {
		return
	}
	if updated, err := app.gitText(ctx, cwd, "log", "-1", "--format=%cI", candidate.HeadOID); err == nil {
		if parsed, parseErr := time.Parse(time.RFC3339, updated); parseErr == nil {
			candidate.LastUpdated = parsed.Format(time.RFC3339)
		}
	}
}

func sweepCandidateID(candidate SweepCandidate) string {
	digest := sha256.Sum256([]byte(candidate.CommonDir + "\x00" + candidate.Path + "\x00" + candidate.Branch))
	return hex.EncodeToString(digest[:8])
}

func sweepCandidateSnapshot(candidate SweepCandidate) string {
	parts := []string{
		candidate.ID, candidate.HeadOID, candidate.DefaultBranch, string(candidate.State), candidate.Status.Fingerprint,
		candidate.ProcessEvidence.State,
	}
	if candidate.PullRequest != nil {
		parts = append(parts, fmt.Sprint(candidate.PullRequest.Number), candidate.PullRequest.HeadRefOID, candidate.PullRequest.BaseRefName)
	}
	parts = append(parts, candidate.ExtraCommits...)
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}

func pathContains(path, root string) bool {
	path, _ = resolvedPath(path)
	root, _ = resolvedPath(root)
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func sortSweepCandidates(candidates []SweepCandidate, attribute string) {
	stateRank := func(state SweepState) int {
		for index, candidate := range sweepStateOrder {
			if candidate == state {
				return index
			}
		}
		return len(sweepStateOrder)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		switch attribute {
		case "size":
			if left.SizeBytes != right.SizeBytes {
				return left.SizeBytes > right.SizeBytes
			}
		case "updated":
			if left.LastUpdated != right.LastUpdated {
				return left.LastUpdated > right.LastUpdated
			}
		case "repository":
			if left.Repository != right.Repository {
				return left.Repository < right.Repository
			}
		case "path":
			return left.Path < right.Path
		default:
			if stateRank(left.State) != stateRank(right.State) {
				return stateRank(left.State) < stateRank(right.State)
			}
		}
		return left.Repository < right.Repository || left.Repository == right.Repository && left.Path < right.Path
	})
}
