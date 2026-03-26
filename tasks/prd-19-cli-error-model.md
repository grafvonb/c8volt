# PRD: Review and Refactor CLI Error Code Usage

## Overview

Define and implement one shared CLI failure model for all existing `c8volt` commands so humans, shell automation, and AI agents can reliably understand what kind of failure happened, whether retrying makes sense, and how exit codes behave. The work must reuse the existing `c8volt/ferrors` boundary, preserve command-tree compatibility and `--no-err-codes`, and prove the final contract through subprocess-based CLI regression tests and repository-required validation.

## Goals

- Define one shared CLI error classification model across all existing commands.
- Define one shared exit-code mapping for the failure classes in scope.
- Preserve understandable operator-facing failure messages while making them more consistent.
- Make machine-facing failure semantics stable enough for scripts and AI agents to distinguish invalid input, unsupported behavior, local setup issues, remote failures, and internal failures.
- Keep `--no-err-codes` compatible as an exit-code override.
- Preserve the existing Cobra command structure, current repository patterns, and incremental-refactor constraints.
- Finish with targeted validation and `make test`.

## User Stories

### US-001: Lock the shared CLI failure-classification boundary
**Description:** Define the shared CLI error classes, the normalization boundary, and the exit-code policy in the existing shared error layer before sweeping command families.

**Acceptance Criteria:**
- `c8volt/ferrors/errors.go` defines the settled shared CLI error classes and their normalization entry points.
- The failure model defines one shared exit-code mapping for the in-scope failure classes.
- The design preserves `--no-err-codes` as a final exit-code override rather than a bypass of failure classification.

### US-002: Normalize root and bootstrap failures through the shared model
**Description:** Route root command startup, configuration, and service-bootstrap failures through the shared model so early command failures follow the same contract as command execution failures.

**Acceptance Criteria:**
- `cmd/root.go` routes root pre-run failures through the shared CLI failure model.
- `cmd/cmd_cli.go` uses the same shared classification path for bootstrap and CLI-construction failures.
- A subprocess-based CLI test proves a representative bootstrap or configuration failure uses the expected non-success behavior.

### US-003: Make operator-facing failures consistent across read and mutation commands
**Description:** Apply the shared failure model across read-only, state-changing, and validation-heavy command families so similar failures produce consistent operator-facing output.

**Acceptance Criteria:**
- Representative `get`, `run`, `deploy`, `cancel`, `delete`, `expect`, `walk`, `embed`, and `config` failure paths route through the shared model.
- Equivalent invalid-input, local-precondition, unsupported, malformed-response, and remote-failure cases do not rely on one-off per-command wording.
- Subprocess-based CLI tests cover representative read-only and state-changing command failures and verify consistent failure behavior.

### US-004: Make exit semantics stable for scripts and AI agents
**Description:** Apply the shared exit-code mapping across the CLI so machine callers can react consistently to invalid input, unsupported behavior, retryable remote failures, and permanent faults.

**Acceptance Criteria:**
- The same failure class yields the same exit-code class across representative command families.
- Unsupported-version or unsupported-operation failures are distinguishable from unavailable, timeout, or conflict failures.
- `--no-err-codes` returns exit code `0` while preserving failure output.
- Subprocess-based CLI tests assert the exit-code behavior for representative invalid, unsupported, unavailable, timeout, conflict, and not-found paths.

### US-005: Keep the shared error model maintainable for future command work
**Description:** Refactor repeated wrapping and mapping logic into repository-native shared helpers so future command work extends one coherent model instead of reintroducing command-family drift.

**Acceptance Criteria:**
- Repeated error-wrapping logic is consolidated into repository-native shared helpers rather than duplicated across command families.
- Shared command-validation sentinels and cross-family mappings are covered by focused regression tests.
- The final implementation leaves one obvious extension point for future CLI failure behavior changes.

### US-006: Align scripting documentation with shipped failure behavior
**Description:** Update the user-facing scripting and error-code documentation so published guidance matches the actual shared CLI failure contract.

**Acceptance Criteria:**
- `README.md` documents the shipped shared error-code and failure-semantics behavior.
- `docs/index.md` reflects the same scripting and `--no-err-codes` contract.
- If Cobra help text or flag descriptions change, generated docs under `docs/cli/` are regenerated in the same change.

### US-007: Prove completion with targeted CLI regression checks and repository validation
**Description:** Finish the refactor with explicit proof that the shared failure model is wired correctly and does not regress existing command behavior.

**Acceptance Criteria:**
- Targeted validation in `specs/19-cli-error-model/quickstart.md` is updated or confirmed against the final implementation.
- `go test ./cmd/... ./c8volt/ferrors/... -count=1` passes.
- `make test` passes.

## Functional Requirements

- FR-001: The implementation MUST define one shared CLI error classification model for all existing `c8volt` commands.
- FR-002: The shared failure model MUST keep the number of CLI error classes small enough to remain understandable and stable.
- FR-003: Invalid input and invalid flag combinations MUST be distinguishable from configuration and local precondition failures.
- FR-004: Unsupported version and unsupported operation failures MUST be classified predictably.
- FR-005: Internal sentinel errors, malformed-response failures, and representative remote HTTP or API failures MUST map into the shared CLI error model.
- FR-006: The implementation MUST define one shared exit-code mapping for the in-scope CLI error classes.
- FR-007: `--no-err-codes` MUST remain compatible by forcing exit code `0` without bypassing failure classification and rendering.
- FR-008: Similar failures across command families MUST produce consistent operator-facing message structure and comparable failure semantics.
- FR-009: The implementation MUST cover the existing CLI command surface, including root pre-run, read-only commands, state-changing commands, validation-heavy commands, utility commands, and nested subcommands.
- FR-010: The refactor MUST reuse the existing `c8volt/ferrors`, `internal/exitcode`, `internal/domain`, `internal/services`, and `cmd/` patterns instead of introducing a parallel framework.
- FR-011: Automated regression coverage MUST include invalid-input, local-precondition, unsupported-operation or version, internal, malformed-response, and remote-failure paths.
- FR-012: Failure semantics intended for scripting and AI-agent use MUST be proven through subprocess-based CLI tests where exit behavior depends on `os.Exit`.
- FR-013: User-facing scripting documentation MUST be updated when the shipped failure contract changes.
- FR-014: Repository validation MUST include targeted CLI or shared-error tests plus `make test`.

## Non-Goals

- Redesigning successful output payloads or normal success rendering.
- Renaming commands, changing the Cobra tree, or introducing new command hierarchies.
- Introducing a new standalone CLI error framework outside the existing repository-native boundaries.
- Expanding the feature into unrelated business capabilities.
- Removing or weakening `--no-err-codes`.
- Broad architectural rewrites beyond the shared failure-model refactor.

## Implementation Notes

- Use `c8volt/ferrors/errors.go` as the canonical boundary for shared CLI failure normalization and exit-code mapping.
- Reuse existing sentinels and repository-native error sources from `cmd/cmd_errors.go`, `internal/domain/errors.go`, and `internal/services/errors.go` before inventing new abstractions.
- Treat `cmd/root.go` and `cmd/cmd_cli.go` as compatibility-critical bootstrap paths because root pre-run failures are part of the public CLI failure contract.
- Apply the shared model across command families under `cmd/`, including `get`, `run`, `deploy`, `cancel`, `delete`, `expect`, `walk`, `embed`, and `config`.
- Prefer subprocess-based CLI tests for failure behavior because `ferrors.HandleAndExit` terminates via `os.Exit`.
- Keep `README.md` and `docs/index.md` aligned with the shipped behavior; only regenerate `docs/cli/` if help text or flag descriptions change.
- Preserve repository guidance from `AGENTS.md`, including incremental refactoring, behavior preservation where possible, and `make test` before completion.

## Validation

- Add or update focused shared-error tests in `c8volt/ferrors/errors_test.go`.
- Add or update subprocess-based CLI regression coverage in the relevant `cmd/*_test.go` files for representative root, get, run, deploy, cancel, delete, expect, walk, embed, and config failure paths.
- Verify `--no-err-codes` behavior with subprocess-based CLI tests.
- Update or confirm the final validation path in `specs/19-cli-error-model/quickstart.md`.
- Run `go test ./cmd/... ./c8volt/ferrors/... -count=1`.
- Run `make test`.
- If help text changes, run `make docs` and verify generated docs match the shipped behavior.

## Traceability

- GitHub Issue: #19
- GitHub URL: https://github.com/grafvonb/c8volt/issues/19
- GitHub Title: refactor(cli): review and refactor error code usage for humans, automation, and ai agents
- Feature Name: 19-cli-error-model
- Feature Directory: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model
- Spec Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/spec.md
- Plan Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/plan.md
- Tasks Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/19-cli-error-model/tasks.md
- PRD Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/tasks/prd-19-cli-error-model.md
- Source Status: derived from Speckit artifacts

## Assumptions / Open Questions

- The existing `internal/exitcode` values are assumed to remain the compatibility baseline unless implementation exposes a compelling reason to document a change explicitly.
- Generated CLI reference docs are assumed not to require regeneration unless Cobra help text or flag descriptions change during implementation.
