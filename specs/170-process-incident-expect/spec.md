# Feature Specification: Process Instance Incident Expectation

**Feature Branch**: `170-process-incident-expect`  
**Created**: 2026-05-05  
**Status**: Draft  
**Input**: User description: "GitHub issue #170: feat(expect): add process-instance --incident expectation"

## GitHub Issue Traceability

- **Issue Number**: 170
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/170
- **Issue Title**: feat(expect): add process-instance --incident expectation

## Clarifications

### Session 2026-05-05

- No critical ambiguities detected worth formal clarification. The GitHub issue defines the command surface, accepted boolean values, missing-instance behavior, combined `--state` and `--incident` semantics, stdin key pipelining, help text, regression coverage, and preservation requirements.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Wait For Incident Presence (Priority: P1)

As a CLI user monitoring a process instance, I want `c8volt expect process-instance` and `c8volt expect pi` to wait until the selected instance has an incident so that automation can block until incident state is reached.

**Why this priority**: Waiting for `Incident: true` is the primary new expectation requested by the issue and delivers the core user value independently.

**Independent Test**: Run the process-instance expectation command against a selected instance whose incident marker changes to `true`, then verify the command succeeds only after the selected instance is present and reports `Incident: true`.

**Acceptance Scenarios**:

1. **Given** a selected process instance exists with `Incident: false`, **When** the user runs `c8volt expect pi --key <key> --incident true`, **Then** the command continues waiting.
2. **Given** the selected process instance exists with `Incident: true`, **When** the user runs `c8volt expect process-instance --key <key> --incident true`, **Then** the command succeeds.

---

### User Story 2 - Wait For Incident Absence (Priority: P2)

As a CLI user monitoring recovery, I want `--incident false` to wait until selected process instances are present without incidents so that scripts can continue only after incident state is cleared.

**Why this priority**: The boolean flag must support both requested values, and the false case has distinct missing-instance semantics that need independent verification.

**Independent Test**: Run the expectation command against a selected instance whose incident marker is false and against a missing instance, then verify false only succeeds for present instances with `Incident: false`.

**Acceptance Scenarios**:

1. **Given** a selected process instance exists with `Incident: false`, **When** the user runs `c8volt expect pi --key <key> --incident false`, **Then** the command succeeds.
2. **Given** a selected process instance is missing, **When** the user runs `c8volt expect pi --key <key> --incident false`, **Then** the command continues waiting and does not treat absence as incident-free.

---

### User Story 3 - Combine State And Incident Expectations (Priority: P3)

As a CLI user with stricter readiness criteria, I want `--state` and `--incident` to work together so that a process instance must satisfy both expectations before the command succeeds.

**Why this priority**: Combined expectations preserve the existing command shape while allowing incident checks to compose with the existing state expectation.

**Independent Test**: Run the expectation command with both `--state` and `--incident`, then verify it waits until every selected instance satisfies both conditions.

**Acceptance Scenarios**:

1. **Given** a selected process instance satisfies the requested state but not the requested incident value, **When** both flags are provided, **Then** the command continues waiting.
2. **Given** a selected process instance satisfies both the requested state and the requested incident value, **When** both flags are provided, **Then** the command succeeds.

---

### User Story 4 - Preserve Key Pipelining (Priority: P4)

As a CLI user composing commands, I want `--incident` to work with stdin key pipelining so that `get pi --keys-only | expect pi --incident true -` remains a supported automation pattern.

**Why this priority**: The existing `-` target behavior is already part of the command contract and must remain usable with the new expectation flag.

**Independent Test**: Pipe process-instance keys from `c8volt get pi --keys-only` into `c8volt expect pi --incident true -`, then verify keys are read from stdin and `--key` is not required.

**Acceptance Scenarios**:

1. **Given** process instance keys are piped on stdin, **When** the user runs `c8volt expect pi --incident true -`, **Then** the command reads the piped keys and applies the incident expectation to them.
2. **Given** the user runs `c8volt expect pi --state active -`, **When** keys are piped on stdin, **Then** the existing state-only stdin behavior remains supported.

---

### User Story 5 - Validate Expectation Input And Help (Priority: P5)

As a CLI user, I want invalid expectation input to fail clearly and help text to document the new flag so that scripts and humans can discover and diagnose the new behavior.

**Why this priority**: The command must remain safe and understandable once the new flag is added, but these checks can be delivered after the core waiting behavior is defined.

**Independent Test**: Inspect help output and run invalid invocations, then verify accepted values and requirement rules are enforced with clear errors.

**Acceptance Scenarios**:

1. **Given** the user passes `--incident maybe`, **When** the command validates input, **Then** it fails with a clear invalid-input message.
2. **Given** the user runs `c8volt expect pi --key <key>` without `--state` or `--incident`, **When** the command validates input, **Then** it fails clearly because at least one expectation flag is required.
3. **Given** the user views help for `expect process-instance` or `expect pi`, **When** the flag list is shown, **Then** `--incident true|false` is documented.

### Edge Cases

- Missing process instances must not satisfy `--incident false`.
- `--incident` accepts exactly `true` and `false`; other values fail during command validation.
- Invocations without any expectation flag must fail even when a process instance key or stdin target is present.
- When multiple process instances are selected, every selected instance must satisfy the incident expectation before success.
- When both `--state` and `--incident` are supplied, every selected instance must satisfy both expectations before success.
- Existing `--state absent` semantics must remain unchanged.
- Existing canceled and terminated state compatibility must remain unchanged.
- Existing `expect pi --state active -` stdin behavior must remain covered by tests.
- Read-only command classification must remain unchanged.
- Current automation unsupported behavior must remain unchanged.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose an `--incident` expectation flag on `c8volt expect process-instance` and `c8volt expect pi`.
- **FR-002**: The `--incident` flag MUST accept exactly the values `true` and `false`.
- **FR-003**: The command MUST reject any other `--incident` value with a clear invalid-input message.
- **FR-004**: The command MUST require at least one expectation flag, either `--state` or `--incident`.
- **FR-005**: `--incident true` MUST succeed only after every selected process instance is present and has `Incident: true`.
- **FR-006**: `--incident false` MUST succeed only after every selected process instance is present and has `Incident: false`.
- **FR-007**: A missing process instance MUST NOT be treated as satisfying `--incident false`.
- **FR-008**: When both `--state` and `--incident` are provided, the command MUST require both expectations to be satisfied for every selected process instance.
- **FR-009**: The system MUST preserve existing `--state` behavior, including `absent` semantics.
- **FR-010**: The system MUST preserve existing canceled and terminated state compatibility.
- **FR-011**: The system MUST preserve stdin key pipelining with `-` for process-instance expectations.
- **FR-012**: Stdin key pipelining MUST work when `--incident` is the only expectation flag.
- **FR-013**: Stdin key pipelining MUST work when `--state` and `--incident` are combined.
- **FR-014**: Help text for process-instance expectation commands MUST document `--incident true|false`.
- **FR-015**: Automated tests MUST cover waiting for `Incident: true`.
- **FR-016**: Automated tests MUST cover waiting for `Incident: false`.
- **FR-017**: Automated tests MUST cover that missing process instances do not satisfy `--incident false`.
- **FR-018**: Automated tests MUST cover combined `--state` and `--incident` expectations.
- **FR-019**: Automated tests MUST cover invalid `--incident` values.
- **FR-020**: Automated tests MUST cover failure when no expectation flag is provided.
- **FR-021**: Automated tests MUST cover `c8volt get pi --keys-only | c8volt expect pi --incident true -`.
- **FR-022**: Automated tests MUST continue covering existing `expect pi --state active -` behavior.
- **FR-023**: The feature MUST preserve read-only command classification.
- **FR-024**: The feature MUST preserve current automation unsupported behavior.

### Key Entities *(include if feature involves data)*

- **Process Instance Selection**: The key-based or stdin-provided set of process instances that the expectation command monitors.
- **Process Instance**: The selected process-instance record whose state and incident marker are evaluated by expectation flags.
- **Incident Expectation**: The requested boolean incident value, either `true` or `false`, that every selected present process instance must match.
- **State Expectation**: The existing requested lifecycle state expectation that may be used alone or combined with the incident expectation.
- **Expectation Result**: The success, waiting, or invalid-input outcome produced after evaluating the selected process instances against requested expectations.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show `c8volt expect pi --key <key> --incident true` waits until `Incident` is true.
- **SC-002**: Automated tests show `c8volt expect pi --key <key> --incident false` waits until a present process instance has `Incident` false.
- **SC-003**: Automated tests show a missing process instance does not satisfy `--incident false`.
- **SC-004**: Automated tests show combined `--state` and `--incident` invocations require both expectations to be satisfied.
- **SC-005**: Automated tests show stdin key pipelining works with `--incident true` and does not require `--key`.
- **SC-006**: Automated tests show invalid values such as `--incident maybe` fail with a clear invalid-input message.
- **SC-007**: Automated tests show running without `--state` or `--incident` fails clearly.
- **SC-008**: Help output documents `--incident true|false` for process-instance expectation commands.
- **SC-009**: Existing tests continue to cover `expect pi --state active -`.
- **SC-010**: Relevant process-instance expectation command tests and process wait tests pass.

## Assumptions

- The affected command surface remains `c8volt expect process-instance` and its `expect pi` alias.
- Existing process-instance selection behavior remains the source of truth for `--key` and stdin `-` inputs.
- Existing public/domain process-instance models already expose an `Incident` boolean that reflects the incident marker to evaluate.
- Existing wait, polling, timeout, and error reporting behavior remain the source of truth unless this feature explicitly changes validation or expectation matching.
- Existing command documentation is generated from command metadata when repository tooling provides a generation path.
