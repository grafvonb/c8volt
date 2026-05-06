# Feature Specification: Validate Process Definition Selectors for Process-Instance Commands

**Feature Branch**: `175-validate-pd-selectors`  
**Created**: 2026-05-06  
**Status**: Draft  
**Input**: User description: "GitHub issue #175: fix(pi): validate BPMN process definition selectors before process-instance operations"

## GitHub Issue Traceability

- **Issue Number**: 175
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/175
- **Issue Title**: fix(pi): validate BPMN process definition selectors before process-instance operations

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Detect Missing Selectors Before Empty Results (Priority: P1)

As an operator searching process instances by BPMN process ID, I want c8volt to distinguish a missing or invisible process definition from an existing process definition with zero process instances so that I do not misdiagnose a typo, tenant mismatch, version mismatch, or permission issue as an empty result set.

**Why this priority**: This is the core false-positive bug and provides the smallest independently useful correction for read-only process-instance inspection.

**Independent Test**: Run `c8volt get pi --bpmn-process-id <missing>` and verify c8volt validates the process-definition selector first, reports that no visible process definition matches, and does not print only `found: 0`.

**Acceptance Scenarios**:

1. **Given** no visible process definition matches a supplied BPMN process ID, **When** the user runs `c8volt get pi --bpmn-process-id <missing>`, **Then** c8volt fails before process-instance search output and explains that no visible process definition matched the selector.
2. **Given** a visible process definition exists for the supplied BPMN process ID but has no matching process instances, **When** the user runs `c8volt get pi --bpmn-process-id <existing>`, **Then** c8volt preserves the valid empty-result behavior and may print `found: 0`.
3. **Given** `--pd-version`, `--pd-version-tag`, or an effective tenant context narrows the selector, **When** no visible process definition matches that full selector context, **Then** c8volt reports the selector validation failure instead of treating the process-instance result set as simply empty.

---

### User Story 2 - Provide Safe Human and Automation Diagnostics (Priority: P2)

As a user running c8volt interactively or from automation, I want missing process-definition selectors to produce output appropriate to the execution mode so that humans can recover quickly and scripts never block on prompts.

**Why this priority**: Clear diagnostics and non-blocking automation behavior are required before the validation can be applied safely across command modes.

**Independent Test**: Exercise a missing BPMN process ID in interactive human output and in automation-oriented modes, then verify interactive output can offer visible process-definition listing while `--json`, `--automation`, `--keys-only`, and non-TTY usage fail clearly without prompting.

**Acceptance Scenarios**:

1. **Given** human interactive output is available and no visible process definition matches, **When** c8volt reports the selector failure, **Then** it shows the missing selector values and may offer to list visible process definitions in the existing process-definition list format.
2. **Given** the user accepts the visible-definition prompt, **When** c8volt lists visible process definitions, **Then** it uses the existing process-definition list format and includes the final `found: <n>` summary.
3. **Given** the command runs with `--json`, `--automation`, `--keys-only`, or non-TTY input/output where prompting would block, **When** selector validation fails, **Then** c8volt returns a clear error without prompting.
4. **Given** structured output is requested, **When** selector validation fails, **Then** the error behavior remains compatible with existing structured output conventions.

---

### User Story 3 - Guard Mutating Process-Instance Commands (Priority: P3)

As an operator canceling or deleting process instances by BPMN process ID, I want c8volt to validate the process-definition selector before mutation so that a mistyped or invisible selector fails safely and does not look like a successful no-op.

**Why this priority**: Mutating commands need the same selector correctness guarantee as read-only commands, but they depend on the shared validation and diagnostic behavior from earlier stories.

**Independent Test**: Run `c8volt cancel pi --bpmn-process-id <missing>` and `c8volt delete pi --bpmn-process-id <missing>` in non-interactive mode and verify each command fails before mutation flow with the shared selector validation diagnostic.

**Acceptance Scenarios**:

1. **Given** no visible process definition matches the supplied selector, **When** the user runs `c8volt cancel pi --bpmn-process-id <missing>`, **Then** c8volt fails before canceling any process instances.
2. **Given** no visible process definition matches the supplied selector, **When** the user runs `c8volt delete pi --bpmn-process-id <missing>`, **Then** c8volt fails before deleting any process instances.
3. **Given** the supplied selector matches a visible process definition with zero process instances, **When** the user runs a guarded mutating command, **Then** existing no-match process-instance behavior is preserved.

---

### User Story 4 - Prevent Partial Multi-ID Starts (Priority: P4)

As an operator starting process instances for one or more BPMN process IDs, I want c8volt to validate every provided BPMN process ID before starting anything so that a mixed valid/missing request cannot create partial process instances by accident.

**Why this priority**: Multi-ID all-or-nothing validation prevents accidental partial work, but it can be implemented after the shared single-selector validation contract exists.

**Independent Test**: Run `c8volt run pi --bpmn-process-id <existing> --bpmn-process-id <missing>` and verify c8volt reports all missing selectors and starts no process instances.

**Acceptance Scenarios**:

1. **Given** one or more provided BPMN process IDs do not match visible process definitions, **When** the user runs `c8volt run pi` with multiple BPMN process IDs, **Then** c8volt reports the missing selector values and starts no process instances.
2. **Given** multiple provided BPMN process IDs all match visible process definitions, **When** the user runs `c8volt run pi`, **Then** existing start behavior proceeds.
3. **Given** multiple missing BPMN process IDs are supplied, **When** validation fails, **Then** c8volt lists each missing BPMN process ID in the diagnostic.

### Edge Cases

- A BPMN process ID is misspelled by one or more characters.
- The BPMN process ID exists in the cluster but is filtered out by `--pd-version`.
- The BPMN process ID exists in the cluster but is filtered out by `--pd-version-tag`.
- The BPMN process ID exists in another tenant but is not visible in the effective tenant context.
- The BPMN process ID is not visible to the current credentials.
- The BPMN process ID exists and is visible, but no process instances match the process-instance query.
- Multiple BPMN process IDs are provided and only some are visible.
- Multiple BPMN process IDs are provided and none are visible.
- Interactive prompting is unavailable or unsafe because stdout/stdin is non-TTY.
- Structured output or automation-oriented modes are active.
- Visible process-definition listing itself returns zero entries or fails after the user accepts the prompt.
- Existing process-instance command behavior without `--bpmn-process-id` remains unchanged.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST validate visible process definitions before process-instance operations when `--bpmn-process-id` is provided.
- **FR-002**: The validation MUST use the same selector context as the process-instance command, including BPMN process ID or IDs, `--pd-version`, `--pd-version-tag`, and effective tenant context.
- **FR-003**: If no visible process definition matches the selector context, the command MUST fail before continuing to process-instance search, mutation, or start behavior.
- **FR-004**: `c8volt get pi` MUST distinguish a missing or invisible process definition from an existing visible process definition with zero matching process instances.
- **FR-005**: Existing valid empty-result behavior MUST be preserved when the process-definition selector is visible and the process-instance result set is empty.
- **FR-006**: Selector validation MUST apply consistently to at least `c8volt get pi`, `c8volt cancel pi`, `c8volt delete pi`, and `c8volt run pi`.
- **FR-007**: Commands that accept multiple BPMN process IDs MUST validate all provided BPMN process IDs before starting or mutating any process instances.
- **FR-008**: If any BPMN process ID in a multi-ID request is not visible, the command MUST fail without partial process-instance starts or mutations.
- **FR-009**: Human interactive diagnostics MUST show the provided selector values that failed validation.
- **FR-010**: Human interactive diagnostics MUST explain that the selector may not exist, may not match version/tag/tenant filters, or may not be visible to the current credentials.
- **FR-011**: Human interactive output MAY offer to list visible process definitions after a validation failure.
- **FR-012**: If the user accepts the visible-definition prompt, c8volt MUST list accessible process definitions using the existing process-definition list output format.
- **FR-013**: Automation-oriented modes, including `--json`, `--automation`, `--keys-only`, and non-TTY usage where prompting would block, MUST fail clearly without interactive prompting.
- **FR-014**: Structured-output error behavior MUST remain compatible with existing repository conventions for machine-readable command failures.
- **FR-015**: Automated tests MUST cover missing process-definition selector behavior for read-only process-instance search.
- **FR-016**: Automated tests MUST cover the existing visible process-definition with zero matching process instances case.
- **FR-017**: Automated tests MUST cover version, version-tag, and tenant selector context in process-definition validation.
- **FR-018**: Automated tests MUST cover multi-ID validation before any starts or mutations.
- **FR-019**: Automated tests MUST cover interactive prompt eligibility and non-interactive no-prompt behavior.
- **FR-020**: User-facing help or documentation MUST stay in sync if command behavior, diagnostics, or examples change.

### Key Entities *(include if feature involves data)*

- **Process Definition Selector**: The user-provided BPMN process ID or IDs plus optional process-definition version, process-definition version tag, and effective tenant context.
- **Visible Process Definition**: A deployed process definition that matches the selector context and is accessible with the current credentials.
- **Process-Instance Operation**: A `get pi`, `cancel pi`, `delete pi`, or `run pi` command path that uses a BPMN process ID to search, mutate, or start process instances.
- **Selector Validation Result**: The command decision that either confirms all requested BPMN process IDs are visible or identifies missing/invisible IDs before continuing.
- **Diagnostic Listing Offer**: An interactive human-output recovery prompt that can list visible process definitions using the existing process-definition list format.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show `get pi --bpmn-process-id <missing>` fails with a missing visible process-definition diagnostic instead of only printing `found: 0`.
- **SC-002**: Automated tests show `get pi --bpmn-process-id <existing>` still reports a valid empty process-instance result when the process definition is visible but no instances match.
- **SC-003**: Automated tests show validation includes `--pd-version`, `--pd-version-tag`, and effective tenant context.
- **SC-004**: Automated tests show `cancel pi` and `delete pi` fail before mutation when the BPMN process ID is not visible.
- **SC-005**: Automated tests show `run pi` validates all provided BPMN process IDs before creating any process instances.
- **SC-006**: Automated tests show multi-ID requests fail without partial work when at least one BPMN process ID is not visible.
- **SC-007**: Automated tests show interactive human output can offer visible process-definition listing and renders that listing in the existing process-definition list format when accepted.
- **SC-008**: Automated tests show `--json`, `--automation`, `--keys-only`, and non-TTY execution fail clearly without prompting.
- **SC-009**: Existing process-instance command tests pass when `--bpmn-process-id` is absent.
- **SC-010**: Relevant command, service, facade, documentation generation, and repository validation checks pass.

## Assumptions

- The affected command names include both full command names and aliases where the repository exposes them, such as `process-instance` and `pi`.
- Validation should reuse existing process-definition search/listing capabilities and command output patterns where practical.
- Tenant handling should follow the repository's existing effective tenant resolution rules.
- Interactive prompting should follow existing prompt eligibility conventions rather than introducing a new prompt policy.
- The visible-definition offer is a recovery aid for human output, not a required step for automation modes.
- The first implementation should avoid changing behavior for process-instance commands that do not provide `--bpmn-process-id`.
