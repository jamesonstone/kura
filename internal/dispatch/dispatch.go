package dispatch

import (
	"path/filepath"
	"strings"
)

const GitWorktree = "git-wt"

func Alias(argv0 string) string {
	name := filepath.Base(argv0)
	name = strings.TrimSuffix(strings.ToLower(name), ".exe")
	switch name {
	case GitWorktree:
		return GitWorktree
	default:
		return ""
	}
}
