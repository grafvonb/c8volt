# Feature Specification: Job Lookup And Updates

**Feature Branch**: `180-job-update-lookup`  
**Created**: 2026-05-07  
**Status**: Draft  
**Input**: User description: "GitHub issue #180: feat(update): add job update and lookup commands"

## GitHub Issue Traceability

- **Issue Number**: 180
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/180
- **Issue Title**: feat(update): add job update and lookup commands

## Clarifications

### Session 2026-05-07

- Q: Should timeout updates be confirmed by comparing the returned deadline against the requested duration? -> A: No; confirm retries only. Timeout updates report accepted/submitted after mutation because deadline-based confirmation is too timing-sensitive.

### Session 2026-05-08

- Q: Should job updates inherit the newer mutation UX used by process-instance variable updates? -> A: Yes. `update job` must support `--dry-run`, build a compact preflight plan from the current job state, require interactive confirmation for material mutations unless explicitly auto-confirmed or automated, skip mutation for retry-only no-op requests, and keep JSON output stable by rejecting `--json --verbose` for this state-changing command.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect A Job By Key (Priority: P1)

As a Camunda operator investigating an incident, I want to use the `jobKey` already exposed by incident-aware process-instance output to inspect the matching job directly so that I can diagnose runtime job state without leaving c8volt.

**Why this priority**: Job lookup is the foundation for diagnosis and for the default post-update confirmation path.

**Independent Test**: Run `c8volt get job --key <job-key>` against a supported Camunda 8.8 or 8.9 environment and verify the command returns the matching job details, including operational metadata needed for incident diagnosis.

**Acceptance Scenarios**:

1. **Given** a valid job key exists on Camunda 8.8 or 8.9, **When** the operator runs `c8volt get job --key <job-key>`, **Then** the command returns matching job details including key, state, retries, deadline when present, process instance key, element instance key, error fields when present, and tenant metadata when available.
2. **Given** no job exists for the supplied key, **When** the operator runs `c8volt get job --key <job-key>`, **Then** the command reports a not-found outcome suitable for human-readable and JSON modes.
3. **Given** `--key` is omitted, **When** the lookup command runs, **Then** it fails validation before calling Camunda.

---

### User Story 2 - Update Job Retries With Confirmation (Priority: P2)

As a Camunda operator, I want to set a job retry count and confirm the read model reflects the requested value so that recovery work can be automated safely.

**Why this priority**: Retry adjustment is the simplest state-changing job update and proves the accepted, confirmed, and failed result flow.

**Independent Test**: Run `c8volt update job --key <job-key> --retries 3` and verify the command reports success only after job lookup observes retries `3`.

**Acceptance Scenarios**:

1. **Given** a valid job key on Camunda 8.8 or 8.9, **When** the operator runs `c8volt update job --key <job-key> --retries 3`, **Then** the command submits retries `3` and waits until job lookup reports retries `3`.
2. **Given** the mutation request fails before acceptance, **When** retries are submitted, **Then** the command reports mutation failure and does not run confirmation.
3. **Given** the mutation is accepted but the requested retry count is not observed before waiter exhaustion, **When** confirmation ends, **Then** the command reports confirmation failure instead of success.
4. **Given** the current retry count already matches the requested retry count and no timeout update is requested, **When** the operator runs the update without `--dry-run`, **Then** the command reports that there is nothing to update and does not ask for confirmation or submit a mutation.
5. **Given** the requested retry count differs from the current retry count, **When** the operator runs the update interactively without `--auto-confirm`, `--automation`, or `--dry-run`, **Then** the command shows a compact plan and asks for confirmation before submitting the mutation.

---

### User Story 3 - Update Job Timeout Without Deadline Confirmation (Priority: P3)

As a Camunda operator, I want to extend or adjust a job timeout using duration input and receive accepted/submitted output after Camunda accepts the mutation so that operational remediation remains scriptable without relying on timing-sensitive deadline comparison.

**Why this priority**: Timeout update behavior has distinct input conversion semantics, while avoiding deadline confirmation prevents false confidence from client/server time drift or read-model timing differences.

**Independent Test**: Run `c8volt update job --key <job-key> --timeout 5m` and verify the submitted timeout is converted to milliseconds and accepted/submitted output is returned without deadline confirmation.

**Acceptance Scenarios**:

1. **Given** a valid job key on Camunda 8.8 or 8.9, **When** the operator runs `c8volt update job --key <job-key> --timeout 5m`, **Then** the command submits timeout as milliseconds and reports accepted/submitted output after mutation acceptance without deadline confirmation.
2. **Given** both `--retries` and `--timeout` are supplied, **When** the update command runs, **Then** both changes are submitted in one request and the command confirms only the requested retry count.
3. **Given** an invalid timeout duration is supplied, **When** the update command validates inputs, **Then** it fails before calling Camunda.
4. **Given** a timeout update is requested in dry-run mode, **When** the command builds the plan, **Then** it reports the timeout submission intent without comparing the requested duration to the observed deadline.

---

### User Story 4 - Preview Job Updates Before Mutation (Priority: P3)

As a Camunda operator, I want to preview job update effects before submitting mutations so that shell quoting, automation, and operational changes can be reviewed safely.

**Why this priority**: Job updates are state-changing. The dry-run and confirmation gate prevent accidental mutation and align this command with the current variable update UX.

**Independent Test**: Run `c8volt update job --key <job-key> --retries 3 --dry-run` and verify the command loads current job state, renders a compact plan, and submits no mutation.

**Acceptance Scenarios**:

1. **Given** a valid job key on Camunda 8.8 or 8.9, **When** the operator runs `c8volt update job --key <job-key> --retries 3 --dry-run`, **Then** the command reports whether retries would change or are already unchanged and submits no mutation.
2. **Given** a timeout update is included in `--dry-run`, **When** the plan is rendered, **Then** the command reports the timeout duration that would be submitted and does not claim deadline confirmation.
3. **Given** `--json --dry-run` is used, **When** the command succeeds, **Then** JSON output includes the full stable plan payload regardless of verbosity defaults.
4. **Given** `--json --verbose` is supplied for `update job`, **When** the command validates flags, **Then** it fails before lookup or mutation because JSON output must remain one stable view.
5. **Given** a non-dry-run JSON job update would mutate state, **When** neither `--auto-confirm` nor `--automation` is supplied, **Then** the command fails before lookup or mutation instead of prompting inside JSON output.

---

### User Story 5 - Return After Accepted Update Without Waiting (Priority: P4)

As an automation user, I want `--no-wait` to return as soon as a job update request is accepted so that scripts can choose lower latency when read-model confirmation is unnecessary.

**Why this priority**: The no-wait path is explicitly required and must preserve the same result vocabulary without polling.

**Independent Test**: Run `c8volt update job --key <job-key> --retries 3 --no-wait` and verify the command reports submitted output without invoking confirmation polling.

**Acceptance Scenarios**:

1. **Given** `--no-wait` is supplied and the update request is accepted, **When** the command runs, **Then** it returns submitted output without polling job lookup.
2. **Given** `--no-wait` is supplied and the mutation request fails, **When** the command runs, **Then** it reports mutation failure instead of submitted or confirmed output.
3. **Given** `--no-wait` is supplied for a material interactive update without `--auto-confirm`, `--automation`, or `--dry-run`, **When** the command reaches the mutation gate, **Then** it still asks for confirmation before submitting and only skips post-mutation polling.

---

### User Story 6 - Preserve Boundaries And Existing Behavior (Priority: P5)

As a maintainer, I want job functionality isolated from existing process-instance and incident behavior so that the new command surface does not regress current diagnostic workflows.

**Why this priority**: Isolation and regression coverage protect existing commands while adding a new state-changing command family.

**Independent Test**: Run targeted tests for existing process-instance incident output and process-instance variable updates, then verify no job lookup, update, or confirmation behavior is added to unrelated service areas.

**Acceptance Scenarios**:

1. **Given** `c8volt get pi --with-incidents` already exposes `jobKey`, **When** the new job commands are added, **Then** the existing incident output remains unchanged.
2. **Given** `c8volt update pi --vars` already has planning, dry-run, submitted, confirmed, confirmation-failed, and mutation-failed result semantics, **When** job updates are added, **Then** process-instance variable update behavior remains unchanged.
3. **Given** Camunda 8.7 is configured, **When** job lookup or update is attempted, **Then** unsupported-version errors are reported before unsupported mutation paths are used.

### Edge Cases

- A job key is required for both lookup and update commands.
- `update job` fails validation when neither `--retries` nor `--timeout` is supplied.
- Invalid retry counts fail before calling Camunda.
- Invalid timeout durations fail before calling Camunda.
- Duration input such as `60s`, `5m`, and `1h` is accepted and converted to milliseconds for the update request.
- Timeout updates are not confirmed by comparing observed deadline timestamps against requested durations.
- When both retries and timeout are supplied, confirmation waits only for the requested retry count.
- `--dry-run` loads current job state, renders the planned update, and submits no mutation.
- Retry-only requests with no retry change report that nothing needs updating and skip both the confirmation prompt and mutation.
- Timeout requests are treated as material update intent because the observed deadline is not a reliable equality predicate for the requested duration.
- Non-dry-run material updates require explicit confirmation in interactive mode unless `--auto-confirm` or `--automation` is supplied.
- Non-dry-run JSON updates that would mutate state require `--auto-confirm` or `--automation` instead of prompting.
- `--json --verbose` is invalid for `update job`, including dry-run mode, so JSON output stays one stable view.
- `--no-wait` skips confirmation polling but still distinguishes accepted mutation from mutation failure.
- Confirmation timeout or retry exhaustion reports confirmation failure without claiming success.
- Mutation failure and confirmation failure are distinguishable in human-readable and JSON output.
- Camunda 8.7 is unsupported for this feature beyond explicit unsupported-version errors.
- Job fail, complete, BPMN error, variable update, retryBackOff, and bulk update flows are out of scope.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add `c8volt get job --key <job-key>`.
- **FR-002**: The system MUST add `c8volt update job --key <job-key>` under the existing `c8volt update` root command.
- **FR-003**: Both job commands MUST require `--key`.
- **FR-004**: Job lookup MUST support Camunda 8.8 and 8.9.
- **FR-005**: Job updates MUST support Camunda 8.8 and 8.9 only.
- **FR-006**: Camunda 8.7 usage MUST fail with unsupported-version errors before unsupported mutation paths are used.
- **FR-007**: Job lookup MUST return job details including key, state, retries, deadline when present, process instance key, element instance key, error fields when present, and tenant metadata when available.
- **FR-008**: Job lookup MUST report a not-found outcome suitable for both human-readable and JSON modes when no job exists for the supplied key.
- **FR-009**: `update job` MUST require at least one update flag from `--retries` or `--timeout`.
- **FR-010**: `update job` MUST support `--retries <count>`.
- **FR-011**: `update job` MUST support `--timeout <duration>`.
- **FR-012**: Timeout duration input MUST accept values such as `60s`, `5m`, and `1h` and convert them to Camunda milliseconds.
- **FR-013**: Invalid retry or timeout values MUST fail before calling Camunda.
- **FR-014**: Job updates MUST NOT expose `retryBackOff`.
- **FR-015**: Job updates MUST support submitting retries and timeout in one update request when both flags are supplied.
- **FR-016**: By default, job updates MUST confirm requested retries through the same lookup behavior as `c8volt get job --key <job-key>` when retries are supplied.
- **FR-017**: Retry confirmation MUST wait until job lookup reports the requested retry count.
- **FR-018**: Timeout-only updates MUST report accepted/submitted output after mutation acceptance and MUST NOT claim deadline confirmation.
- **FR-019**: Timeout updates MUST NOT be confirmed by comparing observed deadline timestamps against requested durations.
- **FR-020**: When retries and timeout are both supplied, confirmation MUST verify the requested retry count only while reporting timeout as submitted in the same mutation.
- **FR-021**: `--no-wait` MUST return submitted output after update acceptance and skip confirmation polling.
- **FR-022**: Confirmation timeout or retry exhaustion MUST report confirmation failure without claiming success.
- **FR-023**: Mutation failure and confirmation failure MUST be distinguishable in human-readable and JSON output.
- **FR-024**: Job update results MUST follow the same submitted, confirmed, confirmation-failed, and mutation-failed pattern as process-instance variable updates where confirmation is available; timeout-only updates use submitted or mutation-failed outcomes.
- **FR-025**: Job lookup, job updates, and job update confirmation MUST stay within a dedicated job service boundary.
- **FR-026**: Job service code MUST include shared API/factory behavior, versioned services for 8.7, 8.8, and 8.9, and compile-time conformance checks matching existing service package patterns.
- **FR-027**: Job functionality MUST NOT be mixed into process-instance services, incident services, or facade-only helpers.
- **FR-028**: Command output MUST use existing command context logger and activity plumbing for submitted and confirmed output.
- **FR-029**: Existing `get pi --with-incidents` behavior MUST remain unchanged and continue to expose `jobKey`.
- **FR-030**: Existing `update pi --vars` dry-run, planning, confirmation, submitted, confirmed, confirmation-failed, and mutation-failed behavior MUST remain unchanged.
- **FR-031**: User-facing help and generated documentation MUST describe job lookup, job update flags, version support, no-wait behavior, confirmation outcomes, and examples.
- **FR-032**: Automated tests MUST cover successful job lookup on Camunda 8.8 and 8.9.
- **FR-033**: Automated tests MUST cover not-found job lookup output in human-readable and JSON modes.
- **FR-034**: Automated tests MUST cover retries update with confirmation, timeout update without deadline confirmation, combined retries and timeout update with retries-only confirmation, and `--no-wait`.
- **FR-035**: Automated tests MUST cover mutation failure, confirmation failure, invalid inputs, missing required flags, and Camunda 8.7 unsupported-version behavior.
- **FR-036**: Automated tests MUST cover unchanged `get pi --with-incidents` and `update pi --vars` behavior.
- **FR-037**: `update job` MUST support `--dry-run` for all valid retry and timeout update requests.
- **FR-038**: `update job --dry-run` MUST load the current job state through the same lookup behavior as `c8volt get job --key <job-key>` and MUST NOT submit a mutation.
- **FR-039**: `update job` MUST build a technical plan aggregate before material mutations, including job key, current retry count when visible, requested retry count when supplied, timeout duration when supplied, timeout milliseconds when supplied, retry change status, and whether mutation was submitted.
- **FR-040**: Human plan output MUST be compact and use short change syntax, for example retry before/after and timeout submission intent, without noisy unchanged-field listings.
- **FR-041**: Retry-only updates MUST detect when the requested retry count already matches the visible current retry count and MUST report that there is nothing to update without prompting or mutating.
- **FR-042**: Timeout requests MUST be treated as material update intent because timeout equality is not determined from the observed deadline.
- **FR-043**: Interactive non-dry-run material job updates MUST ask for confirmation before mutation unless `--auto-confirm` or `--automation` is supplied.
- **FR-044**: Non-dry-run JSON job updates that would mutate state MUST require `--auto-confirm` or `--automation` and MUST fail before lookup or mutation when neither is supplied.
- **FR-045**: `--json --dry-run` MUST return the same full JSON plan payload regardless of verbose defaults.
- **FR-046**: `update job` MUST reject `--json --verbose`, including dry-run mode, before lookup or mutation.

### Key Entities *(include if feature involves data)*

- **Job Key**: A user-supplied Camunda job identifier used as the lookup and update target.
- **Job Detail**: The returned job information needed for diagnosis, including identity, state, retry count, deadline, process and element instance relationships, error metadata, and tenant metadata when available.
- **Job Update Request**: A single accepted mutation request that may include retries, timeout, or both.
- **Job Update Plan**: The pre-mutation aggregate containing current job state, requested changes, no-op classification, dry-run status, and mutation-submission status.
- **Timeout Duration**: User input such as `60s`, `5m`, or `1h` converted to milliseconds for submission.
- **Observed Deadline**: The timestamp returned by job lookup for display and diagnosis; it is not used to confirm timeout updates.
- **Job Update Result**: The command outcome indicating submitted, confirmed, confirmation-failed, or mutation-failed state.
- **Unsupported Version Error**: The explicit failure returned when the command is used against Camunda 8.7 for unsupported job functionality.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show `c8volt get job --key <job-key>` returns the expected job details on Camunda 8.8 and 8.9.
- **SC-002**: Automated tests show missing jobs produce not-found output suitable for both human-readable and JSON modes.
- **SC-003**: Automated tests show `c8volt update job --key <job-key> --retries 3` submits retries `3` and confirms by observing retries `3`.
- **SC-004**: Automated tests show `c8volt update job --key <job-key> --timeout 5m` submits milliseconds and returns accepted/submitted output without deadline confirmation.
- **SC-005**: Automated tests show supplying both `--retries` and `--timeout` submits both changes in one request and confirms only the requested retry count.
- **SC-006**: Automated tests show `--no-wait` returns submitted output without confirmation polling.
- **SC-007**: Automated tests show mutation failure and confirmation failure are reported as distinct outcomes in human-readable and JSON output.
- **SC-008**: Automated tests show invalid retries, invalid timeout values, missing `--key`, and missing update flags fail before calling Camunda.
- **SC-009**: Automated tests show Camunda 8.7 fails with unsupported-version errors before unsupported mutation paths are used.
- **SC-010**: Automated tests show `get pi --with-incidents` still exposes `jobKey` unchanged.
- **SC-011**: Automated tests show `update pi --vars` behavior and confirmation semantics remain unchanged.
- **SC-012**: Automated tests or static checks show job lookup, update, and confirmation behavior are not added to process-instance or incident service APIs.
- **SC-013**: Help output, examples, and generated CLI documentation accurately describe the new commands, required flags, update options, `--no-wait`, version support, and confirmation outcomes.
- **SC-014**: Relevant targeted command and service tests pass, followed by the closest broader repository validation command.
- **SC-015**: Automated tests show `c8volt update job --key <job-key> --retries 3 --dry-run` renders a plan and submits no mutation.
- **SC-016**: Automated tests show retry-only no-op requests report nothing to update and skip confirmation prompts and mutation.
- **SC-017**: Automated tests show timeout dry-run output reports submission intent without deadline comparison or mutation.
- **SC-018**: Automated tests show material interactive job updates prompt before mutation unless auto-confirmed or automated.
- **SC-019**: Automated tests show non-dry-run JSON job updates require `--auto-confirm` or `--automation`, while `--json --dry-run` returns a stable full plan payload.
- **SC-020**: Automated tests show `update job --json --verbose` is rejected before lookup or mutation.

## Assumptions

- Operators obtain the relevant job key from existing diagnostic output such as `c8volt get pi --with-incidents`.
- Camunda 8.8 and 8.9 generated clients expose job search and job update capabilities needed for lookup and supported updates.
- A job update timeout is interpreted as a duration from the current moment, while searched job details expose a deadline timestamp that is not used for confirmation.
- Existing command registration, validation, error mapping, output rendering, metadata, waiter/backoff, and docs generation patterns remain the source of truth for consistent behavior.
- Confirmation may need to tolerate normal read-model delay between accepted mutation and observed job details.
- Documentation and generated CLI docs should be updated through existing command metadata and regeneration paths when available.
