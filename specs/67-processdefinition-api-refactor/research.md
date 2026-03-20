# Research: Review and Refactor Internal Service Processdefinition API Implementation

## Decision 1: Keep the existing API-plus-factory-plus-versioned-service structure

- **Decision**: Refactor within the current `api -> factory -> versioned service` pattern instead of introducing a new shared package or replacing versioned services with one generic implementation.
- **Rationale**: The repository already uses this structure for internal services, and the issue explicitly forbids crucial structural changes, package renames, and layout changes. Localized cleanup is the safest way to improve readability while preserving behavior.
- **Alternatives considered**:
  - Introduce a new cross-version shared processdefinition base type: rejected because it increases abstraction weight and broadens the refactor surface.
  - Collapse v87 and v88 into one implementation: rejected because the generated clients and supported capabilities still differ in important ways.

## Decision 2: Extract only small shared helpers for duplicated behavior

- **Decision**: Limit refactoring to the smallest shared setup, response-validation, and result-normalization helpers that make the code easier to read without hiding version-specific differences.
- **Rationale**: The current duplication is concentrated around client setup, logging, HTTP-response handling, and latest-result selection. Small helpers reduce copy-paste risk while keeping version-specific service logic explicit and debuggable.
- **Alternatives considered**:
  - Leave all duplication in place: rejected because it does not satisfy the issue goal of improving readability and maintainability.
  - Build a generic adapter framework for generated clients: rejected because it would be a structural redesign rather than an incremental repository-native refactor.

## Decision 3: Treat generated-client coverage review as mandatory and XML retrieval as the leading capability candidate

- **Decision**: Review supported generated processdefinition client capabilities across v87 and v88 and treat XML retrieval as the leading missing-capability candidate because both versions expose an XML retrieval method that the shared service does not currently surface.
- **Rationale**: The issue requires a generated-client coverage review and asks for reasonable missing functionality where appropriate. XML retrieval is bounded, clearly related to the existing processdefinition service, and available in both supported versions, which makes it a stronger candidate than version-specific or structurally broader additions.
- **Alternatives considered**:
  - Force a new capability regardless of fit: rejected because it could create unnecessary behavior change.
  - Skip the coverage review and only refactor current methods: rejected because it would miss an explicit acceptance criterion.
  - Add a capability that only exists or behaves materially differently in one supported version: rejected because the shared service surface must stay stable across versions.

## Decision 4: Preserve the current mixed-version semantics where they already differ

- **Decision**: Keep the current version-specific behavior differences that are already part of the service contract, such as v8.7 rejecting statistics requests while v8.8 can enrich results with statistics.
- **Rationale**: The issue asks for refactoring and reasonable capability additions, not behavioral convergence across versions. Preserving existing supported differences avoids accidental regressions while still allowing clearer code structure.
- **Alternatives considered**:
  - Normalize all version differences into one behavior model: rejected because that would likely change current behavior and expand scope beyond a low-risk refactor.
  - Remove v8.8 statistics support to align with v8.7: rejected because it would be a regression.

## Decision 5: Use existing service and factory tests as the primary validation surface

- **Decision**: Validate the refactor by updating the existing service tests for both supported versions and the processdefinition factory tests, with broader repository validation through `make test`.
- **Rationale**: The repository already has focused unit coverage around versioned processdefinition services, and the constitution requires automated validation before completion. These tests cover the preserved success, error, latest-selection, and version-specific statistics behaviors that the refactor must keep stable.
- **Alternatives considered**:
  - Add only a small smoke test: rejected because it would not protect the detailed behavior being preserved.
  - Depend on manual verification: rejected because `make test` is mandatory before completion.

## Decision 6: Keep documentation changes conditional on actual user-visible impact

- **Decision**: Do not plan a documentation update unless the generated-client coverage review results in a CLI-visible processdefinition capability or behavior change.
- **Rationale**: The current feature is scoped to an internal service refactor, and the constitution allows documentation to remain unchanged for internal-only work as long as that choice is explicit in the plan.
- **Alternatives considered**:
  - Always update README and CLI docs: rejected because it would add churn without matching a shipped behavior change.
  - Never update documentation even if a user-visible capability is added: rejected because it would violate the constitution.
