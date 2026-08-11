# Feature Specification: Slow Analysis Progress After Confirmation

**Feature Branch**: `265-slow-analysis-progress`

**Created**: 2026-08-04

**Status**: Draft

**GitHub Issue**: [#265](https://github.com/grafvonb/c8volt/issues/265) - fix(progress): show slow-analysis progress after confirmation

**Input**: GitHub issue #265 reports that broad `ops analyse slow-process-instances` runs show preflight scope and confirmation, then can appear idle for a long time after the operator confirms.

## Issue Traceability

- **GitHub Issue**: #265
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/265
- **Issue Title**: fix(progress): show slow-analysis progress after confirmation
- **Milestone**: v4.2.2
- **Label**: refactoring

## Clarifications

### Session 2026-08-04

- Q: What should trigger a default human durable milestone during post-confirmation slow analysis? → A: Elapsed time plus progress: show a milestone only when enough time has passed and discovery or timeline counters have advanced.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See Progress After Confirmation (Priority: P1)

As a Camunda operator, I want a broad slow-process analysis to keep showing visible progress after I confirm the expensive run so I can tell c8volt is actively discovering matches and loading timeline data.

**Why this priority**: This is the reported defect. The command already warns about scope before confirmation, but trust is lost when the terminal appears frozen immediately after the operator chooses to continue.

**Independent Test**: Run or simulate a broad slow-process analysis that matches thousands of process instances, confirm the prompt, and verify that default human output visibly reports ongoing analysis progress until the command completes or fails.

**Acceptance Scenarios**:

1. **Given** a broad slow-process analysis has shown scope and asks for confirmation, **When** the operator answers yes, **Then** c8volt continues to show high-level progress for the analysis workflow.
2. **Given** the command is discovering many matching process instances, **When** discovery continues after confirmation and work advances after a noticeable silence interval, **Then** the operator sees a sparse durable progress milestone in default human mode.
3. **Given** the command is loading runtime element timelines for the selected process instances, **When** nested lookup work is underway, **Then** the visible progress remains tied to the slow-analysis workflow rather than disappearing or being replaced by lower-value activity.

---

### User Story 2 - Preserve Quiet Machine Output (Priority: P2)

As an automation author, I want progress improvements to stay out of machine-readable results so existing scripts that use JSON, keys-only, quiet, or automation modes remain stable.

**Why this priority**: c8volt is used interactively and in scripts. Fixing an interactive freeze must not corrupt stdout or change deterministic automation behavior.

**Independent Test**: Run the same broad slow-process scenario in human, JSON, keys-only, quiet, and automation modes, and verify that progress appears only where the selected output contract allows it.

**Acceptance Scenarios**:

1. **Given** JSON output is requested, **When** slow analysis runs after confirmation or auto-confirmation, **Then** stdout remains one valid JSON result without progress text.
2. **Given** keys-only output is requested, **When** slow analysis runs, **Then** stdout contains only keys and no progress, prompt, or milestone text.
3. **Given** quiet or automation behavior is active, **When** slow analysis performs long discovery or timeline work, **Then** human progress text is suppressed according to the existing mode rules.

---

### User Story 3 - Keep Detailed Diagnostics Available (Priority: P3)

As a troubleshooting operator, I want verbose and debug modes to keep providing detailed durable progress while default human mode stays compact.

**Why this priority**: Sparse default milestones solve the stuck-terminal concern, while verbose and debug modes remain the appropriate place for detailed operational trace information.

**Independent Test**: Run the slow-process analysis in verbose and debug modes and verify that detailed progress remains visible without changing default human output into noisy per-request detail.

**Acceptance Scenarios**:

1. **Given** verbose mode is enabled, **When** slow analysis performs discovery and runtime element loading, **Then** durable detailed progress remains available.
2. **Given** debug mode is enabled, **When** nested runtime lookups occur, **Then** diagnostic progress remains available without suppressing the high-level analysis workflow.
3. **Given** default human mode is used, **When** long analysis phases run, **Then** progress is compact and does not show endpoint names, cursors, or per-resource debug detail.

### Edge Cases

- A broad selection matches thousands of process instances and requires multiple discovery pages.
- The operator confirms the preflight prompt and the next phase takes long enough that an idle terminal would look stuck.
- Runtime element lookup activity begins inside a broader slow-analysis workflow and must not hide the higher-level workflow status.
- The terminal does not support transient activity display, or output is redirected, so progress still needs sparse durable milestones where allowed.
- A selection matches zero or very few process instances; progress remains concise and does not add misleading long-running milestones.
- Slow analysis fails during discovery or timeline loading; the operator sees clear failure output without stale progress implying success.
- Machine-oriented modes are combined with broad selectors and confirmation bypass; progress must not leak into stdout.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Default human mode MUST visibly report progress after confirmation for broad slow-process analysis runs.
- **FR-002**: Slow-process analysis progress MUST remain focused on the high-level analysis workflow while nested discovery, lookup, or timeline-loading work is active.
- **FR-003**: Default human mode MUST include sparse durable milestones for long post-confirmation phases when enough time has passed since the last visible milestone and discovery or timeline counters have advanced, so non-transient terminals and redirected human stderr do not appear idle without printing timer-only chatter.
- **FR-004**: Milestone pacing MUST be governed by shared command progress policy rather than behavior unique to slow-process analysis.
- **FR-005**: Milestone pacing thresholds MUST be named and understandable in planning and review artifacts rather than hidden as unexplained numeric values.
- **FR-006**: Default human progress MUST stay compact and avoid endpoint names, request details, cursors, and per-resource lifecycle chatter.
- **FR-007**: Verbose and debug modes MUST continue to expose detailed durable progress for discovery and runtime timeline work.
- **FR-008**: JSON output MUST remain one valid machine-readable result and MUST NOT include progress text on stdout.
- **FR-009**: Keys-only output MUST remain one key per line and MUST NOT include progress, prompt, or milestone text on stdout.
- **FR-010**: Quiet and automation modes MUST preserve their existing suppression and deterministic-output behavior.
- **FR-011**: Services MUST continue to provide structured progress information without owning human milestone wording, output-mode gating, or operator-facing rendering policy.
- **FR-012**: Slow-process command validation MUST prove that progress remains visible after the confirmation prompt in a broad-scope run.
- **FR-013**: Shared progress validation MUST prove that milestone pacing is reusable and output-mode safe.
- **FR-014**: User-facing help or documentation MUST be updated only if the operator-visible progress wording or documented behavior changes.

### Key Entities *(include if feature involves data)*

- **Slow Analysis Workflow**: The operator-level task of finding slow process instances and loading enough timeline detail to analyze them.
- **Post-Confirmation Phase**: The expensive work that begins after an operator approves the broad slow-analysis scope.
- **High-Level Activity Scope**: The visible workflow status that should remain active while lower-level discovery or lookup work happens.
- **Durable Milestone**: A compact progress line written to the human progress channel so an operator can see that long work is advancing even without transient activity.
- **Milestone Pacing Policy**: Shared rules that decide when durable milestones are worth showing in default human mode, requiring both a noticeable elapsed interval and forward progress in discovery or timeline work.
- **Command Output Mode**: The selected output contract, including default human output, verbose/debug diagnostics, JSON, keys-only, quiet, and automation modes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In large fake-volume slow-analysis tests, 100% of default human runs show at least one visible post-confirmation progress update before final completion when work exceeds the elapsed-time threshold and discovery or timeline counters advance.
- **SC-002**: In nested runtime-element lookup scenarios, 100% of validated runs preserve or restore the high-level slow-analysis activity while the workflow remains active.
- **SC-003**: JSON-mode slow-analysis tests produce exactly one valid result with no progress text on stdout.
- **SC-004**: Keys-only slow-analysis tests produce only keys and no progress, prompt, or milestone text on stdout.
- **SC-005**: Quiet and automation-mode tests preserve existing deterministic output behavior with no human progress leakage.
- **SC-006**: Verbose and debug tests continue to show detailed durable progress for discovery and runtime element loading.
- **SC-007**: Shared progress policy tests cover milestone pacing boundaries and pass without depending on slow-analysis-specific rules.
- **SC-008**: Documentation and generated help remain aligned with shipped behavior; if no user-visible help wording changes, the plan records that no docs update is required.

## Assumptions

- Operators run slow-process analysis against broad BPMN process identifiers that may match thousands of process instances.
- The preflight scope and confirmation behavior from the earlier ops-scale progress work remains correct and should not be redesigned by this feature.
- Progress text belongs on the existing human progress channel, not in command result stdout.
- Detailed request, cursor, and per-resource information remains diagnostic and belongs in verbose or debug behavior.
- Shared progress behavior is preferable because other long-running commands can reuse the same milestone pacing rules.
- This feature is a focused follow-up to specs 259 and 262, not a full redesign of progress across every command family.

## Out of Scope

- Changing slow-process analysis selection semantics or result ranking.
- Adding new machine-readable result fields solely for progress.
- Replacing the existing preflight confirmation prompt.
- Redesigning all c8volt progress output beyond the shared pacing behavior needed for this fix.
