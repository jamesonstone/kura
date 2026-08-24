# PROJECT PROGRESS SUMMARY

## FEATURE PROGRESS TABLE

| ID | FEATURE | PATH | PHASE | PAUSED | CREATED | SUMMARY |
| -- | ------- | ---- | ----- | ------ | ------- | ------- |
| 0001 | interactive-script-installer | `docs/specs/0001-interactive-script-installer` | complete | no | 2026-08-10 | Make Kura a distributable storehouse of host-installable commands. Running the `kura` binary opens a terminal multi-select where Space toggles tools and Enter installs every selected tool to its declared host destination. The first catalog entry extracts Kit's current `git wt` behavior and installs it as the Git-discoverable `git-wt` command with its manual page. |
| 0002 | worktree-sweep | `docs/specs/0002-worktree-sweep` | complete | no | 2026-08-24 | Add one fleet-wide `git wt sweep` command that discovers linked Git worktrees across bounded user and provider roots, proves GitHub merge state, reports disk usage and safety classifications, and removes only the exact worktrees selected through the authority appropriate to their risk. |
| 0003 | sweep-progress-readability | `docs/specs/0003-sweep-progress-readability` | complete | no | 2026-08-24 | Make fleet sweep visibly active during slow operations, reduce GitHub API round-trips, and add compact age and local-file context without weakening removal authority. |
| 0004 | sweep-batched-revalidation | `docs/specs/0004-sweep-batched-revalidation` | complete | no | 2026-08-24 | Replace per-candidate fleet refreshes with one selected-set revalidation plus immediate target-local safety checks. |
| 0005 | sweep-interactive-failures | `docs/specs/0005-sweep-interactive-failures` | complete | no | 2026-08-24 | Render exact failures, provide retry/exclusion triage, and support bulk STALE retirement with branch preservation. |

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
- **APPROACH**: 1. Define typed configuration, reports, candidates, evidence, snapshots, actions, and failures. 2. Discover bounded built-in, configured, and nested Claude roots and deduplicate exact physical Git common directories. 3. Resolve GitHub defaults and pull requests through fail-closed repository evidence and classify every registered worktree. 4. Calculate bounded disk usage and best-effort live-process evidence. 5. Render one report through human, plain, JSON, menu, and multi-selector interfaces. 6. Revalidate immutable snapshots before automatic clean removal or confirmed local-file/history hard deletion. 7. Restore Kura-owned Makefile installation of the current binary as `git-wt` plus its manual page. 8. Add TTY-only first-run and explicit configuration wizards with typed paths, comment-preserving diff review, backup, and atomic writes. 9. Validate real Git, PTY, configuration, installation, fleet, race, lint, release, source-size, vulnerability, and secret boundaries before ready-PR delivery.
- **OPEN ITEMS**: none
- **POINTERS**: `docs/specs/0002-worktree-sweep/SPEC.md`

### sweep-progress-readability

- **STATUS**: complete
- **PAUSED**: no
- **INTENT**: Make fleet sweep visibly active during slow operations, reduce GitHub API round-trips, and add compact age and local-file context without weakening removal authority.
- **APPROACH**: 1. Animate a TTY-only stderr status line throughout long sweep phases. 2. Deduplicate GitHub evidence by repository identity and resolve default branches plus PR pages in bounded GraphQL batches. 3. Show two-calendar-month stale annotations, compact local-file hints, consistent terminology, and counted actions without changing authority. 4. Validate terminal, batch, rendering, safety, installation, and release contracts before ready-PR delivery.
- **OPEN ITEMS**: none
- **POINTERS**: `docs/specs/0003-sweep-progress-readability/SPEC.md`

### sweep-batched-revalidation

- **STATUS**: complete
- **PAUSED**: no
- **INTENT**: Replace per-candidate fleet refreshes with one selected-set revalidation plus immediate target-local safety checks.
- **APPROACH**: 1. Build and index one size-free report after confirmation. 2. Preserve missing, drifted, or unauthorized candidates independently before mutation. 3. Refresh process evidence once, then recheck exact registration, branch, head, and local status per target. 4. Prove the fleet evidence resolver runs once for multi-target apply and retain every existing removal guardrail. 5. Validate and deliver issue #9 from `GH-9` as a ready PR.
- **OPEN ITEMS**: none
- **POINTERS**: `docs/specs/0004-sweep-batched-revalidation/SPEC.md`

### sweep-interactive-failures

- **STATUS**: complete
- **PAUSED**: no
- **INTENT**: Render exact failures, provide retry/exclusion triage, and support bulk STALE retirement with branch preservation.
- **APPROACH**: 1. Reuse a compact sanitized failure renderer across human output and interactive completion. 2. Add failure retry and exact reviewed exclusions without repository/file repair. 3. Use path-qualified selector membership so Space preserves multiple toggles. 4. Make stale unproven worktrees interactive-only retirable after status/process inspection while preserving local branches. 5. Add review and bulk STALE actions with exact confirmation. 6. Prove selection, safety, branch recovery, and automation boundaries before ready-PR delivery.
- **OPEN ITEMS**: none
- **POINTERS**: `docs/specs/0005-sweep-interactive-failures/SPEC.md`

## LAST UPDATED

2026-08-24 15:55:19 EDT
