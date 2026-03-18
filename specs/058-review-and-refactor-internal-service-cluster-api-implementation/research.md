# Research: Review and Refactor Cluster Service

## Decision 1: Keep the existing factory-plus-versioned-service structure

- **Decision**: Refactor within the current `api -> factory -> versioned service` pattern instead of introducing a new shared package or broader abstraction layer.
- **Rationale**: The repository already uses the same pattern for other internal services, and the issue explicitly forbids crucial structural changes, package renames, and layout changes. Localized cleanup is the safest path to reduce duplication while preserving behavior.
- **Alternatives considered**:
  - Introduce a new cross-version shared cluster base type: rejected because it increases abstraction weight and broadens the refactor surface.
  - Collapse versioned services into one implementation: rejected because generated-client response types differ and the repository consistently keeps version-specific services separate.

## Decision 2: Normalize shared behavior through small helpers, not new layers

- **Decision**: Extract only the smallest shared construction or response-handling patterns that improve readability without hiding version differences.
- **Rationale**: The current duplication is limited and mostly involves repeated setup, logging, and response validation. Small helpers preserve clear ownership while reducing copy-paste risk.
- **Alternatives considered**:
  - Leave all duplication in place: rejected because it does not meet the issue goal of improving readability and maintainability.
  - Build a generic adapter framework for generated clients: rejected because it would be a structural redesign rather than an incremental refactor.

## Decision 3: Treat generated-client coverage review as mandatory, new capability as conditional

- **Decision**: Review the supported generated cluster clients for additional useful operations, but only add one if it is clearly low risk, fits the existing service surface, and does not force user-visible redesign.
- **Rationale**: The issue requires a coverage review but only asks for reasonable missing functionality where appropriate. The plan must allow for “no new capability” if the review finds nothing worth exposing.
- **Alternatives considered**:
  - Force a new capability regardless of fit: rejected because it could create unnecessary behavior change.
  - Skip the review and only refactor current topology code: rejected because it would miss an explicit acceptance criterion.

## Decision 4: Use existing unit, factory, and fake-server integration tests as the validation strategy

- **Decision**: Validate the refactor by updating the existing service tests for both supported versions, the cluster factory tests, and integration tests that already exercise cluster workflows through the fake server.
- **Rationale**: The repository already has this coverage structure, and it matches the constitution’s preference for realistic command or service execution paths where wiring matters.
- **Alternatives considered**:
  - Add only unit tests: rejected because factory selection and integration behavior are part of the preserved surface.
  - Depend on manual verification: rejected because `make test` is mandatory before completion.

## Decision 5: Keep documentation changes conditional on user-visible behavior

- **Decision**: Do not plan a documentation update unless the generated-client coverage review results in a CLI-visible capability or output change.
- **Rationale**: The feature is primarily internal, and the constitution explicitly allows documentation to remain unchanged for internal-only work as long as that choice is stated.
- **Alternatives considered**:
  - Always update README and CLI docs: rejected because it would add churn without matching behavior changes.
  - Never update documentation even if CLI behavior changes: rejected because it violates the constitution.
