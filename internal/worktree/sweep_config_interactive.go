package worktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

func (a *App) runSweepConfigCommand(ctx context.Context, args []string) error {
	_ = ctx
	configPath := ""
	seenConfig := false
	for index := 0; index < len(args); index++ {
		name, inline, hasInline := strings.Cut(args[index], "=")
		if name != "--config" || seenConfig {
			return fmt.Errorf("usage: git wt sweep config [--config <path>]")
		}
		seenConfig = true
		if hasInline {
			configPath = inline
		} else {
			index++
			if index >= len(args) {
				return fmt.Errorf("--config requires a value")
			}
			configPath = args[index]
		}
		if strings.TrimSpace(configPath) == "" {
			return fmt.Errorf("--config requires a non-empty path")
		}
	}
	if !a.isTerminal() {
		return fmt.Errorf("sweep configuration requires terminal input and output")
	}
	home, resolved, err := a.resolveSweepConfigPath(configPath)
	if err != nil {
		return err
	}
	_, err = a.runSweepConfigWizard(home, resolved)
	return err
}

func (a *App) offerFirstSweepConfig() error {
	if !a.isTerminal() {
		return nil
	}
	home, configPath, err := a.resolveSweepConfigPath("")
	if err != nil {
		return err
	}
	if _, err := os.Lstat(configPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect sweep config %s: %w", configPath, err)
	}
	_, err = a.runSweepConfigWizard(home, configPath)
	return err
}

func (a *App) runSweepConfigWizard(home, configPath string) (bool, error) {
	document, err := readSweepConfigDocument(configPath)
	if err != nil {
		return false, err
	}
	if !document.exists {
		if _, err := fmt.Fprintf(a.out, "No sweep configuration exists at %s.\n", sanitizeTerminalField(configPath)); err != nil {
			return false, err
		}
		create, err := promptSweepYesNo(a.stdin, a.out, "Create one now?", true)
		if err != nil || !create {
			if err == nil {
				_, err = fmt.Fprintln(a.out, "Using in-memory defaults for this sweep; no configuration was written.")
			}
			return false, err
		}
		return a.createSweepConfigWizard(home, configPath, document)
	}
	return a.editSweepConfigWizard(home, configPath, document)
}

func (a *App) createSweepConfigWizard(
	home string,
	configPath string,
	document sweepConfigDocument,
) (bool, error) {
	raw := document.raw
	include, err := promptSweepYesNo(a.stdin, a.out, "Include the default worktree locations?", true)
	if err != nil {
		return false, err
	}
	raw.Sweep.IncludeBuiltinRoots = boolSweepPointer(include)
	if include {
		if err := writeSweepBuiltinSummary(a.out, home); err != nil {
			return false, err
		}
	}
	for {
		add, err := promptSweepYesNo(a.stdin, a.out, "Would you like to add more paths?", false)
		if err != nil {
			return false, err
		}
		if !add {
			break
		}
		if err := a.promptSweepAddPath(home, &raw); err != nil {
			return false, err
		}
	}
	return a.confirmSweepConfigWrite(configPath, document, raw)
}

func (a *App) editSweepConfigWizard(
	home string,
	configPath string,
	document sweepConfigDocument,
) (bool, error) {
	raw := document.raw
	for {
		if err := writeSweepConfigSummary(a.out, configPath, raw); err != nil {
			return false, err
		}
		if _, err := fmt.Fprint(a.out, "[d] toggle defaults  [a] add path  [r] remove path  [v] review/save  [q] cancel: "); err != nil {
			return false, err
		}
		choice, err := readSweepPromptLine(a.stdin)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "d":
			include := raw.Sweep.IncludeBuiltinRoots == nil || *raw.Sweep.IncludeBuiltinRoots
			raw.Sweep.IncludeBuiltinRoots = boolSweepPointer(!include)
		case "a":
			if err := a.promptSweepAddPath(home, &raw); err != nil {
				return false, err
			}
		case "r":
			if err := a.promptSweepRemovePath(&raw); err != nil {
				return false, err
			}
		case "v", "":
			return a.confirmSweepConfigWrite(configPath, document, raw)
		case "q":
			_, err := fmt.Fprintln(a.out, "Configuration unchanged.")
			return false, err
		default:
			if _, err := fmt.Fprintln(a.out, "Choose d, a, r, v, or q."); err != nil {
				return false, err
			}
		}
	}
}
