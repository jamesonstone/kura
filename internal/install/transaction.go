package install

import (
	"fmt"
	"path/filepath"
)

func (service *Service) apply(plans []plannedArtifact) error {
	if len(plans) == 0 {
		return nil
	}
	writes := make([]stagedWrite, 0, len(plans))
	for _, plan := range plans {
		write, err := service.stage(plan)
		if err != nil {
			service.removeStages(writes)
			return err
		}
		writes = append(writes, write)
	}
	for index := range writes {
		if err := service.commit(&writes[index]); err != nil {
			rollbackErr := service.rollback(writes[:index+1])
			service.removeStages(writes[index+1:])
			if rollbackErr != nil {
				return fmt.Errorf("commit installation: %w; rollback: %v", err, rollbackErr)
			}
			return fmt.Errorf("commit installation: %w", err)
		}
	}
	for _, write := range writes {
		if write.backup != "" {
			if err := service.files.remove(write.backup); err != nil {
				return fmt.Errorf("installation committed but remove backup %q: %w", write.backup, err)
			}
		}
	}
	return nil
}

func (service *Service) stage(plan plannedArtifact) (stagedWrite, error) {
	directory := filepath.Dir(plan.result.Path)
	if err := service.files.mkdirAll(directory, 0o755); err != nil {
		return stagedWrite{}, fmt.Errorf("create install directory %q: %w", directory, err)
	}
	file, err := service.files.createTemp(directory, ".kura-stage-*")
	if err != nil {
		return stagedWrite{}, fmt.Errorf("stage %q: %w", plan.result.Path, err)
	}
	staged := file.Name()
	cleanup := func() { _ = service.files.remove(staged) }
	if err := file.Chmod(plan.mode); err != nil {
		_ = file.Close()
		cleanup()
		return stagedWrite{}, fmt.Errorf("set staged mode for %q: %w", plan.result.Path, err)
	}
	if _, err := file.Write(plan.content); err != nil {
		_ = file.Close()
		cleanup()
		return stagedWrite{}, fmt.Errorf("write staged file for %q: %w", plan.result.Path, err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		cleanup()
		return stagedWrite{}, fmt.Errorf("sync staged file for %q: %w", plan.result.Path, err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return stagedWrite{}, fmt.Errorf("close staged file for %q: %w", plan.result.Path, err)
	}
	return stagedWrite{destination: plan.result.Path, staged: staged, expected: plan.expected}, nil
}

func (service *Service) commit(write *stagedWrite) error {
	if err := service.verifyExpectation(write.destination, write.expected); err != nil {
		return err
	}
	if write.expected.exists {
		backup, err := service.reserveBackup(filepath.Dir(write.destination))
		if err != nil {
			return err
		}
		if err := service.files.rename(write.destination, backup); err != nil {
			return fmt.Errorf("backup %q: %w", write.destination, err)
		}
		write.backup = backup
	}
	if err := service.files.rename(write.staged, write.destination); err != nil {
		if write.backup != "" {
			if restoreErr := service.files.rename(write.backup, write.destination); restoreErr != nil {
				return fmt.Errorf("install %q: %w; immediate restore: %v", write.destination, err, restoreErr)
			}
			write.backup = ""
		}
		return fmt.Errorf("install %q: %w", write.destination, err)
	}
	write.staged = ""
	write.committed = true
	return nil
}

func (service *Service) reserveBackup(directory string) (string, error) {
	file, err := service.files.createTemp(directory, ".kura-backup-*")
	if err != nil {
		return "", fmt.Errorf("reserve backup in %q: %w", directory, err)
	}
	name := file.Name()
	if closeErr := file.Close(); closeErr != nil {
		_ = service.files.remove(name)
		return "", fmt.Errorf("close backup placeholder %q: %w", name, closeErr)
	}
	if err := service.files.remove(name); err != nil {
		return "", fmt.Errorf("release backup placeholder %q: %w", name, err)
	}
	return name, nil
}

func (service *Service) verifyExpectation(path string, expected fileExpectation) error {
	actual, err := service.inspectFile(path)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("destination %q changed after installation planning", path)
	}
	return nil
}

func (service *Service) rollback(writes []stagedWrite) error {
	var firstErr error
	for index := len(writes) - 1; index >= 0; index-- {
		write := writes[index]
		if write.committed {
			if err := service.files.remove(write.destination); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("remove %q: %w", write.destination, err)
			}
		}
		if write.backup != "" {
			if err := service.files.rename(write.backup, write.destination); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("restore %q: %w", write.destination, err)
			}
		}
		if write.staged != "" {
			_ = service.files.remove(write.staged)
		}
	}
	return firstErr
}

func (service *Service) removeStages(writes []stagedWrite) {
	for _, write := range writes {
		if write.staged != "" {
			_ = service.files.remove(write.staged)
		}
	}
}
