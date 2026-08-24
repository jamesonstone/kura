package worktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSweepRepositoriesDeduplicatesCommonDirectory(t *testing.T) {
	fixture := newGitFixture(t)
	path, _ := createPublishedLane(t, fixture, "topic/sweep")
	config := sweepTestConfig(t, fixture)
	config.Roots = append(config.Roots, filepath.Dir(path))

	repositories, failures := fixture.app.discoverSweepRepositories(context.Background(), config)
	if len(failures) != 0 {
		t.Fatalf("discovery failures: %#v", failures)
	}
	if len(repositories) != 1 || len(repositories[0].entries) != 1 {
		t.Fatalf("repositories = %#v", repositories)
	}
	if !samePath(repositories[0].entries[0].path, path) || !samePath(repositories[0].primary, fixture.primary) {
		t.Fatalf("unexpected repository: %#v", repositories[0])
	}
}

func TestDiscoverSweepRepositoriesHonorsExclusion(t *testing.T) {
	fixture := newGitFixture(t)
	path, _ := createPublishedLane(t, fixture, "topic/excluded")
	config := sweepTestConfig(t, fixture)
	config.ExcludeRoots = []string{path}

	repositories, failures := fixture.app.discoverSweepRepositories(context.Background(), config)
	if len(failures) != 0 {
		t.Fatalf("discovery failures: %#v", failures)
	}
	if len(repositories) != 0 {
		t.Fatalf("excluded repositories = %#v", repositories)
	}
}

func TestDiscoverClaudeRootsIsBoundedToRepositoryMarkers(t *testing.T) {
	projectRoot := t.TempDir()
	repository := filepath.Join(projectRoot, "owner", "repo")
	claudeRoot := filepath.Join(repository, ".claude", "worktrees")
	if err := os.MkdirAll(filepath.Join(repository, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claudeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "not-a-repo", ".claude", "worktrees"), 0o755); err != nil {
		t.Fatal(err)
	}
	roots, failures := discoverClaudeRoots(projectRoot, nil)
	if len(failures) != 0 || len(roots) != 1 || roots[0] != claudeRoot {
		t.Fatalf("roots=%#v failures=%#v", roots, failures)
	}
}

func TestSweepRootContainmentResolvesSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	link := filepath.Join(root, "escaped")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	if pathWithinRoot(link, root) {
		t.Fatalf("symlink escape %s was treated as inside %s", link, root)
	}
}

func TestDiscoverSweepRepositoriesReportsBrokenMarker(t *testing.T) {
	root := t.TempDir()
	lane := filepath.Join(root, "owner", "repo", "GH-1")
	if err := os.MkdirAll(lane, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lane, ".git"), []byte("gitdir: /missing/common/worktrees/GH-1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewApp(os.Stdout, os.Stderr)
	_, failures := app.discoverSweepRepositories(context.Background(), SweepConfig{Roots: []string{root}})
	if len(failures) != 1 || failures[0].Operation != "common-directory" {
		t.Fatalf("failures = %#v", failures)
	}
}
