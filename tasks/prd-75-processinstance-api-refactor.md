# PRD: Review and Refactor Internal Service Processinstance API Implementation

## Overview

Refactor the internal processinstance service to improve readability and maintainability while preserving current create, get, search, wait, walk, cancel, and delete behavior across supported Camunda versions. The work must stay within the existing API, factory, versioned-service, waiter, and walker structure, explicitly review generated-client coverage, and only add one small missing capability if it remains low risk, fits the current service surface, and defines explicit unsupported-version behavior where support differs.

## Goals

- Preserve current external behavior of the internal processinstance service across supported versions.
- Reduce avoidable duplication and make shared versus version-specific responsibilities easier to follow.
- Keep waiter and walker behavior clear and compatible with current callers.
- Reuse existing repository-native helpers and patterns instead of introducing new abstraction layers.
- Review supported generated processinstance clients and make an explicit add-or-defer decision for one bounded missing capability.
- Strengthen automated regression coverage and finish with repository-required validation, including `make test`.

## User Stories

### US-001: Lock the shared processinstance service contract
**Description:** Define and stabilize the shared processinstance service surface, version routing expectations, helper boundaries, and generated-client contract requirements before refactoring implementation details.

**Acceptance Criteria:**
- `internal/services/processinstance/api.go` explicitly defines the preserved shared service surface for the refactor.
- `internal/services/processinstance/factory_test.go` verifies supported version routing and unsupported version failure behavior.
- `internal/services/processinstance/v87/contract.go` and `internal/services/processinstance/v88/contract.go` reflect the generated-client operations required by the settled service surface.

### US-002: Refactor waiter and walker backed v8.7 processinstance behavior without regression
**Description:** Simplify the v8.7 processinstance implementation so create, search, cancel, delete, and traversal flows are easier to maintain while keeping observable behavior unchanged.

**Acceptance Criteria:**
- `internal/services/processinstance/v87/service.go` preserves current create, get, search, cancel, delete, and family-traversal behavior for existing callers.
- Success-path payload validation follows repository-native patterns where applicable and still rejects malformed success responses.
- Automated tests in `internal/services/processinstance/v87/service_test.go` cover preserved success, malformed-response, and error behavior.

### US-003: Refactor waiter and walker backed v8.8 processinstance behavior without regression
**Description:** Simplify the v8.8 processinstance implementation while preserving current create, get, search, wait-for-state, cancel, delete, and traversal semantics.

**Acceptance Criteria:**
- `internal/services/processinstance/v88/service.go` preserves current create, get, search, cancel, delete, and polling-backed state confirmation behavior.
- Shared payload validation and helper interaction paths are clearer without changing current success or failure semantics.
- Automated tests in `internal/services/processinstance/v88/service_test.go` cover preserved success paths, malformed-success handling, and wait-related behavior.

### US-004: Preserve helper semantics while making polling and traversal logic safer to maintain
**Description:** Keep waiter and walker behavior stable, but make their invariants explicit through focused cleanup and direct regression coverage.

**Acceptance Criteria:**
- `internal/services/processinstance/waiter/waiter.go` preserves current timeout, absent-state, and retry-limit behavior.
- `internal/services/processinstance/walker/walker.go` preserves current ancestry, descendants, family, cycle-detection, and orphan-handling behavior.
- Automated tests in `internal/services/processinstance/waiter/waiter_test.go` and `internal/services/processinstance/walker/walker_test.go` verify those invariants directly.

### US-005: Complete the generated-client coverage decision with explicit partial-version handling
**Description:** Review the supported generated processinstance clients and either expose one bounded missing capability or explicitly record a no-addition decision, with defined unsupported-version behavior if only one version can support the capability.

**Acceptance Criteria:**
- The final generated-client coverage review is recorded in `specs/75-processinstance-api-refactor/research.md`.
- Any added capability fits the existing service surface and does not require package or layout changes.
- If an added capability is only available in one supported version, the unsupported version returns a defined error and that behavior is covered by tests.
- If no capability is added, the bounded rationale is recorded explicitly instead of being implied.

### US-006: Prove completion with durable regression coverage and explicit documentation impact
**Description:** Expand and tighten regression coverage so future maintainers can safely modify the processinstance service, and make the final documentation decision explicit.

**Acceptance Criteria:**
- Processinstance service tests cover preserved success paths, malformed-success handling, edge-case state transitions, helper invariants, and any approved capability addition.
- `specs/75-processinstance-api-refactor/quickstart.md` records the final validation sequence and documentation-impact decision.
- `go test ./internal/services/processinstance/... -race -count=1` passes.
- `make test` passes.

### US-007: Update user-facing documentation only if behavior becomes operator-visible
**Description:** Keep the feature internal-only by default, but update user-facing documentation in the same change if the final capability decision affects operator-visible workflows or outputs.

**Acceptance Criteria:**
- If no user-visible workflow changes, the feature artifacts explicitly record that documentation remains unchanged.
- If a user-visible workflow changes, `README.md` and affected generated CLI docs under `docs/cli/` are updated in the same change.
- The final validation path makes the documentation decision explicit rather than leaving it implicit.

## Functional Requirements

- FR-001: The internal processinstance service MUST preserve current external create, get, search, wait, walk, cancel, and delete behavior unless a small, intentional missing capability is explicitly added.
- FR-002: The refactor MUST remain within the existing `api -> factory -> versioned service` structure under `internal/services/processinstance`, keeping `waiter` and `walker` as repository-native helper packages.
- FR-003: The implementation MUST reduce avoidable duplication and simplify service control flow only where the result stays low risk and easy to understand.
- FR-004: The existing version-selection path MUST remain the single routing mechanism for supported processinstance service behavior.
- FR-005: Supported generated processinstance client capabilities MUST be reviewed against the shared service surface.
- FR-006: Any newly exposed processinstance capability MUST fit current service boundaries without package renames or layout changes.
- FR-007: If a newly exposed processinstance capability is only available in one supported version, the unsupported version MUST return a defined, tested error path.
- FR-008: Success-path payload validation MUST remain behaviorally compatible and should reuse existing repository-native helpers where applicable.
- FR-009: Waiter and walker behavior MUST remain compatible with current callers, including absent-state handling, timeout behavior, cycle detection, orphan handling, and family traversal semantics.
- FR-010: Current version-specific differences MUST not be normalized into unintended behavior changes just because both supported versions are being refactored together.
- FR-011: Automated tests MUST be added or updated for preserved success paths, malformed-success responses, helper invariants, version-specific behaviors, and any approved capability addition.
- FR-012: The final implementation MUST make its user-visible impact explicit, including stating when no CLI or documentation changes are required.
- FR-013: Repository validation MUST include targeted processinstance-service tests and `make test`.

## Non-Goals

- Renaming packages or changing package layout.
- Introducing new dependencies, new service layers, or parallel abstraction structures.
- Broad redesign of the internal service architecture.
- Forcing a new processinstance capability if the generated-client review does not justify one.
- Rewriting waiter or walker semantics as part of the refactor.
- Changing user-visible command behavior unless a bounded new capability is intentionally surfaced.

## Implementation Notes

- Use `internal/services/processinstance/api.go`, `internal/services/processinstance/factory.go`, `internal/services/processinstance/v87/service.go`, and `internal/services/processinstance/v88/service.go` as the canonical refactor surface.
- Treat `internal/services/processinstance/waiter/waiter.go` and `internal/services/processinstance/walker/walker.go` as compatibility-critical helpers whose semantics must stay stable even if call sites are simplified.
- Use the existing refactored cluster service as the reference pattern for acceptable cleanup shape.
- Prefer `internal/services/common.RequirePayload` over duplicating nil-payload checks where it fits the current success-path contract, while preserving repository guidance for zero-value payload validation on single-object responses.
- Keep the generated-client coverage review mandatory, but keep any actual capability addition conditional and bounded to one coherent slice.
- The leading capability candidates come from generated processinstance endpoints already exposed by the supported clients, such as sequence-flow, flow-node-statistics, or variable-search operations; the final implementation may still defer all of them if none fits the current service surface cleanly enough.
- Add missing focused v8.7 and helper-level tests instead of relying only on existing factory or user-facing command coverage.
- Keep documentation changes conditional: only update `README.md` and regenerate CLI docs if the final capability decision becomes user-visible.
- Preserve repository guidance from `AGENTS.md`, including incremental refactoring, behavior preservation, adjacent test updates, and final `make test`.

## Validation

- Update `internal/services/processinstance/factory_test.go`.
- Add or update `internal/services/processinstance/v87/service_test.go`.
- Add or update `internal/services/processinstance/v88/service_test.go`.
- Add helper-level regression coverage in `internal/services/processinstance/waiter/waiter_test.go` and `internal/services/processinstance/walker/walker_test.go` if helper behavior or call sites are refactored.
- Record the final validation sequence in `specs/75-processinstance-api-refactor/quickstart.md`.
- Run `go test ./internal/services/processinstance/... -race -count=1`.
- Run `make test`.
- If a user-visible workflow changes, update `README.md`, regenerate relevant docs under `docs/cli/`, and verify the new documentation matches shipped behavior.

## Traceability

- GitHub Issue: #75
- GitHub URL: https://github.com/grafvonb/c8volt/issues/75
- GitHub Title: Review and refactor internal service processinstance api implementation
- Feature Name: 75-processinstance-api-refactor
- Feature Directory: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor
- Spec Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/spec.md
- Plan Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/plan.md
- Tasks Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/75-processinstance-api-refactor/tasks.md
- PRD Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/tasks/prd-75-processinstance-api-refactor.md
- Source Status: derived from Speckit artifacts

## Assumptions / Open Questions

- The generated-client coverage review may conclude that no new capability should be added; that is acceptable if the rationale remains explicit in feature artifacts and tests still prove preserved behavior.
- A partial-version capability addition is allowed by the current feature clarification, but the exact capability choice is still open and must remain bounded to one low-risk slice.
- The feature is expected to remain internal-only unless the final generated-client coverage decision introduces a user-visible processinstance workflow.
- No `contracts/` artifact exists for this feature because the current plan does not commit to an external interface change.
