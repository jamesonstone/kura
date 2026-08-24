# PROJECT PROGRESS SUMMARY

## FEATURE PROGRESS TABLE

| ID | FEATURE | PATH | PHASE | PAUSED | CREATED | SUMMARY |
| -- | ------- | ---- | ----- | ------ | ------- | ------- |
| 0001 | interactive-script-installer | `docs/specs/0001-interactive-script-installer` | complete | no | 2026-08-10 | Make Kura a distributable storehouse of host-installable commands. Running the `kura` binary opens a terminal multi-select where Space toggles tools and Enter installs every selected tool to its declared host destination. The first catalog entry extracts Kit's current `git wt` behavior and installs it as the Git-discoverable `git-wt` command with its manual page. |
| 0002 | worktree-sweep | `docs/specs/0002-worktree-sweep` | complete | no | 2026-08-24 | Add one fleet-wide `git wt sweep` command that discovers linked Git worktrees across bounded user and provider roots, proves GitHub merge state, reports disk usage and safety classifications, and removes only the exact worktrees selected through the authority appropriate to their risk. |

## PROJECT INTENT

Kura is an offline Go command storehouse that lets users select and safely install reviewed host tools from one distributable binary while preserving explicit ownership, destinations, and rollback behavior.

## GLOBAL CONSTRAINTS

See `docs/CONSTITUTION.md` for project-wide constraints and principles.

## FEATURE SUMMARIES

### interactive-script-installer

- **STATUS**: complete
- **PAUSED**: no
- **INTENT**: Make Kura a distributable storehouse of host-installable commands. Running the `kura` binary opens a terminal multi-select where Space toggles tools and Enter installs every selected tool to its declared host destination. The first catalog entry extracts Kit's current `git wt` behavior and installs it as the Git-discoverable `git-wt` command with its manual page.
- **APPROACH**: 1. Bootstrap the Go module and define a validated embedded JSON catalog whose artifact model supports both generic script assets and self-executable command aliases. 2. Implement the terminal multi-selector and a small CLI that separates interactive discovery from deterministic non-interactive installation. 3. Implement destination resolution, ownership manifests, preflight collision checks, same-directory staging, rollback-capable commit, exact reporting, and `PATH` guidance. 4. Transfer the live Kit worktree package, tests, and manpage; wire `argv[0]` dispatch so an installed `git-wt` alias executes the preserved app. 5. Add focused catalog, selector, installer, CLI, dispatch, and isolated-host integration tests without growing any source or test file above 300 lines. 6. Add README usage, project testing documentation, Make targets, GoReleaser, pull-request CI, and tag-triggered release publishing. 7. Validate the complete affected behavior, curate durable repository memory, and deliver issue #1 from `GH-1` as a ready pull request.
- **OPEN ITEMS**: none
- **POINTERS**: `docs/specs/0001-interactive-script-installer/SPEC.md`

### worktree-sweep

- **STATUS**: complete
- **PAUSED**: no
- **INTENT**: Add one fleet-wide `git wt sweep` command that discovers linked Git worktrees across bounded user and provider roots, proves GitHub merge state, reports disk usage and safety classifications, and removes only the exact worktrees selected through the authority appropriate to their risk.
- **APPROACH**: 1. Define typed configuration, reports, candidates, evidence, snapshots, actions, and failures. 2. Discover bounded built-in, configured, and nested Claude roots and deduplicate exact physical Git common directories. 3. Resolve GitHub defaults and pull requests in one batch per repository and classify every registered worktree. 4. Calculate bounded disk usage and best-effort live-process evidence. 5. Render one report through human, plain, JSON, menu, and multi-selector interfaces. 6. Revalidate immutable snapshots before automatic clean removal or confirmed dirty/history hard deletion. 7. Restore Kura-owned Makefile installation of the current binary as `git-wt` plus its manual page. 8. Add TTY-only first-run and explicit configuration wizards with typed paths, comment-preserving diff review, backup, and atomic writes. 9. Validate real Git, PTY, configuration, installation, fleet, race, lint, release, source-size, vulnerability, and secret boundaries before ready-PR delivery.
- **OPEN ITEMS**: hosted pull-request checks pending after delivery
- **POINTERS**: `docs/specs/0002-worktree-sweep/SPEC.md`

## LAST UPDATED

2026-08-24 12:19:04 EDT
