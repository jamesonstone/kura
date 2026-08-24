```text
██╗  ██╗██╗   ██╗██████╗  █████╗
██║ ██╔╝██║   ██║██╔══██╗██╔══██╗
█████╔╝ ██║   ██║██████╔╝███████║
██╔═██╗ ██║   ██║██╔══██╗██╔══██║
██║  ██╗╚██████╔╝██║  ██║██║  ██║
╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝

                         installable commands from one trusted storehouse
```

Kura is a Go command storehouse for developers who want to discover and install reviewed host tools from one offline, distributable binary.

<!-- BEGIN KIT-MANAGED README BADGES -->
[![Last commit](https://img.shields.io/github/last-commit/jamesonstone/kura)](https://github.com/jamesonstone/kura/commits) [![Open issues](https://img.shields.io/github/issues/jamesonstone/kura)](https://github.com/jamesonstone/kura/issues) [![Pull requests](https://img.shields.io/github/issues-pr/jamesonstone/kura)](https://github.com/jamesonstone/kura/pulls) [![CI](https://github.com/jamesonstone/kura/actions/workflows/ci.yml/badge.svg)](https://github.com/jamesonstone/kura/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/release/jamesonstone/kura)](https://github.com/jamesonstone/kura/releases)
<!-- END KIT-MANAGED README BADGES -->

## Install Kura

With Go 1.25 or newer:

```sh
go install github.com/jamesonstone/kura/cmd/kura@latest
```

Prebuilt macOS, Linux, and Windows archives are also published on the [GitHub Releases page](https://github.com/jamesonstone/kura/releases).

Make sure the directory containing `kura` is on `PATH`, then start the selector:

```sh
kura
```

Use the arrow keys or `j`/`k` to move, Space to select any number of tools, and Enter to install them. Kura shows every resulting path and warns when the selected executable directory is not on `PATH`.

## Available tools

### `git wt`

Kura includes the safe, project-oriented Git worktree command extracted from Kit. Selecting it installs:

- `git-wt` beside the discoverable Kura executable, or in the user-local bin directory when Kura was launched by path;
- `git-wt.1` in the user-local `man1` directory on macOS and Linux.

Git automatically discovers executables named `git-<command>` on `PATH`, so the installed tool is invoked as:

```sh
git wt help
git wt list
git wt issue 123
git wt sync --dry-run
git wt sweep
```

The extracted command preserves Kit's conservative worktree, environment-link, removal, and synchronization behavior. Kura does not remove or alter Kit's current copy.

#### Fleet worktree sweep

`git wt sweep` discovers linked worktrees across bounded user and provider
roots, uses authenticated GitHub evidence to classify merged branches, measures
their approximate disk usage, and groups them by removal safety:

- `REMOVE READY` is clean, exact-head, same-repository work merged into the
  GitHub default branch;
- `MERGED + LOCAL FILES` has the same merge proof but contains tracked,
  staged, untracked, ignored, or submodule material;
- `MERGED + LOCAL COMMITS` has a merged pull request but a different local
  head;
- `PROTECTED / ACTIVE` and `UNPROVEN / NOT MERGED` are never selectable;
- `STALE METADATA` is native Git administrative state whose path is gone.

On a terminal, bare sweep shows an animated progress line while it discovers,
queries, classifies, measures, and removes worktrees. It then prints the grouped
report and offers guided actions with exact candidate counts. Progress uses
stderr and is omitted from JSON and redirected output.
Use `--interactive` or `-i` for the colorized multi-selector: arrows or `j`/`k`
move, Space toggles, `/` filters, `s` changes sort, `e` explains, and Enter
opens the exact removal review. A second Enter confirms the unchanged target
snapshot. `MERGED + LOCAL FILES` and divergent commits are interactive-only and
display their recovery loss before the confirmed force operation.

After confirmation, sweep revalidates the complete selected set with one
size-free fleet refresh instead of rebuilding the fleet once per worktree. It
then refreshes process evidence once and immediately rechecks each target's
exact registration, branch, head, and local-status fingerprint before removal.
One drifted target is preserved without weakening independent safe targets.
After apply, interactive sweep always prints a compact completion block. It
reports removed, metadata-pruned, and preserved/failed counts, then lists each
new apply failure with its operation, repository/path, and exact error before
the command returns its aggregate nonzero status.

The action menu offers `[f] address failures`: retry discovery/GitHub evidence
immediately, or select exact retired paths to add to reviewed configuration
exclusions. Exclusion writes retain the normal diff, confirmation, backup, and
atomic persistence contract; this workflow never deletes orphaned contents or
guesses repository identity. `[s] review STALE` opens a stale-only selector.
Protected and unproven rows remain blocked, and every selectable row retains
its existing exact review and confirmation requirements.

Grouped and selector output show each worktree's last commit date. A `STALE`
annotation means that date is older than two calendar months; it is an
informational SSD-cleanup hint and never grants removal authority. Local-file
lanes show a compact basename/path hint plus an overflow count, while the exact
confirmation review continues to list every status entry.

For unattended maintenance:

```sh
git wt sweep --auto
git wt sweep --auto --json
git wt sweep --dry-run --json
```

`--auto` removes only `REMOVE READY` worktrees and exact stale metadata. It
never deletes local files, divergent commits, or remote branches. Sweep does
not fetch, fast-forward defaults, change remotes, or manage a scheduler.
GitHub default-branch and pull-request evidence is deduplicated by repository
identity and fetched in bounded multi-repository page batches; incomplete or
failed evidence remains non-removable.

On the first bare terminal run, sweep offers to create its optional config,
asks whether to include the four built-in locations, and then repeatedly asks
whether to add another typed path. Additional paths are recorded as worktree
pools, project roots for nested `.claude/worktrees`, or excluded subtrees. The
wizard reviews the proposed YAML and asks once more before writing, then
continues directly into sweep.

```sh
git wt sweep
git wt sweep config
git wt sweep config --config "$HOME/custom/git-wt.yaml"
```

`git wt sweep config` reopens an existing file and can toggle built-ins, add or
remove typed paths, and show the complete old/new diff. Updates preserve YAML
comments and unrelated supported settings, retain one mode-`0600` `.bak`, and
replace the mode-`0600` config atomically beneath a mode-`0700` directory.
Missing future directories are accepted with a warning. Invalid YAML, unknown
fields, unsafe paths, and symlink config files fail closed.

Automation, JSON, explicit dry-run, and redirected invocations never prompt or
create configuration. They continue using in-memory built-ins when the default
file is missing. An explicitly missing operational `--config` remains an
error; use the `sweep config --config` command to create it intentionally.

The config path is `$XDG_CONFIG_HOME/kura/git-wt.yaml`, or
`~/.config/kura/git-wt.yaml` when XDG config is unset:

```yaml
version: 1
sweep:
  include_builtin_roots: true
  roots:
    - ~/additional-worktrees
  project_roots:
    - ~/go/src/github.com
  exclude_roots:
    - ~/go/src/github.com/example/repository/.claude/worktrees
  process_check: best_effort
  jobs: 4
  github_timeout: 10s
  sizes:
    enabled: true
    jobs: 4
```

Built-in roots are `~/worktrees`, `~/.codex/worktrees`, `~/Documents/Codex`,
and `~/.claude-worktrees`. Configured project roots are searched only for
nested `.claude/worktrees`; sweep never recursively scans the entire home
directory. Repeated `--root`, `--project-root`, and `--exclude-root` flags add
one-run scope. `--sort`, `--only`, `--no-sizes`, `--color`, `--jobs`,
`--timeout`, `--verbose`, and `--explain` refine reporting.

## Non-interactive use

```sh
kura list
kura install git-wt
kura install --all
kura install --bin-dir "$HOME/bin" --man-dir "$HOME/share/man/man1" git-wt
kura version
```

`KURA_BIN_DIR`, `KURA_MAN_DIR`, and `KURA_STATE_DIR` provide equivalent environment overrides. An existing unowned or locally modified destination is refused. `--force` is the explicit override when replacement is intentional.

## Installation safety

Kura is offline and user-scoped. It does not fetch scripts, elevate privileges, edit shell configuration, or install system-wide files by itself.

Before changing the host, Kura resolves every selected artifact and checks the complete destination set. It records the tool and digest of installed files so future versions can safely update unchanged Kura-owned artifacts. Writes are staged in their destination directories and committed with rollback if an artifact or ownership-state update fails.

## Adding a generic script

The embedded catalog lives at `internal/catalog/assets/catalog.json`. A generic script needs only:

1. the reviewed script beneath `internal/catalog/assets/scripts/`;
2. a catalog tool entry whose artifact uses `source: "embedded"`, `destination: "bin"`, a safe filename, and executable mode `493` (`0755`).

Platform filters can limit an artifact to `darwin`, `linux`, or `windows`. Compiled aliases such as `git-wt` additionally require a dispatcher in the Kura binary; ordinary embedded scripts do not.

## Development

```sh
make build
git wt sweep --dry-run
make install
make check
make test-race
make lint
make release-check
make snapshot
```

`make build` builds `bin/kura` and refreshes the user-local `git-wt` alias and
manual page transactionally through Kura's ownership state from that same
current binary, matching Kit's pre-extraction
developer workflow. `make install` additionally installs `kura` itself. Override
`KURA_PREFIX`, `KURA_BIN_DIR`, `KURA_MAN_DIR`, or `KURA_STATE_DIR` when the
user-local defaults are not the intended destinations.

The pull-request workflow runs formatting, vet, full and race-enabled tests, lint, source-size enforcement, and a cross-platform GoReleaser snapshot. Tags matching `v*` publish Kura archives and checksums through GoReleaser.

## Maintainers

Maintained with 🪖 and ❤️ by [Jameson](https://github.com/jamesonstone) (`jamesonstone`).
