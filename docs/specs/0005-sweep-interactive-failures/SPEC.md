---
kit_metadata_version: 1
artifact: "spec"
workflow_version: 3
phase: "validation"
feature:
  id: "0005"
  slug: "sweep-interactive-failures"
  dir: "0005-sweep-interactive-failures"
relationships:
  - type: builds_on
    target: 0004-sweep-batched-revalidation
references:
  - id: safety-guardrails
    name: Safety guardrails
    type: ruleset
    target: docs/references/rules/safety-guardrails.md
    relation: constrains
    read_policy: must
    used_for: literal partial-failure and destructive-action reporting
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
    used_for: interactive, partial-success, installation, and release validation
    status: active
  - id: github-pr-delivery
    name: GitHub PR delivery
    type: ruleset
    target: docs/references/rules/github-pr-delivery.md
    relation: constrains
    read_policy: must
    used_for: issue 11, GH-11, and ready pull-request delivery
    status: active
delivery_intent: issue_branch_pr_ready
---
# SPEC

## PURPOSE

Show exact interactive sweep apply failures before returning the aggregate
nonzero command error.

## CONTEXT

- Candidate failures are recorded in `SweepReport.Failures` during apply.
- `runSweepTerminal` currently returns the joined apply error immediately, so
  it skips the completion counts and never renders those recorded failures.
- The outer interactive command intentionally avoids printing the complete
  report a second time and ultimately exposes only `sweep completed with N
  failure(s)`.
- Initial discovery/GitHub failures were already shown in the first grouped
  report and should not be duplicated after apply.
- GitHub issue #11 and exact branch/worktree `GH-11` own the repair.

## REQUIREMENTS

- REQ-001: After every confirmed interactive apply, print one compact
  `SWEEP COMPLETION` block whether apply succeeds, partially succeeds, or fails.
- REQ-002: Show removed-worktree, pruned-metadata, and preserved/failed target
  counts before returning nonzero.
- REQ-003: Render every failure added during apply with its operation,
  repository/path identity, and exact sanitized error.
- REQ-004: Do not repeat the full candidate report or failures already shown
  before selection.
- REQ-005: Preserve report persistence and the aggregate nonzero exit whenever
  any discovery, interaction, apply, or persistence failure exists.
- REQ-006: Keep automatic, JSON, redirected, dry-run, explanation, cancellation,
  selector, progress, and removal behavior unchanged.
- REQ-007: A completion-output write error is recorded fail closed even when
  apply also failed.
- REQ-008: Keep every Go source and test file at or below 300 physical lines.

## ACCEPTANCE

- AC-001: A confirmed total apply failure prints zero removed, one
  preserved/failed target, the exact target path, and the exact drift error.
- AC-002: A confirmed partial apply prints both the successful removal count
  and the independently preserved target failure before returning nonzero.
- AC-003: Failure rendering sanitizes repository, path, operation, and error
  fields and remains readable without color.
- AC-004: Successful interactive apply still prints completion counts and no
  failure heading.
- AC-005: Existing JSON/human parity, persistence, cancellation, automatic
  application, and sweep safety tests remain green.
- AC-006: Full checks, race tests, lint, release builds, security scans,
  installation smoke tests, and hosted PR checks pass.

## ACCEPTED PLAN

1. Extract reusable compact failure and completion renderers.
2. Capture the pre-apply failure boundary in `runSweepTerminal`.
3. Always render completion after apply, then return the original apply error.
4. Add total-failure, partial-success, successful, and sanitization tests.
5. Update guidance, validate, install, and deliver the ready GH-11 PR.

## DECISIONS

- Completion renders only failures appended during apply; the initial report
  remains the sole rendering of discovery and GitHub failures.
- Exact errors are terminal-sanitized rather than summarized away.
- The aggregate CLI error remains intentionally terse after the detailed block.

## VALIDATION

- PASS: focused real-Git terminal tests prove exact total-failure output,
  partial success plus independent failure, successful completion without a
  failure heading, pre-apply failure non-duplication, and control sanitization.
- PASS: `make check` passed formatting, vet, the complete test suite, and the
  300-line source audit. The worktree package passed in 70.467 seconds.
- PASS: `make test-race` passed the complete race-enabled suite; the worktree
  package passed in 81.938 seconds.
- PASS: `make lint` reported zero findings. `make release-check` validated the
  GoReleaser configuration and `make snapshot` built all six release archives.
- PASS: `mandoc -T lint`, `git diff --check`, Gitleaks directory/history scans,
  and `govulncheck ./...` passed. Govulncheck found no called vulnerabilities.
- PASS: `make build` installed commit `1990598` through Kura's transactional
  developer path. A fresh login shell resolved `/Users/jamesonstone/.local/bin/git-wt`,
  reported Kura module provenance and exact version ldflags, and completed an
  isolated versioned JSON dry-run against only the GH-11 lane with no failures.
- PENDING: hosted pull-request validation runs after delivery.

## NOTES

- 2026-08-24: Implementation started in issue #11 and worklane `GH-11` from
  merged feature 0004 on `origin/main`.
