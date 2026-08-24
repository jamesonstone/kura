package worktree

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func persistSweepReport(stateRoot string, report SweepReport) error {
	return persistSweepReportNamed(stateRoot, report, "")
}

func persistSweepReview(stateRoot string, report SweepReport) error {
	return persistSweepReportNamed(stateRoot, report, ".review")
}

func persistSweepReportNamed(stateRoot string, report SweepReport, suffix string) error {
	if err := os.MkdirAll(stateRoot, 0o700); err != nil {
		return fmt.Errorf("create sweep state directory: %w", err)
	}
	contents, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode sweep report: %w", err)
	}
	contents = append(contents, '\n')
	temporary, err := os.CreateTemp(stateRoot, ".sweep-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary sweep report: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(contents); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	destination := filepath.Join(stateRoot, report.RunID+suffix+".json")
	if err := os.Rename(temporaryPath, destination); err != nil {
		return fmt.Errorf("commit sweep report: %w", err)
	}
	return nil
}
