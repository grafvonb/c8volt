# Feature Specification: Run Confirmation Observes Real Process Instance States

**Feature Branch**: `225-run-observable-keys`

**Created**: 2026-05-23

**Status**: Draft

**GitHub Issue**: [#225](https://github.com/grafvonb/c8volt/issues/225) - fix(run): accept observable PI states and support keys-only output

**Input**: GitHub issue #225 plus mandatory Ralph implementation context: every planning, task-generation, and Ralph implementation handoff must read and apply `specs/ralph-implementation-rules.md`; Ralph must not be launched unless implementation instructions include `--implementation-context specs/ralph-implementation-rules.md`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Confirm Fast Process Instance Runs (Priority: P1)

As a CLI user running a process that completes very quickly, I want run-style commands to treat any observable real process instance lifecycle state as successful creation confirmation so completed processes do not time out after succeeding.

**Why this priority**: This fixes the core failure where successful fast process starts are reported as timeouts.

**Independent Test**: Run a process definition that completes immediately and verify the run command succeeds without waiting for an `active` observation.

**Acceptance Scenarios**:

1. **Given** a process definition that completes before the waiter observes `active`, **When** a user starts it with `c8volt run pi`, **Then** the command succeeds after observing `completed`.
2. **Given** a process instance that is observable as `active`, **When** a user starts it with a run-style command, **Then** the command succeeds as it does today.
3. **Given** a created process instance cannot be observed or is reported as absent, **When** the run-style command waits for confirmation, **Then** the command does not treat that missing observation as success.

---

### User Story 2 - See The Observed Process Instance State (Priority: P2)

As a CLI user inspecting run results, I want the rendered process instance details to include the actual observed state so I can distinguish long-running active instances from fast completed or terminal instances.

**Why this priority**: Once confirmation accepts more than `active`, users need visible evidence of the state that satisfied confirmation.

**Independent Test**: Run commands that render process instance details and verify normal and JSON output include the observed state.

**Acceptance Scenarios**:

1. **Given** a run-style command observes a created process instance as `completed`, **When** the command renders process instance details, **Then** the output includes `completed` as the process instance state.
2. **Given** a run-style command observes a created process instance as `active`, **When** the command renders process instance details, **Then** the output includes `active` as the process instance state.

---

### User Story 3 - Pipe Created Keys Into Strict Expectations (Priority: P3)

As a CLI user composing commands, I want `c8volt run pi --keys-only` to print only created process instance keys so lifecycle-specific assertions can remain in `expect pi`.

**Why this priority**: This enables reliable shell pipelines without adding expectation flags to run commands or weakening `expect pi`.

**Independent Test**: Pipe `c8volt run pi --keys-only` into `c8volt expect pi --state <state> -` and verify the downstream expectation owns the strict state assertion.

**Acceptance Scenarios**:

1. **Given** a fast process definition, **When** a user runs `c8volt run pi -b C89_NoOpCompletion_Process --keys-only | c8volt expect pi --state completed -`, **Then** the pipeline succeeds when the instance completes quickly.
2. **Given** a long-running process definition, **When** a user pipes `c8volt run pi --keys-only` into `c8volt expect pi --state active -`, **Then** the pipeline succeeds only when the downstream expectation observes `active`.
3. **Given** `c8volt run pi --keys-only` creates one or more process instances, **When** the command writes output, **Then** it prints one process instance key per line and no extra text.

### Edge Cases

- A created process instance observed as `active`, `completed`, `canceled`, or `terminated` is a successful run confirmation.
- An absent, not-found, or otherwise non-observable process instance is not a successful run confirmation.
- Unknown states are not considered successful unless they are explicitly observable real lifecycle states supported by the existing process instance model.
- Observing a terminal state such as `completed` must not emit a warning merely because the state is not `active`.
- `expect pi` remains strict; it must not inherit the broader run-confirmation success semantics.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Run-style process instance creation commands MUST treat an observed created process instance in a real lifecycle state as successful creation confirmation.
- **FR-002**: Successful observed states MUST include at least `active`, `completed`, `canceled`, and `terminated`.
- **FR-003**: Run-style process instance creation commands MUST NOT treat absent, not-found, or unknown non-observable states as successful creation confirmation.
- **FR-004**: Run-style process instance creation commands MUST NOT warn solely because the observed successful state is not `active`.
- **FR-005**: Rendered process instance details for run-style commands MUST include the actual observed process instance state in normal output.
- **FR-006**: JSON output for run-style commands that render process instance details MUST include the actual observed process instance state.
- **FR-007**: `c8volt run pi` MUST support `--keys-only` output for created process instances.
- **FR-008**: `c8volt run pi --keys-only` MUST print only created process instance keys, one key per line.
- **FR-009**: The feature MUST NOT add `--expected-status` or an equivalent lifecycle expectation flag to `run`.
- **FR-010**: `expect pi` MUST preserve strict explicit state expectations and MUST NOT use the broader run-confirmation semantics.
- **FR-011**: User-facing help, documentation, or examples MUST show the pipeline pattern where `run pi --keys-only` feeds `expect pi --state <state> -`.

### Key Entities *(include if feature involves data)*

- **Created Process Instance**: The process instance produced by `run pi`, `deploy --run`, or `embed deploy --run`; key attributes are its unique key and observed lifecycle state.
- **Observed Lifecycle State**: The state returned when the created process instance is visible after creation; accepted confirmation states include `active`, `completed`, `canceled`, and `terminated`.
- **Keys-Only Output**: A line-oriented command output mode containing only created process instance keys for downstream command composition.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A fast process definition that reaches `completed` before `active` is observed succeeds instead of timing out.
- **SC-002**: Existing long-running process instance starts that are observed as `active` continue to succeed.
- **SC-003**: Run-style output that includes process instance details exposes the actual observed state in both normal and JSON output.
- **SC-004**: `c8volt run pi --keys-only` output is valid as direct stdin for `c8volt expect pi --state <state> -`.
- **SC-005**: `expect pi` continues to fail when the explicitly expected state does not match the actual process instance state.

## Assumptions

- The relevant run-style commands are `c8volt run pi`, `c8volt deploy --run`, and `c8volt embed deploy --run`.
- The existing process instance model can represent the observable states needed for confirmation and rendering.
- Strict lifecycle validation belongs in `expect pi`, not in `run`.
- Documentation updates should follow the repository's generated-docs workflow when command metadata changes.
- Ralph planning, task generation, and implementation must receive `specs/ralph-implementation-rules.md` as implementation context.
