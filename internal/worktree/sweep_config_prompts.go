package worktree

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

func (a *App) promptSweepAddPath(home string, raw *sweepYAML) error {
	var expanded string
	for {
		if _, err := fmt.Fprint(a.out, "Path (absolute or ~/...): "); err != nil {
			return err
		}
		value, err := readSweepPromptLine(a.stdin)
		if err != nil {
			return err
		}
		if !isSafeSweepInputPath(value) {
			_, _ = fmt.Fprintln(a.out, "Enter a non-empty absolute or ~/ path without control characters.")
			continue
		}
		expanded, err = expandSweepPath(home, value)
		if err != nil {
			_, _ = fmt.Fprintln(a.out, sanitizeTerminalField(err.Error()))
			continue
		}
		if info, statErr := os.Stat(expanded); errors.Is(statErr, os.ErrNotExist) {
			if _, err := fmt.Fprintf(a.out, "Warning: %s does not exist yet; it will be retained for future discovery.\n", sanitizeTerminalField(expanded)); err != nil {
				return err
			}
		} else if statErr != nil {
			return fmt.Errorf("inspect path %s: %w", expanded, statErr)
		} else if !info.IsDir() {
			_, _ = fmt.Fprintf(a.out, "Path is not a directory: %s\n", sanitizeTerminalField(expanded))
			continue
		}
		break
	}
	stored := compactSweepPath(home, expanded)
	for {
		if _, err := fmt.Fprint(a.out, "Path type: [w] worktree pool  [p] project root  [e] excluded subtree (default w): "); err != nil {
			return err
		}
		kind, err := readSweepPromptLine(a.stdin)
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(kind)) {
		case "", "w":
			raw.Sweep.Roots = appendUniqueSweepPath(raw.Sweep.Roots, stored)
			return nil
		case "p":
			raw.Sweep.ProjectRoots = appendUniqueSweepPath(raw.Sweep.ProjectRoots, stored)
			return nil
		case "e":
			raw.Sweep.ExcludeRoots = appendUniqueSweepPath(raw.Sweep.ExcludeRoots, stored)
			return nil
		default:
			_, _ = fmt.Fprintln(a.out, "Choose w, p, or e.")
		}
	}
}

func (a *App) promptSweepRemovePath(raw *sweepYAML) error {
	type entry struct {
		kind  string
		index int
		path  string
	}
	var entries []entry
	for index, path := range raw.Sweep.Roots {
		entries = append(entries, entry{"worktree pool", index, path})
	}
	for index, path := range raw.Sweep.ProjectRoots {
		entries = append(entries, entry{"project root", index, path})
	}
	for index, path := range raw.Sweep.ExcludeRoots {
		entries = append(entries, entry{"excluded subtree", index, path})
	}
	if len(entries) == 0 {
		_, err := fmt.Fprintln(a.out, "No configured paths are available to remove.")
		return err
	}
	for index, candidate := range entries {
		if _, err := fmt.Fprintf(a.out, "  %d. %-18s %s\n", index+1, candidate.kind, candidate.path); err != nil {
			return err
		}
	}
	selected := entry{}
	for {
		if _, err := fmt.Fprint(a.out, "Remove path number (blank cancels): "); err != nil {
			return err
		}
		value, err := readSweepPromptLine(a.stdin)
		if err != nil || strings.TrimSpace(value) == "" {
			return err
		}
		number, parseErr := strconv.Atoi(strings.TrimSpace(value))
		if parseErr != nil || number < 1 || number > len(entries) {
			_, _ = fmt.Fprintf(a.out, "Choose a path number from 1 through %d.\n", len(entries))
			continue
		}
		selected = entries[number-1]
		break
	}
	switch selected.kind {
	case "worktree pool":
		raw.Sweep.Roots = removeSweepPath(raw.Sweep.Roots, selected.index)
	case "project root":
		raw.Sweep.ProjectRoots = removeSweepPath(raw.Sweep.ProjectRoots, selected.index)
	default:
		raw.Sweep.ExcludeRoots = removeSweepPath(raw.Sweep.ExcludeRoots, selected.index)
	}
	return nil
}

func (a *App) confirmSweepConfigWrite(
	configPath string,
	document sweepConfigDocument,
	raw sweepYAML,
) (bool, error) {
	contents, err := renderSweepConfig(document, raw)
	if err != nil {
		return false, err
	}
	if document.exists && string(contents) == string(document.contents) {
		_, err := fmt.Fprintln(a.out, "Configuration is unchanged.")
		return false, err
	}
	if err := writeSweepConfigDiff(a.out, document.contents, contents); err != nil {
		return false, err
	}
	write, err := promptSweepYesNo(a.stdin, a.out, "Write this configuration?", true)
	if err != nil || !write {
		if err == nil {
			_, err = fmt.Fprintln(a.out, "Configuration unchanged.")
		}
		return false, err
	}
	if err := writeSweepConfig(configPath, document, contents); err != nil {
		return false, err
	}
	_, err = fmt.Fprintf(a.out, "Saved sweep configuration: %s\n", sanitizeTerminalField(configPath))
	return true, err
}

func writeSweepBuiltinSummary(writer io.Writer, home string) error {
	if _, err := fmt.Fprintln(writer, "Default locations:"); err != nil {
		return err
	}
	for _, path := range sweepBuiltinRoots(home) {
		if _, err := fmt.Fprintf(writer, "  %s\n", sanitizeTerminalField(compactSweepPath(home, path))); err != nil {
			return err
		}
	}
	return nil
}

func writeSweepConfigSummary(writer io.Writer, path string, raw sweepYAML) error {
	include := raw.Sweep.IncludeBuiltinRoots == nil || *raw.Sweep.IncludeBuiltinRoots
	if _, err := fmt.Fprintf(writer, "\nSweep configuration: %s\nBuilt-in roots: %t\n", sanitizeTerminalField(path), include); err != nil {
		return err
	}
	for _, group := range []struct {
		label  string
		values []string
	}{{"Worktree pools", raw.Sweep.Roots}, {"Project roots", raw.Sweep.ProjectRoots}, {"Excluded subtrees", raw.Sweep.ExcludeRoots}} {
		if _, err := fmt.Fprintf(writer, "%s:\n", group.label); err != nil {
			return err
		}
		for _, value := range group.values {
			if _, err := fmt.Fprintf(writer, "  %s\n", sanitizeTerminalField(value)); err != nil {
				return err
			}
		}
	}
	return nil
}

func promptSweepYesNo(input io.Reader, output io.Writer, prompt string, defaultYes bool) (bool, error) {
	suffix := " [Y/n]: "
	if !defaultYes {
		suffix = " [y/N]: "
	}
	for {
		if _, err := fmt.Fprint(output, prompt+suffix); err != nil {
			return false, err
		}
		value, err := readSweepPromptLine(input)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "":
			return defaultYes, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			if _, err := fmt.Fprintln(output, "Please answer y or n."); err != nil {
				return false, err
			}
		}
	}
}

func readSweepPromptLine(input io.Reader) (string, error) {
	var builder strings.Builder
	var one [1]byte
	for {
		count, err := input.Read(one[:])
		if count == 1 {
			if one[0] == '\n' {
				return strings.TrimSuffix(builder.String(), "\r"), nil
			}
			builder.WriteByte(one[0])
		}
		if err != nil {
			if errors.Is(err, io.EOF) && builder.Len() != 0 {
				return builder.String(), nil
			}
			return "", err
		}
	}
}

func boolSweepPointer(value bool) *bool { return &value }

func appendUniqueSweepPath(paths []string, value string) []string {
	for _, existing := range paths {
		if existing == value {
			return paths
		}
	}
	return append(paths, value)
}

func removeSweepPath(paths []string, index int) []string {
	return append(paths[:index], paths[index+1:]...)
}

func isSafeSweepInputPath(value string) bool {
	if strings.TrimSpace(value) == "" || filepath.Clean(value) == "." {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}
