# Feature Specification: Process Instance Variable Updates

**Feature Branch**: `179-update-pi-vars`  
**Created**: 2026-05-07  
**Status**: Draft  
**Input**: User description: "GitHub issue #179: feat(update): add process-instance variable updates"

## GitHub Issue Traceability

- **Issue Number**: 179
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/179
- **Issue Title**: feat(update): add process-instance variable updates

## Clarifications

### Session 2026-05-07

- No critical ambiguities detected worth formal clarification. The GitHub issue defines the command surface, required selectors, variable payload validation, Camunda version support, confirmation behavior, waiter controls, output modes, metadata expectations, exclusions, and required regression coverage.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Update Variables For One Process Instance (Priority: P1)

As a Camunda operator with a process instance key, I want `c8volt update process-instance` and `c8volt update pi` to set process-instance-scope variables and confirm the requested values are visible so that operational scripts can safely mutate process data.

**Why this priority**: Single-key update with confirmation is the core workflow and proves the new command family, payload validation, mutation request, and read-model confirmation path.

**Independent Test**: Run `c8volt update pi --key <key> --vars '{"foo":"bar"}'` against an active supported-version process instance, then verify the command reports success only after `get pi --key <key> --with-vars` would show `foo` with the requested value.

**Acceptance Scenarios**:

1. **Given** an active Camunda 8.8 or 8.9 process instance, **When** the operator runs `c8volt update pi --key <key> --vars '{"foo":"bar"}'`, **Then** the command submits the variable update and waits until the requested value is visible through process-instance variable lookup.
2. **Given** the requested variable does not already exist, **When** the operator updates the process instance, **Then** the variable is created and confirmed visible.
3. **Given** the requested variable already exists, **When** the operator updates it with a different JSON value, **Then** the previous value is replaced and the requested value is confirmed visible.
4. **Given** the operator uses the full command name `update process-instance`, **When** the same key and payload are supplied, **Then** behavior matches the `update pi` alias.

---

### User Story 2 - Update Multiple Selected Process Instances (Priority: P2)

As an automation user, I want to update every selected process instance with the same variable map from repeated `--key` values and stdin keys so that batch operations can be scripted safely.

**Why this priority**: Multi-key selection is required for the command to match existing key handling patterns and support operational automation beyond a single instance.

**Independent Test**: Run the update command with duplicate keys from repeated `--key` flags and newline-separated stdin input, then verify each unique process instance receives the same variable map and is reported independently.

**Acceptance Scenarios**:

1. **Given** multiple `--key` values and one `--vars` payload, **When** the command runs, **Then** the same variables are applied to every unique key.
2. **Given** newline-separated keys are provided through stdin using `-`, **When** the command runs, **Then** stdin keys are read using existing stdin key behavior and included in the update target set.
3. **Given** the same key appears more than once across flags and stdin, **When** the command validates targets, **Then** the key is updated once using existing validation and deduplication behavior.
4. **Given** one target fails and fail-fast is enabled, **When** the batch runs, **Then** remaining work stops according to the existing fail-fast semantics.

---

### User Story 3 - Return Accepted Output Without Waiting (Priority: P3)

As an automation user, I want `--no-wait` to return after the mutation request is accepted so that scripts can choose lower latency when read-model confirmation is unnecessary.

**Why this priority**: The no-wait path is explicitly required and provides a useful independent mode after the accepted mutation request is available.

**Independent Test**: Run `c8volt update pi --key <key> --vars '{"foo":"bar"}' --no-wait`, then verify the command reports accepted/update-submitted output without polling variable visibility.

**Acceptance Scenarios**:

1. **Given** `--no-wait` is supplied for a supported process instance, **When** the update request is accepted, **Then** the command returns submitted output without variable confirmation.
2. **Given** multiple keys are supplied with `--no-wait`, **When** the update requests are accepted, **Then** each key is reported independently without confirmation polling.
3. **Given** a mutation request fails before acceptance, **When** `--no-wait` is supplied, **Then** the command reports the failure for that process instance instead of claiming submission.

---

### User Story 4 - Reject Invalid Or Unsupported Updates (Priority: P4)

As a CLI user, I want malformed payloads, missing targets, and unsupported Camunda versions to fail before unsafe mutation attempts so that update commands remain predictable and scriptable.

**Why this priority**: Validation protects a new state-changing command from accidental or unsupported mutations.

**Independent Test**: Run invalid invocations for malformed JSON, non-object JSON, missing variables, missing keys/stdin, and Camunda 8.7, then verify each fails before calling the variable update endpoint.

**Acceptance Scenarios**:

1. **Given** `--vars` is missing, **When** the command runs, **Then** it fails with a clear validation error.
2. **Given** `--vars` is malformed JSON or a JSON value that is not an object, **When** the command runs, **Then** it fails before any mutation request is sent.
3. **Given** no process instance key is supplied by `--key` or stdin `-`, **When** the command runs, **Then** it fails using existing target-selector validation behavior.
4. **Given** the active Camunda configuration is 8.7, **When** the command runs, **Then** it fails with an unsupported-version error before attempting mutation.

### Edge Cases

- `--vars` accepts only JSON object input; arrays, strings, numbers, booleans, and null are rejected.
- The same normalized JSON value must compare equal even when the request and returned variable JSON use different formatting.
- Only requested variable names are checked during confirmation; unrelated existing variables do not affect success.
- Requested values may include nested objects, arrays, strings, numbers, booleans, or null inside the top-level object.
- Keys supplied by repeated `--key` flags and stdin `-` are merged and deduplicated using existing behavior.
- Missing stdin input for `-` fails consistently with existing stdin key handling.
- Camunda 8.7 is unsupported for this command and fails before mutation.
- Confirmation timeout or retry exhaustion reports confirmation failure for the affected key.
- Batch output reports each process instance independently in human-readable and JSON modes.
- Existing `run --vars` and `get process-instance --with-vars` behavior remains unchanged.
- The command must not delete variables or update local element-scoped variables.
- The command must not search for process instances by filters before updating.
- The command must not prompt interactively before mutation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add a new root command family `c8volt update`.
- **FR-002**: The system MUST add `c8volt update process-instance` with alias `pi`.
- **FR-003**: The update process-instance command MUST require `--vars` with JSON object input.
- **FR-004**: The command MUST reject malformed JSON and JSON values whose top-level value is not an object before attempting mutation.
- **FR-005**: The command MUST require at least one process instance key from one or more `--key` flags, newline-separated stdin keys via `-`, or both.
- **FR-006**: The command MUST merge and deduplicate keys from `--key` and stdin using existing key validation and deduplication behavior.
- **FR-007**: The command MUST apply the same variable map to every selected process instance.
- **FR-008**: For Camunda 8.8 and 8.9, the command MUST use the generated Camunda client capability for `PUT /element-instances/{elementInstanceKey}/variables`.
- **FR-009**: The command MUST use the selected process instance key as the `elementInstanceKey`.
- **FR-010**: The command MUST reject Camunda 8.7 configurations with an unsupported-version error before attempting mutation.
- **FR-011**: By default, after submitting an update, the command MUST confirm visibility by reusing the same backend lookup path as `c8volt get process-instance --key <key> --with-vars`.
- **FR-012**: Confirmation MUST wait until every requested variable name has the requested normalized JSON value visible for each updated process instance.
- **FR-013**: Confirmation MUST compare normalized JSON values rather than raw string formatting.
- **FR-014**: Confirmation MUST ignore unrelated variables not included in the requested `--vars` object.
- **FR-015**: The command MUST reuse the existing process-instance state confirmation waiter/backoff style, including command activity, timeout/retry configuration, worker fan-out, `--workers`, `--fail-fast`, and `--no-worker-limit` where applicable.
- **FR-016**: The command MUST support `--no-wait` to return after the update request is accepted without visibility confirmation.
- **FR-017**: Human-readable output MUST report per-process-instance results suitable for single-key and multi-key operation.
- **FR-018**: JSON output MUST report per-process-instance results suitable for automation.
- **FR-019**: The command MUST be marked as state-changing using existing command metadata patterns.
- **FR-020**: The command MUST be marked automation-compatible using existing command metadata patterns.
- **FR-021**: Existing `run --vars` behavior MUST remain unchanged.
- **FR-022**: Existing `get process-instance --with-vars` behavior MUST remain unchanged except for reuse by confirmation.
- **FR-023**: Automated tests MUST cover successful Camunda 8.8 variable update and confirmation.
- **FR-024**: Automated tests MUST cover successful Camunda 8.9 variable update and confirmation.
- **FR-025**: Automated tests MUST cover adding a new variable and updating an existing variable.
- **FR-026**: Automated tests MUST cover repeated `--key` values, stdin key input via `-`, merging, and deduplication.
- **FR-027**: Automated tests MUST cover `--no-wait` accepted/submitted output without confirmation polling.
- **FR-028**: Automated tests MUST cover malformed JSON, non-object JSON, missing `--vars`, and missing target keys.
- **FR-029**: Automated tests MUST cover Camunda 8.7 unsupported-version rejection before mutation.
- **FR-030**: Automated tests MUST cover confirmation failure when requested values are not visible before timeout or retry exhaustion.
- **FR-031**: User-facing command help and generated documentation MUST describe the new command, alias, required flags, stdin key behavior, `--no-wait`, version support, and examples.

### Key Entities *(include if feature involves data)*

- **Process Instance Key**: A user-supplied Camunda process instance identifier used as the update target and passed as the element-instance key for variable updates.
- **Variable Update Payload**: The top-level JSON object supplied through `--vars`; each property name is a variable name and each property value is the requested JSON value.
- **Update Target Set**: The unique collection of process instance keys assembled from repeated `--key` flags and optional stdin key input.
- **Update Result**: The per-process-instance outcome indicating whether the mutation request was accepted, confirmed, failed during mutation, or failed during confirmation.
- **Confirmation Lookup**: The post-mutation process-instance variable read path equivalent to `get process-instance --key <key> --with-vars`.
- **Normalized Variable Value**: The canonical JSON comparison form used to determine whether a returned variable value matches the requested value.
- **Unsupported Version Error**: The explicit failure returned when the command is used against Camunda 8.7.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show `c8volt update pi --key <key> --vars '{"foo":"bar"}' adds a new variable and confirms it through the process-instance variable lookup path.
- **SC-002**: Automated tests show the command updates an existing variable value and confirms the new value by normalized JSON comparison.
- **SC-003**: Automated tests show the full `update process-instance` command name and `update pi` alias behave identically.
- **SC-004**: Automated tests show multiple unique keys receive the same variable map and are reported independently.
- **SC-005**: Automated tests show keys from stdin `-` are accepted and deduplicated with repeated `--key` values.
- **SC-006**: Automated tests show `--no-wait` returns accepted/submitted results without confirmation polling.
- **SC-007**: Automated tests show malformed JSON, non-object JSON, missing `--vars`, and missing target keys fail before mutation.
- **SC-008**: Automated tests show Camunda 8.7 fails with an unsupported-version error before mutation.
- **SC-009**: Automated tests show confirmation failure is reported when requested variable values are not visible before waiter exhaustion.
- **SC-010**: Automated tests show existing `run --vars` and `get process-instance --with-vars` behavior remains unchanged.
- **SC-011**: Help output, examples, and generated CLI documentation accurately describe command usage, alias, required selectors, `--no-wait`, version support, and automation/state-changing metadata.
- **SC-012**: Relevant targeted command tests and the repository's closest broader validation command pass.

## Assumptions

- Camunda 8.8 and 8.9 generated clients expose element-instance variable update support for the required endpoint.
- The generated client method name may not match the external operation name exactly, but it represents the variable update endpoint.
- Existing process-instance key parsing, stdin key parsing, deduplication, worker, fail-fast, and waiter patterns are the source of truth for consistent behavior.
- The process-instance variable read model may lag behind the mutation request, which is why confirmation uses the existing waiter/backoff style by default.
- Returned variable values may be serialized JSON strings, so confirmation compares normalized JSON values.
- Documentation and generated CLI docs should be updated through existing command metadata and regeneration paths when available.
