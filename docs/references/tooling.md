# Tooling Reference

## Purpose

- Record durable repo-wide tooling notes, command references, and local development expectations
- Keep short-lived implementation notes in feature docs instead of here

## Current State

- `make check` is the canonical local formatting, vet, full-test, and source-size gate.
- `make test-race` and `make lint` are required before delivery.
- `make release-check` validates `.goreleaser.yaml`; `make snapshot` cross-builds every supported release archive beneath ignored `dist/`.
- `internal/catalog/assets/catalog.json` is the canonical embedded tool registry. Generic scripts live beneath `internal/catalog/assets/scripts/`; compiled aliases also require `internal/dispatch` support.
- GitHub pull requests run `.github/workflows/ci.yml`. Semantic `vMAJOR.MINOR.PATCH` tags publish macOS, Linux, and Windows archives plus checksums through `.github/workflows/release.yml`.
