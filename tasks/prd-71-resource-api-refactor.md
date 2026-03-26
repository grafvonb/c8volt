# PRD: Review and Refactor Internal Service Resource API Implementation

## Overview

Refactor the internal resource service to improve readability and maintainability while preserving current deploy, delete, and deployment-confirmation behavior across supported Camunda versions. The work must stay within the existing API, factory, and versioned-service structure, explicitly review generated-client coverage, and only add a small missing capability if it is supported across versions, low risk, and fully verifiable.

## Goals

- Preserve current external behavior of the internal resource service across supported versions.
- Reduce avoidable duplication and make version-specific responsibilities easier to follow.
- Reuse existing repository-native helpers and patterns instead of introducing new abstraction layers.
- Review the supported generated resource clients and make an explicit add-or-defer decision for one bounded missing capability.
- Strengthen automated regression coverage and finish with repository-required validation, including `make test`.

## User Stories

### US-001: Lock the shared resource service contract
**Description:** Define and stabilize the shared resource service surface, version routing expectations, and generated-client contract boundaries before refactoring the implementation.

**Acceptance Criteria:**
- `internal/services/resource/api.go` explicitly defines the preserved shared service surface for the refactor.
- `internal/services/resource/factory_test.go` verifies supported version routing and unsupported version failure behavior.
- `internal/services/resource/v87/contract.go` and `internal/services/resource/v88/contract.go` reflect the generated-client operations required by the settled service surface.

### US-002: Refactor v8.7 resource behavior without regression
**Description:** Simplify the v8.7 resource service implementation so deploy and delete flows are easier to maintain while keeping observable behavior unchanged.

**Acceptance Criteria:**
- `internal/services/resource/v87/service.go` preserves existing deploy and delete behavior for current callers.
- Success-path payload validation follows repository-native patterns and still rejects malformed success responses.
- Automated tests in `internal/services/resource/v87/service_test.go` cover preserved deploy, delete, and malformed-response behavior.

### US-003: Refactor v8.8 resource behavior without regression
**Description:** Simplify the v8.8 resource service implementation while preserving current deploy, delete, and wait-for-confirmation behavior.

**Acceptance Criteria:**
- `internal/services/resource/v88/service.go` preserves deploy, delete, and `--no-wait` versus confirmation-poll behavior.
- Shared payload validation and response handling are clearer without changing current success or failure semantics.
- Automated tests in `internal/services/resource/v88/service_test.go` cover deploy, delete, malformed-success, and confirmation-path behavior.

### US-004: Complete the generated-client coverage decision
**Description:** Review the supported generated resource clients and either expose one bounded missing capability or explicitly record a no-addition decision.

**Acceptance Criteria:**
- The final generated-client coverage review is recorded in `specs/71-resource-api-refactor/research.md`.
- Any added capability is available in both supported versions, fits the existing service surface, and does not require package or layout changes.
- If no capability is added, the bounded rationale is recorded explicitly instead of being implied.
- The shared service API and both versioned contracts and services reflect the final add-or-defer decision consistently.

### US-005: Prove completion with durable regression coverage
**Description:** Expand and tighten regression coverage so future maintainers can safely modify the resource service without rediscovering hidden behavior.

**Acceptance Criteria:**
- Resource service tests cover preserved success paths, malformed-success handling, edge-case delete behavior, and version-specific differences.
- `specs/71-resource-api-refactor/quickstart.md` records the final validation sequence and documentation-impact decision.
- `go test ./internal/services/resource/... -race -count=1` passes.
- `make test` passes.

### US-006: Handle documentation impact only if behavior becomes user-visible
**Description:** Keep the feature internal-only by default, but update user-facing documentation in the same change if the final capability decision affects operator-visible workflows.

**Acceptance Criteria:**
- If no user-visible workflow changes, the feature artifacts explicitly record that documentation remains unchanged.
- If a user-visible workflow changes, `README.md` and the affected generated CLI docs under `docs/cli/` are updated in the same change.
- The final PRD validation path makes the documentation decision explicit rather than leaving it implicit.

## Functional Requirements

- FR-001: The internal resource service MUST preserve current external deploy and delete behavior unless a small, intentional missing capability is explicitly added.
- FR-002: The refactor MUST remain within the existing `api -> factory -> versioned service` structure under `internal/services/resource`.
- FR-003: The implementation MUST reduce avoidable duplication and simplify service control flow only where the result stays low risk and easy to understand.
- FR-004: The existing version-selection path MUST remain the single routing mechanism for supported resource service behavior.
- FR-005: Supported generated resource client capabilities MUST be reviewed against the shared service surface.
- FR-006: Any newly exposed resource capability MUST exist across supported versions and fit current service boundaries without package renames or layout changes.
- FR-007: Success-path payload validation MUST remain behaviorally compatible and should reuse existing repository-native helpers where applicable.
- FR-008: Current version-specific differences, including v8.8 deployment confirmation behavior and the current delete semantics, MUST not be normalized into unintended behavior changes.
- FR-009: Automated tests MUST be added or updated for preserved success paths, malformed-success responses, version-specific behaviors, and any approved capability addition.
- FR-010: The final implementation MUST make its user-visible impact explicit, including stating when no CLI or documentation changes are required.
- FR-011: Repository validation MUST include targeted resource-service tests and `make test`.

## Non-Goals

- Renaming packages or changing package layout.
- Introducing new dependencies, new service layers, or parallel abstraction structures.
- Broad redesign of the internal service architecture.
- Forcing a new resource capability if the generated-client review does not justify one.
- Normalizing current version-specific behavior differences just because both versions are being refactored.
- Changing user-visible command behavior unless a bounded new capability is intentionally surfaced.

## Implementation Notes

- Use `internal/services/resource/api.go`, `internal/services/resource/factory.go`, and the existing v8.7/v8.8 service packages as the canonical refactor surface.
- Treat the existing refactored cluster service as the reference pattern for acceptable cleanup shape.
- Prefer `internal/services/common.RequirePayload` over duplicating nil-payload checks where it fits the current success-path contract.
- Keep the generated-client coverage review mandatory, but keep any actual capability addition conditional.
- Resource metadata lookup is the leading candidate for a low-risk missing capability because both supported generated clients expose read-by-key operations; raw content retrieval is lower priority because it would likely need broader shape decisions.
- Add missing focused v8.7 service tests instead of relying only on v8.8 coverage and factory tests.
- Keep documentation changes conditional: only update `README.md` and regenerate CLI docs if the final capability decision becomes user-visible.
- Preserve repository guidance from `AGENTS.md`, including incremental refactoring, behavior preservation, adjacent test updates, and final `make test`.

## Validation

- Update `internal/services/resource/factory_test.go`.
- Add or update `internal/services/resource/v87/service_test.go`.
- Update `internal/services/resource/v88/service_test.go`.
- Record the final validation sequence in `specs/71-resource-api-refactor/quickstart.md`.
- Run `go test ./internal/services/resource/... -race -count=1`.
- Run `make test`.
- If a user-visible workflow changes, update `README.md`, regenerate relevant docs under `docs/cli/`, and verify the new documentation matches shipped behavior.

## Traceability

- GitHub Issue: #71
- GitHub URL: https://github.com/grafvonb/c8volt/issues/71
- GitHub Title: Review and refactor internal service resource api implementation
- Feature Name: 71-resource-api-refactor
- Feature Directory: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor
- Spec Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/spec.md
- Plan Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/plan.md
- Tasks Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/71-resource-api-refactor/tasks.md
- PRD Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/tasks/prd-71-resource-api-refactor.md
- Source Status: derived from Speckit artifacts

## Assumptions / Open Questions

- The generated-client coverage review may conclude that no new capability should be added; that is acceptable if the rationale remains explicit in feature artifacts and tests still prove preserved behavior.
- Resource metadata lookup is the most likely bounded addition, but the final implementation may still defer it if the cross-version shape or caller value is not strong enough.
- The feature is expected to remain internal-only unless the final generated-client coverage decision introduces a user-visible resource workflow.
- No `contracts/` artifact exists for this feature because the current plan does not commit to an external interface change.
