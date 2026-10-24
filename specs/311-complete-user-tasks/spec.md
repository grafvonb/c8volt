# Feature Specification: Complete User Tasks by Explicit Key

**Feature Branch**: `311-complete-user-tasks`

**Created**: 2026-10-24

**Status**: Draft

**Input**: [GitHub issue #311](https://github.com/grafvonb/c8volt/issues/311) — Extend `update user-task` with `--complete` for one or more explicit task keys, preserving the variable-update mode introduced by #326.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Complete Explicitly Selected Tasks (Priority: P1)

As an operator, I want to complete one or several known user tasks in one invocation so I can acknowledge finished work without repeating commands or accidentally selecting unrelated tasks.

**Why this priority**: Explicit selection and truthful acceptance reporting are the core value of this feature.

**Independent Test**: With isolated task responses, complete a single task and a collection containing duplicate keys, then verify the selected targets, submission count, and reported acceptance.

**Acceptance Scenarios**:

1. **Given** a valid task key, **When** I run `update user-task --key <key> --complete`, **Then** the command requests completion for that task and reports acceptance only when the backend accepts it; it does not wait for task-state visibility or downstream process completion.
2. **Given** repeated flags, comma-separated keys, and newline-separated stdin selected with positional `-`, **When** I execute completion, **Then** keys are merged using existing input conventions, deduplicated in first-input order, and each unique key receives one logical completion operation.
3. **Given** the aliases `user-tasks`, `ut`, and `uts`, **When** I use any alias, **Then** selection and completion behavior match the canonical command.
4. **Given** absent keys, empty stdin, malformed input, invalid worker values, or `--complete --no-wait`, **When** I invoke completion, **Then** validation fails before any completion request; empty input neither discovers tasks nor reports a successful no-op.
5. **Given** an explicit key outside the configured discovery tenant, **When** the backend authorizes access, **Then** the task can be completed under the existing explicit-key tenant contract; denied access, missing tasks, and rejected completions remain failures.
6. **Given** a task that requires assignment before completion, **When** the backend rejects completion, **Then** the failure is reported and the command does not assign the task implicitly.

---

### User Story 2 - Supply Optional Completion Variables (Priority: P1)

As an operator, I want to supply one optional set of variables with all selected completions so the work outcome and its data are submitted together.

**Why this priority**: Completion must remain distinct from editing effective task variables, even when both modes accept the same payload flags.

**Independent Test**: Complete selected tasks with omitted variables, an empty object, and equivalent inline/file payloads; inspect isolated submissions and confirm that no separate variable update or verification occurs.

**Acceptance Scenarios**:

1. **Given** a JSON object supplied through `--vars` or `--vars-file`, **When** I complete multiple tasks, **Then** every unique task receives the same variables with their JSON value types preserved.
2. **Given** no variable source, **When** I complete tasks, **Then** every unique task receives a completion request without additional variables.
3. **Given** `{}` or values already matching effective task variables, **When** I complete tasks, **Then** completion still occurs for every unique key; the command does not plan variable differences, issue separate scope-variable writes, or poll for variable confirmation.
4. **Given** both variable sources, an unreadable file, malformed JSON, or a non-object value, **When** I invoke completion, **Then** input validation fails before any completion request.

---

### User Story 3 - Retain Existing User-Task Workflows (Priority: P1)

As an existing user, I want the completion option to leave my variable-editing and read workflows unchanged so adopting the feature does not break existing commands or scripts.

**Why this priority**: Completion extends an existing command whose variable planning, no-op behavior, and confirmation guarantees must remain compatible.

**Independent Test**: Exercise existing variable-update scenarios without `--complete`, including dry-run, unchanged variables, inherited scopes, waiting and `--no-wait`, then check existing task read workflows using isolated dependencies.

**Acceptance Scenarios**:

1. **Given** an invocation without `--complete`, **When** I supply variables, **Then** existing variable-source requirements, planning, scope selection and writes, shared-scope handling, confirmation, verification, no-op behavior, and output schemas remain unchanged.
2. **Given** variable-update mode, **When** I use `--no-wait`, **Then** it remains supported with the existing distinction between accepted writes and confirmed values; omitting a required variable source remains invalid.
3. **Given** existing task resolution, user-task reads and variable features, or `get pi --has-user-tasks`, **When** those commands run, **Then** their selection and result behavior remain unchanged.
4. **Given** Camunda 8.8, 8.9, or 8.10, **When** completion is requested, **Then** the same user-facing contract applies; unsupported versions retain existing generic rejection behavior.
5. **Given** command help and user documentation, **When** I inspect user-task update or its parent command group, **Then** descriptions and examples explain both modes, optional completion variables, completion acceptance, and the completion restriction on `--no-wait` consistently.

---

### User Story 4 - Preview, Confirm, and Control Bulk Execution (Priority: P2)

As an operator, I want to review the full selection once and control concurrent execution so bulk completion is deliberate and failures leave understandable results.

**Why this priority**: Operators need predictable control over a mutation that can affect many tasks and cannot be rolled back by the command.

**Independent Test**: Use isolated responses for several tasks to exercise dry-run, real-terminal confirmation, sequential and concurrent submission, failure, timeout, cancellation, and fail-fast scheduling.

**Acceptance Scenarios**:

1. **Given** a valid completion selection, **When** I use `--dry-run`, **Then** the preview identifies the unique task keys and intended completion, submits no mutation, does not prompt, and never claims acceptance.
2. **Given** an invocation eligible for interactive confirmation, **When** I proceed, **Then** one confirmation covers the entire frozen selection and appears as plain text on configured or inherited stderr; declining or reaching EOF submits no mutation and preserves existing abort behavior.
3. **Given** `--auto-confirm` or `--automation`, **When** I submit a valid selection, **Then** existing unattended execution and output guardrails apply without a prompt.
4. **Given** `--workers 1`, **When** I complete several tasks, **Then** requests execute sequentially in input order; a higher valid worker bound limits simultaneous operations, and `--no-worker-limit` retains existing worker-selection precedence.
5. **Given** a failure among several tasks, **When** fail-fast is disabled, **Then** remaining tasks continue under existing bulk-failure conventions; with `--fail-fast`, new scheduling stops after the observed failure, already submitted work is not described as canceled or rolled back, and unscheduled tasks are identified as not attempted.
6. **Given** accepted, failed, timed-out, canceled, and unattempted work, **When** execution ends, **Then** individual outcomes and the overall result retain available evidence and the established failure exit status; timeout or cancellation does not assert that the backend definitely rejected the mutation.

---

### User Story 5 - Consume Reliable Results and Diagnostics (Priority: P2)

As an automation author or operator, I want results and diagnostics to follow existing command conventions so scripts remain reliable and I can investigate failures without exposing transport details during ordinary use.

**Why this priority**: Partial failure must remain actionable in both interactive operations and unattended workflows.

**Independent Test**: Execute completion with isolated successful and repeated failing responses across supported output modes and logger settings, capturing stdout and stderr separately.

**Acceptance Scenarios**:

1. **Given** human, JSON, keys-only, quiet, or automation settings in supported combinations, **When** completion previews or results are rendered, **Then** existing mode precedence applies; selected-format results stay on stdout, keys-only output contains only task keys one per line, and quiet suppresses informational prose while preserving explicitly requested machine output.
2. **Given** full success, validation failure, or partial execution failure with JSON output, **When** the command exits, **Then** stdout contains exactly one shared envelope followed by EOF, retaining applicable task outcomes and machine-readable error evidence without a second error renderer appending output.
3. **Given** repeated backend failures containing distinctive methods, URLs, bodies, and resource keys, **When** normal or verbose-only output is selected, **Then** one compact aggregate error groups repeated causes, includes counts and a bounded useful reason, uses the configured command logger, and exposes no raw transport details on either stream; quiet retains this compact error.
4. **Given** configured or inherited logging destinations and either active plain or structured logging, **When** a diagnostic is emitted, **Then** its severity and destination honor command configuration without duplication or diagnostic records contaminating stdout.
5. **Given** `--debug` or configured DEBUG logging, independently of verbose and subject to existing flag precedence, **When** sequential or bounded concurrent completion runs, **Then** existing redacted diagnostics describe actual completion exchanges and any retries without changing result stdout; elapsed HTTP measurements do not claim to measure task-state visibility or downstream completion.

### Edge Cases

- A collection reduced to one unique key still completes that task exactly once logically and retains deterministic first-input ordering in its result evidence.
- Invalid input late in a list or stdin stream must prevent all completion requests, even if earlier keys were valid.
- Empty stdin with no flag keys is invalid; valid flag keys combined with empty stdin remain usable under existing input rules.
- An empty object is valid completion data; JSON arrays, scalars, and `null` are not objects and must be rejected.
- A task may disappear, become ineligible, or lose authorization between preview and submission; it must not be reported as a successful no-op.
- Variable payloads may contain nested objects, arrays, booleans, numbers, strings, and null values; these values must retain their types inside the required outer object.
- Concurrent operations may finish out of order; outcomes must remain associated with their original keys and presented in deterministic selection order.
- Fail-fast may leave in-flight requests and unattempted keys in the same result; neither category may be silently converted to success.
- Timeout or cancellation after submission may leave backend acceptance unknown; results must preserve that uncertainty and already known partial outcomes.
- Terminal stdin with redirected stdout must preserve existing prompt eligibility and keep confirmation text on configured stderr.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `update user-task` MUST offer `--complete` and preserve aliases `user-tasks`, `ut`, and `uts`; without it, existing variable-update behavior MUST remain unchanged.
- **FR-002**: Completion MUST require at least one explicit key, supporting repeated/comma-separated `--key`/`-k` and positional `-` for newline-separated stdin under existing parsing rules. Keys MUST be deduplicated in first-input order before execution.
- **FR-003**: The complete local input MUST be validated before completion requests. Missing or malformed keys, empty selection, invalid worker values, conflicting flags, invalid variable sources, and invalid variable content MUST fail without mutation or discovery fallback.
- **FR-004**: Completion MUST accept zero or one variable source, `--vars` or `--vars-file`, containing a JSON object. It MUST preserve value types and apply the same supplied object to every selected task.
- **FR-005**: Omitted variables, `{}`, and values already matching task data MUST NOT suppress completion. Completion MUST submit variables as part of completion, without separate scope-variable updates, variable-difference planning, or variable verification.
- **FR-006**: Each unique key MUST receive one logical native task-completion operation, subject to existing applicable transport conventions. Completion MUST NOT implicitly assign tasks or perform unrelated repairs.
- **FR-007**: Explicit-key completion MUST preserve backend authorization and the existing explicit-key tenant contract. Missing tasks, denied access, and rejected completion MUST remain failures.
- **FR-008**: `--dry-run` MUST preview the frozen unique selection without mutation, confirmation prompts, or acceptance claims. Interactive mutation MUST ask once for that entire selection and preserve existing decline and EOF behavior.
- **FR-009**: Confirmation MUST be plain text on configured or inherited stderr. Existing automation, auto-confirm, output guardrails, and prompt eligibility MUST remain in force.
- **FR-010**: Completion MUST honor existing `--workers`/`-w`, `--no-worker-limit`, timeout, and applicable backoff conventions; `--workers 1` MUST execute sequentially, and a selected worker bound MUST limit concurrent operations.
- **FR-011**: `--fail-fast` MUST stop scheduling additional tasks after an observed failure under existing bulk conventions. Already submitted operations MUST NOT be described as rolled back or canceled solely because scheduling stopped.
- **FR-012**: Completion MUST return after request outcomes are available, reporting accepted submissions without independently verified completion. It MUST NOT poll task state or wait for downstream execution, and MUST reject `--complete --no-wait` while retaining variable-update `--no-wait`.
- **FR-013**: Results MUST identify each unique task and its accepted, failed, timed-out, canceled, or unattempted outcome as applicable, in selection order, with an overall result and preserved partial evidence. Timeout and cancellation MUST NOT imply definite backend rejection; failures MUST retain the established error class and exit behavior.
- **FR-014**: Completion MUST preserve existing human, JSON, keys-only, quiet, and automation conventions and precedence. Stable machine-readable results MUST use exactly one shared JSON envelope, including validation and partial failures, with no subsequent error renderer output.
- **FR-015**: Selected-format results MUST use stdout; diagnostics MUST use the configured command-context logger with appropriate severity and destination. Multi-task failure MUST emit one compact aggregate diagnostic containing counts and grouped, bounded reasons without duplicate emission; quiet MUST retain the error.
- **FR-016**: Normal and verbose-only diagnostics MUST NOT expose raw joined errors, transport chains, HTTP methods, URLs, bodies, or per-key transport failures on either stream. Established machine-readable error evidence MUST remain available without changing its schema to address human formatting.
- **FR-017**: Completion MUST inherit existing DEBUG diagnostics, redaction, destination, and flag precedence through `--debug` or configured DEBUG logging independently of verbose. Diagnostics MUST cover actual exchanges and retries, preserve result stdout, and add no `--api-diagnostics` flag or separate timing report.
- **FR-018**: Completion MUST support Camunda 8.8, 8.9, and 8.10 with existing generic unsupported-version handling and preserve task resolution, reads, variable updates, and `get pi --has-user-tasks` behavior.
- **FR-019**: Command and parent-group help, examples, README guidance, and generated CLI documentation MUST describe both modes accurately, including optional variables, explicit selection, preview, execution controls, acceptance-only semantics, and the `--no-wait` restriction.

### Key Entities *(include if feature involves data)*

- **Completion selection**: The frozen, ordered list of unique explicit task keys covered by one preview or confirmation.
- **Completion variables**: An optional shared object of named values submitted with each task's completion, distinct from a plan to edit effective variable scopes.
- **Task completion outcome**: Evidence for one selected key, distinguishing accepted submission, failure or uncertain submission, and work not attempted.
- **Completion result**: Ordered task outcomes and overall counts/status that preserve partial success and failure evidence without claiming independently verified task or process completion.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can submit one task or a collection in one invocation; across successful single, repeated, comma-separated, stdin, and combined-input scenarios, 100% of unique valid keys have one logical completion operation and duplicate keys add none.
- **SC-002**: Across the acceptance scenarios, 100% of invalid selections and dry-runs submit zero mutations; eligible interactive execution requires exactly one confirmation for a valid selection and none for dry-run or invalid input.
- **SC-003**: Omitted variables, empty objects, and inline/file payloads produce the intended completion in every accepted-input scenario, with zero separate variable edits or completion-confirmation polls.
- **SC-004**: Every selected key is accounted for in mixed-outcome scenarios, with no failed, uncertain, or unattempted task presented as accepted; operators can distinguish preview, accepted submission, and failure from the result alone.
- **SC-005**: Every supported machine-output scenario yields one parseable result without prompt or diagnostic contamination; partial failures retain their evidence and failure exit behavior. Repeated failures yield exactly one compact human error with no raw transport details at normal or verbose-only levels.
- **SC-006**: All existing variable-update and task-read regression scenarios retain their observable behavior across supported versions, including unchanged-data no-ops and accepted-versus-confirmed distinctions.
- **SC-007**: Sequential scenarios never exceed one active completion and bounded concurrent scenarios never exceed the selected bound; enabling supported DEBUG diagnostics leaves selected-format result output unchanged.

## Assumptions

- The source is issue #311 as retrieved on 2026-10-24, including its assignment/unassignment follow-up in #351. The current variable-update feature (#326) and shared diagnostics (#305) are existing dependencies.
- Existing input, confirmation, worker-selection, error, output, tenant, and transport conventions are the baseline unless this issue explicitly differs. In particular, completion rejects empty selection and `--no-wait`, even where another command permits them.
- Backend acceptance is the completion mode's promised outcome. Task-state visibility and downstream process execution may occur later and are not verified by this command.
- Scope excludes search/filter selection, assignment and unassignment, task attribute changes, automatic assignment, per-task variable payloads, changes to variable-update scope or verification semantics, changes to `update job` or task reads, completion polling, benchmarks, synthetic diagnostic traffic, and aggregated latency reports.
- Architecture, helper reuse, version mapping, and generated-documentation obligations in the linked issue remain planning constraints. This specification defines user-visible behavior; implementation design belongs in the plan and tasks.
- Implementation acceptance uses only isolated test-owned dependencies and CLI/subprocess/terminal tests. Interactive cases require real terminal stdin and separately captured stdout/stderr; diagnostic cases cover normal, verbose, quiet, JSON, both DEBUG entry points, configured/inherited destinations, and active plain/structured logging. Tests must detect direct-print routing regressions and raw aggregate leakage.
- Live Camunda, cluster, smoke, and environment-backed validation are excluded from implementation completion gates. Any real-engine behavior not established by isolated validation must be reported as unverified coverage.
