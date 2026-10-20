# Feature Specification: Bulk Job Completion

**Feature Branch**: `345-bulk-job-completion`

**Created**: 2026-10-20

**Status**: Draft

**Input**: [GitHub issue #345 — feat(job): support bulk completion by filters, multiple keys, and stdin](https://github.com/grafvonb/c8volt/issues/345), plus the operator's request for delivery from the v4.3.4 release baseline independently of accumulated develop changes.

**Release Baseline**: `v4.3.4`, commit `019e4c0bd865b9cf675422b4d8c97c489c36f694`; intended pull-request base: `main`. This specification does not select a new release number or authorize publication.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Complete a Selected Listener Backlog (Priority: P1)

An operator has thousands of process instances whose cancellation is waiting for pending task listeners. They select the intended listener jobs by their existing job-search attributes and tenant, review one batch preview, and explicitly complete the jobs without issuing a command for each job.

The motivating case has three `removeTaskData` jobs in `CREATED` state, with kind `TASK_LISTENER`, event `CANCELING`, retries remaining, and no incidents, distributed across KYC and two Enquiry subprocesses. Completing the jobs acknowledges the listeners' work; it does not execute their external cleanup logic.

**Why this priority**: This directly removes the manual operational bottleneck while making the affected scope and consequence visible.

**Independent Test**: Select a mixed-tenant, mixed-job backlog spanning multiple pages. Preview it and execute it once. Only matching jobs in the intended tenant scope are selected and receive completion attempts, with one outcome per selected job.

**Acceptance Scenarios**:

1. **Given** matching and nonmatching jobs, **When** the operator uses `update job --listener-event-type CANCELING --kind TASK_LISTENER --state CREATED --type removeTaskData --complete --dry-run`, **Then** all filters combine with AND, the preview identifies the selected jobs and their context, and no operational state changes.
2. **Given** a configured tenant, **When** the operator searches for completion candidates, **Then** discovery respects that tenant; a named `--tenant` overrides it, and explicit empty tenant or `--all-tenants` follows established accessible-tenant discovery rules without broadening authorization.
3. **Given** more matches than fit on one page, **When** the operator executes the batch, **Then** discovery finishes before the first mutation, selected keys are deduplicated and frozen, and the command's own completions cannot cause undiscovered jobs to be skipped.
4. **Given** a positive `--limit`, **When** discovery reaches that many unique eligible jobs, **Then** this intentional bounded selection is complete and may execute; the preview states that the selection was limited. `--batch-size` controls page size, not the total selection.
5. **Given** discovery fails, is interrupted, or cannot establish the complete requested scope, **When** the command exits, **Then** it submits no mutations and does not describe partial discovery as a successful empty batch.
6. **Given** a confirmed batch, **When** completing a listener creates another sequential listener, **Then** the new job is excluded from this batch and can be selected by a later invocation.
7. **Given** no matching jobs, **When** normal or dry-run execution completes discovery, **Then** the command succeeds without mutation or confirmation and renders the command-appropriate empty result.

---

### User Story 2 - Complete an Explicit Set of Jobs (Priority: P2)

An operator already has job keys from an investigation or a saved selection. They supply repeated or comma-separated keys, newline-separated stdin keys, or both explicit forms, and apply completion to each unique job through the same batch behavior as filter selection.

**Why this priority**: Enables reproducible execution of reviewed selections and existing shell workflows without forcing rediscovery by filters.

**Independent Test**: Submit duplicate keys through flags and stdin, preview the combined selection, and execute it. Each unique eligible key receives at most one intentional completion submission in the absence of transport uncertainty.

**Acceptance Scenarios**:

1. **Given** repeated and comma-separated `--key` values, **When** the operator requests completion, **Then** every unique valid key is included once and the batch receives one confirmation decision.
2. **Given** newline-separated keys supplied with `-`, optionally combined with explicit `--key` values, **When** the operator requests completion, **Then** all input is read and validated before any writes, and duplicate keys do not duplicate work or outcome rows.
3. **Given** explicit-key or stdin input combined with search selectors, **When** the command is validated, **Then** it rejects the ambiguous selection before mutation. An unqualified `update job --complete` also fails rather than selecting every visible job.
4. **Given** malformed keys or an unreadable explicit target during preparation, **When** preparing the selected set, **Then** the command reports the error and performs no writes against the remaining targets.
5. **Given** explicitly selected authorized keys in different tenants, **When** preparing completion, **Then** existing explicit-key authorization semantics are preserved and actual tenant context is shown; discovery tenant settings do not silently remove explicit targets.
6. **Given** the existing single-key completion command or another existing job update mode, **When** invoked with previously supported arguments, **Then** its established behavior and output contract remain compatible. New collection and selector forms are limited to completion in this feature.

---

### User Story 3 - Operate a Batch with Trustworthy Outcomes (Priority: P2)

An operator or unattended script needs a bounded batch operation whose results remain useful when workers race with it, some requests fail, or the operation is interrupted. They must be able to distinguish completed jobs, accepted requests, skipped targets, and unresolved work without inferring that related processes have terminated.

**Why this priority**: A large recovery must remain auditable and usable even when only part of the selected work succeeds.

**Independent Test**: Execute a prepared selection with mixed successful completions, concurrent changes, rejected requests, and uncertain outcomes. Verify each selected key is accounted for, concurrency is bounded, and unresolved failures produce a nonzero exit status without discarding successful results.

**Acceptance Scenarios**:

1. **Given** an interactive nonempty batch, **When** execution is approved, **Then** one mutation confirmation is written to configured stderr and stdout contains only selected-format results. The preview and confirmation explain that listener completion does not run listener business logic.
2. **Given** `--auto-confirm` or automation mode, **When** a batch runs, **Then** it requires no interactive input and preserves the same scope, validation, and reporting rules.
3. **Given** a job changes after preparation, **When** the command rechecks it, **Then** a job that no longer matches filters receives no completion submission; an already completed job is reported as independently observed completion without attributing the completion to this invocation; canceled or other terminal jobs are not labeled successfully completed.
4. **Given** a concurrent worker completes a job, a completion receives 404/409, or a request times out, **When** results are reconciled, **Then** disappearance or transport failure alone is not treated as proof of completion, and uncertainty is preserved when observations cannot establish the outcome.
5. **Given** configured workers and mixed failures, **When** execution proceeds, **Then** concurrency never exceeds its effective bound. By default independent jobs continue; `--fail-fast` stops new scheduling after failure while preserving outcomes for in-flight and unstarted work.
6. **Given** caller cancellation or deadline expiry during execution, **When** the command ends, **Then** accumulated results remain available and every frozen target is accounted for, including unstarted or uncertain work.
7. **Given** JSON mode, **When** a bulk operation succeeds or partially fails, **Then** stdout contains exactly one shared result envelope with aggregate and per-job results, and no confirmation or progress text.
8. **Given** successful completion acknowledgments for listeners, **When** the batch reports its outcome, **Then** it distinguishes accepted from confirmed completion and makes no claim that related process cancellation or history deletion has finished.

### Edge Cases

- Explicit `--retries 0` is a search predicate when combined with selector-based completion, not a request to change retries; legacy keyed `--complete --retries` incompatibility remains unchanged. The mode-specific distinction must be documented and tested.
- Selection controls such as `--limit`, `--batch-size`, `--workers`, or tenant settings alone do not constitute a job selector. Read-only display flags such as `get job --total` are not completion selectors.
- Empty stdin supplied deliberately with `-` yields an empty selected set; missing selection is an input error. Invalid keys are not silently dropped.
- Sparse pages with continuation are not end-of-discovery; duplicate observations do not consume a unique-target limit twice.
- Jobs in another tenant or outside filters must not consume the eligible-target limit. Failure to enforce the requested tenant scope blocks execution.
- Ongoing external worker activity can change search results during discovery; the command must not promise a transactional snapshot or unlimited backlog exhaustion. Known incomplete discovery blocks writes.
- A worker may claim a job between recheck and completion. Recheck is not a lock; backend acceptance and observations determine the reported outcome.
- `CREATED` and remaining retries do not prove a worker has never attempted a job or that the job is abandoned. Pending listeners may legitimately be processing.
- `CANCELING` can originate from an interrupting BPMN event and does not establish root-process cancellation.
- A selection may include jobs that cannot be completed in their current state. The command must not fix them implicitly by resetting retries, resolving incidents, or issuing cancellation commands.
- Quiet output suppresses informational prose without suppressing explicitly requested JSON; no-wait must not turn an empty result into an accepted mutation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `update job --complete` MUST support repeated/comma-separated keys, newline-separated stdin keys using `-`, and filter-based selection. Explicit flags and stdin MAY combine; explicit inputs and search selectors MUST NOT combine.
- **FR-002**: Filter selection MUST support the existing job-search predicates: state, type, process-instance key, element-instance key, element ID, worker, retries, kind, and listener-event type, with existing normalization and AND semantics. Completion MUST require at least one explicit key source or job predicate. Invalid or ambiguous input MUST fail before mutation.
- **FR-003**: Filter discovery MUST honor configured and explicit tenant selection and accessible-all-tenant behavior. The job-search tenant gap MUST be verified and corrected for the shared discovery behavior used by `get job` and bulk completion. Explicit-key behavior MUST retain backend authorization and report actual tenant context without applying discovery filters to those keys.
- **FR-004**: `--batch-size` MUST affect discovery page size only; `--limit` MUST cap unique eligible targets across pages. Omission of a limit MUST traverse the available complete matching scope. Sparse pages and duplicates MUST NOT prematurely complete discovery or reduce the requested eligible selection.
- **FR-005**: All discovery and preparation MUST finish before writes. The command MUST freeze and deduplicate its selection before confirmation and MUST NOT mutate a partially discovered selection after an error, abort, or unresolved completeness condition. A deliberately reached limit is a completed bounded scope.
- **FR-006**: Dry-run MUST expose the selected count and available per-job process-instance, tenant, type, kind, event, and state context while making zero mutations. Listener previews and confirmations MUST explicitly explain that completing the job acknowledges business work without executing that work. Human summaries MUST stay compact; detailed context MUST be accessible in the preview and structured output.
- **FR-007**: Execution MUST ask for mutation confirmation once per nonempty batch when interactive confirmation is eligible, using configured stderr. Auto-confirm and automation MUST be prompt-free. Declining confirmation MUST produce no mutations and MUST NOT be mislabeled successful completion.
- **FR-008**: Before each completion attempt, the command MUST recheck the job against the prepared selection and current completion eligibility. Changed, completed, other-terminal, missing, and unreadable targets MUST receive distinct truthful dispositions; filter changes MUST prevent submission. Rechecks MUST NOT add new keys or silently replace removed targets to fill a limit.
- **FR-009**: Execution MUST use bounded configurable concurrency, deduplicate work, honor deadlines and caller cancellation, and support `--fail-fast`. Without fail-fast, independent targets MUST continue after individual failures. Unstarted selected targets MUST remain visible in the report.
- **FR-010**: Outcomes MUST distinguish no submission, accepted submission, rejected submission, and uncertain submission, separately from observed completion. Concurrent completion and 404/409/transport errors MUST be reconciled using available observations; absence alone MUST NOT establish completion. Repeated attempts MUST NOT conceal a possibly accepted earlier request or attribute another worker's completion to this invocation.
- **FR-011**: Bulk results MUST account for every frozen key and preserve partial outcomes on failure or interruption. Confirmed already-completed targets MAY be successful no-ops; changed nonmatching, other-terminal, unreadable, rejected, and unresolved targets MUST remain explicit and make the overall invocation unsuccessful when selected completion remains unsatisfied. A response accepting completion MUST NOT be labeled independently confirmed without evidence.
- **FR-012**: Bulk JSON MUST emit one shared result envelope containing aggregate counts and deterministic per-job outcomes, including partial-failure results. The existing single explicit-key response contract MUST remain compatible. A filter or stdin selection uses the bulk contract even when only one job matches. Stdout MUST remain free of interactive control and diagnostic text.
- **FR-013**: A completed empty selection MUST succeed without mutation or confirmation and use the existing mode-aware output conventions, including quiet plus JSON. Preserve supported output modes; this feature does not require adding keys-only output to `update job`. If keys-only is supported by the implementation, empty output MUST contain zero bytes and nonempty output MUST follow the established key-stream contract.
- **FR-014**: Existing single-key completion and unrelated retries, timeout, failure, and BPMN-error operations MUST remain compatible. New bulk selection MUST be rejected for unrelated mutation modes rather than implicitly extending them. Existing payload restrictions MUST remain in force, including restrictions for task-listener completion.
- **FR-015**: Completion MUST NOT automatically reset retries, resolve incidents, cancel/resume processes, delete history, or repeatedly discover newly created jobs. Completion reports MUST NOT imply confirmed process termination. Any required follow-up recovery is a separate invocation.
- **FR-016**: Camunda 8.8, 8.9, and 8.10 MUST receive equivalent supported bulk-completion behavior with their existing version-specific restrictions preserved. Earlier-version behavior on the release baseline MUST not be broadened or removed incidentally.
- **FR-017**: User-facing help, command metadata, README guidance, and generated CLI documentation MUST describe selectors, tenant scope, limits, explicit input combinations, listener consequences, result semantics, and the difference between job completion and process termination.

### Key Entities *(include if feature involves data)*

- **Job selection**: Explicit keys or a conjunction of job predicates, with discovery tenant scope and any eligible-target limit.
- **Prepared completion batch**: The frozen unique job keys, observed job context, selection provenance, and evidence that preparation reached its requested scope.
- **Completion candidate**: A selected job with its owning process instance, tenant, type, kind, event, state, and current eligibility observations.
- **Per-job outcome**: Selected key, submission disposition, observed completion disposition, diagnostic cause, and evidence distinguishing this invocation's action from concurrent changes.
- **Batch outcome**: Aggregate counts and deterministic per-job results, including partial execution, unstarted work, and overall exit status.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator can preview and process a controlled backlog of at least 1,000 eligible jobs spanning multiple result pages using one bulk execution invocation and at most one mutation confirmation, without manually issuing per-job commands.
- **SC-002**: In stable controlled selections, 100% of eligible jobs within the requested limit are accounted for exactly once, with zero completion submissions to jobs outside the selected keys, filters, or tenant scope.
- **SC-003**: Every dry-run, declined confirmation, failed/incomplete discovery, and successful empty-selection case performs zero operational mutations.
- **SC-004**: Every selected key has a reported outcome after mixed-success execution, fail-fast, and interruption. Unresolved failures yield a nonzero exit status, and no uncertain or merely accepted outcome is falsely described as confirmed completion.
- **SC-005**: Interactive checks demonstrate at most one mutation confirmation; unattended checks require no user input; structured-output checks decode exactly one result envelope with no trailing control text.
- **SC-006**: An isolated fake-server acceptance scenario represents three pending canceling listener jobs owned by distinct process instances plus a nonmatching control. Across adapters 8.8/8.9/8.10, preview submits no mutations, execution completes only the frozen matching jobs once, preserves their process context, and reports accepted submission separately from observed completion. The test rejects process mutation/repair/history endpoints and makes no claim of real-engine cancellation or listener business execution.
- **SC-007**: Existing single-job completion and unrelated job-update acceptance scenarios retain their behavior, and the supported completion scenarios pass for Camunda 8.8, 8.9, and 8.10 through appropriate version-specific coverage.

## Validation Scope

User correction, 2026-10-20: implementation validation is limited to isolated tests, mocks, test-owned HTTP servers and local subprocess/terminal checks. The earlier live-recovery requirement in SC-006 is superseded. No real Camunda environment, Kubernetes discovery, connectivity probe, deployment or cleanup is permitted or required to complete this feature. Real-engine cancellation behavior remains unverified and is outside implementation acceptance.

## Assumptions

- This is an explicit operator recovery action. The operator determines whether external listener work has already been performed, will be performed separately, or can be waived. c8volt does not execute `removeTaskData` business logic.
- Filter selection may target any supported completable job kind; the motivating listener use case does not silently restrict the general job command to listeners.
- The first release expands selection only for `--complete`. Other job mutation modes keep their single-key behavior.
- Existing single-job completion returns acceptance without independent confirmation. Preserving that behavior is permitted when confirmation is explicitly reported as skipped; bulk execution must not silently claim stronger proof. Bounded reconciliation is required for uncertain outcomes, not a promise to wait indefinitely for whole process families.
- Search results reflect externally changing state. The frozen batch defines this invocation's scope; it is not a durable snapshot, persistent exactly-once guarantee, or promise that a later query will find no pending jobs.
- Job selection and completion require existing connectivity and authorization. Authorization failure or missing visibility must remain explicit rather than causing silent scope broadening.
- The source issue's reuse and layering constraints are binding planning inputs under repository guidance. Implementation structure, transport details, and task decomposition belong in the subsequent plan rather than this user-behavior specification.
- Work is isolated on the v4.3.4 baseline for a focused follow-up release. It must not depend on unreleased develop changes or on feature #341's cancellation cleanup changes. Forward integration to develop and the new release number are separate release decisions; the v4.3.4 tag remains unchanged.
- This specification adds no automatic listener bypass to ordinary cancellation, no automatic incident repair, no process deletion, and no unbounded recovery loop.
