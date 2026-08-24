---
kit_metadata_version: 1
artifact: "spec"
workflow_version: 3
phase: "complete"
feature:
  id: "0003"
  slug: "sweep-progress-readability"
  dir: "0003-sweep-progress-readability"
relationships:
  - type: builds_on
    target: 0002-worktree-sweep
references:
  - id: safety-guardrails
    name: Safety guardrails
    type: ruleset
    target: docs/references/rules/safety-guardrails.md
    relation: constrains
    read_policy: must
    used_for: preserving fail-closed classification and removal authority
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
    used_for: terminal, GitHub batching, rendering, and installation validation
    status: active
  - id: github-pr-delivery
    name: GitHub PR delivery
    type: ruleset
    target: docs/references/rules/github-pr-delivery.md
    relation: constrains
    read_policy: must
    used_for: issue 7, GH-7, and ready pull-request delivery
    status: active
delivery_intent: issue_branch_pr_ready
---
# SPEC

## PURPOSE

Make fleet sweep visibly active during slow operations, reduce GitHub API
round-trips, and add compact age and local-file context without weakening the
existing proof or confirmation boundary.

## CONTEXT

- GitHub evidence, bounded discovery, size measurement, and revalidation may
  take long enough that a bare terminal appears stuck.
- Sweep currently resolves the default branch and pull requests separately for
  every distinct clone, including duplicate clones of the same repository.
- Group labels say `MERGED + LOCAL FILES`, while the action menu says `dirty`.
- Candidates already record commit time and exact status lines, but the normal
  views do not surface age or a compact local-file hint.
- GitHub issue #7 and exact branch/worktree `GH-7` own the implementation.

## REQUIREMENTS

- REQ-001: Show an animated, TTY-only progress line during discovery,
  repository evidence, classification, sizing, revalidation, and removal.
- REQ-002: Progress is written outside report output and is absent from JSON,
  redirected output, and persisted reports.
- REQ-003: Deduplicate GitHub evidence by repository identity and fetch default
  branch plus pull-request pages in bounded multi-repository GraphQL batches.
- REQ-004: A failed batch is isolated through bounded per-repository fallback,
  except global authentication or rate-limit failures never fan out; missing,
  truncated, or invalid evidence remains fail closed.
- REQ-005: Use `Merged + Local Files` terminology consistently in action menus
  and add the number of candidate worktrees each option would remove.
- REQ-006: Display the last commit date in grouped and selector output. Mark a
  candidate stale after two calendar months, using the report generation time.
- REQ-007: Age-based staleness is informational only and never changes state,
  selectability, automatic authority, or removal behavior.
- REQ-008: Display a sanitized, bounded local-file summary in normal human and
  selector views while retaining exact status lines in destructive review.
- REQ-009: After execution, report worktrees removed, metadata pruned, and
  preserved/failed actions separately.
- REQ-010: Preserve all existing command, JSON safety, confirmation, and
  removal contracts; keep every Go source and test file at or below 300 lines.

## ACCEPTANCE

- AC-001: Tests prove progress animates on human TTY runs, clears cleanly, and
  never contaminates JSON or non-terminal output.
- AC-002: Tests prove multiple repositories share one GraphQL batch, duplicate
  identities share evidence, pagination is complete, and failures fail closed.
- AC-003: Tests prove the two-calendar-month boundary and verify it has no
  effect on selection or automatic removal authority.
- AC-004: Human and selector tests cover updated columns, stale annotations,
  compact sanitized local-file hints, consistent terminology, and menu counts.
- AC-005: Focused tests, full checks, race tests, lint, release validation,
  source-size audit, installation smoke tests, and hosted PR checks pass.

## ACCEPTED PLAN

1. Add a reusable terminal progress renderer and instrument long sweep phases.
2. Replace per-clone default/PR calls with deduplicated GraphQL page batches.
3. Add derived age and compact status rendering helpers.
4. Update grouped output, selector, menu, review completion, and documentation.
5. Validate from source through the installed `git wt sweep` command and
   deliver the ready GH-7 pull request.

## DECISIONS

- Progress goes to stderr so stdout remains a stable report stream.
- Calendar-month staleness is a visual signal, not a cleanup category.
- Exact local status remains in the final destructive review; ordinary views
  show basenames and an overflow count to preserve readability.
- GitHub pagination stops and fails closed instead of silently accepting more
  than the supported evidence ceiling.

## VALIDATION

- PASS: `make check` passed formatting, vet, the complete test suite, and the
  300-line source audit. The real-Git/PTY worktree package passed in 73.886s.
- PASS: `make test-race` passed the complete race-enabled suite; the worktree
  package passed in 73.258s.
- PASS: `make lint` reported zero findings. `make release-check` validated the
  GoReleaser configuration and `make snapshot` built all six release archives.
- PASS: focused tests cover progress lifecycle and JSON isolation, counted
  terminology, calendar-month staleness without authority changes, compact
  status paths, action totals, repository-identity deduplication, batched
  GraphQL evidence, pagination, failure isolation, and no rate-limit fan-out.
- PASS: a live authenticated read-only run from the current `git-wt` binary
  exercised the GraphQL schema and decoder against `jamesonstone/kura` with no
  failures. An isolated real PTY run visibly animated every report phase and
  rendered the last-updated column without modifying a worktree.
- PASS: `mandoc -T lint`, `git diff --check`, Gitleaks directory/history scans,
  and `govulncheck ./...` passed. Govulncheck found no called vulnerabilities.
- PASS: `make build` installed commit `fed5872` through Kura's transactional
  developer path. A fresh login shell resolved `/Users/jamesonstone/.local/bin/git-wt`,
  reported Kura module provenance, displayed the sweep command, and completed
  an isolated versioned JSON dry-run with no failures.
- PASS: PR #8 implementation head `b0fd62f` was ready, `MERGEABLE`, and `CLEAN`; maintainer
  assignment, Validate, Lint, and GoReleaser snapshot checks all completed
  successfully with no review comments or change requests.

## NOTES

- 2026-08-24: Implementation started in issue #7 and worklane `GH-7`.
- 2026-08-24: PR #8 delivered ready for review after local, installation, live
  GitHub/PTY, security, release, and hosted validation passed.
