# PRD: Add Cluster License Command

## Traceability

- Feature name: `63-cluster-license`
- Source status: Derived from Spec Kit artifacts
- Spec: [specs/63-cluster-license/spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/spec.md)
- Plan: [specs/63-cluster-license/plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/plan.md)
- Tasks: [specs/63-cluster-license/tasks.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/63-cluster-license/tasks.md)

## Problem Statement

The repository already supports cluster license retrieval in the internal cluster service layer, but operators cannot access that capability through the CLI's nested `get cluster` command tree. The project needs a low-risk CLI feature that exposes the existing license capability at `c8volt get cluster license`, keeps the change aligned with current Cobra and output patterns, and ships with the tests and documentation updates required for a user-visible command.

## Goal

Expose existing cluster license retrieval through a discoverable nested CLI command while preserving established output, error handling, and exit behavior for `get` commands.

## Operator and Maintainer Stories

### Story 1: Retrieve Cluster License Details

As an operator, I want to run `c8volt get cluster license` so that I can inspect the connected cluster's license status without making raw API calls.

### Story 2: Understand Failures and Command Usage

As an operator, I want help output and failure behavior for cluster license retrieval to match nearby CLI commands so that I can use the command confidently in interactive and scripted workflows.

### Story 3: Maintain Confidence Through Tests and Docs

As a contributor, I want automated coverage and user-facing documentation for the new command so that the feature can be reviewed and maintained without hidden behavior changes.

## Scope

### In Scope

- Add the nested command path `c8volt get cluster license` beneath the existing `get cluster` hierarchy.
- Reuse the current cluster service capability and domain model for license retrieval.
- Print the existing cluster license payload through the CLI's standard structured output behavior.
- Preserve current `get` command error reporting and exit semantics for license retrieval failures.
- Make `license` discoverable in `c8volt get` and `c8volt get cluster` help output.
- Add or update focused command tests for help discovery, successful execution, and failing execution.
- Update user-facing documentation where cluster read commands are surfaced.
- Regenerate CLI reference docs from Cobra metadata.
- Finish with targeted Go validation and `make test`.

### Non-Goals

- Redesigning the internal cluster service, domain model, or Camunda client integration.
- Adding a legacy or alternate direct command path such as `c8volt get cluster-license`.
- Changing the observable behavior of existing cluster read commands.
- Introducing new dependencies, abstractions, or parallel command structures.
- Broad documentation rewrites beyond the command surfaces affected by this feature.

## Current Pain Points

- Operators cannot retrieve cluster license details through the CLI even though the internal capability already exists.
- Users exploring the `get` command tree cannot discover license retrieval as a supported cluster workflow.
- A new user-visible command needs explicit help, tests, and docs to avoid regressions in behavior or discoverability.
- Optional or malformed upstream license responses need defined CLI handling so maintainers do not infer behavior from implementation details alone.

## Target Outcome

- `c8volt get cluster license` becomes the only supported user-visible command path for cluster license retrieval.
- Successful license results reuse the CLI's standard structured output behavior and existing domain payload.
- Failure behavior matches the repository's established `get` command handling and exit semantics.
- Help output under `get` and `get cluster` makes the command discoverable without reading source code.
- README, `docs/index.md`, and generated CLI docs stay aligned with the shipped command behavior.
- Reviewers can validate completion through focused command tests, retained service-level regression confidence, regenerated docs, and `make test`.

## Requirements

### Functional Requirements

- **AC-001**: The CLI must expose cluster license retrieval at `c8volt get cluster license`.
- **AC-002**: The command must reuse the existing cluster service capability to retrieve the connected cluster's license information, including validity and license type, and include additional license attributes when provided by the connected Camunda version.
- **AC-003**: Successful command execution must preserve the CLI's established structured output behavior for `get` commands.
- **AC-004**: License retrieval failures must preserve the CLI's established error reporting and exit semantics for `get` commands.
- **AC-005**: The `get` help tree must make `license` discoverable through `c8volt get --help` and `c8volt get cluster --help`.
- **AC-006**: The command must define and preserve observable behavior when optional license fields are absent.
- **AC-007**: The command must surface malformed or empty successful responses clearly rather than silently inventing missing values.
- **AC-008**: The feature must introduce only the nested path `c8volt get cluster license` and must not add a legacy or alternate direct command such as `c8volt get cluster-license`.
- **AC-009**: The feature must remain bounded to CLI exposure of the existing license capability and must not redesign the underlying cluster service behavior.
- **AC-010**: Automated tests must cover help discovery, successful license retrieval, and representative failure behavior.
- **AC-011**: User-facing documentation and generated CLI references affected by the new command must be updated in the same change.
- **AC-012**: Final validation must define how maintainers verify the feature through targeted command tests, docs regeneration, and `make test`.

### Invariants

- No change to the underlying cluster service contract beyond reusing the existing license retrieval capability.
- No new direct or deprecated compatibility command path for cluster license retrieval.
- No new dependencies or non-native abstractions.
- No behavioral regression in existing `get cluster` command workflows.
- No hand-edited generated CLI docs when the repository already provides `make docs`.

## Implementation Notes

- Reuse the repository's current Cobra command patterns under `cmd/`, specifically the existing `get` and `get cluster` hierarchy.
- Keep the implementation localized to command wiring, shared execution behavior, focused `cmd` tests, and user-facing documentation updates.
- Use the existing cluster domain payload and standard JSON or structured output helper rather than introducing command-specific translation layers.
- Preserve inherited root and `get` flags on the new nested command.
- Follow the repository testing conventions for Cobra commands, including explicit temp `--config` usage and subprocess assertions when failures exit via `os.Exit`.
- Regenerate CLI reference pages from Cobra metadata with `make docs` rather than editing `docs/cli/` output by hand.

## Execution Shaping

### Work Package 1: Establish the Command Entry Point

- Review the current `get` and `get cluster` command tree and adjacent test surfaces.
- Add the base `get cluster license` Cobra command in the existing command layout.
- Wire the command through the current CLI construction path without introducing a parallel structure.

### Work Package 2: Deliver the User-Visible Retrieval Flow

- Connect the new command to the existing cluster license service.
- Preserve successful output formatting and handling for missing optional fields.
- Encode malformed-response handling explicitly where the command surface requires it.

### Work Package 3: Lock Down Discoverability and Failure Semantics

- Add help discovery coverage for `get`, `get cluster`, and `get cluster license`.
- Add failure-path coverage that verifies the established exit-code and error-handling behavior.
- Refine command help text and descriptions so the nested path is obvious and consistent.

### Work Package 4: Align Documentation and Prove Completion

- Update README and `docs/index.md` anywhere cluster read commands are surfaced.
- Regenerate CLI docs for the new command tree.
- Run targeted command and cluster service validation, then finish with `make test`.

## Acceptance Criteria by Story

### Story 1 Acceptance Criteria

- `c8volt get cluster license` executes successfully against a valid environment and returns the existing cluster license payload.
- Optional license fields can be absent without causing the command to invent values or fail unnecessarily.
- Command-level tests cover the successful execution path.

### Story 2 Acceptance Criteria

- `c8volt get --help` and `c8volt get cluster --help` make the `license` subcommand discoverable.
- `c8volt get cluster license --help` describes the supported nested command path clearly.
- Representative upstream, transport, or authentication failures preserve the CLI's established failure and exit behavior.

### Story 3 Acceptance Criteria

- README, `docs/index.md`, and generated CLI references show `c8volt get cluster license` wherever affected cluster read commands are documented.
- Reviewers can verify success, failure, and help behavior through automated tests without reading implementation details.
- The repository validation flow includes targeted Go tests, docs regeneration, and `make test`.

## Validation

- Update or add focused command coverage in `cmd/get_test.go`.
- Retain existing cluster service regression coverage under `internal/services/cluster/...` as supporting protection for the reused license capability.
- Run `go test ./cmd/... -race -count=1`.
- Run `go test ./internal/services/cluster/... -race -count=1`.
- Run `make docs`.
- Run `make test`.

## Assumptions / Open Questions

- The internal cluster service already exposes the required license retrieval capability for supported Camunda versions, so no service redesign is required for this feature.
- The new command should live only under the existing `get cluster` hierarchy and should not create a legacy compatibility path.
- The command surface may need to make malformed successful responses explicit during implementation if the current helper behavior is ambiguous.

## Source Availability

- Available: `spec.md`, `plan.md`, `tasks.md`
- Also referenced for context: `research.md`, `data-model.md`, `quickstart.md`, `contracts/cli-command-contract.md`
- Absent: None
