package worktree

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

func (a *App) populateSweepProcessSnapshot(ctx context.Context, config *SweepConfig) {
	if config.ProcessCheck == "disabled" {
		return
	}
	if _, err := a.lookPath("lsof"); err != nil {
		config.processError = "lsof is not installed"
		return
	}
	processCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	output, err := a.run(processCtx, os.TempDir(), "lsof", "-a", "-d", "cwd", "-Fn")
	if processCtx.Err() != nil {
		config.processError = "process inspection timed out"
		return
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, "n/") {
			continue
		}
		path := strings.TrimPrefix(line, "n")
		path, _ = resolvedPath(path)
		if !seen[path] {
			seen[path] = true
			config.processPaths = append(config.processPaths, path)
		}
	}
	sort.Strings(config.processPaths)
	if err != nil && len(config.processPaths) == 0 {
		config.processError = fmt.Sprintf("process inspection failed: %v", err)
	}
}

func inspectSweepProcess(path string, config SweepConfig) SweepProcessEvidence {
	if config.ProcessCheck == "disabled" {
		return SweepProcessEvidence{State: "disabled"}
	}
	if config.processError != "" {
		return SweepProcessEvidence{State: "unavailable", Detail: config.processError}
	}
	for _, processPath := range config.processPaths {
		if pathContains(processPath, path) {
			return SweepProcessEvidence{
				State:  "active",
				Detail: "a live process has its current directory inside this worktree",
			}
		}
	}
	return SweepProcessEvidence{State: "clear"}
}
