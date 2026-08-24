package worktree

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (a *App) resolveSweepConfigPath(override string) (string, string, error) {
	home, err := a.homeDir()
	if err != nil {
		return "", "", fmt.Errorf("determine home directory: %w", err)
	}
	configPath := strings.TrimSpace(override)
	if configPath == "" {
		configHome := strings.TrimSpace(a.getenv("XDG_CONFIG_HOME"))
		if configHome == "" {
			configHome = filepath.Join(home, ".config")
		}
		configPath = filepath.Join(configHome, "kura", "git-wt.yaml")
	}
	if !isSafeSweepInputPath(configPath) {
		return "", "", fmt.Errorf("sweep config path contains an empty value or control character")
	}
	configPath, err = expandSweepPath(home, configPath)
	if err != nil {
		return "", "", err
	}
	return home, configPath, nil
}

func compactSweepPath(home, path string) string {
	path = filepath.Clean(path)
	relative, err := filepath.Rel(home, path)
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		if relative == "." {
			return "~"
		}
		return "~/" + filepath.ToSlash(relative)
	}
	return path
}
