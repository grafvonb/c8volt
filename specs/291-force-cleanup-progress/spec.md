# Feature Specification: Force-Cleanup Progress During Process-Definition Purge

**Feature Branch**: Current branch `develop`; no feature branch created or switched.

**Feature ID**: `291-force-cleanup-progress`

**Created**: 2026-09-04

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/291"

**Source**: [Issue #291 — fix(progress): expose force-cleanup stages during all-process-definitions purge](https://github.com/grafvonb/c8volt/issues/291). Related specification: [#285 semantic progress milestones](../285-semantic-progress-milestones/spec.md).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the Actual Force-Cleanup Stage (Priority: P1)

As an operator running `c8volt ops purge all-process-definitions --force`, I want the activity to identify the work currently happening after confirmation so that a large process-instance cleanup does not appear frozen on “deleting process definitions.”

**Why this priority**: Cancellation, draining, and history deletion can occupy most of the workflow before any process definition is deleted.

**Independent Test**: Execute a forced purge with multiple affected root trees and process definitions, including a period when cancelled instances remain active. Observe activity from confirmation through final deletion.

**Acceptance Scenarios**:

1. **Given** a confirmed purge requires force cleanup, **When** cancellation starts, **Then** activity identifies cancellation of process-instance root trees with completed/total roots and the known affected-instance count before the first root finishes.
2. **Given** cancellation is in progress, **When** a matching root outcome arrives, **Then** activity immediately advances cancellation counters and records failures without advancing history or definition deletion counters.
3. **Given** cancellation finishes and the workflow waits for active instances to disappear, **When** draining starts, **Then** activity says it is waiting for active process instances to drain and remains truthful until the wait ends.
4. **Given** active instances have drained, **When** history deletion starts and root outcomes arrive, **Then** activity identifies process-instance history deletion and updates completed/total root-tree counts after every matching completion.
5. **Given** required cleanup has succeeded, **When** process-definition deletion starts and outcomes arrive, **Then** activity identifies definition deletion and reports completed/total definitions without carrying root-tree counters into this stage.
6. **Given** nested requests or polling occur, **When** they publish activity, **Then** the current workflow stage stays visible without being replaced by lower-level request detail.

---

### User Story 2 - Retain Useful Progress and Failure Evidence (Priority: P2)

As an operator watching a long purge or reviewing its terminal output, I want paced aggregate milestones, immediate failures, and optional item outcomes so that I can assess progress without excessive output.

**Why this priority**: Durable evidence supports long runs and diagnosis after transient activity is repainted.

**Independent Test**: Exercise successes and failures before and after the 10-second milestone threshold, through multiple cleanup stages, in default and verbose human modes.

**Acceptance Scenarios**:

1. **Given** default human mode and progressing cleanup, **When** a matching completion reaches the next 10-second pacing threshold, **Then** a compact durable milestone describes the actual stage and its aggregate counts.
2. **Given** the workflow is draining or no new item has completed, **When** time passes, **Then** elapsed time alone produces no repeated durable informational milestone or fabricated completion.
3. **Given** a root or definition fails, **When** its failed outcome arrives, **Then** the failure is immediately visible regardless of milestone timing, and subsequent progress retains the failure count for that stage.
4. **Given** verbose mode, **When** a root cancellation, root history deletion, or definition deletion finishes, **Then** exactly one completion line for that item and stage identifies its outcome, with no duplicate paced aggregate informational line.
5. **Given** durable progress has been activated and unreported completions remain, **When** the workflow finishes, **Then** remaining progress is flushed once under the existing final-flush policy; a clean workflow finishing before 10 seconds emits no durable milestone.

---

### User Story 3 - Preserve Existing Purge Contracts (Priority: P3)

As an automation author or operator, I want additional visibility to preserve existing outputs and destructive-operation safeguards so that current scripts and cleanup procedures continue to work.

**Why this priority**: This progress correction depends on maintaining established purge behavior.

**Independent Test**: Run equivalent purge cases across supported human, verbose, JSON, automation, quiet, and applicable keys-only modes; compare output contracts, confirmation, requests, outcomes, and exit behavior with established behavior.

**Acceptance Scenarios**:

1. **Given** JSON, automation, or an applicable keys-only mode, **When** nested cleanup progresses or fails, **Then** no human progress text is introduced and established results and error contracts remain intact.
2. **Given** quiet mode, **When** successful items complete, **Then** progress stays suppressed; **When** an item fails, **Then** the existing immediate warning remains visible.
3. **Given** interactive confirmation, declined confirmation, auto-confirmation, dry-run, or a non-force purge, **When** the command executes, **Then** existing safety decisions remain intact and activity describes only stages actually entered.
4. **Given** the same selection and options, **When** stage reporting is enabled, **Then** the frozen mutation scope, worker counts, backend requests, final result, report, and exit behavior remain unchanged.
5. **Given** help or progress describes `--batch-size`, **When** the operator reads it, **Then** it refers only to discovery pages and never implies mutation batching.

### Edge Cases

- No definitions are selected, or selected definitions need no force cleanup: retain empty-result behavior or proceed to definition deletion without inventing cleanup stages.
- Multiple definitions share a root tree: count each unique root once per applicable cleanup stage, while definition deletion uses its own total.
- Completions arrive concurrently or out of order: counters remain exact within each stage and do not mix resource units.
- Affected-instance totals or per-root counts are unavailable: omit unsupported counts; never infer zero or present planned scope as completed work.
- Cancellation or history deletion fails, draining times out, or the operator interrupts the run: retain the actual last stage and visible errors, and never announce unentered stages or successful cleanup.
- All items finish quickly: transient stage changes remain accurate without mandatory durable stage announcements.
- A request is accepted without confirmed completion under existing behavior: use submitted wording rather than claiming a confirmed lifecycle outcome.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: After confirmation or its existing authorized bypass, human activity MUST identify each stage as entered: cancelling process-instance root trees, waiting for active process instances to drain, deleting process-instance histories, and deleting process definitions.
- **FR-002**: Activity MUST reflect actual execution; it MUST NOT announce definition deletion while force cleanup is active or announce a skipped stage.
- **FR-003**: Cancellation and history deletion MUST use unique root trees as their completion unit; definition deletion MUST use process definitions. Each stage MUST display its own trustworthy completed/total counts, including zero completed when the stage begins.
- **FR-004**: Transient activity MUST update after every matching completion, with exact completed and failed counts despite concurrent or out-of-order outcomes. A completion in one stage MUST NOT increment another stage's counters.
- **FR-005**: Cancellation activity MUST include a known affected-instance count. Labels MUST distinguish planned affected scope from cumulative completed affected work; aggregate completed affected counts MUST be omitted if any contributing count is unknown.
- **FR-006**: Draining MUST have explicit waiting activity from wait entry until exit without fabricating completion counts, percentages, or estimated duration.
- **FR-007**: The current stage MUST occupy the existing stable workflow activity during nested requests and polling. Activity MUST stop for confirmation prompts and on command exit.
- **FR-008**: Default durable informational milestones MUST remain completion-driven, compact, and paced at 10 seconds across the workflow. Stage transitions and elapsed time alone MUST NOT create unpaced informational lines. Existing immediate-warning and final-flush exceptions MUST remain intact.
- **FR-009**: Matching failed outcomes MUST be immediately visible in human modes that permit warnings, independent of pacing. Failures MUST remain represented in subsequent aggregates for their stage, and MUST NOT be described as successful work.
- **FR-010**: Verbose mode MUST emit one identity-and-outcome completion line per root or definition within each applicable stage, replacing paced informational milestones. Wording MUST preserve the distinction between submitted, confirmed, and failed work.
- **FR-011**: Progress MUST preserve established JSON, automation, keys-only where supported, and quiet contracts: no human progress in machine modes; successful progress suppressed in quiet mode with immediate failure warnings retained. This feature MUST NOT add output-mode support or flags.
- **FR-012**: Confirmation, dry-run, force eligibility, selection, mutation safety, worker counts, backend requests, retry and wait behavior, result ordering, final output, audit reports, and exit behavior MUST remain unchanged.
- **FR-013**: `--batch-size` MUST remain exclusively a discovery-page setting in behavior and wording. Default progress MUST avoid endpoint, cursor, request, and per-item diagnostic detail.
- **FR-014**: Command-level acceptance coverage MUST exercise the real nested force-cleanup callback sequence, including stage entry, root cancellation completions, draining, root history-deletion completions, and definition-deletion completions. Coverage limited to isolated definition completions is insufficient.
- **FR-015**: Acceptance coverage MUST include pacing boundaries, immediate failures, verbose outcomes, output suppression, shared roots, skipped stages, interruption or failed draining, and unchanged mutation behavior. Command help, README guidance, and generated CLI documentation MUST match the resulting progress behavior.

### Key Entities

- **Purge Workflow**: The confirmed, frozen set of process definitions and required cleanup work, governed by existing safeguards.
- **Cleanup Stage**: The current operator-visible operation with its own resource unit, total, completed count, failure count, and optional trustworthy affected count; draining is a waiting stage.
- **Root Tree**: A unique process-instance root and its affected descendants, counted once within cancellation and once within history deletion when those stages execute.
- **Item Outcome**: A root or definition result associated with a stage and described as submitted, confirmed, or failed.
- **Durable Milestone**: A compact diagnostic line reporting observed stage progress under the established pacing and output policy.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In every full force-cleanup acceptance run, operators see all four actual stages in execution order, with cancellation visible before the first root completes and draining visible throughout its wait.
- **SC-002**: For every acceptance work set, displayed completed/total and failed counts are exact after each matching outcome, never exceed that stage's total, and never include work from another stage.
- **SC-003**: Default human output emits at most one paced informational milestone per 10 seconds, excluding existing final-flush exceptions; idle waits generate zero timer-only milestones and failures are visible when received.
- **SC-004**: Verbose runs show exactly one completion outcome per item per applicable stage and zero duplicate paced informational lines.
- **SC-005**: Every supported machine-output regression case contains zero added human progress text; quiet cases contain zero successful progress messages while retaining immediate failure warnings.
- **SC-006**: Equivalent before-and-after purge cases preserve 100% of confirmation decisions, mutation targets, backend requests, worker settings, final results, reports, and exit outcomes.

## Assumptions

- This specification narrows the existing #285 progress contract to the nested force-cleanup visibility gap in all-process-definitions purge; redesigning other commands is out of scope.
- Existing frozen plans and actual outcomes supply stage totals and affected counts. Additional backend requests solely to enrich progress are out of scope.
- History deletion operates on root trees; completed/total measures root-tree outcomes rather than individually counted history records.
- Ten seconds remains the fixed informational pacing interval. Clean short runs remain silent on the durable progress channel; activated durable progress retains the existing final-flush rules.
- Stage changes affect transient activity immediately. No new durable heartbeat is required during draining.
- Existing output-mode availability, aliases, authentication, tenant selection, and mutation permissions remain authoritative.
- Planning will apply repository architecture, validation, and documentation rules to implementation details; this specification introduces no service restructuring or mutation changes.
