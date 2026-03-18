# PRD: Add Cluster License Command

## Traceability

- Feature name: `63-cluster-license`
- Source status: Derived from Spec Kit artifacts
- Spec: [specs/63-cluster-license/spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/spec.md)
- Plan: [specs/63-cluster-license/plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/plan.md)
- Tasks: [specs/63-cluster-license/tasks.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/tasks.md)

## Problem Statement

Cluster license information is already available through the internal cluster service, but operators do not yet have a repository-native CLI command to retrieve it. The repository needs a low-risk way to expose that capability through the existing `get cluster` hierarchy, preserve current cluster-read behavior patterns, and make the new command discoverable and verifiable through tests and documentation.

## Goal

Add `c8volt get cluster license` as the sole new user-visible command path for cluster license retrieval, reuse the existing service and output patterns, and make the feature safe to implement and review through focused command tests, documentation updates, CLI doc regeneration, and final validation.

## Operator and Maintainer Stories

### Story 1: Retrieve Cluster License Details

As an operator, I want to run `c8volt get cluster license` so that I can inspect the connected cluster's license status from the CLI without making raw API calls.

### Story 2: Understand Failures and Command Usage

As an operator, I want help output and failure behavior for cluster license retrieval to match the rest of the CLI so that I can use the command confidently in interactive and scripted workflows.

### Story 3: Maintain Confidence Through Tests and Docs

As a contributor, I want automated coverage and user-facing documentation for the new command so that the change can be maintained and reviewed without hidden behavior changes.

## Scope

### In Scope

- Add `license` beneath the existing `get cluster` Cobra hierarchy.
- Expose cluster license retrieval at `c8volt get cluster license`.
- Reuse the existing internal cluster service API and `domain.License` output shape.
- Preserve structured success output and established CLI failure/exit semantics.
- Make the command discoverable through `get` and `get cluster` help output.
- Add or update focused command tests for success, help discovery, and failing execution.
- Update `README.md` and `docs/index.md` where cluster read commands are surfaced.
- Regenerate CLI docs under `docs/cli/`.
- Finish with targeted Go test runs and `make test`.

### Non-Goals

- Introducing a legacy or alternate direct command such as `c8volt get cluster-license`.
- Redesigning the cluster service layer, versioned cluster clients, or domain model.
- Adding new dependencies, abstractions, or parallel command hierarchies.
- Editing generated site output under `docs/_site/` by hand.
- Broad documentation rewrites unrelated to the new cluster license command.

## Current Pain Points

- Operators cannot retrieve cluster license details through the public CLI even though the underlying capability already exists.
- The `get cluster` command group lacks a license-specific read path, making the capability undiscoverable through normal help navigation.
- Adding a new user-visible command without explicit tests and docs would create regression and maintenance risk.
- The command surface needs a clear decision about supported paths so later work does not accidentally introduce extra compatibility baggage.

## Target Outcome

- `c8volt get cluster license` is available as a nested command under `get cluster`.
- The command returns the existing structured license payload, including optional fields when present.
- Success and failure behavior align with current `get` command patterns.
- `get` and `get cluster` help make the new command discoverable.
- README, docs index, and generated CLI docs reflect the shipped command behavior.
- Reviewers can verify completion through focused command tests, existing service regression coverage, regenerated docs, and `make test`.

## Requirements

### Functional Requirements

- **AC-001**: The CLI must expose cluster license retrieval at `c8volt get cluster license`.
- **AC-002**: The command must reuse the existing cluster license retrieval capability already provided by the internal cluster service.
- **AC-003**: Successful command output must preserve the CLI's structured JSON output behavior for `domain.License`, including optional fields when present.
- **AC-004**: Missing optional license fields must remain absent rather than being replaced with invented placeholder values.
- **AC-005**: Command failures must preserve established CLI error reporting and exit semantics for `get` workflows.
- **AC-006**: `c8volt get` and `c8volt get cluster` help output must make `license` discoverable as a supported cluster subcommand.
- **AC-007**: The feature must introduce only the nested `c8volt get cluster license` command path and must not add a direct or legacy `c8volt get cluster-license` path.
- **AC-008**: Automated tests must cover successful command execution, help discovery, and representative failing execution.
- **AC-009**: User-facing documentation that surfaces cluster read commands must be updated in the same change set, and CLI docs must be regenerated from Cobra metadata.
- **AC-010**: Final validation must include targeted Go test runs, docs regeneration, and `make test`.

### Invariants

- No new dependencies.
- No redesign of internal cluster service boundaries.
- No alternate command hierarchy outside `get cluster`.
- No hand-editing of generated documentation output when repository generation paths already exist.
- No behavioral regressions to existing cluster read commands.

## Implementation Notes

- Reuse the repository's current Cobra command composition and keep the new command under `get cluster`.
- Route the command directly to `cli.GetClusterLicense(cmd.Context())` and print the result through the existing JSON helper pattern.
- Keep command-level coverage in `cmd/get_test.go`, since the main change is in the public CLI surface.
- Rely on existing service-level license tests in `internal/services/cluster/v87/service_test.go` and `internal/services/cluster/v88/service_test.go` as regression coverage for supported versions.
- Update source docs such as `README.md` and `docs/index.md`, then regenerate `docs/cli/` with `make docs`.
- Keep the feature narrowly scoped to command exposure, tests, and docs rather than service refactoring.

## Execution Shaping

### Work Package 1: Establish the Command Entry Point

- Review the current `get`, `get cluster`, and cluster service touch points.
- Add the base `get cluster license` Cobra command and shared handler.

### Work Package 2: Deliver the Operator-Facing Behavior

- Implement successful license retrieval and structured output.
- Prove the success path through focused command tests.

### Work Package 3: Lock Down Discoverability and Failure Semantics

- Add help-discovery coverage and failing execution coverage.
- Refine help text and command behavior so it matches neighboring `get` commands.

### Work Package 4: Align Documentation and Final Validation

- Update README and docs index references where cluster read commands are described.
- Regenerate CLI docs for the new command.
- Run targeted validation and `make test`.

## Acceptance Criteria by Story

### Story 1 Acceptance Criteria

- `c8volt get cluster license` executes successfully with a valid config.
- The command prints the structured license payload using the repository's standard output pattern.
- Optional license fields remain optional and are not synthesized by the command.

### Story 2 Acceptance Criteria

- `c8volt get` help and `c8volt get cluster` help expose the new command.
- A failing `c8volt get cluster license` execution yields the expected CLI failure semantics and exit code.
- The command surface remains limited to the nested path without a direct `cluster-license` compatibility command.

### Story 3 Acceptance Criteria

- README, `docs/index.md`, and generated CLI docs reflect the new command where cluster read commands are documented.
- Command-level tests cover help, success, and failure scenarios.
- Reviewers can verify the feature outcome without reading source internals beyond the touched command area.

## Validation

- Update `cmd/get_test.go`.
- Add `cmd/get_cluster_license.go`.
- Regenerate affected CLI docs under `docs/cli/`.
- Run `go test ./cmd/... -race -count=1`.
- Run `go test ./internal/services/cluster/... -race -count=1`.
- Run `make docs`.
- Run `make test`.

## Assumptions / Open Questions

- The existing internal cluster service remains the source of truth for supported cluster license fields and version-specific behavior.
- The feature remains limited to the nested command path clarified in the spec; adding a legacy direct command is intentionally out of scope.
- README and docs index updates should stay narrowly focused on cluster-read command discoverability rather than broader documentation restructuring.

## Source Availability

- Available: `spec.md`, `plan.md`, `tasks.md`
- Also referenced for context: `research.md`, `data-model.md`, `quickstart.md`, `contracts/cli-command-contract.md`
- Absent: None
