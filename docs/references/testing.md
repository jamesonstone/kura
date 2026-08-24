# Testing Reference

## Purpose

- Record the project's durable commands, suites, environments, automation, and evidence expectations
- Follow `rules/testing-and-environment-validation.md` for the mandatory cross-project testing and production-safety contract
- Keep feature-specific testing details in the current feature's `SPEC.md` VALIDATION and OUTCOME sections; legacy staged flows may still use `PLAN.md` or `TASKS.md`

## Code-Level Validation

| Layer | Command | PR workflow or check | Required | Notes |
| --- | --- | --- | --- | --- |
| Formatting | `make fmt-check` | `validate` | yes | Checks all Go source and tests without rewriting them |
| Static analysis | `make vet` | `validate` | yes | Runs `go vet ./...` |
| Unit and integration | `make test` | `validate` | yes | Includes the complete extracted worktree suite and isolated built-binary installation |
| Race detection | `make test-race` | `validate` | yes | Runs the complete suite with the Go race detector |
| Lint | `make lint` | `lint` | yes | Uses golangci-lint v2 |
| Source size | `make source-size` | `validate` | yes | Enforces the 300-line limit across `cmd` and `internal` Go files |
| Release | `make release-check && make snapshot` | `snapshot` | yes | Validates GoReleaser and cross-builds supported archives |

## High-Level Suites

| Suite | Type | Environment | Command | Automation | Evidence |
| --- | --- | --- | --- | --- | --- |
| Kura installation | integration | isolated local filesystem | `go test ./internal/integration -count=1 -v` | PR `validate` job | Go test output and CI logs |

## Environment Preflights

- The installation test builds the current `cmd/kura`, installs into temporary bin/man/state roots, executes the alias directly and through `git wt`, verifies the manpage, and proves an idempotent reinstall.
- Sweep tests use temporary real Git clones and linked worktrees plus injected GitHub evidence to prove bounded discovery, common-directory and repository-identity deduplication, batched/paginated GitHub evidence with fail-closed fallback, every classification, immutable drift checks, automatic non-force removal, confirmed local-file/history removal, age annotations, compact status hints, TTY-only progress, local branch lifecycle, stale metadata, deterministic JSON/human output, and selector controls without touching host worktrees.
- The developer-install smoke test must prove that the Makefile-installed `git-wt` reports Kura module provenance and that a fresh shell resolves `git wt sweep` immediately.
- Configuration tests prove first-run create/decline and continuation, TTY-only prompting, automation non-mutation, explicit custom paths, default opt-in/out, all typed path categories, deduplication, existing add/remove/toggle/cancel flows, comment and unrelated-setting preservation, complete diff review, missing-path warnings, strict invalid/symlink refusal, mode-0600 backup/config files, mode-0700 parent creation, and atomic replacement cleanup.
- Git must be available on `PATH`; tests do not require GitHub, network access, elevated permissions, or a persistent host install.
- Production validation is not applicable. Kura is a local CLI and release artifact, not a deployed service.

## Credentials And Test Data

- Code-level and integration tests require no credentials or secrets.
- Tests use language-managed temporary directories and do not retain synthetic resources.

## Evidence And Retention

- Go test output and the GoReleaser snapshot remain in CI logs and workflow artifacts; local `dist/` output is ignored.
- A production `RUN_STATUS.md` map is not applicable because Kura has no deployed environment or production suite.

## Automation And Fallbacks

- `.github/workflows/ci.yml` runs every required pull-request check and cross-platform snapshot build.
- `.github/workflows/release.yml` repeats release gates and publishes tag artifacts with GoReleaser.

## Known Gaps

- `git wt --help` rendering is asserted when the host provides a man viewer; the installed manpage is always compared byte-for-byte and `git wt help` always verifies Git command discovery.
- Interactive terminal behavior is covered through a real pseudo-terminal on macOS and Linux and is skipped explicitly on unsupported hosts.
