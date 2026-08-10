# Feature Specification: Command Mode And Concern Reorganization

**Feature Branch**: `270-cmd-mode-reorg`

**Created**: 2026-08-10

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/270"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Maintainers Can Navigate Command Modes Safely (Priority: P1)

As a c8volt maintainer, I need distinct command modes and lifecycles to have clear ownership boundaries so I can change watch, polling, progress, reporting, and ordinary command behavior without accidentally modifying unrelated flows.

**Why this priority**: The highest risk identified by the issue is that command files mix multiple modes and concerns, making behavior-preserving changes harder to review and more likely to regress.

**Independent Test**: Can be tested by reorganizing one command mode at a time, reviewing the moved ownership boundaries, and running the targeted tests for that command mode to confirm no user-facing behavior changed.

**Acceptance Scenarios**:

1. **Given** a command has a distinct mode with its own lifecycle, **When** maintainers inspect the command ownership, **Then** the mode-specific execution, state, timing, retry, status, request-building, and refresh behavior are grouped separately from ordinary command wiring.
2. **Given** a base command file is reviewed after reorganization, **When** maintainers inspect its responsibilities, **Then** it remains focused on command construction, flags, validation, top-level dispatch, and ordinary execution.
3. **Given** a behavior-preserving reorganization slice is complete, **When** representative human, JSON, keys-only, quiet, verbose, automation, prompt, and exit-code scenarios are checked where applicable, **Then** they match the pre-reorganization behavior.

---

### User Story 2 - Renderers Stay Focused On Presentation (Priority: P2)

As a maintainer working on command output, I need rendering ownership to be organized by resource and presentation concern so output changes can be made without crossing into backend orchestration or facade calls.

**Why this priority**: Output contracts are central to c8volt's operator trust and script safety, and the issue identifies renderer files that currently mix resource views, layout selection, and non-rendering responsibilities.

**Independent Test**: Can be tested by reviewing each reorganized renderer area and verifying that output tests for the affected resource pass without changes to output text, fields, ordering, or machine-readable contracts.

**Acceptance Scenarios**:

1. **Given** resource-specific output exists for process instances, process definitions, incidents, resources, and tenants, **When** renderer ownership is inspected, **Then** each resource's view construction and rendering are located in the matching focused area.
2. **Given** a renderer area is reorganized, **When** maintainers inspect its responsibilities, **Then** it contains presentation and view-model behavior only, not backend orchestration, mutation planning, or facade calls.
3. **Given** generic flat-row layout behavior is shared across resources, **When** maintainers look for layout ownership, **Then** it is located in a focused shared rendering area rather than hidden inside resource-specific mode selection.

---

### User Story 3 - Complex Command Workflows Have Reviewable Ownership (Priority: P3)

As a maintainer changing large command workflows, I need planning, selection, direct-key execution, worker outcomes, progress, and reports to be separated by concern so future fixes remain small, testable, and reviewable.

**Why this priority**: The issue calls out several large workflows whose current shape makes it difficult to distinguish command wiring, user-facing presentation, operational planning, and workflow execution responsibilities.

**Independent Test**: Can be tested by completing each reorganization slice independently, reviewing that the moved concerns have focused ownership, and running the affected command, workflow, and report tests before moving to the next slice.

**Acceptance Scenarios**:

1. **Given** a command supports both selector-based execution and direct-key execution, **When** the command is reorganized, **Then** each path is independently identifiable and can be tested without inspecting unrelated execution paths.
2. **Given** a workflow reports progress or writes reports, **When** its ownership is reviewed, **Then** progress selection, milestone pacing, terminal formatting, user-facing rendering, and report serialization are separated by concern while preserving workflow semantics.
3. **Given** a helper appears unused, **When** maintainers consider removing it, **Then** the helper is removed only after production and test callers are checked and the affected tests still pass.

### Edge Cases

- A file is large but still cohesive; size alone must not force a split.
- A split would create an artificial abstraction that is harder to understand than the original ownership.
- A command mode shares setup with ordinary execution but owns distinct lifecycle state, timing, polling, progress, or request-building behavior.
- A renderer currently performs non-rendering work that must move without changing output contracts.
- A helper appears unused in production code but is still needed by tests, examples, or subprocess scenarios.
- User-visible output could change unintentionally due to moved formatting, ordering, prompt, progress, or report behavior.
- Generated documentation or command capability metadata could change unintentionally during behavior-preserving moves.
- A potential ownership correction is larger than mechanical reorganization and should become follow-up work rather than being hidden in this feature.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The reorganization MUST preserve all existing CLI flags, aliases, help text, output text, output fields, ordering, prompts, exit behavior, command metadata, and backend calls unless an intentional follow-up change is explicitly separated from this feature.
- **FR-002**: Base command ownership MUST remain focused on command construction, flag definition, validation, top-level dispatch, and ordinary execution.
- **FR-003**: Distinct command modes with their own execution lifecycle, timing, retry, polling, status, progress, repaint, or request-building behavior MUST have focused ownership separate from ordinary command wiring.
- **FR-004**: Process-definition watch ownership MUST be separated from ordinary process-definition lookup ownership while preserving the behavior established by the process-definition watch repaint work.
- **FR-005**: Resource view ownership MUST be organized so process-instance, process-definition, incident, resource, and tenant rendering can be reviewed and tested by resource.
- **FR-006**: View ownership MUST be limited to view-model construction, layout, and rendering behavior; it MUST NOT own facade calls, backend orchestration, traversal, polling, mutation planning, or workflow execution.
- **FR-007**: Generic flat-row layout behavior MUST have focused shared rendering ownership rather than being embedded in resource-specific render-mode selection.
- **FR-008**: Process-instance dry-run presentation MUST remain separate from dry-run planning and facade interaction.
- **FR-009**: Process-instance paging and progress support MUST be divided by search request construction, paging progress, shared search progress, and mutation-result ownership where those responsibilities are distinct.
- **FR-010**: Job update ownership MUST be divided so command wiring, request parsing, worker-outcome handling, and planning concerns can be reviewed independently.
- **FR-011**: Process-instance cancel and delete workflows MUST keep selector or search execution distinguishable from direct-key execution while preserving destructive-operation safety and reporting semantics.
- **FR-012**: Root command ownership MUST separate root wiring, configuration resolution, and service installation responsibilities.
- **FR-013**: Slow-process analysis ownership MUST separate command behavior, validation behavior, and progress behavior.
- **FR-014**: Operations progress and reporting ownership MUST separate progress mode selection, milestone pacing, terminal formatting, user-facing rendering, report-file behavior, Markdown report behavior, JSON report behavior, and workflow-specific report serialization where those concerns differ.
- **FR-015**: Test ownership MUST follow production ownership for reorganized command modes, renderers, workflows, helpers, and subprocess scenarios.
- **FR-016**: Dead command helpers MUST be removed only after confirming they have no production or test callers.
- **FR-017**: Backend traversal, retries, polling, mutation planning, worker execution, and version-specific behavior MUST remain outside command ownership.
- **FR-018**: The existing single command package boundary MUST be retained unless reorganization reveals a proven ownership boundary that justifies a separately planned change.
- **FR-019**: The feature MUST use small, reviewable slices grouped by concern, and file moves MUST be completed before any behavioral or ownership corrections in the same area.
- **FR-020**: Any ownership correction that exceeds mechanical reorganization MUST be documented as a follow-up rather than blended into behavior-preserving file moves.

### Key Entities *(include if feature involves data)*

- **Command Mode**: A distinct user-facing execution path such as watch, polling, streaming, follow, batch, direct-key execution, selector-driven execution, dry-run, or interactive operation.
- **Base Command Area**: The command ownership area responsible for construction, flags, validation, top-level dispatch, and ordinary execution.
- **Renderer Area**: The ownership area responsible for view-model construction, layout decisions, and terminal or machine-readable presentation.
- **Workflow Concern**: A separable operational responsibility such as planning, selection, paging, progress, worker outcomes, report generation, or report serialization.
- **Behavior Contract**: The observable CLI promises for flags, aliases, help, output, prompts, exit codes, metadata, backend calls, and automation-safe behavior.
- **Reorganization Slice**: A small, reviewable unit of work grouped by one command mode, resource renderer, workflow, report concern, or test ownership boundary.
- **Follow-Up Ownership Correction**: A non-mechanical change identified during reorganization that needs separate planning because it may affect behavior, architecture, or review scope.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of reorganized command modes with distinct lifecycle behavior have focused ownership separate from ordinary command wiring.
- **SC-002**: 100% of reorganized renderer areas are free of facade calls, backend orchestration, traversal, polling, mutation planning, and workflow execution responsibilities.
- **SC-003**: 100% of touched tests are aligned with the production ownership area they verify.
- **SC-004**: 100% of removed helpers have documented production and test caller checks before removal.
- **SC-005**: Targeted tests pass for each touched command mode, renderer, workflow, or report area before that reorganization slice is considered complete.
- **SC-006**: The command package test suite passes after each major reorganization slice.
- **SC-007**: Final validation passes the full repository test target and whitespace checks before completion.
- **SC-008**: Generated CLI documentation shows no unintended differences after behavior-preserving reorganization.
- **SC-009**: Representative human, JSON, keys-only, quiet, verbose, automation, prompt, and exit-code outputs remain unchanged for 100% of applicable changed command paths.
- **SC-010**: Maintainers can identify the owner of command wiring, mode lifecycle, rendering, paging progress, mutation outcomes, configuration resolution, service installation, and report serialization for every reorganized area without inspecting unrelated command files.

## Assumptions

- The feature is scoped to GitHub issue #270 and builds on the command-layer ownership improvements from issue #254.
- The process-definition watch behavior introduced by issue #268 is treated as the compatibility baseline for watch-related reorganization.
- This feature is primarily a behavior-preserving maintainability effort; intentional behavior changes belong in separate follow-up work.
- Existing tests, generated documentation, command metadata, and representative CLI output provide the compatibility baseline.
- The command package remains one package for this feature unless a separate issue and plan justify a stronger package boundary.
- Documentation changes are not expected unless validation reveals an unintended generated documentation difference or a planned follow-up changes user-visible behavior.
