package worktree

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type branchUpstream struct {
	remote string
	merge  string
}

func (a *App) readBranchUpstream(ctx context.Context, repo repository, branch string) (branchUpstream, error) {
	remote, err := a.gitConfigValue(ctx, repo, "branch."+branch+".remote")
	if err != nil {
		return branchUpstream{}, err
	}
	merge, err := a.gitConfigValue(ctx, repo, "branch."+branch+".merge")
	if err != nil {
		return branchUpstream{}, err
	}
	return branchUpstream{remote: remote, merge: merge}, nil
}

func (a *App) gitConfigValue(ctx context.Context, repo repository, key string) (string, error) {
	missing := "__kit_wt_sync_missing_config__"
	output, err := a.git(ctx, repo.top, "config", "--default", missing, "--get", key)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", key, err)
	}
	value := strings.TrimSpace(string(output))
	if value == missing {
		return "", nil
	}
	return value, nil
}

func (a *App) setProofBranchUpstream(ctx context.Context, repo repository, branch string) error {
	if _, err := a.git(ctx, repo.top, "config", "branch."+branch+".remote", "kit-wt-sync-proof"); err != nil {
		return fmt.Errorf("set proof branch remote: %w", err)
	}
	if _, err := a.git(ctx, repo.top, "config", "branch."+branch+".merge", "refs/heads/"+branch); err != nil {
		unsetErr := a.restoreBranchConfig(ctx, repo, "branch."+branch+".remote", "")
		return errors.Join(fmt.Errorf("set proof branch merge: %w", err), unsetErr)
	}
	return nil
}

func (a *App) restoreBranchUpstream(
	ctx context.Context,
	repo repository,
	branch string,
	original branchUpstream,
) error {
	return errors.Join(
		a.restoreBranchConfig(ctx, repo, "branch."+branch+".remote", original.remote),
		a.restoreBranchConfig(ctx, repo, "branch."+branch+".merge", original.merge),
	)
}

func (a *App) restoreBranchConfig(ctx context.Context, repo repository, key, value string) error {
	if value == "" {
		_, err := a.git(ctx, repo.top, "config", "--unset-all", key)
		if err != nil && !strings.Contains(err.Error(), "exit status 5") {
			return fmt.Errorf("unset %s: %w", key, err)
		}
		return nil
	}
	if _, err := a.git(ctx, repo.top, "config", key, value); err != nil {
		return fmt.Errorf("restore %s: %w", key, err)
	}
	return nil
}

func joinBranchDeletionError(deleteErr, restoreErr error) error {
	if restoreErr == nil {
		return deleteErr
	}
	return errors.Join(deleteErr, restoreErr)
}
