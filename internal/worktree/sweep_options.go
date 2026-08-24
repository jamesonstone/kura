package worktree

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseSweepOptions(args []string) (SweepOptions, error) {
	options := SweepOptions{Sort: "state", Color: "auto", Jobs: 4, Timeout: 10 * time.Second}
	seen := make(map[string]bool)
	for index := 0; index < len(args); index++ {
		arg := args[index]
		name, inline, hasInline := strings.Cut(arg, "=")
		value := func() (string, error) {
			if hasInline {
				return inline, nil
			}
			index++
			if index >= len(args) {
				return "", fmt.Errorf("%s requires a value", name)
			}
			return args[index], nil
		}
		switch name {
		case "--auto", "--interactive", "-i", "--json", "--dry-run", "--no-sizes", "--verbose":
			if hasInline || seen[name] {
				return SweepOptions{}, fmt.Errorf("invalid or duplicate sweep flag %q", arg)
			}
			seen[name] = true
			switch name {
			case "--auto":
				options.Auto = true
			case "--interactive", "-i":
				options.Interactive = true
			case "--json":
				options.JSON = true
			case "--dry-run":
				options.DryRun = true
			case "--no-sizes":
				options.NoSizes = true
			case "--verbose":
				options.Verbose = true
			}
		case "--config", "--only", "--sort", "--color", "--jobs", "--timeout", "--explain":
			if seen[name] {
				return SweepOptions{}, fmt.Errorf("duplicate sweep flag %q", name)
			}
			seen[name] = true
			parsed, err := value()
			if err != nil {
				return SweepOptions{}, err
			}
			if err := applySweepValue(&options, name, parsed); err != nil {
				return SweepOptions{}, err
			}
		case "--root", "--project-root", "--exclude-root":
			parsed, err := value()
			if err != nil {
				return SweepOptions{}, err
			}
			switch name {
			case "--root":
				options.Roots = append(options.Roots, parsed)
			case "--project-root":
				options.ProjectRoots = append(options.ProjectRoots, parsed)
			case "--exclude-root":
				options.ExcludeRoots = append(options.ExcludeRoots, parsed)
			}
		default:
			return SweepOptions{}, fmt.Errorf("unknown sweep flag %q", arg)
		}
	}
	if seen["--interactive"] && seen["-i"] {
		return SweepOptions{}, fmt.Errorf("duplicate interactive sweep flag")
	}
	if options.Auto && options.Interactive || options.Interactive && options.JSON || options.Auto && options.DryRun {
		return SweepOptions{}, fmt.Errorf("--auto, --interactive, --json, and --dry-run have an incompatible combination")
	}
	return options, nil
}

func applySweepValue(options *SweepOptions, name, value string) error {
	switch name {
	case "--config":
		options.ConfigPath = value
	case "--only":
		state := SweepState(value)
		if !validSweepState(state) {
			return fmt.Errorf("unknown sweep state %q", value)
		}
		options.Only = state
	case "--sort":
		if value != "state" && value != "size" && value != "updated" && value != "repository" && value != "path" {
			return fmt.Errorf("--sort must be state, size, updated, repository, or path")
		}
		options.Sort = value
	case "--color":
		if value != "auto" && value != "always" && value != "never" {
			return fmt.Errorf("--color must be auto, always, or never")
		}
		options.Color = value
	case "--jobs":
		jobs, err := strconv.Atoi(value)
		if err != nil || jobs < 1 || jobs > 32 {
			return fmt.Errorf("--jobs must be an integer from 1 through 32")
		}
		options.Jobs = jobs
	case "--timeout":
		duration, err := time.ParseDuration(value)
		if err != nil || duration <= 0 {
			return fmt.Errorf("--timeout must be a positive duration")
		}
		options.Timeout = duration
	case "--explain":
		options.Explain = value
	}
	return nil
}

func validSweepState(state SweepState) bool {
	for _, candidate := range sweepStateOrder {
		if candidate == state {
			return true
		}
	}
	return false
}
