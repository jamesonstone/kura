---
kit_metadata_version: 1
artifact: "spec"
workflow_version: 3
phase: "complete"
feature:
  id: "0006"
  slug: "sweep-stale-wip-guard"
  dir: "0006-sweep-stale-wip-guard"
relationships:
  - type: builds_on
    target: 0005-sweep-interactive-failures
references:
  - id: deletion-safety
    name: Deletion safety
    type: ruleset
    target: docs/references/rules/deletion-safety.md
    relation: constrains
    read_policy: must
    used_for: recoverable worktree retirement versus work-in-progress preservation
    status: active
  - id: source-file-size
    name: Source file size
    type: ruleset
    target: docs/references/rules/source-file-size.md
    relation: constrains
    read_policy: must
    used_for: keeping Go sources and tests within the repository limit
    status: active
  - id: testing-and-environment-validation
    name: Testing and environment validation
    type: ruleset
    target: docs/references/rules/testing-and-environment-validation.md
    relation: constrains
    read_policy: must
    used_for: real-Git stale retirement and WIP preservation tests
    status: active
  - id: github-pr-delivery
    name: GitHub PR delivery
    type: ruleset
    target: docs/references/rules/github-pr-delivery.md
    relation: constrains
    read_policy: must
    used_for: issue 22, GH-22, and ready pull-request delivery
    status: active
delivery_intent: issue_branch_pr_ready
---
# SPEC

## PURPOSE

Retire unused linked worktrees sooner while refusing to delete work in
progress. Operators who always create new lanes rarely return to old checkouts
and can rebuild a worktree from its remote branch.

## CONTEXT

- Feature 0005 made age-stale unproven worktrees interactively retirable after
  two calendar months, including dirty trees, while preserving the local
  branch and blocking `--auto`.
- That window leaves abandoned clean lanes around for months.
- Tracked, staged, untracked, or submodule files are work in progress and must
  not be force-removed.
- An open pull request is also work in progress even when the last commit is
  old.
- Ignored-only build output is not work in progress.
- A published remote branch remains the supported restore path after the
  worktree is removed.

## REQUIREMENTS

- REQ-001: Mark a worktree `STALE` when its last commit is older than 14 days
  relative to report generation.
- REQ-002: Interactive STALE retirement remains available only for unproven
  worktrees whose GitHub reason is `pull-request-missing` or
  `pull-request-not-merged`.
- REQ-003: Do not grant STALE retirement for open PRs, GitHub-unavailable
  rows, ambiguous/fork/detached unproven rows, or protected/active targets.
- REQ-004: Do not grant STALE retirement when local status has tracked,
  staged, untracked, or submodule files.
- REQ-005: Ignored-only status may still use forced worktree removal after
  exact interactive confirmation.
- REQ-006: Preserve the local branch as a recovery ref. Never delete a remote
  branch. `--auto` still cannot retire unproven STALE worktrees.
- REQ-007: Keep every Go source and test file at or below 300 physical lines.

## NON-GOALS

- Do not change GitHub-merged `remove-ready` or merged-local-file/commit
  authority.
- Do not fetch, fast-forward, or delete remote branches.
- Do not give `--auto` STALE unproven authority.

## ACCEPTANCE

- AC-001: Tests prove the 14-day stale boundary.
- AC-002: A clean published unproven lane older than 14 days is interactively
  retirable and keeps its local branch.
- AC-003: An aged unproven lane with untracked or tracked files is not
  retirable and remains on disk.
- AC-004: An aged open-PR lane is not STALE-retirable.
- AC-005: `--auto` still rejects unproven STALE retirement.
- AC-006: `make check`, `make test-race`, and `make lint` pass.

## ACCEPTED PLAN

1. Change the stale age helper from two calendar months to 14 days.
2. Refuse STALE retirement unless the unproven reason is missing or
   closed-unmerged PR evidence.
3. Inspect local status and skip retirement when work-in-progress files exist.
4. Keep ignored-only dirty trees on the existing confirmed force-worktree path.
5. Update canonical sweep docs, manpage, and 0003/0005 stale wording.

## DECISIONS

- Use 14 days instead of two calendar months so abandoned lanes surface in the
  same sprint cadence as new `GH-<n>` worktrees.
- Treat tracked, staged, untracked, and submodule files as work in progress.
  Ignored-only build output is not WIP and may still use confirmed force
  removal.
- Limit unproven STALE retirement to missing or closed-unmerged pull-request
  evidence so open PRs and fail-closed GitHub rows stay preserved.
- Keep `--auto` from gaining STALE unproven authority; interactive exact
  review remains the confirmation boundary.
- Preserve the local branch so a published remote branch can rebuild the
  worktree.

## DISCOVERIES

- Feature 0005 retired dirty stale-unproven worktrees. That deleted untracked
  local files and is incompatible with the work-in-progress guard.
- Open PRs classify as unproven `pull-request-open` and would otherwise inherit
  age-based retirement unless explicitly excluded.

## VALIDATION

- PASS: focused stale-boundary, WIP, open-PR, reason-allowlist, and clean
  stale-retirement tests.
- PASS: `make check`
- PASS: `make lint`
- PASS: `make test-race`

## OUTCOME

- STALE means older than 14 days.
- Interactive retirement of clean unproven lanes with missing or closed
  pull-request evidence preserves the local branch.
- Work-in-progress files and open PRs are not STALE-retirable.
- `--auto` still cannot retire unproven STALE worktrees.

## REPOSITORY MEMORY

- Retained this spec because the 14-day window, WIP definition, and open-PR
  exclusion are policy that tests encode but later agents need as rationale.
- Canonical operator docs live in `docs/references/worktrees.md`.

## NOTES

- 2026-09-07: Issue #22 and worklane `GH-22` own this policy change after PR
  #21 merged.
