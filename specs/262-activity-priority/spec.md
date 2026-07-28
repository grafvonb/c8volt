# Feature Specification: Preserve High-Level Activity

**Feature Branch**: `262-activity-priority`

**Created**: 2026-07-28

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/262"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Keep Workflow Progress Visible (Priority: P1)

An operator runs a long-running Camunda operation and sees the meaningful command-level progress remain visible while the command performs nested request, wait, polling, or lookup work.

**Why this priority**: This is the core UX defect. Large-scale operations are only trustworthy when operators can keep seeing the highest-value status, such as what resource set is being processed and how far the command has progressed.

**Independent Test**: Can be fully tested by running or simulating a long-running command with nested Camunda lookups and verifying the visible activity line stays on the command workflow message instead of switching to lower-level wait or request wording.

**Acceptance Scenarios**:

1. **Given** a process-instance deletion is showing high-level delete progress, **When** the command performs nested process-instance loads or waits, **Then** the visible activity remains focused on the delete workflow.
2. **Given** a slow-process analysis is showing discovery or timeline progress, **When** the command performs nested runtime-element searches, **Then** the visible activity remains focused on the analysis workflow.
3. **Given** a bulk run command is showing start progress for multiple process instances, **When** individual create or confirmation calls happen, **Then** the visible activity remains focused on the bulk run workflow.

---

### User Story 2 - Preserve Useful Fallback Activity (Priority: P2)

An operator runs a Camunda-backed command that does not have a broader workflow progress message and still receives compact, resource-aware activity text while c8volt waits for Camunda.

**Why this priority**: Simple commands still benefit from feedback. The fallback should be understandable, but it must not compete with richer workflow progress when such progress exists.

**Independent Test**: Can be fully tested by running representative simple Camunda-backed commands and verifying their transient activity text names the resource being loaded, searched, deployed, updated, or deleted.

**Acceptance Scenarios**:

1. **Given** no higher-level activity is active, **When** c8volt waits for a Camunda resource search, **Then** the visible activity names the searched resource family.
2. **Given** no higher-level activity is active, **When** c8volt waits for a Camunda resource load, **Then** the visible activity names the loaded resource family.
3. **Given** c8volt uses a known Camunda operation, **When** transient activity is displayed, **Then** generic wording such as "submitting Camunda API request" is not shown.

---

### User Story 3 - Keep Scripted Output Stable (Priority: P3)

An automation user runs commands in non-interactive modes and receives the same deterministic output as before, without transient progress or activity text contaminating machine-readable streams.

**Why this priority**: c8volt must remain safe for scripts and pipelines. Improving interactive activity must not change JSON, keys-only, quiet, or automation output contracts.

**Independent Test**: Can be fully tested by running representative commands in machine-oriented modes and verifying stdout and stderr remain free of transient activity text.

**Acceptance Scenarios**:

1. **Given** a command runs with JSON output, **When** nested Camunda work occurs, **Then** the result remains valid JSON without activity text.
2. **Given** a command runs with keys-only output, **When** nested Camunda work occurs, **Then** stdout contains only keys.
3. **Given** a command runs in quiet or automation mode, **When** nested Camunda work occurs, **Then** transient activity is suppressed as before.

### Edge Cases

- A lower-level activity starts before a higher-level workflow message is known; once the workflow message appears, it becomes the visible activity.
- A higher-level workflow completes while lower-level activity is still in progress; the visible activity can fall back to the remaining lower-level scope if appropriate.
- Multiple activities at the same importance level overlap; the visible message remains stable and balanced until scopes complete.
- A command writes a durable warning, prompt, error, or final outcome while activity is visible; the durable output remains readable and is not mixed with spinner text.
- A known Camunda operation has no command-level workflow progress; the fallback activity still uses resource-aware wording.
- An unknown future Camunda operation is encountered; generic fallback wording may appear, but only when no better label is known.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST preserve the highest-value active command or workflow activity message over nested lower-level activity messages.
- **FR-002**: The system MUST allow lower-level request, wait, and polling activity to run without replacing a visible higher-level workflow message.
- **FR-003**: The system MUST show lower-level fallback activity when no higher-level workflow or batch activity is active.
- **FR-004**: The system MUST restore the most appropriate remaining activity message when a higher-level activity completes while other activity is still active.
- **FR-005**: The system MUST keep default human-mode activity transient and compact.
- **FR-006**: The system MUST preserve existing durable preflight, prompt, warning, verbose, debug, and final outcome wording.
- **FR-007**: The system MUST keep JSON, keys-only, quiet, and automation outputs free of transient activity text.
- **FR-008**: Known Camunda resource operations used by c8volt commands MUST have resource-aware fallback activity wording.
- **FR-009**: Generic fallback wording MUST remain available for unknown operations, but known c8volt command paths SHOULD NOT rely on it.
- **FR-010**: Representative long-running command families MUST be covered by tests that prove high-level activity is not overwritten by nested lower-level activity.
- **FR-011**: Representative simple Camunda-backed commands MUST be covered by tests that prove fallback activity remains useful when no higher-level workflow progress exists.

### Key Entities *(include if feature involves data)*

- **Activity Message**: A transient, operator-visible status line that describes what c8volt is currently waiting on or processing.
- **Activity Scope**: A bounded unit of work that can start, update, and finish while a command is running.
- **Workflow Activity**: A high-value activity message tied to an operator task, such as discovery, analysis, bulk mutation, dependency expansion, or exact progress through a known work set.
- **Fallback Activity**: A lower-value activity message tied to a single Camunda request, wait, or lookup, shown only when no higher-value activity is active.
- **Command Output Mode**: The selected output contract, including default human output, verbose/debug diagnostics, JSON, keys-only, quiet, and automation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In representative long-running command scenarios, 100% of nested lower-level activity attempts preserve the visible higher-level workflow message while that workflow remains active.
- **SC-002**: In representative simple Camunda-backed scenarios without higher-level workflow progress, 100% of known resource operations show resource-aware fallback activity instead of generic API wording.
- **SC-003**: Existing machine-oriented output contract tests for JSON, keys-only, quiet, and automation modes continue to pass with no transient activity text in result streams.
- **SC-004**: At least one representative command from each high-risk family is covered: process-instance get/search, process-instance mutation, process-definition mutation, deployment, wait/expect, and ops analysis or repair.
- **SC-005**: Operators observing long-running commands can identify the active high-level task and progress phase for the full duration of nested work in validated scenarios.
- **SC-006**: Unknown future Camunda operations still produce a fallback activity message rather than no activity at all when terminal activity is otherwise enabled.

## Assumptions

- This feature is a follow-up to issue #259 and preserves the same UX goals for preflight, progress, prompt, dry-run, and final outcome behavior.
- The primary users are Camunda operators and automation authors running c8volt in both interactive and scripted environments.
- The feature applies to terminal activity indicators and transient progress only; durable output wording is changed only where necessary to preserve consistency.
- The command list will be audited by behavior category, not by adding bespoke logic to every command.
- Representative command coverage is sufficient when the shared behavior is centralized and all command families use the same activity mechanism.
