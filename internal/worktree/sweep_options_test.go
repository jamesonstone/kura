package worktree

import (
	"testing"
	"time"
)

func TestParseSweepOptions(t *testing.T) {
	options, err := parseSweepOptions([]string{
		"--interactive", "--root", "/tmp/one", "--root=/tmp/two",
		"--project-root", "/src", "--exclude-root=/skip", "--only", "merged-local-files",
		"--sort", "size", "--color=always", "--jobs", "8", "--timeout=3s",
		"--no-sizes", "--verbose", "--explain", "/tmp/one/lane",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !options.Interactive || options.Sort != "size" || options.Color != "always" ||
		options.Jobs != 8 || options.Timeout != 3*time.Second || !options.NoSizes || !options.Verbose {
		t.Fatalf("unexpected options: %#v", options)
	}
	if len(options.Roots) != 2 || options.Only != SweepMergedLocalFiles || options.Explain == "" {
		t.Fatalf("missing parsed values: %#v", options)
	}
}

func TestParseSweepOptionsRejectsUnsafeCombinations(t *testing.T) {
	for _, args := range [][]string{
		{"--auto", "--interactive"},
		{"--interactive", "-i"},
		{"--interactive", "--json"},
		{"--auto", "--dry-run"},
		{"--jobs", "0"},
		{"--color", "sometimes"},
		{"--only", "unknown"},
		{"-int"},
	} {
		if _, err := parseSweepOptions(args); err == nil {
			t.Fatalf("expected %v to fail", args)
		}
	}
}
