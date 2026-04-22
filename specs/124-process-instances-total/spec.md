# Feature Specification: Process Instances Total Output

**Feature Branch**: `124-process-instances-total`  
**Created**: 2026-04-22  
**Status**: Draft  
**Input**: User description: "GitHub issue #124: feat: add `--total` flag to return number of found process instances"

## GitHub Issue Traceability

- **Issue Number**: 124
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/124
- **Issue Title**: feat: add `--total` flag to return number of found process instances

## Clarifications

### Session 2026-04-22

- Q: How should `--total` behave when the backend-reported total is capped and only represents a lower bound? → A: Return the backend-reported count as-is, even when it is only a lower bound.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Return Count Only (Priority: P1)

As a CLI user running scripts or checks, I want the process-instances list command to return only the total number of matches so that I can consume the result without parsing table or detail output.

**Why this priority**: Count-only output is the core value requested in the issue and directly supports automation, validation, and monitoring use cases.

**Independent Test**: Run the relevant process-instances list command with filters that match a known number of instances and verify that `--total` returns exactly one numeric value with no surrounding descriptive text.

**Acceptance Scenarios**:

1. **Given** a process-instances list query matches one or more instances, **When** the operator runs the command with `--total`, **Then** the command outputs a single numeric value representing the total matches.
2. **Given** a process-instances list query matches no instances, **When** the operator runs the command with `--total`, **Then** the command outputs `0` as the only output value.
3. **Given** the operator uses `--total` in automation, **When** standard command output is captured, **Then** the result is machine-friendly and does not include matching-instance details.

---

### User Story 2 - Preserve Existing List Behavior (Priority: P2)

As a CLI user who inspects matching instances, I want the command to behave exactly as it does today when I do not ask for count-only output so that existing workflows and expectations remain unchanged.

**Why this priority**: The feature is additive, and preserving current behavior avoids regressions for interactive and existing scripted usage.

**Independent Test**: Run the same process-instances list command without `--total` before and after the change and verify that the existing output structure remains unchanged.

**Acceptance Scenarios**:

1. **Given** the operator runs the process-instances list command without `--total`, **When** results are returned, **Then** the command shows the same instance-detail output it already provides today.
2. **Given** existing scripts or documentation rely on the current default output, **When** `--total` is not present, **Then** they continue to work without modification.

---

### User Story 3 - Understand the New Flag Quickly (Priority: P3)

As a CLI user discovering command options, I want the help text to explain the `--total` flag clearly so that I know it switches the command to count-only output.

**Why this priority**: Clear help text reduces misuse and makes the new behavior discoverable without requiring release notes or source inspection.

**Independent Test**: Inspect the relevant command help and confirm it describes `--total` as returning only the number of found process instances.

**Acceptance Scenarios**:

1. **Given** a user views command help, **When** the `--total` option is listed, **Then** the description makes clear that the command returns only the count of found process instances.
2. **Given** a user compares the normal command and the `--total` variant, **When** they read the help text, **Then** the difference in output intent is unambiguous.

### Edge Cases

- When the filtered query matches zero process instances, the command must still emit a numeric count-only result rather than empty output.
- When `--total` is combined with filters that would normally return many rows, the command must still avoid emitting matching-instance details.
- When callers expect machine-friendly output, the count-only result must not add headers, labels, or explanatory prose beyond the number itself.
- If the command already supports other output-affecting options, the count-only mode must have a clear and predictable outcome that does not silently fall back to detail output.
- When the backend reports a capped total that is only a lower bound to the true match count, `--total` must still return that numeric backend-provided value rather than forcing a full recount or failing.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a `--total` flag on the relevant process-instances list command.
- **FR-002**: When `--total` is provided, the system MUST output only a single numeric value representing the total number of matching process instances.
- **FR-003**: When `--total` is provided and no process instances match, the system MUST output `0` as the only result value.
- **FR-004**: When `--total` is provided, the system MUST suppress the normal matching-instance detail output for that command invocation.
- **FR-005**: When `--total` is not provided, the system MUST preserve the command's existing externally observable output behavior.
- **FR-006**: The system MUST make the `--total` output machine-friendly by avoiding additional descriptive text in count-only mode unless an existing CLI convention requires it for the specific command family.
- **FR-006a**: When the backend-reported total is capped and represents only a lower bound, the system MUST return that backend-provided numeric value unchanged in `--total` mode.
- **FR-007**: The system MUST describe the purpose of `--total` in the relevant command help text or user-facing command documentation.
- **FR-008**: Automated test coverage MUST verify both the preserved default output behavior and the count-only behavior introduced by `--total`.

### Key Entities *(include if feature involves data)*

- **Process Instance Match Set**: The full set of process instances that satisfy the list command's current filters for a given invocation.
- **Count-Only Output**: The single numeric result value returned when `--total` is requested instead of full instance details.
- **Default Detail Output**: The command's existing non-`--total` result presentation, which must remain unchanged by this feature.
- **Command Help Entry**: The user-facing option description that explains when and why to use `--total`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests confirm that the relevant process-instances list command returns exactly one numeric value when `--total` is used.
- **SC-002**: Automated tests confirm that a no-match query returns `0` as the only output when `--total` is used.
- **SC-003**: Automated tests confirm that the command's default output remains unchanged when `--total` is not provided.
- **SC-003a**: Automated tests confirm that when the backend exposes a capped total as a lower bound, `--total` returns that numeric lower-bound value unchanged.
- **SC-004**: Command help or user-facing option text clearly states that `--total` returns only the number of found process instances.

## Assumptions

- The issue's "relevant command" refers to the existing process-instances list command surface already used to enumerate matching process instances.
- Count-only mode is intended for scripting, validation, and monitoring workflows where a raw numeric value is easier to consume than detail output.
- The feature is additive and should not change filtering semantics, matching rules, or default output formatting outside `--total` mode.
- Any required CLI conventions for numeric output should be preserved as long as the result remains a single machine-friendly count value.
- When backend totals are capped, the backend-reported numeric total is still useful enough for the count-only workflow even if it is a lower bound rather than an exact count.
