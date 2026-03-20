# PRD: Review and Refactor Internal Service Processdefinition API Implementation

## Traceability

- Feature name: `67-processdefinition-api-refactor`
- Source status: Derived from Spec Kit artifacts
- Spec: [specs/67-processdefinition-api-refactor/spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/67-processdefinition-api-refactor/spec.md)
- Plan: [specs/67-processdefinition-api-refactor/plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/67-processdefinition-api-refactor/plan.md)
- Tasks: [specs/67-processdefinition-api-refactor/tasks.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/67-processdefinition-api-refactor/tasks.md)

## Problem Statement

The internal processdefinition service currently carries avoidable duplication across supported Camunda versions, which makes the code harder to read, reason about, and extend safely. The repository needs a low-risk refactor that improves maintainability, preserves current externally observable behavior, explicitly reviews generated-client coverage, and only adds a missing capability when that addition clearly fits the existing service boundaries.

## Goal

Improve the maintainability of the processdefinition service while preserving current search, latest, and get behavior, keeping the existing repository-native structure, and making the generated-client coverage decision and validation path explicit enough for safe implementation and review.

## Maintainer Stories

### Story 1: Safer Processdefinition Service Maintenance

As a maintainer, I want the internal processdefinition service to be easier to read and reason about so that supported-version changes can be made with lower regression risk.

### Story 2: Verified Generated-Client Coverage

As a maintainer, I want the supported generated processdefinition capabilities reviewed against the service surface so that a useful missing capability can be added only when it clearly fits the existing boundaries.

### Story 3: Durable Regression Protection

As a contributor, I want focused automated coverage around the refactored processdefinition service so that future cleanup work does not silently break preserved behavior or version-specific differences.

## Scope

### In Scope

- Refactor the current processdefinition service implementation under the existing API, factory, and versioned-service structure.
- Reduce avoidable duplication between supported-version implementations where that cleanup remains low risk and repository-native.
- Preserve current processdefinition search, latest, and get behavior unless a small intentional capability addition is explicitly approved.
- Review supported generated processdefinition client capabilities and make an explicit add-or-defer decision.
- Treat XML retrieval as the leading bounded capability candidate because it appears in both supported generated clients.
- Add or update focused unit and factory tests for preserved behavior, version-specific differences, and any approved capability addition.
- Update user-facing documentation only if the final generated-client coverage decision introduces a user-visible workflow or output change.

### Non-Goals

- Renaming packages or changing package layout.
- Broad redesign of the internal service architecture.
- Introducing new dependencies, shared abstraction layers, or parallel service structures.
- Forcing a new processdefinition capability when the coverage review does not justify one.
- Converging existing version-specific behavior differences that are already part of the service contract.
- Changing command names, flags, or CLI behavior unless a narrow new capability is intentionally exposed beyond internal use.

## Current Pain Points

- Similar logic exists across versioned processdefinition services, increasing maintenance noise and the chance of inconsistent follow-up changes.
- Current response handling, latest-result selection, and generated-client interaction are more repetitive than necessary.
- Generated-client coverage has not yet been explicitly reviewed as part of the processdefinition service contract.
- Validation expectations need to be explicit so the refactor stays low risk and reviewable.

## Target Outcome

- The processdefinition service remains in the current API-plus-factory-plus-versioned-service shape.
- Shared responsibilities are clearer and duplication is reduced without hiding necessary version-specific differences.
- The shared service surface is explicitly reviewed against supported generated processdefinition capabilities.
- Any adopted missing capability is small, supported across v8.7 and v8.8, and covered by automated tests.
- Reviewers can confirm completion through focused service and factory tests, the documented capability decision, and `make test`.

## Requirements

### Functional Requirements

- **AC-001**: The processdefinition service must keep the existing version-selection entry point and current module boundaries.
- **AC-002**: The implementation must improve readability and maintainability without package renames or layout changes.
- **AC-003**: Avoidable duplication between supported-version implementations must be reduced where that reduction stays low risk and easy to understand.
- **AC-004**: Existing processdefinition workflows must preserve observable behavior, including search ordering, latest-selection behavior, result mapping, and current error handling, unless a small intentional capability addition is documented.
- **AC-005**: Supported-version response handling must remain correct for both generated-client variants.
- **AC-006**: Existing version-specific behavior differences, including v8.7 rejecting statistics requests and v8.8 enriching results with statistics, must be preserved unless an intentional change is explicitly approved and tested.
- **AC-007**: Supported generated processdefinition capabilities must be reviewed against the service surface, with an explicit add-or-defer decision.
- **AC-008**: Any added missing capability must fit the current service boundaries, exist across supported versions, preserve compatibility expectations, and be covered by automated tests.
- **AC-009**: The final work must define the effect on user-visible command behavior and explicitly state when no CLI behavior changes occur.
- **AC-010**: The final work must define how maintainers verify preserved behavior and any intentional addition through automated tests and observable service outcomes.
- **AC-011**: Documentation must be updated in the same change only when user-visible processdefinition workflows or outputs change.

### Invariants

- No crucial structural changes.
- No package renames or package layout changes.
- No new dependency introduction.
- No behavioral regressions in current processdefinition search, latest, or get paths.
- No conflicts with existing versioned packages or names used elsewhere in the project.

## Implementation Notes

- Reuse the repository’s current internal service pattern in `internal/services/processdefinition`.
- Prefer small shared helpers or localized normalization improvements over a new abstraction layer.
- Treat the generated-client coverage review as mandatory, but treat a new capability as conditional.
- Use the cluster service refactor as the structural reference for acceptable cleanup patterns.
- Keep validation centered on `internal/services/processdefinition/factory_test.go`, `internal/services/processdefinition/v87/service_test.go`, `internal/services/processdefinition/v88/service_test.go`, and `make test`.
- If the approved generated-client coverage decision becomes user-visible, align it with current `get` command and Cobra patterns, then update `README.md` and generated CLI docs.
- Preserve the existing convention that generated-adjacent changes should be driven from the source surface and regenerated where the repository already supports that workflow.

## Execution Shaping

### Work Package 1: Stabilize the Shared Contract

- Lock the reviewed processdefinition service surface in the shared API.
- Expand or clarify contract and factory coverage before refactoring core logic.
- Confirm the candidate generated-client additions that are in scope for review.

### Work Package 2: Refactor for Maintainability

- Refactor the shared entry point and version-specific processdefinition logic.
- Reduce duplication in request construction, response validation, and latest-result handling.
- Preserve current mixed-version behavior where it is already part of the supported contract.

### Work Package 3: Complete the Generated-Client Coverage Review

- Review supported generated processdefinition capabilities against the current service surface.
- Either add one justified missing capability, with XML retrieval as the leading candidate, or record a bounded no-addition decision.
- Keep the decision traceable in code, tests, and feature artifacts.

### Work Package 4: Prove Completion

- Update focused regression coverage for preserved success, error, ordering, and edge-case behavior.
- Confirm whether documentation remains internal-only or needs user-visible updates.
- Run the documented validation sequence, including targeted tests and `make test`.

## Acceptance Criteria by Story

### Story 1 Acceptance Criteria

- The processdefinition service area is easier to review because shared responsibilities are clearer and duplication is reduced.
- Existing processdefinition behavior remains unchanged from the caller perspective.
- Supported-version tests and factory tests pass with preserved behavior coverage.

### Story 2 Acceptance Criteria

- The supported generated processdefinition capabilities are explicitly reviewed against the service surface.
- A missing capability is added only if it is useful, low risk, available across supported versions, and compatible with existing boundaries.
- The final generated-client coverage decision is traceable in project artifacts and covered by automated tests.

### Story 3 Acceptance Criteria

- Focused regression validation covers the final behavior of the processdefinition service, including version-specific differences and edge cases.
- The documentation impact is explicit: either no change is required because the work is internal-only, or the affected docs are updated in the same change.
- The final validation path includes targeted tests and `make test`.

## Validation

- Update `internal/services/processdefinition/factory_test.go`.
- Update `internal/services/processdefinition/v87/service_test.go`.
- Update `internal/services/processdefinition/v88/service_test.go`.
- Run `go test ./internal/services/processdefinition/... -race -count=1`.
- Run `make test`.
- If user-visible docs change, run the CLI docs generation flow and verify the updated output.

## Assumptions / Open Questions

- XML retrieval is the leading missing-capability candidate because both supported generated clients expose it, but the coverage review may still conclude that it should not be added at this time.
- The generated-client coverage review may conclude that no new capability should be added; that outcome is acceptable if the rationale remains explicit.
- The feature is expected to remain internal-only unless the approved capability decision introduces a user-visible processdefinition workflow.
- Contracts were not generated during Spec Kit planning because no external interface change is currently committed.

## Source Availability

- Available: `spec.md`, `plan.md`, `tasks.md`
- Also referenced for context: `research.md`, `data-model.md`, `quickstart.md`
- Absent: `contracts/`
