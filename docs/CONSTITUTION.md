# CONSTITUTION

## PRINCIPLES

- Kura is one trusted, reviewable storehouse for host-installable developer commands.
- The shipped binary remains useful offline: catalog metadata, scripts, compiled command behavior, and supporting manual pages are embedded release inputs.
- Host installation preserves user agency through explicit selection, exact path reporting, conservative ownership checks, and actionable guidance instead of hidden configuration changes.

## CONSTRAINTS

- Every installable tool declares its artifacts, destination classes, modes, and platform applicability in the embedded catalog.
- Generic scripts remain data plus catalog entries. Compiled executable aliases additionally require an explicit in-binary dispatcher.
- Installation is user-scoped by default and never fetches scripts, elevates privileges, edits shell startup files, or silently replaces unowned or locally modified destinations.
- Selected artifacts and Kura ownership state are preflighted and committed as one rollback-capable installation transaction.

### Kit-Managed Baseline Rules

<!-- BEGIN KIT-MANAGED BASELINE RULES -->
- Treat `docs/CONSTITUTION.md` as the canonical project contract.
- Keep `AGENTS.md`, `CLAUDE.md`, and `.github/copilot-instructions.md` aligned with the repo-local docs tree.
- Use native agent planning for research, clarification, design, and implementation planning.
- Before implementation, inspect code and repository memory; create or adopt `SPEC.md` when material rationale exists.
- After validation, curate feature rationale, project invariants, reusable practices, and domain knowledge into their scope-appropriate canonical documents.
- Allow a justified `not required` repository-memory decision when code and tests preserve the complete durable truth.
- Before a terminal task completion or handoff response, load `docs/references/rules/agent-completion-output.md` and use its literal status, immediate prioritized action list, and primary task profile.
- Before commit, pull request, issue, comment, or other attribution text, load `docs/references/rules/human-authorship.md`. Only the human user may be displayed as author; do not attribute coding agents, tools, or bots.
- Keep every version-control-eligible handwritten implementation/source and test file at 300 physical lines or less.
- Before delivery, audit the complete affected source/test scope; whole-project reconcile and scheduled maintenance audit the entire repository.
- Exclude documentation files, all `docs/**`, all `.kit/**`, `.kit.yaml`, ignored files, vendored dependencies, and proven generated files.
- Split oversized files by semantic responsibility while preserving stable public entry points and behavior; never use minification or arbitrary numbered chunks to claim compliance.
<!-- END KIT-MANAGED BASELINE RULES -->

## CHANGE CLASSIFICATION

<!-- all work falls into one of two tracks — classify before acting -->

### Repository-Memory Work

<!-- use when: consequential product rationale, architecture, cross-component behavior, or historical decisions must survive -->
<!-- workflow: native plan → create/adopt SPEC.md before code → implement → validate → curate repository memory -->
<!-- legacy staged documents: BRAINSTORM.md, legacy SPEC.md, PLAN.md, TASKS.md only when explicitly chosen -->

### Ad Hoc (Lightweight)

<!-- use when: bug fixes, security reviews, refactors, dependency updates, config changes, small refinements -->
<!-- workflow: understand → implement → verify -->
<!-- docs: update practical canonical docs when behavior changes -->
<!-- do not create feature SPEC.md solely for ceremony; report a justified not-required memory decision -->

### Ad Hoc with Existing Specs

<!-- if change touches code with existing spec docs: update them when rationale, behavior, requirements, or approach changes -->
<!-- leave them unchanged when code and tests communicate the complete durable truth -->

## NON-GOALS

- Acting as a remote plugin marketplace or executing code fetched at install time.
- Managing system packages, acquiring administrator privileges, or rewriting user shell configuration.
- Owning the runtime behavior or lifecycle of installed commands after they are launched.

## DEFINITIONS

- **Tool**: one user-selectable catalog entry such as `git-wt`.
- **Artifact**: one file installed for a tool, sourced from embedded content or the running Kura executable and routed to a declared destination class.
- **Self-executable alias**: a copy of the Kura binary installed under a command-specific filename and dispatched through `argv[0]`.
- **Ownership state**: Kura's user-local map of installed paths to tool IDs, expected modes, and content digests used for fail-closed upgrades.
