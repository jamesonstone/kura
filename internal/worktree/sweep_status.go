package worktree

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

func (a *App) inspectSweepStatus(
	ctx context.Context,
	primary string,
	path string,
) (SweepStatus, error) {
	statusText, err := a.status(ctx, path, true)
	if err != nil {
		return SweepStatus{}, err
	}
	if links, linkErr := inspectManagedEnvironmentLinks(primary, path); linkErr == nil {
		statusText = statusWithoutManagedEnvironmentLinks(statusText, links)
	}
	status := parseSweepStatus(statusText)
	return status, nil
}

func parseSweepStatus(statusText string) SweepStatus {
	status := SweepStatus{}
	for _, line := range strings.Split(strings.TrimSpace(statusText), "\n") {
		if line == "" {
			continue
		}
		status.Lines = append(status.Lines, line)
		if strings.HasPrefix(line, "!! ") {
			status.Ignored++
			continue
		}
		if strings.HasPrefix(line, "?? ") {
			status.Untracked++
			continue
		}
		if len(line) < 2 {
			status.Tracked++
			continue
		}
		if line[0] != ' ' {
			status.Staged++
		}
		if line[0] != ' ' || line[1] != ' ' {
			status.Tracked++
		}
		if strings.Contains(line[:2], "m") || strings.Contains(line[:2], "?") {
			status.Submodules++
		}
	}
	sort.Strings(status.Lines)
	if len(status.Lines) != 0 {
		digest := sha256.Sum256([]byte(strings.Join(status.Lines, "\n")))
		status.Fingerprint = hex.EncodeToString(digest[:])
	}
	return status
}

func (a *App) sweepExtraCommits(
	ctx context.Context,
	cwd string,
	mergedHead string,
	localHead string,
) ([]string, error) {
	if _, err := a.git(ctx, cwd, "cat-file", "-e", mergedHead+"^{commit}"); err != nil {
		return nil, fmt.Errorf("merged PR head is unavailable locally: %w", err)
	}
	output, err := a.gitText(
		ctx,
		cwd,
		"log",
		"--format=%H%x09%s",
		"--no-decorate",
		mergedHead+".."+localHead,
	)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(output, "\n"), nil
}

func sweepStatusDetail(status SweepStatus) string {
	return fmt.Sprintf(
		"tracked=%d staged=%d untracked=%d ignored=%d submodules=%d",
		status.Tracked,
		status.Staged,
		status.Untracked,
		status.Ignored,
		status.Submodules,
	)
}
