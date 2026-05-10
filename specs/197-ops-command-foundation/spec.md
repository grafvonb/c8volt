# Feature Specification: Ops Command Foundation

**Feature Branch**: `197-ops-command-foundation`
**Created**: 2026-05-10
**Status**: Draft
**Input**: GitHub issue #197, "feat(ops): add ops command foundation and shared workflow contracts"
**Issue URL**: https://github.com/grafvonb/c8volt/issues/197
**Issue Title**: feat(ops): add ops command foundation and shared workflow contracts
**Mandatory Implementation Context**: Planning, task generation, and every Ralph implementation iteration MUST read and apply `specs/ralph-implementation-rules.md`. Ralph MUST NOT be launched unless implementation instructions include `--implementation-context specs/ralph-implementation-rules.md`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Discover Ops Command Family (Priority: P1)

As a c8volt maintainer, I can run `c8volt ops --help` and see a clear top-level grouping command for high-level operational workflows without requiring Camunda runtime configuration.

**Why this priority**: The root grouping command is the foundation that all later ops workflows depend on.

**Independent Test**: Run `c8volt ops --help` without a configured Camunda runtime and verify the command is discoverable, help text is useful, and no workflow behavior executes.

**Acceptance Scenarios**:

1. **Given** no runtime Camunda configuration is available, **When** the user runs `c8volt ops --help`, **Then** help output describes the ops command family and exits successfully.
2. **Given** existing top-level commands are available, **When** command discovery or help generation runs, **Then** existing commands remain present and unchanged.

---

### User Story 2 - Discover Execute Grouping Command (Priority: P2)

As a c8volt maintainer, I can run `c8volt ops execute --help` and see a grouping command for predefined operational playbooks that discover target sets and execute existing resource actions.

**Why this priority**: Future execution workflows need a stable parent command before concrete playbooks are added.

**Independent Test**: Run `c8volt ops execute --help` and verify it describes the command group without requiring runtime configuration or executing any playbook.

**Acceptance Scenarios**:

1. **Given** the ops command family exists, **When** the user runs `c8volt ops execute --help`, **Then** help output describes predefined operational playbooks and exits successfully.
2. **Given** no concrete execute workflow has been implemented by this feature, **When** the user inspects `c8volt ops execute`, **Then** no orphan cleanup, retention policy, or smoke test workflow is present.

---

### User Story 3 - Discover Repair Grouping Command (Priority: P3)

As a c8volt maintainer, I can run `c8volt ops repair --help` and see a grouping command for repair and remediation workflows without ambiguous target key semantics.

**Why this priority**: Repair workflows need a clear parent command, and avoiding an ambiguous top-level key prevents future command contract drift.

**Independent Test**: Run `c8volt ops repair --help` and verify it describes repair workflows, performs no remediation, and does not expose an ambiguous top-level `--key` flag.

**Acceptance Scenarios**:

1. **Given** the ops command family exists, **When** the user runs `c8volt ops repair --help`, **Then** help output describes repair and remediation workflows and exits successfully.
2. **Given** target-specific repair workflows are out of scope, **When** the user inspects `c8volt ops repair --help`, **Then** it does not accept or advertise a top-level `--key` flag.

---

### User Story 4 - Establish Shared Ops Workflow Contracts (Priority: P4)

As a maintainer adding future ops workflows, I have lightweight shared conventions for workflow metadata, automation compatibility, dry-run behavior, report output, and step statuses.

**Why this priority**: Shared contracts reduce duplication for upcoming workflow issues while keeping this feature from implementing concrete playbooks.

**Independent Test**: Inspect the source-level contracts and command metadata for the ops command family, then verify future workflow conventions are available without adding concrete workflow behavior.

**Acceptance Scenarios**:

1. **Given** future ops workflows will be state-changing, **When** maintainers add concrete subcommands, **Then** they can reuse documented conventions for mutation metadata, automation compatibility, dry-run behavior, report file handling, report format inference, and structured report rendering.
2. **Given** automation paths need deterministic output, **When** future commands support `--automation --json`, **Then** the contract keeps JSON on stdout and progress or log output off stdout.
3. **Given** this feature is only foundational, **When** shared helpers are added, **Then** they are limited to immediately useful workflow contracts and do not implement concrete operational playbooks.

---

### User Story 5 - Regenerate User-Facing Command Documentation (Priority: P5)

As a user reading generated CLI documentation, I can find the ops command family documentation generated from source command definitions.

**Why this priority**: Documentation should reflect new user-facing command structure while preserving the repository's generated-doc workflow.

**Independent Test**: Regenerate command documentation through the repository generation path and verify ops docs are present while generated files are not hand-edited.

**Acceptance Scenarios**:

1. **Given** generated command documentation exists, **When** documentation is refreshed, **Then** ops command documentation appears from source command metadata.
2. **Given** existing command documentation exists, **When** docs are regenerated, **Then** unrelated command documentation remains consistent with existing behavior.

### Edge Cases

- Grouping commands must not create a CLI client or require Camunda configuration before rendering help.
- Grouping commands must not perform workflow behavior when invoked for help or command discovery.
- `c8volt ops repair` must avoid ambiguous target input such as a top-level `--key`; target-specific repair commands will define their own key semantics later.
- Future status values must be consistent enough for reports to compare steps without each workflow inventing incompatible names.
- Documentation updates must come from source metadata and the existing generation workflow, not manual edits to generated CLI pages.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose a top-level `ops` grouping command for high-level operational workflows.
- **FR-002**: The `ops` grouping command MUST describe the ops command family in help output.
- **FR-003**: The system MUST expose an `ops execute` grouping command for predefined operational playbooks that discover a target set and execute existing resource actions.
- **FR-004**: The system MUST expose an `ops repair` grouping command for repair and remediation workflows.
- **FR-005**: `ops`, `ops execute`, and `ops repair` MUST be grouping commands only and MUST NOT perform concrete workflow behavior directly.
- **FR-006**: Help for `ops`, `ops execute`, and `ops repair` MUST be available without requiring runtime Camunda configuration.
- **FR-007**: `ops repair` MUST NOT accept an ambiguous top-level `--key` flag.
- **FR-008**: Future ops workflows MUST follow the layering model `cmd -> ops command facade -> resource services -> generated Camunda clients`.
- **FR-009**: Ops command facades MUST orchestrate workflows, coordinate steps, aggregate results, and build report models, but MUST NOT own resource-specific API logic, waiters, polling, hierarchy traversal, deletion, or update behavior.
- **FR-010**: Missing primitive resource capabilities needed by future ops workflows MUST be added to the correct resource service or existing facade instead of being implemented ad hoc inside ops commands.
- **FR-011**: The feature MUST establish shared workflow conventions for state-changing command metadata, automation compatibility checks, dry-run behavior, report file behavior, report format behavior, structured report models, deterministic automation JSON stdout, and progress or log output separation.
- **FR-012**: Shared step statuses MUST include at least `planned`, `skipped`, `submitted`, `confirmed`, `confirmation_failed`, `blocked`, and `failed`.
- **FR-013**: Shared helpers or contracts MUST remain lightweight and immediately useful for upcoming ops workflows.
- **FR-014**: The feature MUST NOT implement `ops execute orphan-cleanup`, `ops execute retention-policy`, `ops execute smoke-test`, `ops repair incident`, or `ops repair process-instance`.
- **FR-015**: When generated CLI documentation exists, the feature MUST update source command definitions and regenerate documentation through the repository's existing generation workflow.
- **FR-016**: Planning, task generation, and Ralph implementation instructions MUST include `specs/ralph-implementation-rules.md`; Ralph MUST NOT be launched without passing `--implementation-context specs/ralph-implementation-rules.md`.

### Key Entities

- **Ops Command Family**: The top-level grouping surface for operational workflows.
- **Ops Execute Group**: A grouping surface for future predefined playbooks that discover targets and execute existing resource actions.
- **Ops Repair Group**: A grouping surface for future repair and remediation workflows with target-specific semantics.
- **Ops Workflow Contract**: Shared conventions for metadata, automation compatibility, dry-run semantics, report output, and step status values.
- **Structured Report Model**: A workflow-neutral representation that future ops commands can render as Markdown or JSON.
- **Workflow Step Status**: A constrained status term used to summarize planned, skipped, submitted, confirmed, blocked, failed, or confirmation-failed workflow steps.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `c8volt ops --help`, `c8volt ops execute --help`, and `c8volt ops repair --help` all complete successfully without requiring Camunda runtime configuration.
- **SC-002**: Help output for each new grouping command clearly communicates its purpose in one command invocation.
- **SC-003**: `c8volt ops repair --help` contains no ambiguous top-level `--key` flag.
- **SC-004**: The implementation introduces no concrete ops playbook commands listed as out of scope.
- **SC-005**: Existing top-level command behavior and generated command documentation remain unchanged except for the addition of ops command family documentation.
- **SC-006**: Future workflow issues can add concrete ops subcommands without redefining the `ops`, `ops execute`, or `ops repair` root structure.
- **SC-007**: The generated tasks for this feature are split into independently verifiable work units suitable for one Ralph iteration each.

## Assumptions

- The primary user is a c8volt maintainer or operator who already understands existing command conventions.
- The feature should prefer existing Cobra command, command contract, output rendering, and documentation generation patterns.
- Concrete workflow issues #183, #186, #187, and #188 will supply their own target-specific behavior later.
- The ops command foundation may define small shared contracts only where they are directly needed by upcoming ops workflows.
- The repository's generated CLI documentation path remains the source of truth for user-facing command docs.

## Out of Scope

- Implementing `ops execute orphan-cleanup`.
- Implementing `ops execute retention-policy`.
- Implementing `ops execute smoke-test`.
- Implementing `ops repair incident`.
- Implementing `ops repair process-instance`.
- Adding resource-specific Camunda API logic inside ops commands.
- Adding broad abstractions that are not needed by the first ops workflows.
