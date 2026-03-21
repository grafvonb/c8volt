# Research: Review and Refactor Internal Service Resource API Implementation

## Decision 1: Keep the existing factory-plus-versioned-service structure

- **Decision**: Refactor within the current `api -> factory -> versioned service` pattern instead of introducing a new shared package or broader abstraction layer.
- **Rationale**: The repository already uses the same pattern for other internal services, and the issue explicitly forbids crucial structural changes, package renames, and layout changes. Localized cleanup is the safest way to improve maintainability while preserving behavior.
- **Alternatives considered**:
  - Introduce a new cross-version shared resource base type: rejected because it increases abstraction weight and broadens the refactor surface.
  - Collapse versioned services into one implementation: rejected because generated-client request and response shapes differ meaningfully across v87 and v88.

## Decision 2: Reuse repository-native payload validation helpers

- **Decision**: Prefer `internal/services/common.RequirePayload` for shared HTTP-status-plus-payload validation where the current resource services open-code the same malformed-response check.
- **Rationale**: Repository guidance already prefers `RequirePayload` for generated success responses, and the resource services currently duplicate the same nil-payload guard after status validation. Reusing the existing helper reduces duplication without changing the surrounding service structure.
- **Alternatives considered**:
  - Keep each version-specific nil-payload check inline: rejected because it preserves duplicated response-validation logic the repository already standardized elsewhere.
  - Introduce a new resource-specific response helper: rejected because the common helper already exists and fits the intended use.

## Decision 3: Add resource metadata lookup and defer raw content retrieval

- **Decision**: Extend the shared resource service with a `Get` operation that returns typed resource metadata for a resource key in both supported versions, and explicitly defer raw content retrieval.
- **Rationale**: The issue requires a generated-client coverage review and allows one bounded addition where reasonable. Both v87 and v88 generated clients expose metadata lookup and content retrieval, but metadata lookup fits the existing service pattern cleanly: it reuses standard success-payload validation, maps into a small typed domain object, and does not force binary/text handling decisions into the shared API yet.
- **Alternatives considered**:
  - Force both lookup and content retrieval into the service surface: rejected because that broadens the API without first proving user value.
  - Skip the review and only refactor deploy/delete paths: rejected because it would miss an explicit acceptance criterion.
  - Prefer raw content retrieval as the first addition: rejected because it would add a second new service method with string/binary payload questions and no immediate caller, which is a wider surface than needed for this story.

## Decision 4: Preserve the current deploy/delete semantics exactly while simplifying the code paths

- **Decision**: Treat v87 deploy, v88 deploy-with-optional-wait, and version-specific delete behavior as compatibility-critical paths that must keep their current observable results, including current no-op behavior when consistent delete is requested.
- **Rationale**: The resource service already contains subtly different behavior across versions, especially around deployment confirmation and delete handling. The refactor should make those differences easier to read, not normalize them prematurely into changed behavior.
- **Alternatives considered**:
  - Harmonize v87 and v88 behavior during the refactor: rejected because it risks hidden behavioral change.
  - Change delete behavior for the non-`AllowInconsistent` path: rejected because the issue is a low-risk refactor, not a behavior redesign.

## Decision 5: Expand validation with focused resource service tests, especially for v87

- **Decision**: Validate the refactor with focused versioned service tests, factory tests, and any adjacent integration coverage affected by deployment-confirmation behavior, adding missing v87 service tests rather than relying only on the current v88 coverage.
- **Rationale**: The repository constitution requires automated validation, and the current resource test surface is thinner than nearby service areas. Adding v87-focused tests is the clearest way to protect preserved behavior during the refactor.
- **Alternatives considered**:
  - Depend on factory tests and repository-wide `make test` alone: rejected because they do not cover enough version-specific service behavior.
  - Depend on manual testing of deployments: rejected because `make test` is mandatory before completion.

## Decision 6: Keep documentation updates conditional on user-visible change

- **Decision**: Do not plan a documentation update unless the generated-client coverage review results in a CLI-visible resource capability or output change.
- **Rationale**: The feature is primarily internal, and the constitution explicitly allows documentation to remain unchanged for internal-only work as long as that choice is stated.
- **Alternatives considered**:
  - Always update README and CLI docs: rejected because it would add churn without matching behavior changes.
  - Never update documentation even if a new resource command or output path appears: rejected because it violates the constitution.
