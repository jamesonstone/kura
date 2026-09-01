package worktree

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestHomeAliasesOpenPrimaryWithoutListingWorktrees(t *testing.T) {
	fixture := newGitFixture(t)
	runGit(t, fixture.primary, "branch", "--track", "topic/navigate", "origin/main")
	runWT(t, fixture.app, fixture.primary, "add", "topic/navigate", "--no-link-env")
	lanePath := filepath.Join(fixture.worktreeRoot, "example", "project", "topic", "navigate")

	inner := fixture.app.run
	fixture.app.run = func(ctx context.Context, cwd, name string, args ...string) ([]byte, error) {
		if name == "git" {
			if len(args) >= 2 && args[0] == "worktree" && args[1] == "list" {
				t.Fatalf("home must not list worktrees: git %s", strings.Join(args, " "))
			}
			if len(args) >= 1 && args[0] == "remote" {
				t.Fatalf("home must not resolve origin identity: git %s", strings.Join(args, " "))
			}
		}
		if name == "gh" {
			t.Fatalf("home must not query GitHub: gh %s", strings.Join(args, " "))
		}
		return inner(ctx, cwd, name, args...)
	}

	for _, args := range [][]string{{"home"}, {"h"}, {"-h"}, {"--home"}} {
		var got string
		fixture.app.runShell = func(_ context.Context, path string) error {
			got = path
			return nil
		}
		fixture.out.Reset()
		runWT(t, fixture.app, lanePath, args...)
		if !samePath(got, fixture.primary) {
			t.Fatalf("%v shell path = %q, want filesystem-equivalent path %q", args, got, fixture.primary)
		}
		if strings.Contains(fixture.out.String(), "do you want to create this worktree?") {
			t.Fatalf("%v prompted to create a worktree", args)
		}
	}
}

func TestHomeAliasesRejectExtraArguments(t *testing.T) {
	fixture := newGitFixture(t)
	fixture.app.runShell = func(context.Context, string) error {
		t.Fatal("shell should not start when home has extra arguments")
		return nil
	}
	for _, args := range [][]string{{"home", "extra"}, {"h", "extra"}, {"-h", "extra"}, {"--home", "extra"}} {
		err := fixture.app.Run(context.Background(), fixture.primary, args)
		if err == nil || err.Error() != "home accepts no arguments" {
			t.Fatalf("%v usage error = %v", args, err)
		}
	}
}
