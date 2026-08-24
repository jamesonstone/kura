package worktree

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type sweepRepository struct {
	commonDir string
	primary   string
	entries   []worktreeEntry
	roots     []string
}

func (a *App) discoverSweepRepositories(
	ctx context.Context,
	config SweepConfig,
) ([]sweepRepository, []SweepFailure) {
	roots := append([]string(nil), config.Roots...)
	var failures []SweepFailure
	for _, projectRoot := range config.ProjectRoots {
		discovered, errs := discoverClaudeRoots(projectRoot, config.ExcludeRoots)
		roots = append(roots, discovered...)
		for _, err := range errs {
			failures = append(failures, SweepFailure{Operation: "project-root-discovery", Path: projectRoot, Error: err.Error()})
		}
	}
	roots = uniqueSortedPaths(roots)
	type seed struct {
		path string
		root string
	}
	var seeds []seed
	for _, root := range roots {
		if pathExcluded(root, config.ExcludeRoots) {
			continue
		}
		paths, errs := discoverGitSeeds(root, config.ExcludeRoots)
		for _, path := range paths {
			seeds = append(seeds, seed{path: path, root: root})
		}
		for _, err := range errs {
			failures = append(failures, SweepFailure{Operation: "root-discovery", Path: root, Error: err.Error()})
		}
	}
	targets := make(map[string]*sweepRepository)
	for _, candidate := range seeds {
		commonDir, err := a.gitText(ctx, candidate.path, "rev-parse", "--path-format=absolute", "--git-common-dir")
		if err != nil {
			failures = append(failures, SweepFailure{Operation: "common-directory", Path: candidate.path, Error: err.Error()})
			continue
		}
		commonDir, _ = resolvedPath(commonDir)
		target := targets[commonDir]
		if target == nil {
			entries, listErr := a.worktrees(ctx, candidate.path)
			if listErr != nil || len(entries) == 0 {
				if listErr == nil {
					listErr = errors.New("repository has no registered worktrees")
				}
				failures = append(failures, SweepFailure{Operation: "worktree-list", Path: candidate.path, Error: listErr.Error()})
				continue
			}
			target = &sweepRepository{commonDir: commonDir, primary: filepath.Clean(entries[0].path), entries: entries}
			targets[commonDir] = target
		}
		target.roots = append(target.roots, candidate.root)
	}
	repositories := make([]sweepRepository, 0, len(targets))
	for _, target := range targets {
		target.roots = uniqueSortedPaths(target.roots)
		filtered := target.entries[:0]
		for _, entry := range target.entries {
			if pathWithinAnyRoot(entry.path, roots) && !pathExcluded(entry.path, config.ExcludeRoots) {
				filtered = append(filtered, entry)
			}
		}
		target.entries = filtered
		if len(filtered) != 0 {
			repositories = append(repositories, *target)
		}
	}
	sort.Slice(repositories, func(i, j int) bool { return repositories[i].primary < repositories[j].primary })
	return repositories, failures
}

func discoverGitSeeds(root string, excludes []string) ([]string, []error) {
	info, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, []error{err}
	}
	if !info.IsDir() {
		return nil, []error{fmt.Errorf("configured root is not a directory")}
	}
	var paths []string
	var failures []error
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", path, walkErr))
			return fs.SkipDir
		}
		if !entry.IsDir() {
			return nil
		}
		if path != root && pathExcluded(path, excludes) {
			return fs.SkipDir
		}
		gitPath := filepath.Join(path, ".git")
		gitInfo, statErr := os.Lstat(gitPath)
		if statErr == nil {
			if gitInfo.Mode().IsRegular() {
				paths = append(paths, path)
			}
			return fs.SkipDir
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			failures = append(failures, fmt.Errorf("%s: %w", gitPath, statErr))
			return fs.SkipDir
		}
		if entry.Name() == ".git" {
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		failures = append(failures, err)
	}
	sort.Strings(paths)
	return paths, failures
}

func discoverClaudeRoots(projectRoot string, excludes []string) ([]string, []error) {
	info, err := os.Stat(projectRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("configured project root is not a directory")
		}
		return nil, []error{err}
	}
	var roots []string
	var failures []error
	err = filepath.WalkDir(projectRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", path, walkErr))
			return fs.SkipDir
		}
		if !entry.IsDir() {
			return nil
		}
		if path != projectRoot && pathExcluded(path, excludes) {
			return fs.SkipDir
		}
		gitPath := filepath.Join(path, ".git")
		if _, statErr := os.Lstat(gitPath); statErr == nil {
			claudeRoot := filepath.Join(path, ".claude", "worktrees")
			if claudeInfo, claudeErr := os.Stat(claudeRoot); claudeErr == nil && claudeInfo.IsDir() {
				roots = append(roots, claudeRoot)
			}
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		failures = append(failures, err)
	}
	return uniqueSortedPaths(roots), failures
}

func pathWithinAnyRoot(path string, roots []string) bool {
	for _, root := range roots {
		if pathWithinRoot(path, root) {
			return true
		}
	}
	return false
}

func pathWithinRoot(path, root string) bool {
	pathResolved, _ := resolvedPath(path)
	rootResolved, _ := resolvedPath(root)
	relative, err := filepath.Rel(rootResolved, pathResolved)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func pathExcluded(path string, excludes []string) bool {
	for _, exclude := range excludes {
		if pathWithinRoot(path, exclude) {
			return true
		}
	}
	return false
}

func uniqueSortedPaths(paths []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(path)
		if !seen[path] {
			seen[path] = true
			result = append(result, path)
		}
	}
	sort.Strings(result)
	return result
}
