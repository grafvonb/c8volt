# Feature Specification: Resolve Incident Commands

**Feature Branch**: `181-resolve-incident-commands`  
**Created**: 2026-05-08  
**Status**: Draft  
**Input**: GitHub issue [#181](https://github.com/grafvonb/c8volt/issues/181) - `feat(resolve): add incident resolution commands`

## Source Issue

- **Issue Number**: 181
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/181
- **Issue Title**: feat(resolve): add incident resolution commands

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Resolve Known Incidents (Priority: P1)

As a Camunda operator with one or more known incident keys, I want to resolve those incidents directly from c8volt so I can recover failed work without switching tools.

**Why this priority**: Direct incident-key resolution is the smallest useful command path and provides the foundation for all other resolution flows.

**Independent Test**: Can be tested by running the incident resolution command with one active incident key and verifying the command submits resolution and reports the incident as resolved or no longer active.

**Acceptance Scenarios**:

1. **Given** an active incident key from `c8volt get pi --with-incidents`, **When** the user runs `c8volt resolve incident --key <incident-key>`, **Then** c8volt submits an incident resolution request and waits until the incident is no longer active or is reported resolved.
2. **Given** multiple incident keys, **When** the user runs `c8volt resolve incident --key <key-a> --key <key-b>`, **Then** each unique incident is resolved and reported independently.
3. **Given** newline-separated incident keys on stdin, **When** the command is invoked with `-`, **Then** c8volt reads those keys using existing stdin key behavior and resolves each unique incident.

---

### User Story 2 - Resolve Process Instance Incidents (Priority: P2)

As a Camunda operator with one or more process instance keys, I want c8volt to find and resolve the currently active incidents for each process instance so I can recover process instances without manually copying every incident key.

**Why this priority**: Process-instance resolution is the main workflow shortcut and depends on incident lookup plus the incident-key resolution behavior.

**Independent Test**: Can be tested by running the process-instance resolution command against a process instance with active incidents and verifying only the incidents discovered at command start are attempted and confirmed.

**Acceptance Scenarios**:

1. **Given** a process instance with active incidents, **When** the user runs `c8volt resolve pi --key <process-instance-key>`, **Then** c8volt discovers the active incidents for that process instance, resolves all discovered incidents, and waits until those incidents are no longer active.
2. **Given** a process instance with no active incidents, **When** the user runs `c8volt resolve pi --key <process-instance-key>`, **Then** c8volt reports that no active incidents needed resolution without treating it as a failure.
3. **Given** multiple process instance keys from repeated flags or stdin, **When** the user runs `c8volt resolve pi`, **Then** each unique process instance is processed and reported independently.

---

### User Story 3 - Control Waiting and Failure Reporting (Priority: P3)

As an operator running resolution in manual or automated workflows, I want waiting, timeout, and partial failure behavior to be explicit so I can trust the command result and handle failures target by target.

**Why this priority**: Reliable confirmation and partial failure visibility determine whether the new state-changing command is safe for repeated operational use.

**Independent Test**: Can be tested by exercising default wait behavior, `--no-wait`, one failed resolution among multiple targets, and confirmation timeout.

**Acceptance Scenarios**:

1. **Given** resolution requests are accepted and the user supplied `--no-wait`, **When** the command runs, **Then** c8volt returns submitted output without polling for confirmation.
2. **Given** one resolution request fails among multiple targets, **When** the command completes, **Then** the affected target is reported as failed without hiding successful resolutions for other targets.
3. **Given** waiter timeout or retry exhaustion before resolution is visible, **When** confirmation cannot complete, **Then** the command reports confirmation failure for the affected target.

---

### User Story 4 - Preserve Existing Workflows (Priority: P4)

As a c8volt user, I want the new resolution commands to follow existing command conventions so that current get, update, output, and automation workflows remain predictable.

**Why this priority**: The feature is state-changing and touches shared CLI behavior, so compatibility with existing workflows must be verified even after the core command paths work.

**Independent Test**: Can be tested by checking command metadata, output modes, aliases, worker options, unsupported-version behavior, and existing process-instance commands.

**Acceptance Scenarios**:

1. **Given** a supported Camunda version, **When** the user runs the new commands, **Then** the commands expose human and JSON output suitable for per-target reporting.
2. **Given** an unsupported Camunda version, **When** the user attempts resolution, **Then** c8volt fails with an unsupported-version error before submitting any mutation.
3. **Given** existing `get process-instance --with-incidents`, `get pi --with-incidents`, and `update pi --vars` commands, **When** this feature is added, **Then** their behavior remains unchanged.

### Edge Cases

- Duplicate keys supplied across repeated `--key` flags and stdin are merged using existing key validation and deduplication behavior.
- `resolve pi` resolves only the active incident set discovered at command start; newly created incidents are not chased indefinitely.
- Process instances with no active incidents are reported as skipped or no-op outcomes, not failures.
- Partial failures report the failed incident or process instance while preserving successful target results.
- Confirmation timeout and retry exhaustion are reported per affected target.
- Unsupported Camunda versions fail before mutation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST add a root command family named `resolve`.
- **FR-002**: System MUST add `resolve incident` with alias `inc`.
- **FR-003**: System MUST add `resolve process-instance` with alias `pi`.
- **FR-004**: `resolve incident` MUST accept one or more incident keys from repeated `--key` flags, newline-separated stdin keys via `-`, or both.
- **FR-005**: `resolve pi` MUST accept one or more process instance keys from repeated `--key` flags, newline-separated stdin keys via `-`, or both.
- **FR-006**: Commands MUST reuse existing key validation and deduplication behavior for all key sources.
- **FR-007**: `resolve incident` MUST attempt resolution for each unique supplied incident key.
- **FR-008**: `resolve pi` MUST discover active incidents for each selected process instance using the same incident lookup path used by `c8volt get process-instance --key <key> --with-incidents`.
- **FR-009**: `resolve pi` MUST attempt resolution for all active incidents discovered for the selected process instance at command start.
- **FR-010**: By default, `resolve incident` MUST wait until each supplied incident is no longer active or is reported as resolved.
- **FR-011**: By default, `resolve pi` MUST wait until each selected process instance no longer has the initially discovered active incidents.
- **FR-012**: Commands MUST support `--no-wait` to return after resolution requests are accepted.
- **FR-013**: Commands MUST reuse existing waiter/backoff, command activity, timeout/retry configuration, worker fan-out, `--workers`, `--fail-fast`, and `--no-worker-limit` patterns where applicable.
- **FR-014**: Commands MUST render per-target results suitable for human output.
- **FR-015**: Commands MUST render per-target results suitable for JSON output, including process instance key when applicable, attempted incident keys, resolved incident keys, skipped incident keys, failed incident keys, and confirmation status.
- **FR-016**: Commands MUST be marked as state-changing and automation-compatible following existing command metadata patterns.
- **FR-017**: System MUST use generated Camunda clients for versions where incident resolution endpoints are available.
- **FR-018**: System MUST fail with an unsupported-version error before mutation when the configured Camunda version cannot resolve incidents.
- **FR-019**: Incident lookup and resolution behavior MUST remain in the incident service boundary and MUST NOT be reintroduced into the process-instance service boundary.
- **FR-020**: Existing `get process-instance --with-incidents`, `get pi --with-incidents`, and `update pi --vars` behavior MUST remain unchanged.

### Key Entities *(include if feature involves data)*

- **Resolution Target**: A user-selected incident key or process instance key supplied by flags, stdin, or both.
- **Incident Resolution Attempt**: The outcome of submitting resolution for one incident key, including submitted, resolved, skipped, failed, and confirmation status.
- **Process Instance Resolution Result**: The per-process-instance outcome including the selected process instance key and the incident keys discovered, attempted, resolved, skipped, or failed.
- **Confirmation State**: The observed state after resolution submission, including confirmed resolved, still active, timeout, retry exhaustion, or unsupported version.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can resolve one known active incident with a single `resolve incident --key` command and receive a per-target success result.
- **SC-002**: A user can resolve multiple unique incident keys from repeated flags, stdin, or both, with duplicates reported only once.
- **SC-003**: A user can resolve all incidents discovered at command start for one process instance with a single `resolve pi --key` command.
- **SC-004**: A process instance with no active incidents completes without failure and clearly reports that no incidents required resolution.
- **SC-005**: With `--no-wait`, accepted resolution requests return without confirmation polling.
- **SC-006**: Partial failures, confirmation timeouts, and unsupported-version failures are visible in per-target human and JSON output.
- **SC-007**: Existing process-instance get and update workflows covered by this feature's regression checks remain unchanged.

## Assumptions

- Operators already have permission and connectivity to perform supported Camunda state-changing operations through c8volt.
- Incident resolution endpoints are available only for some configured Camunda versions; unsupported versions must be rejected before mutation.
- The process-instance command path may coordinate incident lookup and resolution, but incident lookup and resolution behavior remains owned by the incident service boundary.
- New commands follow the repository's existing CLI patterns for aliases, key selectors, stdin handling, workers, activity metadata, output modes, and waits.
- Incident creation, job retry, variable mutation, interactive prompts, process-instance searching by filters, and endless polling for incidents created after command start are out of scope.
