package worktree

import (
	"context"
	"testing"
	"time"
)

func sweepTestConfig(t *testing.T, fixture gitFixture) SweepConfig {
	t.Helper()
	return SweepConfig{
		Roots:        []string{fixture.worktreeRoot},
		ProcessCheck: "disabled",
		Jobs:         1,
		Timeout:      5 * time.Second,
		Sizes:        false,
		SizeJobs:     1,
		StateRoot:    t.TempDir(),
	}
}

func configureSweepEvidence(
	fixture gitFixture,
	prs map[string][]SyncPullRequest,
) {
	fixture.app.resolveSyncPRs = staticSyncPRs(prs)
	fixture.app.resolveSweepBatch = func(
		_ context.Context,
		requests []sweepEvidenceRequest,
		_ time.Duration,
		_ *sweepProgress,
	) map[string]sweepResolvedEvidence {
		resolved := make(map[string]sweepResolvedEvidence, len(requests))
		for _, request := range requests {
			byBranch := make(map[string][]SyncPullRequest, len(request.branches))
			for _, branch := range request.branches {
				byBranch[branch] = append([]SyncPullRequest(nil), prs[branch]...)
			}
			resolved[request.identity] = sweepResolvedEvidence{
				identity: request.identity, defaultBranch: "main", pullRequests: byBranch,
			}
		}
		return resolved
	}
}

func buildSweepTestReport(
	t *testing.T,
	fixture gitFixture,
	config SweepConfig,
) SweepReport {
	t.Helper()
	return fixture.app.buildSweepReport(
		context.Background(),
		fixture.primary,
		config,
		SweepOptions{Sort: "state", Jobs: 1, Timeout: 5 * time.Second, NoSizes: true},
	)
}

func findSweepCandidate(t *testing.T, report SweepReport, branch string) SweepCandidate {
	t.Helper()
	for _, candidate := range report.Candidates {
		if candidate.Branch == branch {
			return candidate
		}
	}
	t.Fatalf("no sweep candidate for %s: %#v", branch, report.Candidates)
	return SweepCandidate{}
}
