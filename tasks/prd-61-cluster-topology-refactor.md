# PRD: Refactor Cluster Topology Command

## Traceability

- Feature name: `61-cluster-topology-refactor`
- Source status: Derived from Spec Kit artifacts
- Spec: [specs/61-cluster-topology-refactor/spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/spec.md)
- Plan: [specs/61-cluster-topology-refactor/plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/plan.md)
- Tasks: [specs/61-cluster-topology-refactor/tasks.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/61-cluster-topology-refactor/tasks.md)

## Problem Statement

Cluster topology retrieval currently lives behind the flat command `c8volt get cluster-topology`, which does not fit the repository’s growing `get <resource>` command hierarchy. The repository needs a low-risk command-tree refactor that introduces `c8volt get cluster topology`, keeps existing automation working through the legacy command during migration, and updates help and documentation so users can discover the preferred path without changing the underlying topology behavior.

## Goal

Move cluster topology retrieval into a repository-native nested Cobra hierarchy while preserving current output, exit behavior, and service execution semantics, then make the migration path explicit through tests and documentation.

## Operator and Maintainer Stories

### Story 1: Use the Nested Command Path

As an operator, I want to run `c8volt get cluster topology` so that cluster topology retrieval follows the same command-tree structure as other `get` workflows without changing the result I receive.

### Story 2: Preserve Existing Automation

As an existing user, I want `c8volt get cluster-topology` to keep working during the migration period so that current scripts and habits do not break immediately.

### Story 3: Understand the Preferred and Deprecated Paths

As a contributor or operator, I want help output and documentation to show the new hierarchy and the deprecated compatibility path so I know which command to use going forward.

## Scope

### In Scope

- Introduce a `cluster` parent command beneath `get`.
- Expose topology retrieval at `c8volt get cluster topology`.
- Preserve `c8volt get cluster-topology` as a working compatibility path.
- Keep the legacy path quiet at runtime and document deprecation in help and docs only.
- Reuse one execution path so both commands preserve the same topology behavior.
- Add or update command-level tests for hierarchy, routing, and compatibility behavior.
- Update README examples, command help expectations, and generated CLI docs for the new hierarchy.
- Finish with targeted Go tests, docs regeneration, and `make test`.

### Non-Goals

- Changing the underlying cluster topology retrieval implementation or domain model.
- Redesigning the cluster service packages or Camunda integration.
- Introducing a new top-level `cluster` command outside the `get` tree.
- Printing runtime deprecation warnings for the legacy command.
- Adding new dependencies, parallel command structures, or broad documentation rewrites unrelated to cluster topology.

## Current Pain Points

- `cluster-topology` is a one-off flat command name instead of a discoverable nested resource path.
- Users reading `get` help cannot naturally find topology under a cluster grouping.
- Migration guidance is currently absent, so the preferred future command path is not obvious.
- A user-visible command change needs explicit tests and documentation to avoid accidental compatibility regressions.

## Target Outcome

- `c8volt get cluster topology` is the preferred, documented command path.
- `c8volt get cluster-topology` remains functional during the deprecation period.
- Both command paths route to the same underlying topology retrieval behavior and preserve existing output and exit semantics.
- `c8volt get` and `c8volt get cluster` help make the new hierarchy discoverable.
- README and generated CLI docs reflect the preferred command and the deprecated compatibility path.
- Reviewers can verify completion through focused command tests, existing cluster regression coverage, regenerated docs, and `make test`.

## Requirements

### Functional Requirements

- **AC-001**: The CLI must expose cluster topology retrieval at `c8volt get cluster topology`.
- **AC-002**: The new nested command path must preserve the current observable topology behavior, including success output, failure behavior, and exit semantics.
- **AC-003**: The legacy command `c8volt get cluster-topology` must continue working during the deprecation period.
- **AC-004**: Help output and affected documentation must identify `c8volt get cluster topology` as the preferred path and `c8volt get cluster-topology` as deprecated but supported.
- **AC-005**: The legacy command must not emit a runtime deprecation warning.
- **AC-006**: The preferred and legacy commands must share the same execution behavior rather than diverging into separate implementations.
- **AC-007**: The `get` command tree must expose `cluster` as a discoverable parent command and `topology` as its topology subcommand.
- **AC-008**: User-facing docs that currently reference `cluster-topology` must be updated in the same change set when they would otherwise become stale.
- **AC-009**: Automated tests must cover the new nested path, the legacy compatibility path, and the help/discoverability expectations.
- **AC-010**: Final validation must include targeted command tests, retained cluster regression checks, regenerated docs, and `make test`.

### Invariants

- No change to the actual cluster topology retrieval logic beyond command wiring reuse.
- No new dependencies.
- No broad refactor of internal cluster services.
- No runtime warning output on the compatibility path.
- No documentation drift between shipped command behavior and generated docs.

## Implementation Notes

- Reuse the repository’s Cobra patterns and keep `cluster` under `get`, not at the root.
- Prefer a shared command execution helper or equivalent shared wiring so both command paths remain behaviorally identical.
- Create command-level test coverage in `cmd/` because the main behavior change is in the CLI surface.
- Preserve inherited root and `get` flags across both command paths.
- Treat generated CLI docs as required outputs for this feature, and update README usage only where cluster-topology examples already exist.
- Do not edit generated site output under `docs/_site/`; regenerate docs from source.

## Execution Shaping

### Work Package 1: Establish Shared Command Structure

- Add command test scaffolding.
- Introduce the `get cluster` parent command.
- Extract or stabilize the shared topology execution path used by all command entries.

### Work Package 2: Deliver the Preferred Nested Command

- Register `topology` beneath `get cluster`.
- Preserve current success, failure, and flag behavior.
- Prove the nested command works through focused command tests.

### Work Package 3: Preserve Compatibility

- Rework the legacy `cluster-topology` entry to reuse the shared execution path.
- Keep existing aliases under review.
- Encode deprecation in help and docs only, with no runtime warning.

### Work Package 4: Make the Change Discoverable and Verifiable

- Update README examples and docs source files that currently reference the legacy path.
- Generate nested CLI docs for `get cluster` and `get cluster topology`.
- Run targeted validation and `make test`.

## Acceptance Criteria by Story

### Story 1 Acceptance Criteria

- `c8volt get cluster topology` is available and discoverable through the new `get cluster` hierarchy.
- The nested command produces the same topology output and failure semantics as the original command behavior.
- Command-level tests cover successful and failing nested command execution.

### Story 2 Acceptance Criteria

- `c8volt get cluster-topology` continues to execute successfully during the migration period.
- The compatibility path preserves the same observable behavior as the preferred nested path.
- The legacy path remains quiet at runtime and does not print a deprecation warning.

### Story 3 Acceptance Criteria

- `c8volt get` help exposes `cluster`, and `c8volt get cluster` help exposes `topology`.
- README and generated CLI docs show the preferred nested path and mark the legacy path as deprecated but supported.
- Reviewers can verify the migration guidance without reading source code.

## Validation

- Add or update `cmd/get_cluster_topology_test.go`.
- Retain or touch `internal/services/cluster/factory_test.go` and adjacent cluster service tests only as needed for regression confidence.
- Run `go test ./cmd/... -race -count=1`.
- Run `go test ./internal/services/cluster/... -race -count=1`.
- Run `make docs`.
- Run `make test`.

## Assumptions / Open Questions

- The deprecation period continues beyond this change, so removing `cluster-topology` is out of scope.
- Existing aliases such as `ct`, `cluster-info`, and `ci` should be reviewed during implementation and preserved unless a repository-aligned reason to change them is documented.
- The feature remains limited to command-tree and documentation changes; no service-level topology redesign is expected.

## Source Availability

- Available: `spec.md`, `plan.md`, `tasks.md`
- Also referenced for context: `research.md`, `data-model.md`, `quickstart.md`, `contracts/cli-command-contract.md`
- Absent: None
