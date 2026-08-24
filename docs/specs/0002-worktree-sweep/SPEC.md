---
kit_metadata_version: 1
artifact: "spec"
workflow_version: 3
phase: "complete"
feature:
  id: "0002"
  slug: "worktree-sweep"
  dir: "0002-worktree-sweep"
relationships:
  - type: builds_on
    target: 0001-interactive-script-installer
references:
  - id: safety-guardrails
    name: Safety guardrails
    type: ruleset
    target: docs/references/rules/safety-guardrails.md
    relation: constrains
    read_policy: must
    used_for: exact worktree ownership, destructive-operation, identity, and failure boundaries
    status: active
  - id: source-file-size
    name: Source file size
    type: ruleset
    target: docs/references/rules/source-file-size.md
    relation: constrains
    read_policy: must
    used_for: keeping all Go source and test files at or below 300 lines
    status: active
  - id: testing-and-environment-validation
    name: Testing and environment validation
    type: ruleset
    target: docs/references/rules/testing-and-environment-validation.md
    relation: constrains
    read_policy: must
    used_for: unit, real-Git, terminal, installation, release, and host validation
    status: active
  - id: github-pr-delivery
    name: GitHub PR delivery
    type: ruleset
    target: docs/references/rules/github-pr-delivery.md
    relation: constrains
    read_policy: must
    used_for: issue 5, GH-5, and ready pull-request delivery
    status: active
delivery_intent: issue_branch_pr_ready
---
# SPEC

## PURPOSE

Add one fleet-wide `git wt sweep` command that discovers linked Git worktrees
across bounded user and provider roots, proves GitHub merge state, reports disk
usage and safety classifications, and removes only the exact worktrees selected
through the authority appropriate to their risk. Restore Kura's Makefile host
installation workflow so the current source immediately installs both `kura`
and the Git-discoverable `git-wt` alias plus its manual page.

## CONTEXT

- Kura owns and distributes the extracted `git-wt` implementation through one
  executable that dispatches by `argv[0]`.
- Repository-scoped `git wt sync` already provides rich merged-PR evidence and
  exact-head removal safety, but it starts inside one clone, reconciles that
  clone's default branch, and recognizes only Kura's canonical project root.
- Worktrees also exist below Codex, Claude Code, and user-defined roots. Claude
  defaults to project-local `.claude/worktrees`, while provider and user roots
  can be configured or relocated.
- Recursive invocation once per lane duplicates GitHub work, changes current
  lane protection, and cannot provide one coherent review or confirmation.
- Some merged worktrees contain local files or commits. Those targets are not
  eligible for unattended deletion; they need an exact snapshot, loss outline,
  explicit selection, and a second confirmation after review.
- `make install` currently runs only `go install ./cmd/kura`. It does not install
  the current binary under `git-wt` or install the manual page, so source changes
  cannot be tested immediately through `git wt`.
- GitHub issue #5 and exact branch/worktree `GH-5` own the implementation.

## REQUIREMENTS

- REQ-001: Add `git wt sweep` without changing repository-scoped `sync`
  semantics or adding sweep behavior to `list` and navigation.
- REQ-002: Discover worktrees only below built-in roots, configured roots, and
  `.claude/worktrees` below configured project roots. Never recursively scan the
  entire home directory.
- REQ-003: Built-in roots are `~/worktrees`, `~/.codex/worktrees`,
  `~/Documents/Codex`, and `~/.claude-worktrees`. Missing roots are clean no-ops.
- REQ-004: Read optional versioned YAML from
  `$XDG_CONFIG_HOME/kura/git-wt.yaml`, falling back to
  `~/.config/kura/git-wt.yaml`. Support additive roots, project roots,
  exclusions, process-check policy, GitHub limits, and size settings.
- REQ-005: Validate marker files, physical path containment, exact Git
  registration, the shared common directory, primary worktree identity, and
  repository identity. Deduplicate by physical common directory.
- REQ-006: Use authenticated `gh` evidence without fetching, updating refs, or
  fast-forwarding a default branch. GitHub or decode failure fails closed for
  that repository.
- REQ-007: Automatic merge proof requires one unambiguous same-repository PR
  merged into the repository's current GitHub default branch. The local branch
  name must equal the PR head branch.
- REQ-008: Classify every entry as remove-ready, merged-with-local-files,
  merged-with-local-commits, protected-or-active, unproven-or-not-merged, or
  stale-metadata, with stable machine reasons and human explanations.
- REQ-009: Remove-ready additionally requires local `HEAD` equal to the merged
  PR head OID and no tracked, staged, unstaged, untracked, ignored, or submodule
  material other than verified Kura-managed environment symlinks.
- REQ-010: Primary, current, default-branch, locked, detached, fork-backed,
  wrong-base, missing, ambiguous, open, closed-unmerged, root-escaping, or
  positively live-process entries are never selectable.
- REQ-011: Process ownership inspection is best effort. Positive evidence
  blocks selection; unavailable inspection is recorded but does not alone block.
- REQ-012: Calculate approximate physical disk usage without following
  symlinks, using bounded concurrency. Support size sorting and `--no-sizes`.
- REQ-013: Bare `git wt sweep` on a TTY renders grouped colorized results and
  an action menu. Non-TTY use is deterministic, plain, and non-mutating.
- REQ-014: `--interactive` and `-i` open a multi-selector supporting arrows,
  `j`/`k`, Space, filtering, sorting, explanation, review, cancellation, and
  second-Enter confirmation. Labels remain clear without color.
- REQ-015: `--auto` is non-interactive and may remove only remove-ready entries
  plus exact stale metadata. It never removes local files or commits.
- REQ-016: Merged local-file deletion displays exact paths and categories,
  binds confirmation to a snapshot, revalidates, then may use one forced native
  worktree removal. It never uses `git clean`, reset, stash, or manual recursion.
- REQ-017: Merged local-commit deletion displays exact extra commits and loss,
  binds confirmation to the snapshot, revalidates, then may force-remove the
  worktree and force-delete only the exact selected local branch.
- REQ-018: Branch deletion happens only after successful worktree removal.
  Ordinary candidates use non-force deletion; remote branches are never deleted.
- REQ-019: First Enter opens exact review. Second Enter executes only while the
  path, OID, branch, PR, default branch, status digest, process evidence, and
  target set still match. Drift cancels execution and refreshes the report.
- REQ-020: Human, plain, selector, and versioned JSON output are projections of
  one deterministic report. Partial mutation and all independent failures are
  reported literally and return nonzero.
- REQ-021: Support `--auto`, `--interactive`/`-i`, `--json`, `--dry-run`,
  `--config`, repeatable root/project-root/exclusion flags, `--only`, `--sort`,
  `--no-sizes`, `--color`, `--jobs`, `--timeout`, `--verbose`, and `--explain`.
  Reject incompatible interactive, auto, and JSON combinations.
- REQ-022: Honor `NO_COLOR`. Default color is automatic for TTY output.
- REQ-023: Persist redacted reports and confirmation outcomes below
  `$XDG_STATE_HOME/kura/git-wt/sweeps`, falling back to the user-local state
  directory. Never persist environment contents or secrets.
- REQ-024: Extend Kura's Makefile so one documented command builds current
  source and atomically installs or refreshes `kura`, `git-wt`, and `git-wt.1`
  into user-local destinations, then prove `git wt sweep` resolves immediately.
- REQ-025: Keep every handwritten Go source and test file at or below 300
  physical lines and preserve macOS, Linux, and Windows builds.
- REQ-026: When the default config is absent, bare terminal sweep offers a
  first-run wizard before discovery. It asks whether to create configuration,
  whether to include built-ins, and then repeatedly whether to add another
  typed worktree-pool, project-root, or exclusion path. Saving continues the
  same sweep; declining uses in-memory defaults for that invocation.
- REQ-027: Add `git wt sweep config [--config <path>]` as the supported
  interactive create/update surface. Existing files can toggle built-ins, add
  or remove typed paths, and review the full old/new YAML before final write.
  Operational `--config <missing>` remains an error.
- REQ-028: Automation, JSON, explicit dry-run, and non-TTY runs never prompt or
  create configuration. Config creation/update preserves YAML comments,
  ordering, and unrelated supported fields; accepts missing future directories
  with warnings; rejects invalid, unknown, unsafe, non-regular, or symlink
  targets; writes mode 0600 below a created mode-0700 directory; retains one
  mode-0600 backup; and uses platform-native atomic replacement.

### Non-goals

- Changing Kit or KP.
- Installing or managing a recurring scheduler.
- Deleting remote branches, GitHub issues, pull requests, or provider sessions.
- Scanning the entire home directory or arbitrary folders outside configured roots.
- Removing unproven, active, protected, detached, or default-branch worktrees.
- Fetching repository branches, reconciling default branches, or changing remotes.
- Making Kura's repository rules require use of the optional `git wt` wrapper.

## ACCEPTANCE

- AC-001: Discovery tests cover built-ins, YAML/flag additions, exclusions,
  nested Claude roots, missing roots, spaces, symlinks, physical containment,
  broken markers, common-directory deduplication, and no whole-home scan.
- AC-002: Classification tests cover every state and exact refusal reason,
  including GitHub failures, PR ambiguity, default-base proof, head mismatch,
  dirty categories, locks, protected paths, and process evidence.
- AC-003: Tests prove `--auto` removes only remove-ready entries and stale
  metadata, uses no force, touches no remote branch, and is repeat-idempotent.
- AC-004: Tests prove local-file and local-commit targets cannot execute without
  the exact review/confirmation sequence, and snapshot drift aborts both paths.
- AC-005: Terminal tests prove grouped color/no-color rendering, width safety,
  menu choices, selector navigation, Space multi-select, filter/sort/explain,
  review, second-Enter confirmation, and cancellation.
- AC-006: Human/plain/JSON parity, deterministic ordering, exit states, report
  persistence, redaction, size calculation, and partial failures are covered.
- AC-007: The Makefile workflow installs from current source and a real shell
  resolves `git wt sweep` immediately, including the updated manual page.
- AC-008: Formatting, vet, full tests, race tests, lint, source-size, release
  check, snapshot, integration installation, diff, secret, and docs review pass.
- AC-009: Tests prove first-run save/decline and continuation, all typed path
  categories, empty configuration, missing-path warnings, existing
  add/remove/toggle/cancel, comments and non-root settings, old/new diff,
  backup/modes, atomic cleanup, custom config creation, TTY enforcement, and
  zero prompts/writes from non-interactive operational modes.

## ACCEPTED PLAN

1. Define typed sweep configuration, report, repository, candidate, evidence,
   status, snapshot, failure, and execution models.
2. Build bounded marker discovery and physical common-directory deduplication.
3. Add `gh`-only repository/default/PR resolution and deterministic classification.
4. Add bounded size and best-effort live-process inspection.
5. Render grouped human/plain/JSON reports and persist redacted run evidence.
6. Implement the TTY menu, selector, review, and second-confirmation state machine.
7. Implement revalidated automatic, dirty-file, local-commit, and metadata actions.
8. Wire flags, help, manpage, documentation, and Kura-owned configuration.
9. Restore a source-current Makefile install workflow for `kura`, `git-wt`, and
   the manpage and prove immediate Git command discovery.
10. Validate end to end, curate repository memory, and deliver the ready GH-5 PR.
11. Add the first-run and `sweep config` interactive lifecycle with
    comment-preserving YAML mutation, typed paths, reviewed backup/atomic
    writes, and strict non-interactive separation; rerun delivery validation.

## DECISIONS

- Kura is the sole implementation owner because it owns and distributes the
  generic `git-wt` alias; Kit and KP remain unchanged.
- `sweep` is fleet-scoped instead of overloading repository-scoped `sync` or
  Git's metadata-only `prune` terminology.
- Built-ins plus YAML preserve convenience without falsely treating provider
  paths as portable universal defaults.
- Provider-specific session adapters are excluded. Generic Git/GitHub proof and
  best-effort live-process evidence are the accepted ownership boundary.
- Dirty local files and divergent commits are separate hard-delete classes.
  They remain interactive-only and require exact snapshot-bound confirmation.
- The CLI exposes no general `--force` or `--yes` bypass. Force operations are
  internal consequences of one specifically confirmed candidate class.
- Scheduling stays external. `git wt sweep --auto --json` is the stable surface
  for Codex automation, launchd, cron, or another scheduler.
- The installed alias must be built from the same current Kura source as the
  developer binary so `make` validation cannot accidentally exercise Kit's
  stale host copy.
- Missing default configuration is convenience state, not an automation gate.
  Only a bare TTY offers setup; every scripted surface stays non-interactive and
  keeps the established in-memory default behavior.
- The wizard records built-in inclusion as one boolean and additional paths by
  explicit semantic type. It never guesses from current contents, so future
  missing roots and project-local Claude discovery remain stable.
- Existing user-authored YAML is persistent configuration: supported updates
  preserve comments and field order, present the complete diff, retain one
  backup, and fail closed rather than repairing invalid or symlink files.

## DISCOVERIES

- `make install` currently installs only the `kura` executable through
  `go install`; it does not create `git-wt` or install the manual page.
- The host's current `git-wt` reports Kit as its Go module provenance, so the
  final host smoke test must verify that installation changes ownership to Kura.
- Kura already has real-Git merged-PR resolution, branch proof refs, removal
  rollback, selector input, and terminal rendering that can be reused without
  changing repository-scoped sync behavior.
- The local context resolver reports missing baseline workflow evidence in this
  repository. The current repo-local AGENTS routing, guardrails, delivery,
  testing, worktree, and source-size references were loaded directly instead;
  unrelated managed-file reconciliation remains outside issue #5.
- The installed GitHub CLI accepts the repository positionally for `gh repo
  view`; its `--repo` flag is not available. A real fleet smoke test caught the
  incompatible form before host installation and now has exact regression
  coverage.
- Repository `sync` intentionally performs targeted pull-request fallback per
  branch. Reusing that resolver across hundreds of worktrees was too slow, so
  sweep uses one exact `--state all --limit 1000` batch per repository and
  preserves missing branches rather than weakening proof or issuing serial
  fallbacks.
- Recursive `lsof +D` per candidate was also unsuitable at fleet scale. Sweep
  takes one bounded process-CWD snapshot and maps candidates against it; this
  preserves positive live-process blocking and the accepted best-effort
  unavailable state.
- The real built command completed read-only discovery across 43 repositories
  and 588 linked worktrees in the four built-in roots. It classified 67
  remove-ready, 370 merged-with-local-files, 6 merged-with-local-commits, 2
  protected/active, and 143 unproven entries. One broken marker and four
  unavailable GitHub repositories remained explicit fail-closed findings.
- macOS exposes temporary paths through both `/var` and `/private/var`; config,
  containment, discovery, and tests compare physical path identity.
- The original PR loaded optional YAML but never created it; the host's default
  config path was absent. The follow-up adds one shared wizard for first-run and
  explicit later configuration without changing operational defaults.
- Reading prompts one byte at a time avoids buffered read-ahead consuming the
  subsequent sweep menu when first-run setup and sweep continue in one process.
- YAML node mutation reuses retained sequence-item nodes so file, value, and
  per-path comments survive root-list updates. Windows uses replace-existing,
  write-through file replacement while Unix uses ordinary atomic rename.

## VALIDATION

- PASS: `make check` ran formatting verification, `go vet ./...`, the complete
  `go test ./... -count=1` suite, and the whole-source 300-line audit. The
  worktree package passed in 71.727 seconds.
- PASS: `make test-race` passed the complete race-enabled suite; the worktree
  package passed in 78.454 seconds, including real-Git and PTY sweep coverage.
- PASS: `make lint` reported zero GolangCI-Lint findings.
- PASS: `make release-check` validated GoReleaser and `make snapshot` built six
  macOS, Linux, and Windows archives across amd64 and arm64.
- PASS: isolated Kura installation proved alias dispatch, Kura Go-module
  provenance, exact manpage installation, immediate `git wt sweep` discovery,
  versioned JSON output, and unchanged reinstall.
- PASS: a temporary Makefile prefix proved `make build` transactionally
  installed `git-wt`, its manpage, and Kura ownership state from current source.
- PASS: the user-local `make install` path installed `kura`, replaced the stale
  Kit-built alias with Kura provenance, installed the matching manpage, and a
  fresh shell ran an isolated `git wt sweep --dry-run --json` successfully.
- PASS: real fleet discovery used the installed command contract without
  fetching or mutating any worktree. No real `--auto` or interactive removal
  was executed during validation.
- PASS: `mandoc -T lint`, UTF-8 manual rendering, `git diff --check`, feature
  validation, Gitleaks directory/history scans, and `govulncheck ./...` passed.
  Govulncheck reported no called vulnerabilities; imported/module advisories
  do not affect called code.
- PASS: the configuration follow-up reran `make check` and `make test-race`;
  the worktree package passed in 69.580 and 75.825 seconds. Lint reported zero
  findings, Windows test/binary cross-builds passed, and GoReleaser remained
  valid across all six snapshot targets.
- PASS: focused tests cover first-run create/decline/continue, default opt-in
  and empty opt-out, worktree/project/exclusion path types, deduplication,
  missing-path warnings, invalid-input recovery, existing add/remove/toggle/
  cancel, comment and non-root-setting preservation, old/new line diff,
  mode-0600 backup/config, mode-0700 parent creation, atomic cleanup, strict
  version/unknown/symlink refusal, explicit custom config, and non-TTY/
  automation no-write behavior.
- PASS: real `expect`-driven PTY smoke tests created an isolated config through
  synchronized prompts and proved first-run save reloads configuration,
  continues to the grouped sweep menu, accepts quit, and persists one report.
- PASS: PR #6 head `a946d7a` was ready and `CLEAN` with Validate, Lint,
  GoReleaser snapshot, and maintainer-assignment checks passing before the
  configuration follow-up.
- BASELINE DEBT: `kit check --project` retains unrelated stale managed
  instruction findings that predate issue #5. Both feature specs pass
  `kit check --all`; the new feature-local checks are clean.
- PENDING EXTERNAL: hosted checks for the configuration follow-up cannot be
  observed until its validated commit is pushed to the existing ready PR #6.

## OUTCOME

- Added fleet-wide bounded discovery, physical common-directory deduplication,
  exact GitHub default/PR evidence, stable safety states, status fingerprints,
  divergent-commit inventories, one-shot process evidence, disk sizing,
  immutable candidate snapshots, and redacted persistent run reports.
- Added grouped color/no-color output, deterministic JSON/plain reporting, TTY
  action menu, multi-select navigation, filtering, sorting, explanation,
  exact-target review, and second-Enter confirmation.
- Added unattended remove-ready and stale-metadata execution, confirmed dirty
  worktree removal, confirmed divergent local-branch deletion, immediate drift
  revalidation, ordered local branch cleanup, partial failure reporting, and
  remote-branch preservation.
- Added strict versioned YAML, built-in and configured roots, nested Claude
  discovery, exclusions, best-effort process policy, GitHub/size bounds, and
  the full documented flag surface.
- Added TTY-only first-run configuration and `git wt sweep config`, including
  default opt-in/out, repeatable typed paths, existing add/remove/toggle/cancel,
  missing-path warnings, comment-preserving old/new review, one backup, atomic
  replacement, strict invalid/symlink refusal, and immediate post-save reload.
- Restored source-current Makefile installation: `make build` refreshes the
  transactional Kura-owned `git-wt` alias and manpage; `make install` also
  installs `kura`. The resulting `git wt sweep` is immediately testable.
- Updated user, worktree, tooling, testing, safety, manual-page, and durable
  feature documentation without changing Kit or KP.

## REPOSITORY MEMORY

Created this V3 spec because fleet discovery, provider-root boundaries,
GitHub-only proof, interactive hard-delete authority, confirmation drift,
configuration, installed-alias ownership, and automation behavior are material
product decisions not recoverable from isolated code paths or tests.

Updated `README.md`, `docs/references/worktrees.md`,
`docs/references/tooling.md`, `docs/references/testing.md`, and the safety
guardrails with the validated command, install, automation, evidence, and
human-confirmed hard-delete boundaries. No Constitution change is required:
the feature preserves the existing offline, embedded-command, explicit-user-
agency, fail-closed ownership, and transactional-installation invariants.
