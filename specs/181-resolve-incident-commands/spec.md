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

1. **Given** an active incident key from `c8volt get pi --with-incidents`, **When** the user runs `c8volt resolve incident --key <incident-key>`, **Then** c8volt submits an incident resolution request and waits by reloading incident state through the incident service until the incident is no longer active.
2. **Given** multiple incident keys, **When** the user runs `c8volt resolve incident --key <key-a> --key <key-b>`, **Then** each unique incident is resolved and reported independently.
3. **Given** newline-separated incident keys on stdin, **When** the command is invoked with `-`, **Then** c8volt reads those keys using existing stdin key behavior and resolves each unique incident.

---

### User Story 2 - Resolve Process Instance Incidents (Priority: P2)

As a Camunda operator with one or more process instance keys, I want c8volt to find and resolve the currently active incidents for each process instance so I can recover process instances without manually copying every incident key.

**Why this priority**: Process-instance resolution is the main workflow shortcut and depends on incident lookup plus the incident-key resolution behavior.

**Independent Test**: Can be tested by running the process-instance resolution command against a process instance with active incidents and verifying only the incidents discovered at command start are attempted and confirmed.

**Acceptance Scenarios**:

1. **Given** a process instance with active incidents, **When** the user runs `c8volt resolve pi --key <process-instance-key>`, **Then** c8volt discovers the active incidents for that process instance, resolves all discovered incidents, and waits by reloading the process-instance incident lookup until the initially discovered incidents are no longer active.
2. **Given** a process instance with no active incidents, **When** the user runs `c8volt resolve pi --key <process-instance-key>`, **Then** c8volt reports that no active incidents needed resolution without treating it as a failure.
3. **Given** multiple process instance keys from repeated flags or stdin, **When** the user runs `c8volt resolve pi`, **Then** each unique process instance is processed and reported independently.

---

### User Story 3 - Preview Resolution Plan (Priority: P3)

As a Camunda operator preparing a state-changing recovery action, I want to preview incident resolution effects before submitting mutations so that shell input, automation payloads, and affected incidents can be reviewed safely.

**Why this priority**: Incident resolution changes operational state. A dry-run plan aligns the command family with the repository's newer mutation UX while preserving the issue's no-prompt scope.

**Independent Test**: Can be tested by running `c8volt resolve incident --key <incident-key> --dry-run` and `c8volt resolve pi --key <process-instance-key> --dry-run`, verifying the command loads the relevant current state, renders a compact plan, and submits no mutation.

**Acceptance Scenarios**:

1. **Given** valid incident keys, **When** the user runs `c8volt resolve incident --key <incident-key> --dry-run`, **Then** c8volt reports which unique incident keys would be resolved and submits no mutation.
2. **Given** valid process instance keys, **When** the user runs `c8volt resolve pi --key <process-instance-key> --dry-run`, **Then** c8volt discovers the active incidents for each process instance, reports the incident keys that would be resolved, and submits no mutation.
3. **Given** a process instance with no active incidents, **When** the user runs `c8volt resolve pi --key <process-instance-key> --dry-run`, **Then** c8volt reports that no active incidents would be resolved without treating it as a failure.
4. **Given** `--json --dry-run` is used, **When** the command succeeds, **Then** JSON output includes the full stable plan payload regardless of verbosity defaults.

---

### User Story 4 - Control Waiting and Failure Reporting (Priority: P4)

As an operator running resolution in manual or automated workflows, I want waiting, timeout, and partial failure behavior to be explicit so I can trust the command result and handle failures target by target.

**Why this priority**: Reliable confirmation and partial failure visibility determine whether the new state-changing command is safe for repeated operational use.

**Independent Test**: Can be tested by exercising default wait behavior, `--no-wait`, one failed resolution among multiple targets, and confirmation timeout.

**Acceptance Scenarios**:

1. **Given** resolution requests are accepted and the user supplied `--no-wait`, **When** the command runs, **Then** c8volt returns submitted output without polling for confirmation.
2. **Given** one resolution request fails among multiple targets, **When** the command completes, **Then** the affected target is reported as failed without hiding successful resolutions for other targets.
3. **Given** waiter timeout or retry exhaustion before the reloaded lookup state shows resolution, **When** confirmation cannot complete, **Then** the command reports confirmation failure for the affected target.

---

### User Story 5 - Preserve Existing Workflows (Priority: P5)

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
- `--dry-run` discovers and renders the same target set that a non-dry-run invocation would attempt, but never submits incident resolution requests.
- `--no-wait` has no additional effect in dry-run mode because dry-run submits no mutation.
- `--no-wait` skips post-mutation lookup polling only; it does not change validation, dry-run planning, target discovery, or mutation submission behavior.
- Partial failures report the failed incident or process instance while preserving successful target results.
- Confirmation timeout and retry exhaustion are reported per affected target when reloaded lookup state does not satisfy the requested resolution predicate.
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
- **FR-010**: By default, `resolve incident` MUST wait after accepted mutation by polling incident lookup through the incident service until each supplied incident key is no longer active.
- **FR-011**: By default, `resolve pi` MUST wait after accepted mutation by polling the same process-instance incident lookup path used for discovery until the initially discovered incident keys are no longer active for the selected process instance.
- **FR-012**: Commands MUST support `--no-wait` to return after resolution requests are accepted and skip post-mutation lookup polling.
- **FR-013**: Commands MUST support `--dry-run` to render a lookup-backed pre-mutation plan without submitting incident resolution requests.
- **FR-014**: `resolve incident --dry-run` MUST load current incident state for each unique supplied incident key where supported and report which incidents would be submitted for resolution.
- **FR-015**: `resolve pi --dry-run` MUST discover active incidents for each selected process instance using the same lookup path as non-dry-run resolution and report which incident keys would be resolved.
- **FR-016**: `--json --dry-run` MUST return a stable plan payload that includes mutation-submission status set to false.
- **FR-017**: `--json --verbose` MUST be rejected for resolve commands, including dry-run mode, so JSON output stays one stable view.
- **FR-018**: Commands MUST reuse the existing waiter/backoff, command activity, timeout/retry configuration, worker fan-out, `--workers`, `--fail-fast`, and `--no-worker-limit` patterns where applicable; confirmation MUST follow the same lookup-polling pattern as `update pi --vars`.
- **FR-019**: Commands MUST render per-target results suitable for human output.
- **FR-020**: Commands MUST render per-target results suitable for JSON output, including process instance key when applicable, attempted incident keys, resolved incident keys, skipped incident keys, failed incident keys, confirmation status, dry-run status, and mutation-submission status.
- **FR-021**: Commands MUST be marked as state-changing and automation-compatible following existing command metadata patterns.
- **FR-022**: System MUST use generated Camunda clients for versions where incident resolution endpoints are available.
- **FR-023**: System MUST fail with an unsupported-version error before mutation when the configured Camunda version cannot resolve incidents.
- **FR-024**: Incident lookup and resolution behavior MUST remain in the incident service boundary and MUST NOT be reintroduced into the process-instance service boundary.
- **FR-025**: Existing `get process-instance --with-incidents`, `get pi --with-incidents`, and `update pi --vars` behavior MUST remain unchanged.

### Key Entities *(include if feature involves data)*

- **Resolution Target**: A user-selected incident key or process instance key supplied by flags, stdin, or both.
- **Resolution Plan**: The pre-mutation aggregate used by dry-run and command-local safety checks, including requested targets, discovered incidents, dry-run status, and mutation-submission status.
- **Incident Resolution Attempt**: The outcome of submitting resolution for one incident key, including submitted, resolved, skipped, failed, and confirmation status.
- **Process Instance Resolution Result**: The per-process-instance outcome including the selected process instance key and the incident keys discovered, attempted, resolved, skipped, or failed.
- **Confirmation State**: The observed state after resolution submission from reloading the relevant incident lookup path, including confirmed no longer active, still active, timeout, retry exhaustion, or unsupported version.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can resolve one known active incident with a single `resolve incident --key` command and receive a per-target success result.
- **SC-002**: A user can resolve multiple unique incident keys from repeated flags, stdin, or both, with duplicates reported only once.
- **SC-003**: A user can resolve all incidents discovered at command start for one process instance with a single `resolve pi --key` command.
- **SC-004**: A process instance with no active incidents completes without failure and clearly reports that no incidents required resolution.
- **SC-005**: With `--no-wait`, accepted resolution requests return without confirmation polling.
- **SC-006**: With `--dry-run`, commands render the incidents that would be resolved and submit no mutation.
- **SC-007**: `--json --dry-run` produces a stable plan payload with mutation-submission status false.
- **SC-008**: Partial failures, confirmation timeouts, and unsupported-version failures are visible in per-target human and JSON output.
- **SC-009**: Existing process-instance get and update workflows covered by this feature's regression checks remain unchanged.

## Assumptions

- Operators already have permission and connectivity to perform supported Camunda state-changing operations through c8volt.
- Incident resolution endpoints are available only for some configured Camunda versions; unsupported versions must be rejected before mutation.
- The process-instance command path may coordinate incident lookup and resolution, but incident lookup and resolution behavior remains owned by the incident service boundary.
- New commands follow the repository's existing CLI patterns for aliases, key selectors, stdin handling, workers, activity metadata, dry-run planning, output modes, and waits.
- Incident creation, job retry, variable mutation, interactive prompts, process-instance searching by filters, and endless polling for incidents created after command start are out of scope.
