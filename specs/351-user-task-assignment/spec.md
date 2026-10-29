# Feature Specification: Assign and Unassign User Tasks by Explicit Key

**Feature Branch**: `351-user-task-assignment`

**Created**: 2026-10-28

**Status**: Draft

**Input**: [GitHub issue #351](https://github.com/grafvonb/c8volt/issues/351), updated 2026-10-28 — Extend `update user-task` with explicit assignment and unassignment for one or more task keys, preserving variable updates and the implemented completion workflow from #311.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Assign Explicitly Selected Tasks Safely (Priority: P1)

As an operator, I want to assign one or several known tasks to the same user without accidentally replacing an existing assignee, so ownership changes match my explicit intent.

**Why this priority**: Explicit selection and protection against unintended reassignment are the central value of the feature.

**Independent Test**: Using isolated task responses, assign a single task and a collection with duplicate keys; inspect the selected identities and override policy and compare successful responses with conflicts.

**Acceptance Scenarios**:

1. **Given** one valid task key and a nonblank backend identity, **When** I run `update user-task --key <key> --assign <assignee>`, **Then** that task receives one logical assignment operation for the supplied identity with replacement explicitly prohibited, and success is reported only after the backend's successful response.
2. **Given** keys supplied through repeated flags, comma-separated values, newline-separated stdin with positional `-`, or their supported combination, **When** I assign tasks, **Then** each eligible unique valid key receives the same identity and override policy, duplicates cause no extra logical operation, and results retain first-input order.
3. **Given** a task already assigned to another user, **When** I assign without `--force`, including with `--auto-confirm` or `--automation`, **Then** a successful lookup identifying another assignee skips the task with conflict status before confirmation or mutation; any later native request still prohibits replacement, so a race remains a backend conflict.
4. **Given** an existing assignment, **When** I specify `--force`, **Then** replacement is explicitly permitted and that permission appears in the preview and any interactive confirmation.
5. **Given** assignment to the current assignee, a missing task, an invalid identity, or an authorization failure, **When** the backend accepts or rejects the operation, **Then** its result is preserved; no rejection is converted to synthetic success.
6. **Given** an explicit key outside the configured discovery tenant, **When** the backend authorizes access, **Then** assignment is permitted under the existing explicit-key tenant contract; denied access remains a failure.
7. **Given** no keys, empty stdin without other keys, malformed keys, invalid workers, a blank assignee, or an incompatible mode or payload flag, **When** I invoke assignment, **Then** validation fails before any mutation, without discovery or a successful no-op.

---

### User Story 2 - Remove Explicit Task Assignments (Priority: P1)

As an operator, I want to remove the assignee from selected tasks so they can be reassigned through the normal task workflow.

**Why this priority**: Unassignment is the second core operation and must work independently of assignment and completion.

**Independent Test**: Unassign one task and a deduplicated collection with isolated success and rejection responses; verify selection, outcomes, and absence of assignment or completion operations.

**Acceptance Scenarios**:

1. **Given** one or more explicit task keys, **When** I run `update ut --key <keys> --unassign`, **Then** each unique key receives one logical unassignment operation, and successful responses are reported in selection order.
2. **Given** an already unassigned task, **When** the backend accepts or rejects unassignment, **Then** that result is preserved without a preliminary read, skipped request, or suppressed error.
3. **Given** `--unassign` together with assignment, completion, a variable payload, `--force`, or enabled `--no-wait`, **When** I execute the command, **Then** incompatible input is rejected before mutation.
4. **Given** an alias, stdin selection, a configured discovery tenant, or a backend authorization failure, **When** I unassign tasks, **Then** the same explicit-selection and authorization contract as assignment applies.

---

### User Story 3 - Preview and Approve the Exact Change (Priority: P2)

As an operator, I want to inspect the selected tasks, requested action, and reassignment permission before approving a change, so I can catch selection or ownership mistakes.

**Why this priority**: A useful preview and one clear approval protect both interactive operators and scripted workflows.

**Independent Test**: Exercise dry-run, interactive approval, decline, EOF, auto-confirm, and automation with isolated dependencies and real terminal stdin, capturing result and prompt streams separately.

**Acceptance Scenarios**:

1. **Given** valid assignment or unassignment input, **When** I use `--dry-run`, **Then** the command previews the frozen unique selection and action, including assignee and override permission for assignment, without backend requests, confirmation, or success claims.
2. **Given** interactive execution eligible for confirmation, **When** I review and approve the frozen operation, **Then** one plain confirmation appears on configured or inherited stderr, identifies the action and scope and applicable assignment details, and only the approved selection is submitted.
3. **Given** a declined confirmation or EOF, **When** approval does not succeed, **Then** existing abort behavior is preserved and no mutations occur.
4. **Given** auto-confirm or automation, **When** I execute valid input, **Then** confirmation is bypassed according to existing conventions, while the selected identity and override permission remain unchanged.
5. **Given** any non-dry-run assignment or unassignment selection, **When** execution is prepared, **Then** no completion-specific advisory task reads, completed-task skipping, or unassigned-task warnings occur; protected assignment checks ownership once before approval, and backend mutation responses determine the outcomes of submitted work.

---

### User Story 4 - Understand Bulk Progress and Uncertain Outcomes (Priority: P2)

As an operator, I want every selected task accounted for when some operations fail or take too long, so I can distinguish confirmed results from unresolved or unattempted work.

**Why this priority**: Bulk execution must preserve evidence and prevent unsafe assumptions after partial failure.

**Independent Test**: Use isolated delayed, successful, rejected, timed-out, and canceled responses with sequential and bounded concurrent execution, including fail-fast and applicable retries.

**Acceptance Scenarios**:

1. **Given** `--workers 1`, **When** several tasks are changed, **Then** operations execute sequentially; with a selected larger bound, active operations never exceed that bound, and worker auto-selection follows existing conventions.
2. **Given** a backend response delayed by blocking task listeners, **When** an operation is submitted, **Then** the command awaits its response subject to existing timeouts, reports success only after a documented successful response, and performs no additional polling, read-back, or downstream waiting.
3. **Given** mixed successful and failed operations, **When** execution finishes, **Then** every selected key has an ordered outcome, partial successes remain available, and the established failure class and exit behavior are preserved.
4. **Given** fail-fast execution and a failure, **When** scheduling stops, **Then** no additional work is scheduled after the failure is observed, already submitted outcomes are retained, and unattempted tasks are distinguished from canceled or failed submissions without implying rollback.
5. **Given** timeout, cancellation, or ambiguous transport failure after submission, **When** the result is reported, **Then** backend rejection is not asserted without evidence; cancellation before submission leaves work not attempted.
6. **Given** an ambiguous attempt followed by a rejection during an applicable retry, **When** results are assembled, **Then** the earlier uncertainty remains visible; a later documented success may establish acceptance.

---

### User Story 5 - Use Reliable Automation and Diagnostics (Priority: P2)

As an operator or script author, I want stable results and compact diagnostics in the requested output mode so I can safely consume successful keys and investigate failures.

**Why this priority**: Assignment changes need the same script-safe output and logging guarantees as existing mutations.

**Independent Test**: Execute both operations using isolated responses under human, JSON, keys-only, quiet, verbose, automation, and DEBUG modes; separately capture stdout, stderr, and configured logger destinations.

**Acceptance Scenarios**:

1. **Given** JSON output, including quiet plus JSON, **When** preview, success, validation failure, or partial failure is rendered, **Then** exactly one shared envelope is emitted followed by end of output, with applicable partial evidence and the original exit behavior and no second error rendering.
2. **Given** keys-only dry-run, **When** selection includes duplicates, **Then** stdout contains each unique preview key once in selection order; during execution it contains only keys with successful native responses, one per line. No qualifying keys produces zero bytes, and partial failure retains its failure exit status.
3. **Given** ordinary human execution, **When** a preview, successful summary, or verbose outcome is rendered, **Then** it uses configured INFO logging, leaves stdout empty, honors configured or inherited destinations and plain or structured formatting, and is suppressed at WARN level or under quiet.
4. **Given** repeated backend failures containing distinctive resource keys, methods, URLs, bodies, and transport details, **When** normal or verbose-only execution fails, **Then** one compact aggregate ERROR groups bounded reasons and counts without raw HTTP detail on either stream; quiet suppresses informational output but retains the compact error.
5. **Given** either `--debug` or configured DEBUG logging, **When** isolated exchanges occur with or without verbose, **Then** existing diagnostic visibility, redaction, destination, and flag precedence apply without contaminating result stdout or adding a separate diagnostic mechanism.
6. **Given** interactive control text alongside selected-format output, **When** a prompt is eligible under existing output guardrails, **Then** it remains plain configured stderr text, never a logger record or result payload.

---

### User Story 6 - Preserve Existing Task Workflows (Priority: P2)

As an existing user, I want assignment options to leave variable updates, completion, and task queries unchanged so current commands and scripts continue to work.

**Why this priority**: The feature extends an established command and must not alter its other operations.

**Independent Test**: Run representative existing variable-update, completion, resolution, and task-query scenarios against isolated dependencies across the three supported versions.

**Acceptance Scenarios**:

1. **Given** no active assignment, unassignment, or completion mode, **When** I update variables, **Then** existing payload requirements, scope planning, verification, no-op behavior, output, and `--no-wait` behavior remain unchanged.
2. **Given** completion mode, **When** I complete tasks, **Then** existing advisory reads, completed-task handling, warnings, optional variables, response semantics, and rendering remain unchanged; assignment is never implicit.
3. **Given** existing user-task reads, resolution, variable queries, or `get pi --has-user-tasks`, **When** those commands run, **Then** their observable behavior remains unchanged.
4. **Given** Camunda 8.8, 8.9, or 8.10, **When** I assign or unassign tasks, **Then** the same user-facing contract applies; unsupported versions retain existing generic rejection behavior.
5. **Given** command and parent-group help, README guidance, or generated CLI documentation, **When** I consult it, **Then** examples explain the four separate modes, explicit override permission, local previews, response waiting, and the absence of additional polling.

### Edge Cases

- Duplicate keys across flags and stdin produce one logical operation per unique key, with deterministic first-input order even when responses finish out of order.
- Empty key flag occurrences, empty comma-separated components, malformed keys, and invalid input late in a collection invalidate the whole selection before mutation. Empty stdin alone is invalid; empty stdin combined with valid explicit keys does not erase those keys.
- Blank assignees are invalid. Valid backend identities are not restricted to email addresses or resolved from display names or the authenticated user.
- Explicitly false boolean mode flags do not activate a mode; an explicitly supplied empty assignment value remains invalid rather than falling through to variable updates. Existing false-value and flag-precedence conventions remain the baseline.
- Same-assignee assignment and already-unassigned removal preserve the backend response. A task's state changing during approval does not trigger extra reads or compensating operations; the native override policy still applies.
- All tasks failing yields zero keys-only stdout bytes but a failure exit. Missing input is an error, not a successful empty operation.
- Dry-run cannot assert identity validity, task existence, permission, or guaranteed success, since it performs no backend requests.
- An aborted or canceled invocation is not a successful no-op; an in-flight timeout does not prove rejection or rollback.
- Fail-fast retains in-flight results and does not label unscheduled tasks as submitted. Applicable retries remain part of the same logical per-key operation.
- Malformed or undocumented success responses do not establish success. A documented successful response confirms that operation, not independent read-back or permanent ownership.
- Quiet plus machine output retains requested results under existing precedence. Configured or inherited stderr destinations and terminal stdin with redirected stdout retain existing prompt eligibility and routing.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `update user-task` MUST support assignment through `--assign <assignee>` and unassignment through `--unassign`, retaining `user-tasks`, `ut`, and `uts` aliases and existing completion and variable-update modes.
- **FR-002**: Assignment and unassignment MUST accept one or more explicit keys through repeated or comma-separated `--key`/`-k` and newline-separated stdin selected by positional `-`, deduplicating in first-input order without discovery-based selection.
- **FR-003**: All local input MUST be validated before mutation. Missing selection, empty stdin without other keys, malformed keys, explicit empty key components, invalid worker values, and empty or whitespace-only assignees MUST fail without mutations or successful no-op results. Key validity MUST retain the existing 16-decimal-digit task-key grammar.
- **FR-004**: Assignment, unassignment, and completion MUST be mutually exclusive active modes. Assignment and unassignment MUST reject variable payload flags and enabled `--no-wait`; `--force` MUST be valid only with assignment. Without an active mutation-mode selector, existing variable-update behavior MUST remain unchanged.
- **FR-005**: Assignment MUST apply the same supplied backend identity and override policy to every selected task, without identity lookup, email-only validation, display-name resolution, or inference of the authenticated user.
- **FR-006**: Assignment MUST prohibit replacing an existing assignee unless an interactive replacement question is explicitly approved or `--force` enables replacement. Force skips confirmation; it is valid only with assignment. Approval authorizes replacement at execution time. That policy MUST be enforced with the submitted mutation rather than inferred from a prior read or backend defaults. Automation and auto-confirm MUST NOT grant override permission.
- **FR-007**: Unassignment MUST remove the selected task's assignment through the backend's normal unassignment operation. Both modes MUST preserve native success or rejection semantics for submitted operations. Protected assignment may skip observed different-assignee tasks with failed/not_submitted evidence and conflict exit status; it MUST NOT suppress errors to synthesize success.
- **FR-008**: Each unique key MUST receive at most one logical requested operation per invocation, subject to existing applicable mutation retries. Neither mode may implement reassignment by unassigning first, add automatic rollback, or silently perform completion or variable updates.
- **FR-009**: Both modes MUST preserve backend-authorized explicit-key tenant behavior; a configured discovery tenant MUST NOT introduce filtering or additional discovery for explicit keys.
- **FR-010**: Before approval, both modes MUST freeze locally validated selection and action, including assignee and override permission where applicable. Protected CLI assignment MUST read each unique task once before approval, offer explicit replacement approval for tasks owned by another identity, and stop on lookup failure. Without interactive consent or force, occupied tasks remain skipped. Tasks whose completed lookup already matches the requested identity MUST succeed as unchanged/not_submitted without confirmation or mutation, preserve ordered results, and be excluded from approval counts. A single human result MUST say `user-task <key> is already assigned to "<assignee>"; no change needed.` All-unchanged JSON MUST use succeeded, keys-only MUST emit zero bytes, and quiet MUST suppress the informational summary. Force skips lookup and still submits; dry-run remains a preview. Interactive replacement approval covers the selected batch once; nonterminal input must not be mistaken for approval. Fail-fast plus an observed block prevents all assignment requests. Dry-run, explicit override and unassignment skip these reads. There is no reread after approval; backend override protection remains authoritative.
- **FR-011**: Dry-run MUST preview the frozen action and keys, assignee where applicable, and whether replacement is allowed. It MUST perform zero backend requests, require no confirmation, and make no acceptance or guaranteed-success claim.
- **FR-012**: Eligible interactive execution MUST confirm the frozen operation exactly once, identifying action, scope, assignee, and replacement permission where applicable. Prompts MUST remain plain configured or inherited stderr text, preserve existing eligibility, decline and EOF behavior, and never appear in result stdout or diagnostic records.
- **FR-013**: Auto-confirm and automation MUST retain protected-assignment ownership checks while preserving existing confirmation bypass, output guardrails, and precedence without changing the requested operation, identity, or override permission.
- **FR-014**: Both modes MUST honor existing worker-selection, timeout, and applicable backoff conventions, including `--workers`/`-w`, `--no-worker-limit`, sequential execution with one worker, and a bounded number of active operations.
- **FR-015**: Fail-fast MUST stop new scheduling after an observed failure while preserving already submitted outcomes. It MUST distinguish unscheduled work and MUST NOT imply cancellation or rollback of already submitted changes.
- **FR-016**: Each submitted operation MUST await the native response, including any blocking task-listener processing, subject to existing timeout and cancellation controls. Both modes MUST perform no additional polling, read-back verification, or downstream waiting; `--no-wait` cannot bypass the native request.
- **FR-017**: A submitted mutation MUST require the documented successful backend response to report success. An unchanged outcome instead requires a completed ownership lookup matching the requested assignee and MUST NOT imply submission. Established `accepted` terminology MUST mean observed successful response rather than request dispatch. Human wording MAY describe backend-confirmed assignment or unassignment but MUST NOT imply independent read-back or unchanged future ownership.
- **FR-018**: Results MUST preserve an ordered outcome for every unique selected key, overall counts/status, partial successes, and distinctions among preview, unchanged, accepted, failed, timed-out, canceled, and not-attempted work. Submission evidence MUST separately distinguish not submitted, accepted, rejected, and unknown as applicable, retaining established error classification and exit behavior.
- **FR-019**: Timeout, cancellation, or ambiguous transport failure MUST NOT imply definite rejection. Applicable retries MUST preserve earlier ambiguous submission evidence when followed by a rejection, while a later documented success may establish acceptance. Cancellation before an operation starts MUST leave it not attempted.
- **FR-020**: Both modes MUST preserve human, JSON, keys-only, quiet, verbose, and automation contracts and precedence. JSON results MUST use one stable shared envelope, including validation and partial failures, with no subsequent error renderer and with original failure exits and machine-readable evidence preserved.
- **FR-021**: Keys-only dry-run MUST print unique preview keys in selection order; execution MUST print only keys with successful native responses in that order. Output MUST contain one key per line and no other stdout content, emit zero bytes when none qualify, and retain partial-failure exit behavior and applicable diagnostics.
- **FR-022**: Selected-format results MUST use stdout; human mutation summaries, dry-run summaries, and verbose outcome details MUST use the configured command-context INFO logger. They MUST honor its level, format, destination, inherited configuration, activity routing, and quiet suppression, leaving human stdout empty. Prompts MUST remain independent plain stderr text.
- **FR-023**: Multi-task failure MUST produce one compact aggregate ERROR with counts and useful bounded reasons, grouping repeated causes without duplicate emission. Quiet MUST retain the compact failure while suppressing informational output. Normal and verbose-only diagnostics MUST exclude raw joined errors, transport chains, methods, URLs, bodies, and repeated per-key transport details from both streams.
- **FR-023a**: A single human assignment blocked without interactive consent or force, or by a definite already-assigned rejection, with override disabled, MUST emit one INFO hint explaining that reassignment is not allowed and suggesting `--force`, replacing the failure summary and ERROR. Preserve the conflict exit code and machine evidence. Quiet and WARN suppress the hint; other conflicts, ambiguous submissions, unassignment, override-enabled and bulk operations retain their existing handling.
- **FR-024**: Both modes MUST inherit existing DEBUG visibility and redaction through either `--debug` or configured DEBUG level, independently of verbose and with existing precedence, destination, and result-stream separation. Established machine-readable error evidence MUST remain available without schema changes made solely to alter human formatting.
- **FR-025**: The same user-facing contract MUST apply to Camunda 8.8, 8.9, and 8.10, retaining generic unsupported-version validation. Planning MUST confirm supported-version assignment protection and response semantics without weakening the default non-overriding policy.
- **FR-026**: Existing variable updates, their scope and verification semantics, no-ops and `--no-wait`, completion's preflight and result behavior, user-task resolution and reads, and `get pi --has-user-tasks` MUST retain their observable behavior. Completion MUST never assign implicitly.
- **FR-027**: Command and parent-group descriptions, examples, README guidance, and generated CLI documentation MUST explain all modes, explicit-key input, override permission, local previews, concurrency, partial outcomes, native-response waiting and the reason `--no-wait` is rejected for assignment and unassignment.

### Key Entities *(include if feature involves data)*

- **Task selection**: A frozen, ordered collection of unique explicit task keys covered by one operation and confirmation.
- **Assignment intent**: The selected action, shared backend identity for assignment, and explicit permission or prohibition for replacing existing ownership.
- **Operation preview**: A local description of the selection and intent, without evidence of backend eligibility or successful submission.
- **Task operation outcome**: A selected key's execution status, submission evidence, and applicable failure information, distinguishing success from uncertainty and work not attempted.
- **Bulk result**: Ordered per-task outcomes and overall counts/status, preserving partial evidence and the invocation's failure behavior.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can assign or unassign a single task or collection in one invocation. In every valid selection scenario, each eligible unique key has exactly one logical operation unless scheduling stops, and duplicates add none.
- **SC-002**: In every reassignment scenario without explicit override permission, replacement is prohibited, including unattended execution. Every preview and interactive approval for assignment identifies whether replacement is permitted.
- **SC-003**: Every invalid-input, declined-approval, and dry-run scenario submits zero mutations; dry-run performs zero backend requests. Eligible interactive execution requires exactly one approval, and dry-run requires none.
- **SC-004**: In every mixed-outcome scenario, 100% of selected keys are accounted for in selection order; no uncertain, failed, or unattempted task is presented as successful, and partial failures retain their failure exit behavior.
- **SC-005**: Every supported machine-output scenario is consumable without prompt or diagnostic contamination. Structured output contains exactly one result; keys-only contains exactly the qualifying keys or zero bytes when none qualify.
- **SC-006**: Every successful operation is reported only after backend confirmation, with zero extra state checks or downstream waits. Timeout and cancellation scenarios never assert unsupported rejection or rollback.
- **SC-007**: Sequential acceptance scenarios never exceed one active operation, and bounded concurrent scenarios never exceed the selected limit. Fail-fast preserves all submitted outcomes and identifies remaining unattempted work.
- **SC-008**: In all repeated-failure scenarios, operators receive one compact human error without raw transport detail at normal or verbose-only levels. Informational human results follow configured visibility and destination, and quiet retains failures.
- **SC-009**: All existing variable-update, completion, and task-read regression scenarios retain their observable behavior across supported versions; assignment and unassignment behave consistently across the same versions.

## Assumptions

- The source is issue #351 as retrieved on 2026-10-28, updated at 11:54:57 UTC that day, including the clarified reuse, response-waiting, and keys-only requirements. Issue #311 is implemented and supplies applicable input, execution, submission-evidence, and output conventions; its completion-only preflight behavior is not part of this feature.
- Existing authentication, explicit-key tenant handling, prompt eligibility, output precedence, worker selection, retry, timeout, and failure-exit conventions remain the baseline unless a requirement explicitly differs. No new transport retry policy or permission model is introduced.
- Backend success confirms the requested lifecycle operation, including blocking task listeners, satisfying the requirement to substantiate success claims. No independent read-back or guarantee of permanent ownership is implied. A timeout can leave the outcome unknown.
- Boolean selectors set to false remain inactive, consistent with current completion behavior; an explicitly provided empty assignment string is invalid. Supplied variable payload flags remain incompatible with assignment and unassignment even when their values are empty. Nonblank assignee values are treated as opaque identities without display-name normalization.
- Scope excludes search/filter selection, automatic assignment before completion, combined assign-and-complete, per-task assignees, identity lookup or creation, candidate-user/group changes, other task attributes, completion preflight reuse, assignment-state polling, rollback, changed variable-update semantics, changed completion behavior, and new diagnostic infrastructure.
- Architecture boundaries, concrete helper reuse without a generic mutation framework, supported-version request mapping, existing generated-client reuse, formatting, and documentation regeneration from the source issue remain planning constraints. This specification describes observable behavior; implementation design belongs in the plan and tasks.
- Implementation acceptance uses only unit tests, mocks, test-owned HTTP fakes, and isolated CLI/subprocess/terminal checks. Interactive checks require real terminal stdin and separate stdout/stderr capture, including configured/inherited destinations and redirected stdout. Output checks must cover active plain/structured logging, normal/verbose/quiet/machine modes, both DEBUG entry points, suppression, redaction, and absence of duplicates; they must detect direct-print routing or raw-error leakage regressions.
- Live Camunda, cluster, smoke, and environment-backed validation are not implementation completion gates. Actual real-engine listener execution and other behavior not established by isolated validation must be reported as unverified coverage. Specification-only validation uses document review, structure and link checks, not runtime tests.
