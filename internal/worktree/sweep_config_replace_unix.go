//go:build !windows

package worktree

import "os"

func replaceSweepFile(source, destination string) error {
	return os.Rename(source, destination)
}
