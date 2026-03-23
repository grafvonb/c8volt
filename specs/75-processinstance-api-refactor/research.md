# Research: Review and Refactor Internal Service Processinstance API Implementation

## Decision 1: Keep the existing factory-plus-versioned-service-plus-helper structure

- **Decision**: Refactor within the current `api -> factory -> versioned service` pattern and keep `waiter` and `walker` as focused helper packages instead of introducing a new shared base package or broader abstraction layer.
- **Rationale**: The repository already uses this structure successfully for other service areas, and the issue explicitly forbids crucial structural changes, package renames, and layout changes. Localized cleanup is the safest way to improve maintainability while preserving behavior across create, search, cancel, delete, and tree-walk flows.
- **Alternatives considered**:
  - Introduce a new cross-version shared processinstance base type: rejected because it increases abstraction weight and broadens the refactor surface.
  - Collapse v87 and v88 into one implementation: rejected because generated-client request and response shapes differ materially across versions.
  - Fold waiter or walker logic back into the versioned services: rejected because those helpers already provide useful separation for polling and graph traversal behavior.

## Decision 2: Normalize duplicated response validation through repository-native helpers where they fit

- **Decision**: Prefer existing `internal/services/common` payload-validation helpers when current processinstance services open-code the same HTTP-status-plus-nil-payload checks and the response shapes match the shared helper contracts.
- **Rationale**: The repository guidance already favors shared payload validation helpers, and the current processinstance services repeat the same malformed-response guard in several paths. Reusing repository-native helpers reduces duplication without changing the surrounding service structure or error semantics.
- **Alternatives considered**:
  - Keep each version-specific nil-payload check inline: rejected because it preserves repeated malformed-response logic the repository already standardizes elsewhere.
  - Introduce a new processinstance-specific response helper package: rejected because the repository already has common helpers and the issue does not justify a new abstraction layer.

## Decision 3: Preserve waiter and walker semantics exactly while simplifying the call paths around them

- **Decision**: Treat `waiter.WaitForProcessInstanceState`, `waiter.WaitForProcessInstancesState`, and the walker ancestry, descendant, and family flows as compatibility-critical behavior whose semantics must stay the same even if their callers are simplified.
- **Rationale**: These helpers define the observable proof model for create, cancel, delete, expect, and walk workflows. The refactor should make their use easier to follow without changing current retry timing, absent-state handling, cycle detection, orphan detection, or family traversal behavior.
- **Alternatives considered**:
  - Rewrite waiter behavior around a different polling model: rejected because that would change operational semantics rather than refactor structure.
  - Replace walker recursion or ancestry rules during the refactor: rejected because it would risk behavioral regressions in child-process handling and delete traversal.

## Decision 4: Treat generated-client coverage review as mandatory and bound new capability exposure to one small addition

- **Decision**: Review the supported generated processinstance clients for additional useful operations, but only add one capability if it is clearly low risk, fits the current service surface, and has explicit unsupported-version behavior if only one supported version can expose it cleanly.
- **Rationale**: The issue requires a generated-client coverage review and the spec explicitly allows a partial-version addition when unsupported versions return a defined error. Both generated Operate clients expose extra processinstance-related endpoints such as sequence flows, flow-node statistics, and variable search, but the plan should keep scope narrow until one candidate proves worth exposing.
- **Alternatives considered**:
  - Force multiple new processinstance capabilities into this issue: rejected because it would broaden the API and test surface too much for a refactor-focused change.
  - Skip the review and only refactor current code paths: rejected because it would miss an explicit acceptance criterion.
  - Require all additions to exist across both supported versions: rejected because the spec clarification explicitly allows partial-version additions with a defined unsupported-version path.

## Decision 5: Keep observability behavior stable and limit any additions to targeted capability-specific logging

- **Decision**: Preserve current observability behavior by default, and only add targeted logs when a newly exposed capability or unsupported-version path would otherwise be opaque to maintainers or operators.
- **Rationale**: This feature is a maintainability refactor, not an observability redesign. Keeping existing log behavior stable minimizes churn while still allowing one narrow improvement if a new capability needs a clear unsupported-version or failure explanation.
- **Alternatives considered**:
  - Add new logs and metrics across all refactored processinstance paths: rejected because it widens the feature beyond the issue scope.
  - Prohibit any observability change at all: rejected because one targeted log may be the clearest way to preserve operator understanding for a newly added bounded capability.

## Decision 6: Expand validation with focused service, helper, and factory tests

- **Decision**: Validate the refactor with focused versioned service tests, factory tests, and helper-level waiter or walker tests where behavior is touched, rather than relying only on current command coverage or repository-wide `make test`.
- **Rationale**: The repository constitution requires automated validation, and the current visible processinstance test surface is thinner than the behavioral surface described in the service API and CLI documentation. Adding direct tests around preserved service and helper behavior is the clearest way to protect against regressions.
- **Alternatives considered**:
  - Depend on existing command or integration coverage alone: rejected because it does not isolate version-specific and helper-level behavior well enough for a refactor.
  - Depend on manual CLI verification: rejected because `make test` is mandatory before completion.

## Decision 7: Keep documentation changes conditional on user-visible change

- **Decision**: Do not plan a documentation update unless the generated-client coverage review results in a user-visible processinstance capability, command behavior change, or output change.
- **Rationale**: The feature is primarily internal, and the constitution explicitly allows documentation to remain unchanged for internal-only work as long as that choice is recorded in the plan.
- **Alternatives considered**:
  - Always update README and CLI docs: rejected because it would add churn without matching behavior changes.
  - Never update documentation even if a new processinstance workflow or output path appears: rejected because it violates the constitution.
