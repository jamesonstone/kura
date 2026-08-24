package worktree

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type sweepYAML struct {
	Version int `yaml:"version"`
	Sweep   struct {
		IncludeBuiltinRoots *bool    `yaml:"include_builtin_roots"`
		Roots               []string `yaml:"roots"`
		ProjectRoots        []string `yaml:"project_roots"`
		ExcludeRoots        []string `yaml:"exclude_roots"`
		ProcessCheck        string   `yaml:"process_check"`
		Jobs                int      `yaml:"jobs"`
		GitHubTimeout       string   `yaml:"github_timeout"`
		Sizes               struct {
			Enabled *bool `yaml:"enabled"`
			Jobs    int   `yaml:"jobs"`
		} `yaml:"sizes"`
	} `yaml:"sweep"`
}

func (a *App) loadSweepConfig(options SweepOptions) (SweepConfig, error) {
	home, err := a.homeDir()
	if err != nil {
		return SweepConfig{}, fmt.Errorf("determine home directory: %w", err)
	}
	configPath := strings.TrimSpace(options.ConfigPath)
	if configPath == "" {
		configHome := strings.TrimSpace(a.getenv("XDG_CONFIG_HOME"))
		if configHome == "" {
			configHome = filepath.Join(home, ".config")
		}
		configPath = filepath.Join(configHome, "kura", "git-wt.yaml")
	}
	configPath, err = expandSweepPath(home, configPath)
	if err != nil {
		return SweepConfig{}, err
	}
	raw := sweepYAML{}
	includeBuiltins := true
	raw.Sweep.IncludeBuiltinRoots = &includeBuiltins
	sizes := true
	raw.Sweep.Sizes.Enabled = &sizes
	contents, readErr := os.ReadFile(configPath)
	if readErr == nil {
		decoder := yaml.NewDecoder(bytes.NewReader(contents))
		decoder.KnownFields(true)
		if err := decoder.Decode(&raw); err != nil {
			return SweepConfig{}, fmt.Errorf("decode sweep config %s: %w", configPath, err)
		}
		if raw.Version != 1 {
			return SweepConfig{}, fmt.Errorf("sweep config version must be 1")
		}
	} else if !errors.Is(readErr, os.ErrNotExist) || options.ConfigPath != "" {
		return SweepConfig{}, fmt.Errorf("read sweep config %s: %w", configPath, readErr)
	}
	config := SweepConfig{
		ConfigPath: configPath, ProcessCheck: "best_effort", Jobs: options.Jobs,
		Timeout: options.Timeout, Sizes: !options.NoSizes, SizeJobs: options.Jobs,
	}
	if raw.Sweep.ProcessCheck != "" {
		config.ProcessCheck = raw.Sweep.ProcessCheck
	}
	if config.ProcessCheck != "best_effort" && config.ProcessCheck != "disabled" {
		return SweepConfig{}, fmt.Errorf("process_check must be best_effort or disabled")
	}
	if raw.Sweep.Jobs > 0 && options.Jobs == 4 {
		config.Jobs = raw.Sweep.Jobs
	}
	if raw.Sweep.Sizes.Jobs > 0 {
		config.SizeJobs = raw.Sweep.Sizes.Jobs
	}
	if raw.Sweep.Sizes.Enabled != nil && !options.NoSizes {
		config.Sizes = *raw.Sweep.Sizes.Enabled
	}
	if raw.Sweep.GitHubTimeout != "" && options.Timeout == 10*time.Second {
		config.Timeout, err = time.ParseDuration(raw.Sweep.GitHubTimeout)
		if err != nil || config.Timeout <= 0 {
			return SweepConfig{}, fmt.Errorf("github_timeout must be a positive duration")
		}
	}
	if raw.Sweep.IncludeBuiltinRoots == nil || *raw.Sweep.IncludeBuiltinRoots {
		config.Roots = append(config.Roots, sweepBuiltinRoots(home)...)
	}
	config.Roots = append(config.Roots, raw.Sweep.Roots...)
	config.Roots = append(config.Roots, options.Roots...)
	config.ProjectRoots = append(raw.Sweep.ProjectRoots, options.ProjectRoots...)
	config.ExcludeRoots = append(raw.Sweep.ExcludeRoots, options.ExcludeRoots...)
	if config.Roots, err = normalizeSweepPaths(home, config.Roots); err != nil {
		return SweepConfig{}, err
	}
	if config.ProjectRoots, err = normalizeSweepPaths(home, config.ProjectRoots); err != nil {
		return SweepConfig{}, err
	}
	if config.ExcludeRoots, err = normalizeSweepPaths(home, config.ExcludeRoots); err != nil {
		return SweepConfig{}, err
	}
	stateHome := strings.TrimSpace(a.getenv("XDG_STATE_HOME"))
	if stateHome == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}
	config.StateRoot = filepath.Join(stateHome, "kura", "git-wt", "sweeps")
	return config, nil
}

func sweepBuiltinRoots(home string) []string {
	return []string{
		filepath.Join(home, "worktrees"), filepath.Join(home, ".codex", "worktrees"),
		filepath.Join(home, "Documents", "Codex"), filepath.Join(home, ".claude-worktrees"),
	}
}

func normalizeSweepPaths(home string, paths []string) ([]string, error) {
	unique := make(map[string]bool)
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		expanded, err := expandSweepPath(home, path)
		if err != nil {
			return nil, err
		}
		expanded, err = resolvedPath(expanded)
		if err != nil {
			return nil, fmt.Errorf("resolve sweep path %q: %w", path, err)
		}
		if !unique[expanded] {
			unique[expanded] = true
			result = append(result, expanded)
		}
	}
	sort.Strings(result)
	return result, nil
}

func expandSweepPath(home, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "~" {
		path = home
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("sweep path must be absolute or home-relative: %q", path)
	}
	return filepath.Clean(path), nil
}
