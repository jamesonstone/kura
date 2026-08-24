# Tooling Reference

## Purpose

- Record durable repo-wide tooling notes, command references, and local development expectations
- Keep short-lived implementation notes in feature docs instead of here

## Current State

- `make check` is the canonical local formatting, vet, full-test, and source-size gate.
- `make test-race` and `make lint` are required before delivery.
- `make release-check` validates `.goreleaser.yaml`; `make snapshot` cross-builds every supported release archive beneath ignored `dist/`.
- `make build-kura` builds only `bin/kura`. `make build` also transactionally refreshes the user-local `git-wt` alias and manpage through Kura ownership state from that exact binary; `make install` additionally installs `kura`. `KURA_PREFIX`, `KURA_BIN_DIR`, `KURA_MAN_DIR`, and `KURA_STATE_DIR` override the user-local destinations.
- `internal/catalog/assets/catalog.json` is the canonical embedded tool registry. Generic scripts live beneath `internal/catalog/assets/scripts/`; compiled aliases also require `internal/dispatch` support.
- Fleet worktree maintenance is owned by `internal/worktree/sweep*.go`. One typed report feeds JSON, grouped human output, the selector, immutable confirmation snapshots, persisted run evidence, and automatic execution so safety decisions cannot diverge between interfaces.
- GitHub pull requests run `.github/workflows/ci.yml`. Semantic `vMAJOR.MINOR.PATCH` tags publish macOS, Linux, and Windows archives plus checksums through `.github/workflows/release.yml`.
