package dispatch

import "testing"

func TestAlias(t *testing.T) {
	tests := map[string]string{
		"/usr/local/bin/git-wt":      GitWorktree,
		"git-wt.exe":                 GitWorktree,
		"/usr/local/bin/kura":        "",
		"/tmp/not-really-git-wt.old": "",
	}
	for argv0, want := range tests {
		if got := Alias(argv0); got != want {
			t.Fatalf("Alias(%q) = %q, want %q", argv0, got, want)
		}
	}
}
