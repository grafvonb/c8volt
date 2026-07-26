# Feature Specification: Job Ops Workflow Primitives

**Feature Branch**: `231-job-ops-workflows`
**Created**: 2026-05-25
**Status**: Draft
**Input**: GitHub issue [#231](https://github.com/grafvonb/c8volt/issues/231) - `feat(job): extend get and update job commands for ops workflows`

## GitHub Issue Traceability

- **Issue Number**: 231
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/231
- **Issue Title**: feat(job): extend get and update job commands for ops workflows

## Ralph Implementation Context

- Planning, task generation, and every Ralph implementation iteration MUST read and apply `specs/ralph-implementation-rules.md`.
- Ralph MUST NOT be launched unless the implementation instructions include `--implementation-context specs/ralph-implementation-rules.md`.
- Commit subjects for this issue-backed work MUST use Conventional Commits and end with `#231`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Preserve Keyed Job Lookup And Current Updates (Priority: P1)

As a Camunda operator who already has a job key from incident or process diagnostics, I want existing keyed job lookup and retry/timeout update behavior to keep working so that current recovery workflows do not regress while the command surface expands.

**Why this priority**: Existing low-level job commands are the stable base that new list and worker outcome modes must not break.

**Independent Test**: Run the existing keyed lookup and retry/timeout update flows and verify output, validation, dry-run, confirmation, JSON guardrails, and no-wait behavior remain compatible with the current command contract.

**Acceptance Scenarios**:

1. **Given** a valid job key on a supported runtime, **When** the operator runs `c8volt get job --key <job-key>`, **Then** c8volt returns exactly that job with the existing job detail fields and output modes.
2. **Given** a valid job key and retry update, **When** the operator runs `c8volt update job --key <job-key> --retries <n>`, **Then** c8volt preserves the existing retry mutation, dry-run, confirmation, JSON guardrail, and automation behavior.
3. **Given** a valid job key and timeout update, **When** the operator runs `c8volt update job --key <job-key> --timeout <duration>`, **Then** c8volt preserves the existing timeout mutation behavior and does not claim deadline equality confirmation.

---

### User Story 2 - Search Jobs Without A Known Key (Priority: P2)

As a Camunda operator investigating failures, I want `get job` to list jobs when I do not already know the job key so that I can discover failed, typed, scoped, or worker-owned jobs directly from c8volt.

**Why this priority**: Job discovery is the main new read-only operator value and a prerequisite for future ops workflows that compose job primitives.

**Independent Test**: Run `c8volt get job` without `--key` using each supported search filter independently and in safe combinations, then verify default, JSON, keys-only, and limit behavior.

**Acceptance Scenarios**:

1. **Given** jobs in different states, **When** the operator runs `c8volt get job --state failed`, **Then** c8volt lists only jobs matching the requested state semantics.
2. **Given** jobs with different job types, process instance keys, element instance keys, element IDs, workers, retries, kinds, or listener event types, **When** the operator supplies the matching filters, **Then** c8volt lists only matching jobs or fails locally when a filter value is invalid.
3. **Given** no `--key` is supplied and no search filter is supplied, **When** the operator runs `c8volt get job`, **Then** c8volt uses the command's documented default list behavior with the explicit limit policy instead of requiring a known key.

---

### User Story 3 - Report Technical Job Failure (Priority: P3)

As a Camunda operator acting as a worker substitute, I want `update job --fail` to report a technical failure with retry and message controls so that recovery playbooks can use the same primitive as manual remediation.

**Why this priority**: Technical failure is the most direct worker outcome mode needed by ops workflows.

**Independent Test**: Run `c8volt update job --key <job-key> --fail --retries 0 --message <text>` and the retry-backoff variant, then verify request construction, mutation safety gates, output, and unsupported-version behavior.

**Acceptance Scenarios**:

1. **Given** a valid job key, **When** the operator runs `update job --fail --retries 0 --message <text>`, **Then** c8volt submits a technical failure outcome with the requested retry count and message.
2. **Given** a retry backoff is supplied, **When** the operator runs `update job --fail --retries 2 --retry-backoff 5m --message <text>`, **Then** c8volt submits the backoff value according to the command's documented duration behavior.
3. **Given** `--fail` is combined with another worker outcome mode, **When** command validation runs, **Then** c8volt fails before remote calls.

---

### User Story 4 - Throw A Modeled BPMN Error (Priority: P4)

As a Camunda operator handling modeled business failure, I want `update job --throw-bpmn-error <code>` to throw the intended BPMN error so that manual and ops-driven recovery can follow the process model rather than technical retry semantics.

**Why this priority**: BPMN error handling is distinct from technical failure and must be explicit to avoid dangerous operator ambiguity.

**Independent Test**: Run `c8volt update job --key <job-key> --throw-bpmn-error PAYMENT_DECLINED --message <text>` and verify validation, request construction, output contract, and unsupported-version behavior.

**Acceptance Scenarios**:

1. **Given** a valid job key and BPMN error code, **When** the operator runs `update job --throw-bpmn-error <code> --message <text>`, **Then** c8volt submits a modeled BPMN error outcome for that job.
2. **Given** the BPMN error code is missing, **When** command validation runs, **Then** c8volt fails before remote calls.
3. **Given** `--throw-bpmn-error` is combined with `--fail`, `--complete`, retry update, or timeout update behavior in a conflicting way, **When** command validation runs, **Then** c8volt fails before mutation.

---

### User Story 5 - Complete A Job With Variables (Priority: P5)

As a Camunda operator completing a job manually or through an ops workflow, I want `update job --complete --vars <json>` so that a job can be completed with explicit variables through the same mutation safety model as other updates.

**Why this priority**: Completion rounds out the worker outcome modes and provides a composable primitive for future repair workflows.

**Independent Test**: Run `c8volt update job --key <job-key> --complete --vars '{"approved":true}'` and verify JSON variable parsing, mutation safety gates, output contract, and unsupported-version behavior.

**Acceptance Scenarios**:

1. **Given** a valid job key, **When** the operator runs `update job --complete --vars <json>`, **Then** c8volt submits completion with the provided variables.
2. **Given** `--complete` is supplied without variables, **When** the command runs, **Then** c8volt completes the job with no additional variables.
3. **Given** invalid variable JSON is supplied, **When** command validation runs, **Then** c8volt fails before remote calls.

---

### User Story 6 - Keep Job Primitives Safe, Versioned, And Documented (Priority: P6)

As a maintainer and automation user, I want new job behavior to follow existing command contracts, version gates, documentation, and generated docs so scripts and future ops workflows can rely on stable primitives.

**Why this priority**: The feature changes user-visible command behavior and mutation paths, so safety, compatibility, and docs must be completed before the work is usable.

**Independent Test**: Run command contract tests, validation tests, generated docs checks, and supported-version tests for all new and preserved job paths.

**Acceptance Scenarios**:

1. **Given** Camunda 8.8 or 8.9 is configured, **When** the new get and update job paths run, **Then** c8volt uses supported generated-client capabilities through existing service and facade boundaries.
2. **Given** Camunda 8.7 is configured, **When** a new job search or worker outcome mutation path is requested, **Then** c8volt fails before mutation with a clear unsupported capability error.
3. **Given** command metadata, README examples, and generated CLI docs are inspected, **When** the feature is complete, **Then** they accurately describe the new job search filters, outcome modes, version support, dry-run, confirmation, JSON, and automation behavior.

### Edge Cases

- `get job --key` remains keyed lookup and must not silently switch to list mode.
- `get job` list mode is selected only when `--key` is omitted.
- Keyed lookup cannot be combined with list/search-only filters.
- Job search supports `--state`, `--type`, `--pi-key`, `--element-instance-key`, `--element-id`, `--worker`, `--retries`, `--kind`, `--listener-event-type`, and `--limit`.
- Job terminology uses `elementId` and `elementInstanceKey`; no new `flowNode` or `fni` aliases are introduced for jobs.
- Invalid state, kind, listener event type, retry count, duration, or variable JSON values fail before remote calls.
- `update job --retries` and `update job --timeout` keep their current behavior.
- `--fail`, `--throw-bpmn-error`, and `--complete` are mutually exclusive.
- Worker outcome modes are separate from retry/timeout update modes; `--fail` may use retry count and retry-backoff as technical failure inputs, while `--throw-bpmn-error` and `--complete` cannot be combined with retry or timeout update flags.
- State-changing update modes support dry-run, confirmation, JSON guardrails, no-wait where applicable, and automation behavior consistent with existing `update job`.
- Dry-run never submits a mutation and must clearly show which worker outcome or update would be submitted.
- Human and JSON outputs distinguish validation failure, mutation failure, submitted, confirmed, and unsupported-version outcomes where those states apply.
- Camunda 8.7 fails before unsupported mutation paths are used.

### Forward Real-State Validation Note

The command feature can prove request construction with command/service tests, but final integration acceptance needs real service-task handler jobs, not only listener jobs. Future real-state suites should:

1. Use c8volt commands and embedded service-task BPMN to deploy and start suite-owned process instances.
2. Use Camunda worker APIs directly, when needed, to activate jobs under a deterministic worker name because c8volt is not a general-purpose job worker.
3. Use direct worker API actions to create handler-owned job states for timeout, technical failure, completion, BPMN error, and stale/already-mutated repair scenarios.
4. Verify c8volt commands observe the resulting job state through `get job`, `update job`, and `ops repair` flows.
5. Record direct worker setup as integration evidence and as a spec-owned gap where a safe reusable setup helper or embedded BPMN fixture would reduce future test complexity.

This setup should remain distinct from listener-job coverage. Listener jobs prove listener visibility; handler-owned service-task jobs prove worker outcome behavior and repair semantics.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST preserve `c8volt get job --key <job-key>` as exact keyed lookup.
- **FR-002**: The system MUST preserve current `c8volt update job --key <job-key> --retries <n>` behavior.
- **FR-003**: The system MUST preserve current `c8volt update job --key <job-key> --timeout <duration>` behavior.
- **FR-004**: `get job` without `--key` MUST list jobs using supported search filters.
- **FR-005**: Search/list mode MUST support `--state <state>`.
- **FR-006**: Search/list mode MUST support `--type <job-type>`.
- **FR-007**: Search/list mode MUST support `--pi-key <process-instance-key>`.
- **FR-008**: Search/list mode MUST support `--element-instance-key <element-instance-key>`.
- **FR-009**: Search/list mode MUST support `--element-id <bpmn-element-id>`.
- **FR-010**: Search/list mode MUST support `--worker <worker-name>`.
- **FR-011**: Search/list mode MUST support `--retries <count>`.
- **FR-012**: Search/list mode MUST support `--kind <job-kind>`.
- **FR-013**: Search/list mode MUST support `--listener-event-type <event-type>` for listener-job filtering.
- **FR-014**: Search/list mode MUST support `--limit <count>` and document its default behavior.
- **FR-015**: Job command terminology MUST use `elementId` and `elementInstanceKey` concepts and MUST NOT add new job aliases based on `flowNode` or `fni`.
- **FR-016**: `update job` MUST add `--fail` as an explicit technical failure mode.
- **FR-017**: `--fail` MUST support retry count, retry-backoff duration, and message input needed for technical failure submission.
- **FR-018**: `update job` MUST add `--throw-bpmn-error <error-code>` as an explicit modeled BPMN error mode.
- **FR-019**: `--throw-bpmn-error` MUST support an optional message.
- **FR-020**: `update job` MUST add `--complete` as an explicit job completion mode.
- **FR-021**: `--complete` MUST support `--vars <json>` for completion variables.
- **FR-022**: `--fail`, `--throw-bpmn-error`, and `--complete` MUST be mutually exclusive.
- **FR-023**: Worker outcome modes MUST be separate from retry/timeout update modes; `--fail` MAY use retry count and retry-backoff as technical failure inputs, while `--throw-bpmn-error` and `--complete` MUST NOT be combined with retry or timeout update flags.
- **FR-024**: All state-changing update modes MUST support dry-run without submitting a mutation.
- **FR-025**: All state-changing update modes MUST support confirmation, JSON guardrails, and automation behavior consistent with existing `update job`.
- **FR-026**: All state-changing update modes MUST report human-readable and JSON outcomes through existing command result patterns.
- **FR-027**: Camunda 8.8 and 8.9 MUST be supported for new search and worker outcome behavior through generated clients where those upstream capabilities exist.
- **FR-028**: Camunda 8.7 MUST fail before unsupported new job search or worker outcome mutation paths with a clear unsupported capability error.
- **FR-029**: Job service and facade behavior MUST preserve existing c8volt layering and MUST NOT put generated-client request construction in command code.
- **FR-030**: Command contract metadata MUST describe mutation behavior, automation support, output modes, and required flags for the changed commands.
- **FR-031**: User-facing help, README examples, and generated CLI documentation MUST describe new job search filters, worker outcome modes, version support, and safety controls.
- **FR-032**: Automated tests MUST cover keyed lookup preservation.
- **FR-033**: Automated tests MUST cover job search request construction, validation, output contracts, and limit behavior.
- **FR-034**: Automated tests MUST cover retry and timeout update preservation.
- **FR-035**: Automated tests MUST cover technical failure request construction, validation, dry-run, output, and unsupported-version behavior.
- **FR-036**: Automated tests MUST cover BPMN error request construction, validation, dry-run, output, and unsupported-version behavior.
- **FR-037**: Automated tests MUST cover completion request construction, variable parsing, dry-run, output, and unsupported-version behavior.
- **FR-038**: Automated tests MUST cover mutual exclusion and incompatible flag combinations before remote calls.

### Key Entities *(include if feature involves data)*

- **Job Query**: A keyed lookup or list/search request built from job key, search filters, output modifiers, and limit controls.
- **Job Detail**: The returned job information needed for diagnosis and mutation safety, including identity, state, retries, worker, type, kind, process context, element context, error fields, and tenant metadata when available.
- **Job Search Filter**: Validated filter intent for state, type, process instance key, element instance key, element ID, worker, retries, kind, listener event type, and limit.
- **Job Worker Outcome**: A mutually exclusive mutation mode representing technical failure, modeled BPMN error, or completion.
- **Technical Failure Request**: A worker outcome containing retry count, optional retry backoff, and operator message.
- **BPMN Error Request**: A worker outcome containing modeled error code and optional message.
- **Completion Request**: A worker outcome containing optional completion variables.
- **Mutation Plan**: A dry-run and confirmation payload describing the selected job, requested update or worker outcome, and whether mutation will be submitted.
- **Unsupported Capability Error**: The explicit failure returned before unsupported job search or worker outcome mutation paths on Camunda 8.7.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show `get job --key` still returns exactly one job and preserves existing output contracts.
- **SC-002**: Automated tests show `update job --retries` and `update job --timeout` preserve current validation, dry-run, mutation, and output behavior.
- **SC-003**: Automated tests show `get job` without `--key` lists jobs by each supported search filter and respects explicit limits.
- **SC-004**: Automated tests show job search rejects invalid filter values before remote calls.
- **SC-005**: Automated tests show no new `flowNode` or `fni` aliases are accepted for job commands.
- **SC-006**: Automated tests show `update job --fail` submits the expected technical failure request and handles dry-run, confirmation, JSON, automation, and unsupported-version paths.
- **SC-007**: Automated tests show `update job --throw-bpmn-error` submits the expected BPMN error request and handles dry-run, confirmation, JSON, automation, and unsupported-version paths.
- **SC-008**: Automated tests show `update job --complete --vars <json>` submits the expected completion request and handles variable parsing, dry-run, confirmation, JSON, automation, and unsupported-version paths.
- **SC-009**: Automated tests show `--fail`, `--throw-bpmn-error`, and `--complete` are mutually exclusive.
- **SC-010**: Automated tests show Camunda 8.7 fails before unsupported new job search or worker outcome mutation requests are submitted.
- **SC-011**: Command contract tests show updated mutation, automation, output mode, and required flag metadata for changed commands.
- **SC-012**: README, help text, and generated CLI documentation match the implemented search filters, outcome modes, examples, and version support.
- **SC-013**: Relevant targeted command/service tests pass, followed by the closest broader repository validation command.

## Assumptions

- Operators already have c8volt configuration and Camunda permissions for job read and worker outcome operations in the selected tenant context.
- Existing `get job --key` and retry/timeout update behavior from issue #180 remains the compatibility baseline.
- Camunda 8.8 and 8.9 generated clients contain, or can be refreshed to contain, the upstream operations needed for supported job search and worker outcomes.
- Future `ops` workflows will compose these low-level primitives rather than receiving separate job-specific verbs.
- Documentation and generated CLI docs are updated through existing command metadata and regeneration paths.
