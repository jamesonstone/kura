---
kit_metadata_version: 1
artifact: "spec"
workflow_version: 3
phase: "complete"
feature:
  id: "0004"
  slug: "sweep-batched-revalidation"
  dir: "0004-sweep-batched-revalidation"
relationships:
  - type: builds_on
    target: 0003-sweep-progress-readability
references:
  - id: safety-guardrails
    name: Safety guardrails
    type: ruleset
    target: docs/references/rules/safety-guardrails.md
    relation: constrains
    read_policy: must
    used_for: fail-closed destructive revalidation and exact target ownership
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
    used_for: real-Git, performance-regression, installation, and release validation
    status: active
  - id: github-pr-delivery
    name: GitHub PR delivery
    type: ruleset
    target: docs/references/rules/github-pr-delivery.md
    relation: constrains
    read_policy: must
    used_for: issue 9, GH-9, and ready pull-request delivery
    status: active
delivery_intent: issue_branch_pr_ready
---
# SPEC

## PURPOSE

Eliminate the selected-target multiplier from sweep revalidation while
preserving the exact confirmation and immediate local safety boundary.

## CONTEXT

- Applying 377 selected targets currently calls `buildSweepReport` 377 times.
- Each rebuild repeats bounded root discovery, repository-identity resolution,
  batched GitHub evidence, process inspection, and every candidate classifier.
- The progress work in feature 0003 exposed this pre-existing behavior; it did
  not introduce the repeated refresh.
- GitHub batching within one report cannot help when the complete report is
  rebuilt once per selected candidate.
- GitHub issue #9 and exact branch/worktree `GH-9` own the repair.

## REQUIREMENTS

- REQ-001: Build exactly one size-free fleet report after confirmation for any
  non-empty selected target set.
- REQ-002: Index refreshed candidates by stable ID and compare every selected
  state and immutable snapshot before beginning mutation.
- REQ-003: Missing, changed, filtered, or unauthorized candidates remain
  preserved with exact failures; independent unchanged candidates may proceed.
- REQ-004: Take one fresh process-CWD snapshot after fleet revalidation and
  preserve any selected target whose process state changed or became active.
- REQ-005: Immediately before each worktree mutation, re-resolve exact Git
  registration and require the branch, head OID, and status fingerprint to
  match the refreshed candidate.
- REQ-006: Keep ordinary clean-removal reinspections, force-worktree review
  authority, divergent-branch exact-head checks, metadata pruning, and remote
  branch preservation unchanged.
- REQ-007: Progress distinguishes one selected-set refresh from per-target
  local validation/removal and never contaminates machine output.
- REQ-008: Keep every Go source and test file at or below 300 physical lines.

## ACCEPTANCE

- AC-001: A real-Git multi-target test observes exactly one sweep GitHub
  evidence resolution during apply and both unchanged targets are removed.
- AC-002: Tests prove one missing or drifted target is preserved while an
  independently unchanged target can still be removed.
- AC-003: Tests prove an immediate local status-fingerprint mismatch prevents
  force removal even after set-wide revalidation succeeded.
- AC-004: Existing automatic, local-file, local-commit, snapshot-drift,
  metadata-pruning, progress, JSON, and cancellation tests remain green.
- AC-005: Full checks, race tests, lint, release builds, security scans,
  installation smoke tests, and hosted PR checks pass.

## ACCEPTED PLAN

1. Replace candidate-by-candidate fleet refresh with one indexed report.
2. Refresh process evidence once immediately before the mutation phase.
3. Add exact local status verification inside the worktree removal boundary.
4. Add multi-target call-count, independent-failure, and local-drift tests.
5. Update operator guidance, validate, install, and deliver the ready GH-9 PR.

## DECISIONS

- Revalidation remains fail closed per target instead of making one unrelated
  drift cancel all independently safe targets.
- GitHub evidence and bounded discovery are set-wide; mutable local status and
  exact registration remain target-local at the removal boundary.
- Process inspection remains one bounded global CWD snapshot because recursive
  per-worktree `lsof +D` is both slower and less reliable at fleet scale.

## DISCOVERIES

- The selected-target multiplier was above GitHub batching: rebuilding the
  complete report per candidate repeated fleet discovery, evidence collection,
  process inspection, and classification.
- Set-wide evidence can be refreshed once, but mutable local process, Git
  registration, head, and status evidence still require checks near mutation.
- Stable candidate IDs allow missing or drifted targets to fail closed without
  cancelling independently unchanged targets from the same confirmed set.

## VALIDATION

- PASS: focused real-Git tests prove two selected worktrees trigger exactly one
  apply-time GitHub evidence resolution and both unchanged targets are removed.
- PASS: focused tests prove a missing target is preserved without blocking an
  independent removal, a fresh process snapshot runs once for the target set,
  new active-process evidence fails closed, and status drift introduced after
  fleet refresh is caught at the immediate local removal boundary.
- PASS: `make check` passed formatting, vet, the complete test suite, and the
  300-line source audit. The worktree package passed in 63.729 seconds.
- PASS: `make test-race` passed the complete race-enabled suite; the worktree
  package passed in 70.963 seconds.
- PASS: `make lint` reported zero findings. `make release-check` validated the
  GoReleaser configuration and `make snapshot` built all six release archives.
- PASS: `mandoc -T lint`, `git diff --check`, Gitleaks directory/history scans,
  and `govulncheck ./...` passed. Govulncheck found no called vulnerabilities.
- PASS: `make build` installed commit `37e5399` through Kura's transactional
  developer path. A fresh login shell resolved `/Users/jamesonstone/.local/bin/git-wt`,
  reported Kura module provenance and exact version ldflags, and completed an
  isolated versioned JSON dry-run against only the GH-9 lane with no failures.
- PASS: PR #10 implementation head `9691d64` was ready, `MERGEABLE`, and
  `CLEAN`; maintainer assignment, Validate, Lint, and GoReleaser snapshot checks
  all completed successfully with no review comments or change requests.

## OUTCOME

- Replaced per-candidate report rebuilds with one size-free fleet refresh for
  the confirmed target set and indexed refreshed candidates by stable ID.
- Added one fresh set-wide process snapshot plus immediate target-local Git
  registration, head, and status verification before mutation.
- Preserved missing, drifted, or newly active targets with exact failures while
  allowing independently unchanged targets to proceed.
- Delivered the validated repair through GH-9 and ready PR #10.

## REPOSITORY MEMORY

- Retained this V3 spec because the set-wide versus target-local revalidation
  boundary and independent fail-closed behavior are material safety and
  performance rationale not recoverable from isolated code paths alone.
- Reconciled the implemented decisions, discoveries, validation evidence, and
  GH-9/PR #10 delivery outcome in this spec.

## NOTES

- 2026-08-24: Implementation started in issue #9 and worklane `GH-9` from
  merged feature 0003 on `origin/main`.
- 2026-08-24: PR #10 delivered ready for review after local, race, security,
  release, installation, and hosted validation passed.
