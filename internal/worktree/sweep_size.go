package worktree

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

func populateSweepSizes(ctx context.Context, candidates []SweepCandidate, jobs int) {
	if jobs < 1 {
		jobs = 1
	}
	type work struct{ index int }
	queue := make(chan work)
	var wait sync.WaitGroup
	for range jobs {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for item := range queue {
				if candidates[item.index].State == SweepStaleMetadata {
					continue
				}
				candidates[item.index].SizeBytes, _ = sweepDirectorySizeContext(ctx, candidates[item.index].Path)
			}
		}()
	}
	for index := range candidates {
		select {
		case queue <- work{index: index}:
		case <-ctx.Done():
			close(queue)
			wait.Wait()
			return
		}
	}
	close(queue)
	wait.Wait()
}

func sweepDirectorySize(root string) (int64, error) {
	return sweepDirectorySizeContext(context.Background(), root)
}

func sweepDirectorySizeContext(ctx context.Context, root string) (int64, error) {
	var size int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			if errors.Is(walkErr, os.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func humanSweepBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return formatSweepFloat(value, 0) + units[unit]
	}
	return formatSweepFloat(value, 1) + units[unit]
}

func formatSweepFloat(value float64, decimals int) string {
	if decimals == 0 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.1f", value)
}
