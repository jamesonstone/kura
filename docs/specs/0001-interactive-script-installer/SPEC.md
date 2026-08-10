---
kit_metadata_version: 1
artifact: "spec"
workflow_version: 3
phase: "complete"
feature:
  id: "0001"
  slug: "interactive-script-installer"
  dir: "0001-interactive-script-installer"
relationships: []
references:
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
    used_for: code-level, installer integration, release, and CI validation
    status: active
  - id: github-pr-delivery
    name: GitHub PR delivery
    type: ruleset
    target: docs/references/rules/github-pr-delivery.md
    relation: constrains
    read_policy: must
    used_for: issue 1, GH-1, and ready pull-request delivery
    status: active
  - id: kit-git-wt
    name: Kit git-wt implementation
    type: external-code
    target: https://github.com/jamesonstone/kit/tree/81546aa56dbd9a32031aeaae55aa22c1b068424c/cmd/git-wt
    relation: informs
    read_policy: must
    used_for: preserving the live git-wt command behavior during extraction
    status: active
skills:
  - name: github:yeet
    source: GitHub plugin
    path: github:yeet
    trigger: publish the validated issue branch as a ready pull request
    required: true
delivery_intent: issue_branch_pr_ready
---
# SPEC

## PURPOSE

Make Kura a distributable storehouse of host-installable commands. Running the
`kura` binary opens a terminal multi-select where Space toggles tools and Enter
installs every selected tool to its declared host destination. The first
catalog entry extracts Kit's current `git wt` behavior and installs it as the
Git-discoverable `git-wt` command with its manual page.

## CONTEXT

- Kura currently contains repository scaffolding but no Go module, command,
  tests, release configuration, or product behavior.
- Kit currently owns `cmd/git-wt`, `internal/worktree`, its tests, and
  `docs/man/git-wt.1` in the same module and release archive as Kit.
- Kura must be installable with `go install` or a GoReleaser-built binary, then
  install tools without requiring the source checkout or a Go toolchain.
- Git discovers an external subcommand named `git-wt` on `PATH`; `git wt
  --help` also depends on a `git-wt(1)` manual page.
- A copied standalone `git-wt` binary would duplicate release artifacts and
  make every embedded compiled command a second build. One Kura executable can
  instead dispatch by its installed filename while preserving the command's
  public behavior.
- GitHub issue #1 and branch `GH-1` own this implementation. Kit is a read-only
  source dependency for this change; removing its old copy is separate work.

## REQUIREMENTS

- REQ-001: Running `kura` with no arguments in an interactive terminal shows
  all catalog tools, their descriptions, and selection state.
- REQ-002: Space toggles the focused tool, arrow keys or equivalent navigation
  move focus, Enter confirms every selected tool, and cancellation makes no
  installation changes.
- REQ-003: Provide deterministic non-interactive `list`, `install`, `version`,
  and help surfaces for automation, accessibility, and recovery when no TTY is
  available.
- REQ-004: Define the tool catalog declaratively. Each tool owns one or more
  artifacts with a source, destination class, filename, mode, and optional
  platform filter. Adding an embedded generic script must require only an asset
  plus a catalog entry, not a new installer implementation.
- REQ-005: Support embedded-file artifacts and self-executable aliases. The
  latter copies the running Kura executable and dispatches based on `argv[0]`.
- REQ-006: Register `git-wt` as a self-executable alias and install its manpage
  on Unix. Executing the installed alias must run the extracted worktree app.
- REQ-007: Preserve the live Kit worktree source, tests, safety behavior, help,
  terminal selector, environment-link policy, and manpage semantics, changing
  only module ownership and integration required by Kura.
- REQ-008: Resolve user-scoped default destinations with explicit command-line
  and environment overrides. Prefer the running Kura executable directory only
  when that exact executable is discoverable on `PATH`; otherwise use the
  platform's user-local bin and data locations.
- REQ-009: Plan and validate the complete multi-tool installation before
  replacing any file. Refuse an existing unowned or user-modified destination
  unless the user explicitly supplies `--force`.
- REQ-010: Track installed artifact digests and tool ownership in Kura state so
  an unchanged prior Kura installation can be safely upgraded while locally
  modified files remain protected.
- REQ-011: Stage artifacts in their destination directories and commit them as
  one rollback-capable transaction. A failed write or state update must restore
  replaced files and remove newly installed files.
- REQ-012: Report installed, updated, and unchanged paths exactly. Warn when
  the selected bin directory is not on `PATH` and provide an actionable path
  adjustment without modifying shell startup files.
- REQ-013: Keep the CLI offline and host-local. It must not fetch scripts,
  elevate privileges, change shell configuration, or mutate Kit.
- REQ-014: Build Kura for macOS, Linux, and Windows through GoReleaser and a
  tag-triggered GitHub Actions release workflow. Pull requests must run the
  project code-level checks.
- REQ-015: Keep every handwritten Go source and test file at or below 300
  physical lines.

### Non-goals

- Removing `git-wt` from Kit or changing Kit's release in this pull request.
- Downloading plugins or executing a remote script catalog.
- Installing system-wide files, using `sudo`, or editing `PATH`, `MANPATH`, or
  shell startup files automatically.
- Adding uninstall, automatic update polling, dependency resolution, or
  arbitrary lifecycle hooks before more than one tool demonstrates the need.
- Changing `git wt` worktree safety policy or command semantics during the
  extraction.

## ACCEPTANCE

- AC-001: Selector tests prove focus movement, multi-selection, confirmation,
  empty confirmation, cancellation, input decoding, and deterministic render.
- AC-002: Catalog tests reject duplicate IDs, unsafe artifact names, missing
  embedded assets, invalid destination classes, invalid modes, and unsupported
  self aliases while accepting an added generic script entry.
- AC-003: Installer tests prove new install, unchanged reinstall, owned update,
  collision refusal, explicit force, duplicate destination refusal, platform
  filtering, exact modes, and rollback after injected commit/state failure.
- AC-004: CLI tests prove interactive selection, list, explicit install, all,
  version/help, unknown tool, and non-TTY guidance.
- AC-005: A built Kura binary installs into isolated temporary bin/man/state
  roots; the resulting `git-wt` runs help, Git resolves it as `git wt`, the
  manpage is exact, and a second install is unchanged.
- AC-006: The extracted worktree package's complete tests pass in Kura without
  weakening or deleting safety coverage.
- AC-007: Formatting, full tests, race tests, vet, lint, GoReleaser check and
  snapshot, cross-platform builds, source-size audit, diff review, and secret
  review pass or are reported literally.
- AC-008: CI and release workflows use the module's Go version and publish only
  the Kura archive plus checksums for supported targets.

## ACCEPTED PLAN

1. Bootstrap the Go module and define a validated embedded JSON catalog whose
   artifact model supports both generic script assets and self-executable
   command aliases.
2. Implement the terminal multi-selector and a small CLI that separates
   interactive discovery from deterministic non-interactive installation.
3. Implement destination resolution, ownership manifests, preflight collision
   checks, same-directory staging, rollback-capable commit, exact reporting,
   and `PATH` guidance.
4. Transfer the live Kit worktree package, tests, and manpage; wire `argv[0]`
   dispatch so an installed `git-wt` alias executes the preserved app.
5. Add focused catalog, selector, installer, CLI, dispatch, and isolated-host
   integration tests without growing any source or test file above 300 lines.
6. Add README usage, project testing documentation, Make targets, GoReleaser,
   pull-request CI, and tag-triggered release publishing.
7. Validate the complete affected behavior, curate durable repository memory,
   and deliver issue #1 from `GH-1` as a ready pull request.

## DECISIONS

- Accepted: use one Kura executable for compiled tools and dispatch by installed
  filename. This keeps `go install` and release archives simple while letting
  Git discover `git-wt` normally.
- Accepted: use a declarative embedded JSON catalog. Generic scripts are data
  and assets; the installer remains independent of individual tool behavior.
- Accepted: retain user control of host configuration. Kura reports missing
  `PATH` or manual-page discovery but does not edit shell configuration.
- Accepted: maintain an ownership-and-digest state file and transact artifact
  plus state replacement together. Safe upgrades must not require routinely
  forcing over an existing Kura-managed command.
- Rejected: build and ship `git-wt` as a second release binary. That repeats
  the coupling this extraction is intended to remove and scales poorly as Kura
  gains compiled tools.
- Rejected: make the first catalog remote or plugin-driven. Offline embedded
  assets provide deterministic installation, reviewable provenance, and no
  supply-chain fetch at install time.

## DISCOVERIES

- The live Kit implementation is self-contained outside its module import path
  and depends only on `go-runewidth` and `x/term` at runtime.
- Kit's user-scoped Make install already uses `~/.local/bin` and
  `~/.local/share/man/man1`, which remain compatible fallback destinations.
- Kit's `git-wt` implementation and tests already satisfy the 300-line source
  limit; the largest transferred file is 299 lines.
- Race instrumentation exposed a scheduler-dependent fixed delay in Kit's
  pseudo-terminal cancellation test. Kura preserves the behavior while making
  the test deterministic with an explicit child-to-parent readiness pipe.
- The current official GitHub Action majors observed during implementation are
  checkout v7, setup-go v7, golangci-lint-action v9, and GoReleaser action v7.
- Git's intercepted `git wt --help` path renders the embedded manual page when
  `MANPATH` points to the installed user man root; `git wt help` independently
  proves executable discovery without requiring a man viewer.
- Kit v1.0.122 generated its own Kit-specific product sentence in Kura's first
  project rollup. Repository-memory curation replaced that sentence with the
  implemented Kura product intent while preserving the generated feature data.

## VALIDATION

- PASS: `make check` ran formatting verification, `go vet ./...`, the complete
  `go test ./... -count=1` suite, and the source-size target. The extracted
  worktree package passed in 57.311 seconds.
- PASS: `make test-race` ran the complete race-enabled suite. The extracted
  worktree package passed in 81.833 seconds after the deterministic PTY test
  repair; all Kura installer and integration packages also passed.
- PASS: `golangci-lint run ./...` reported zero issues.
- PASS: the built-binary integration test installed into isolated bin, man,
  and state roots; direct `git-wt help`, Git-discovered `git wt help`,
  conditional `git wt --help`, byte-exact manpage comparison, and unchanged
  reinstall all passed.
- PASS: `goreleaser check` validated one configuration file and the snapshot
  built six archives for macOS, Linux, and Windows on amd64 and arm64. The
  snapshot binary reported `kura 0.0.0-SNAPSHOT-59555b1`.
- PASS: `govulncheck ./...` found zero called vulnerabilities and zero
  vulnerabilities in imported packages. It reported one vulnerability in a
  required module whose vulnerable symbols are not called.
- PASS: the repository-wide eligible source/test audit inspected 88 files and
  found zero files above 300 physical lines; the largest affected file remains
  the transferred 299-line `internal/worktree/remove.go`.
- PASS: `gitleaks dir . --redact --no-banner` covered the complete working tree
  and `gitleaks git --redact --no-banner` covered repository history; both
  reported no leaks.
- PASS: all workflow YAML parsed, the current official action release majors
  were verified from their source repositories, and the release tag gate uses
  exact semantic `vMAJOR.MINOR.PATCH` validation.
- PASS: Kit source commit `81546aa56dbd9a32031aeaae55aa22c1b068424c`
  remained clean. Every transferred worktree file and the manpage matches that
  source byte-for-byte except the intentionally synchronized PTY cancellation
  test.
- BASELINE DEBT: after feature completion generated the missing project
  summary, `kit check --project` reports only eight pre-existing instruction
  scaffold warnings across six files because Kura predates Kit's
  cross-repository program rule. The reviewed dry-run proposes five unrelated
  managed changes (one created, one updated, three merged), so that refresh
  remains outside issue #1. The one feature-local invalid reference relation
  was corrected before completion.
- NOT APPLICABLE: deployment and production validation. Kura is a local CLI
  and release artifact, not a deployed service.
- PENDING EXTERNAL: hosted pull-request checks cannot be observed until the
  validated branch is pushed and the ready pull request exists.

## OUTCOME

- Added the `kura` Go CLI with a real terminal multi-selector plus deterministic
  list, install, version, and help commands.
- Added a strict embedded JSON catalog whose generic scripts require only an
  asset and catalog entry, while compiled aliases require explicit dispatch.
- Added user-local destination resolution, Windows-aware paths and names,
  complete preflight collision checks, ownership digests and modes, atomic
  staging, rollback-capable replacement, exact result reporting, and `PATH`
  guidance.
- Extracted the complete live Kit worktree package and manual page into Kura.
  An installed copy of the Kura executable named `git-wt` dispatches that
  behavior, so Go and GoReleaser distribute only one compiled binary.
- Added focused catalog, selector, CLI, installer safety, generic multi-script,
  PTY, worktree, and built-binary integration coverage.
- Added user and contributor documentation, canonical project testing and
  tooling references, project-wide installation invariants, Make targets,
  pull-request CI, tag-triggered publishing, and six-target GoReleaser archives.
- Kit's existing `git-wt` source and release remain unchanged; its eventual
  removal is intentionally a separate Kit delivery.

## REPOSITORY MEMORY

- Created this V3 spec because catalog extensibility, self-dispatch, safe host
  ownership, transactional replacement, and the read-only Kit extraction
  boundary are material architecture and product rationale not recoverable from
  individual code paths alone.
- Updated `docs/CONSTITUTION.md` with the demonstrated offline-catalog,
  user-agency, declarative-artifact, fail-closed ownership, and transactional
  installation invariants.
- Updated `README.md`, `docs/references/testing.md`, and
  `docs/references/tooling.md` with user operation, generic script extension,
  release, and reusable validation guidance.
- `kit complete 0001-interactive-script-installer` transitioned the reconciled
  spec to complete and generated the first
  `docs/PROJECT_PROGRESS_SUMMARY.md` entry.
