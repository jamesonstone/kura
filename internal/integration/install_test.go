package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuiltKuraInstallsGitWorktreeForGitDiscovery(t *testing.T) {
	root := repositoryRoot(t)
	temporary := t.TempDir()
	binaryName := "kura"
	aliasName := "git-wt"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
		aliasName += ".exe"
	}
	binary := filepath.Join(temporary, binaryName)
	run(t, root, nil, "go", "build", "-o", binary, "./cmd/kura")
	listOutput := run(t, root, nil, binary, "list")
	if !strings.Contains(listOutput, "git-wt") || !strings.Contains(listOutput, "git wt") {
		t.Fatalf("list output = %q", listOutput)
	}

	binDir := filepath.Join(temporary, "bin")
	manDir := filepath.Join(temporary, "man", "man1")
	stateDir := filepath.Join(temporary, "state")
	args := []string{"install", "--bin-dir", binDir, "--man-dir", manDir, "--state-dir", stateDir, "git-wt"}
	installOutput := run(t, root, nil, binary, args...)
	if strings.Count(installOutput, "installed\tgit-wt") != expectedArtifactCount() {
		t.Fatalf("install output = %q", installOutput)
	}
	alias := filepath.Join(binDir, aliasName)
	helpOutput := run(t, root, nil, alias, "help")
	if !strings.Contains(helpOutput, "Usage: git wt") || !strings.Contains(helpOutput, "sync --dry-run") ||
		!strings.Contains(helpOutput, "sweep [flags]") {
		t.Fatalf("alias help output = %q", helpOutput)
	}
	moduleOutput := run(t, root, nil, "go", "version", "-m", alias)
	if !strings.Contains(moduleOutput, "github.com/jamesonstone/kura") {
		t.Fatalf("installed alias module provenance = %q", moduleOutput)
	}

	gitPath := binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	gitOutput := run(t, root, []string{"PATH=" + gitPath}, "git", "wt", "help")
	if !strings.Contains(gitOutput, "Usage: git wt") {
		t.Fatalf("git discovery output = %q", gitOutput)
	}
	emptyRoot := filepath.Join(temporary, "empty-worktrees")
	if err := os.MkdirAll(emptyRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	sweepOutput := run(t, root, []string{
		"PATH=" + gitPath,
		"HOME=" + filepath.Join(temporary, "home"),
		"XDG_CONFIG_HOME=" + filepath.Join(temporary, "config"),
		"XDG_STATE_HOME=" + filepath.Join(temporary, "sweep-state"),
	}, "git", "wt", "sweep", "--dry-run", "--json", "--root", emptyRoot, "--no-sizes")
	if !strings.Contains(sweepOutput, `"schema_version": 1`) || !strings.Contains(sweepOutput, `"result": "report"`) {
		t.Fatalf("installed sweep output = %q", sweepOutput)
	}
	if runtime.GOOS != "windows" {
		installedManpage, err := os.ReadFile(filepath.Join(manDir, "git-wt.1"))
		if err != nil {
			t.Fatal(err)
		}
		sourceManpage, err := os.ReadFile(filepath.Join(root, "internal", "catalog", "assets", "man", "git-wt.1"))
		if err != nil {
			t.Fatal(err)
		}
		if string(installedManpage) != string(sourceManpage) {
			t.Fatal("installed manpage differs from the embedded source")
		}
		if _, err := exec.LookPath("man"); err == nil {
			manualOutput := run(t, root, []string{
				"PATH=" + gitPath,
				"MANPATH=" + filepath.Dir(manDir),
				"GIT_PAGER=cat",
				"MANPAGER=cat",
				"PAGER=cat",
			}, "git", "wt", "--help")
			if !strings.Contains(manualOutput, "GIT-WT(1)") {
				t.Fatalf("git manual output = %q", manualOutput)
			}
		}
	}

	reinstallOutput := run(t, root, nil, binary, args...)
	if strings.Count(reinstallOutput, "unchanged\tgit-wt") != expectedArtifactCount() {
		t.Fatalf("reinstall output = %q", reinstallOutput)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func run(t *testing.T, directory string, environment []string, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = directory
	command.Env = append(os.Environ(), environment...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return string(output)
}

func expectedArtifactCount() int {
	if runtime.GOOS == "windows" {
		return 1
	}
	return 2
}
