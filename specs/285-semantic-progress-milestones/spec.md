# Feature Specification: Semantic Progress Milestones for Long-Running Commands

**Feature Branch**: `285-semantic-progress-milestones`

**Created**: 2026-08-31

**Status**: Draft

**GitHub Issue**: [#285](https://github.com/grafvonb/c8volt/issues/285) - feat(progress): add semantic milestones for long-running commands

**Input**: GitHub issue #285 requests stable semantic progress and paced durable milestones for long-running commands without changing mutation safety or output contracts.

## Issue Traceability

- **GitHub Issue**: #285
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/285
- **Issue Title**: feat(progress): add semantic milestones for long-running commands
- **Milestone**: v4.3.1

## Clarifications

### Session 2026-08-31

- Q: When an affected-item count is known for only some completed work items, what should aggregate progress display? → A: Omit the aggregate count if any contributing count is unknown.
- Q: When an eligible operation succeeds before the first 10-second pacing interval, should it emit a final durable progress milestone? → A: No milestone for clean sub-10-second operations.
- Q: In verbose mode, should per-item completion lines replace paced aggregate informational milestones? → A: Verbose per-item lines replace paced informational milestones.
- Q: How should immediate per-item failure warnings behave in quiet and automation modes? → A: Quiet shows failure warnings; automation suppresses them.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Understand Live Work Without Spinner Churn (Priority: P1)

As a Camunda operator running a long mutation or operational workflow, I want one stable activity message to show meaningful completed, affected, and failed counts so that I can tell whether useful work is progressing without lower-level activity replacing the workflow status.

**Why this priority**: Long-running commands currently preserve a stable spinner but can remain uninformative for minutes. Accurate aggregate progress is the primary operator value and must remain readable under concurrent work.

**Independent Test**: Run an eligible command against a frozen multi-item work set while nested request, wait, polling, and batch activity occurs; verify the workflow activity remains visible and its exact aggregate counters advance after every completed item.

**Acceptance Scenarios**:

1. **Given** a command has frozen 64 work items, **When** 18 items complete, **Then** the stable workflow activity reports 18 of 64 completed while nested activity does not replace it.
2. **Given** completed items expose a trustworthy affected-resource count, **When** each result is incorporated, **Then** the activity reports the cumulative affected count and current failure count.
3. **Given** an affected-resource count is unavailable or not trustworthy, **When** progress is rendered, **Then** that count is omitted rather than estimated or shown as zero.
4. **Given** work completes concurrently and out of input order, **When** completion events arrive, **Then** aggregate completed, affected, and failed counts remain exact and never decrease.

---

### User Story 2 - Retain Durable Evidence During Long Operations (Priority: P2)

As an operator monitoring a command over time or reviewing captured terminal output, I want occasional compact milestones and immediate failure warnings so that important progress remains visible after transient activity is repainted.

**Why this priority**: A transient spinner helps during an interactive run but does not provide durable evidence. Paced milestones improve confidence and troubleshooting without flooding default output.

**Independent Test**: Drive a deterministic sequence of successes and failures across more than one milestone interval; verify paced informational lines, immediate warnings, a final flush, and verbose per-item detail all use the same cumulative facts.

**Acceptance Scenarios**:

1. **Given** an eligible operation continues for at least 10 seconds and makes progress, **When** the next completion crosses the pacing interval, **Then** one compact informational milestone is written to the diagnostic stream.
2. **Given** many items complete rapidly in default mode, **When** less than 10 seconds of progressing operation time separates completions, **Then** the command does not emit one durable informational line per item.
3. **Given** an item fails, **When** its completion is received, **Then** a warning is emitted immediately without waiting for the next informational milestone and cumulative progress is retained.
4. **Given** durable progress has been activated by the pacing threshold or a failure warning and the operation ends with progress accumulated since the last durable line, **When** the workflow finishes, **Then** the final accumulated milestone is flushed exactly once.
5. **Given** verbose mode is enabled, **When** an item completes, **Then** a durable line identifies that item and its outcome while aggregate counters remain correct and no paced aggregate informational milestone duplicates it.

---

### User Story 3 - Preserve Command and Automation Contracts (Priority: P3)

As an operator or automation author using direct keys, stdin keys, search selection, waiting, no-wait submission, or machine-readable output, I want progress semantics to remain consistent without changing command results, safety checks, or parseable output.

**Why this priority**: Progress is valuable only if it does not corrupt scripts, misstate lifecycle outcomes, or alter the confirmed mutation scope.

**Independent Test**: Execute equivalent direct-key and search-selected mutations in waited and no-wait modes across human, verbose, JSON, keys-only, quiet, and automation output; compare results, exit behavior, reports, and progress gating with the existing contracts.

**Acceptance Scenarios**:

1. **Given** a no-wait mutation returns after request acceptance, **When** progress is reported, **Then** the outcome is described as submitted rather than cancelled, deleted, repaired, or confirmed.
2. **Given** a waited mutation has been operationally confirmed, **When** completion is reported, **Then** the command uses the confirmed lifecycle verb appropriate to the operation.
3. **Given** JSON or keys-only output is requested, **When** the command runs, **Then** stdout remains valid and unchanged and no human progress text is emitted.
4. **Given** quiet or automation mode is active, **When** work progresses successfully, **Then** no transient or durable human progress chatter is emitted.
5. **Given** search-selected destructive work requires confirmation, **When** discovery and dependency expansion finish, **Then** planning activity stops before the prompt and confirmed mutation starts a fresh workflow-priority activity scope.
6. **Given** equivalent roots are selected directly, from stdin, or by search, **When** they are mutated, **Then** all paths report the same completion unit and lifecycle semantics.
7. **Given** an item fails, **When** quiet mode is active, **Then** its immediate warning remains visible; **When** automation mode is active, **Then** no human progress warning is emitted and structured errors, results, and reports remain authoritative.

### Edge Cases

- The selected or planned work set is empty; progress adds no artificial completion or affected counts, and final command output retains its established empty-result behavior.
- An eligible operation completes successfully before the first pacing interval; it emits no durable milestone and retains its established final command output.
- The first completion is a failure; the warning is immediate and later success milestones retain the failure count.
- Every item fails; completed and failed totals converge accurately without claiming a successful lifecycle state.
- A no-wait request is accepted for some items and rejected for others; accepted items are submitted and rejected items are failed without using confirmation verbs.
- Completion events arrive simultaneously or out of input order; aggregation is race-safe and counters never exceed the frozen total.
- A process-definition cleanup plan has an affected count for only some items; the aggregate affected count is omitted because the complete scope is not trustworthy.
- Discovery spans multiple pages before confirmation; discovery progress is clearly distinct from mutation completion and is never labeled as mutation batches.
- A prompt is displayed after planning; no spinner corrupts or overwrites the prompt.
- A durable line is emitted while transient activity is active; the activity is cleared and restored without duplicated, interleaved, or corrupted terminal text.
- The context is cancelled or the command exits early; already accumulated failures remain visible and no false final success is reported.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Eligible long-running commands MUST retain one workflow-priority transient activity that nested request, polling, wait, and batch activity cannot replace.
- **FR-002**: For a frozen work set with a trustworthy total, transient activity MUST update after every completed work item and show the exact completed and total counts.
- **FR-003**: Progress MUST maintain cumulative completed, affected, and failed counts from structured completion facts and MUST remain correct under concurrent completion.
- **FR-004**: Progress MUST show an aggregate affected-resource count only when every contributing count for the displayed scope is trustworthy; if any contributing count is unavailable, the entire aggregate affected count MUST be omitted.
- **FR-005**: The completion unit and affected-count label MUST match the command family, such as process definitions, process-instance root trees, affected process instances, repairs, or smoke-test stages.
- **FR-006**: Default human mode MUST emit a compact informational milestone on the first completion that occurs at least 10 seconds after the progress scope starts or after the preceding informational milestone.
- **FR-007**: Default human mode MUST NOT emit one informational line for every rapidly completed item.
- **FR-008**: A failed completion MUST emit an immediate warning independently of informational milestone pacing.
- **FR-009**: Immediate failure warnings MUST preserve cumulative progress so later milestones and the final flush include all completed and failed work.
- **FR-010**: When durable progress has been activated by the pacing threshold or a failure warning and an eligible workflow finishes with unreported accumulated progress, it MUST flush one final durable milestone; a clean workflow finishing before the first pacing threshold MUST emit no durable milestone.
- **FR-011**: Verbose mode MUST emit one durable completion line per item, including operator-facing identity and outcome, instead of paced aggregate informational milestones; transient aggregate activity and immediate failure severity remain available.
- **FR-012**: Durable progress and warnings MUST use the diagnostic stream and MUST preserve the active transient display without corruption or duplicate repainting.
- **FR-013**: Lifecycle wording MUST distinguish submitted, confirmed operation-specific outcomes, and failed outcomes.
- **FR-014**: No-wait execution MUST use submitted wording after request acceptance and MUST NOT claim confirmation.
- **FR-015**: Direct-key, stdin-key, and search-selected execution of the same mutation MUST use the same progress units, status meanings, and aggregate behavior.
- **FR-016**: Search-selected execution MUST report discovery separately, freeze the selected and dependency-expanded scope, stop activity before interactive confirmation, and start a fresh mutation activity after confirmation.
- **FR-017**: Discovery pages MUST NOT be described as mutation batches or completed mutation work.
- **FR-018**: Required process-definition coverage MUST include all-process-definition purge, basic process-definition deletion, and long-running deployment batches with known totals.
- **FR-019**: Required process-instance coverage MUST include cancel and delete operations for explicit keys, stdin keys, search selection, force cancellation before deletion, waited execution, and no-wait execution.
- **FR-020**: Required operational-workflow coverage MUST include retention-policy cleanup, orphan process-instance purge, incident-selected process-instance purge, incident repair, process-instance repair, and smoke-test execution stages.
- **FR-021**: Other long-running start/run, analysis, search, wait, and expect workflows MUST be assessed individually and receive semantic milestones only when they expose useful completed work or stage information.
- **FR-022**: Default-mode progress MUST remain aggregate-first and MUST NOT expose endpoints, cursors, raw requests, or concurrent per-resource lifecycle chatter.
- **FR-023**: Existing low-level per-resource diagnostic detail MUST remain available only where verbose behavior already permits it.
- **FR-024**: JSON and keys-only stdout MUST remain unchanged and parseable, with no transient or durable human progress text.
- **FR-025**: Quiet mode MUST suppress non-error transient and durable human progress but MUST retain immediate failure warnings. Automation mode MUST suppress all transient and durable human progress, including per-item failure warnings, and MUST rely on existing structured errors, results, and audit reports.
- **FR-026**: Final human output, structured results, ordering, confirmation, dry-run, force, fail-fast, no-wait, mutation safety, exit behavior, and report contents MUST remain unchanged.
- **FR-027**: The feature MUST add no progress-related CLI flag and MUST NOT change worker counts or concurrency behavior.
- **FR-028**: Automated coverage MUST exercise concurrent completions, activity arbitration, durable-output clearing and restoration, pacing, immediate failures, final flushing, lifecycle wording, and suppression in every machine-oriented mode.
- **FR-029**: Command help, README guidance, examples, and generated CLI documentation MUST describe any user-visible progress behavior introduced by the feature.

### Key Entities

- **Progress Scope**: One operator-visible long-running workflow with a command-family label, optional frozen total, cumulative completion facts, and an active output policy.
- **Completion Fact**: One structured work-item outcome containing optional identity, operator-facing label, optional trustworthy affected count, and a lifecycle status such as submitted, confirmed, or failed.
- **Progress Aggregate**: The cumulative completed, affected, and failed counts derived from completion facts; it is monotonic for the lifetime of one progress scope.
- **Durable Milestone**: A compact informational or warning line representing progress since the prior durable emission together with current cumulative scope state.
- **Output Policy**: The active human, verbose, JSON, keys-only, quiet, or automation contract that determines whether transient or durable progress may be displayed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of nested-activity tests, the workflow activity remains the visible transient status while request, polling, wait, and batch scopes start and finish beneath it.
- **SC-002**: For every tested frozen work set, completed and failed counters are exact after each completion, never decrease, and never exceed the total.
- **SC-003**: In default mode, progressing operations emit no more than one informational milestone per 10-second pacing interval, excluding immediate failures and the final flush.
- **SC-004**: Every failure warning is emitted on the same completion event that records the failure, without waiting for a later pacing interval.
- **SC-005**: For every workflow whose durable progress was activated, 100% of accumulated progress is represented by the last emitted milestone and no final partial milestone is duplicated; clean sub-10-second workflows emit zero durable milestones.
- **SC-006**: In verbose-mode tests, every completed work item produces exactly one durable identity-and-outcome line and zero paced aggregate informational milestones.
- **SC-007**: For the same mutation scope, direct-key, stdin-key, and search-selected paths produce identical completed, failed, and affected aggregates.
- **SC-008**: In 100% of no-wait acceptance tests, accepted work is labeled submitted and no confirmed lifecycle verb is used.
- **SC-009**: Existing JSON and keys-only regression fixtures remain byte-for-byte parseable with zero human progress lines on stdout.
- **SC-010**: Across all required command families, quiet-mode tests observe zero non-error progress chatter while retaining each immediate failure warning, and automation tests observe zero human progress lines including per-item failure warnings.
- **SC-011**: All existing mutation result, ordering, safety, exit, and report regression tests pass unchanged.
- **SC-012**: Every required command family has at least one automated acceptance scenario covering transient progress, durable progress or justified suppression, failures, and final state.

## Assumptions

- The existing workflow activity priority remains authoritative and continues to prevent lower-priority scopes from replacing the workflow message.
- Ten seconds is the default durable informational pacing interval and does not become a user-configurable flag in this feature.
- The pacing clock begins when the progress scope starts and resets after each informational milestone; informational output remains completion-driven rather than timer-driven.
- Pacing advances only when real completion progress occurs; idle time alone does not create repetitive durable lines.
- Verbose per-item completion replaces default paced informational milestones rather than adding a second aggregate durable stream.
- Clean workflows that finish before the first pacing interval remain silent on the durable progress channel; once pacing or a failure warning activates durable progress, any later unreported accumulation is flushed at completion.
- Existing plans, frozen scopes, and mutation responses remain the authoritative sources for totals, identities, affected counts, and lifecycle outcomes.
- Process-definition affected counts describe cleanup scope rather than claiming that every related process instance has already been mutated.
- Concurrent completion order does not redefine final result ordering or report contents.
- Existing authorization, tenant visibility, error conversion, output-mode selection, and command aliases remain unchanged.
- Quiet mode retains immediate failure warnings; automation represents failures only through its established structured contracts.

## Out of Scope

- Changing worker counts, concurrency scheduling, retry policy, or the command execution model.
- Reusing discovery batch size as a mutation batch size.
- Adding a progress-related CLI flag or configuration option.
- Printing endpoints, cursors, request payloads, or transport details in default progress.
- Moving operator-facing wording into service-layer completion facts.
- Changing confirmation, dry-run, force, fail-fast, wait, or no-wait semantics.
- Redesigning final human output, JSON envelopes, keys-only output, audit reports, or mutation result ordering.
- Re-enabling noisy concurrent per-resource diagnostic logs in default mode.

## Implementation Governance

- Planning, task generation, and every Ralph implementation iteration MUST read and apply `specs/ralph-implementation-rules.md`.
- Ralph MUST NOT be launched unless `--implementation-context specs/ralph-implementation-rules.md` is included in the implementation instructions.
- Commit subjects for this issue-backed work MUST use Conventional Commits format and append `#285` as the final token.
