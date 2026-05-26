# Feature Specification: BPMN Selector Validation for Operational Commands

**Feature Branch**: `207-bpmn-selector-validation`

**Created**: 2026-05-23

**Status**: Draft

**Input**: GitHub issue #207 - "fix(cli): apply BPMN selector validation to every --bpmn-process-id operation"

**Issue Traceability**:

- **Issue Number**: #207
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/207
- **Issue Title**: fix(cli): apply BPMN selector validation to every --bpmn-process-id operation

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Block no-op mutations from missing BPMN selectors (Priority: P1)

As an operator preparing to cancel or delete process instances by BPMN process ID, I need a typo, invisible definition, tenant mismatch, version mismatch, or permission mismatch to fail before any search or mutation planning so I do not mistake a bad selector for a successful empty operation.

**Why this priority**: `cancel pi` and `delete pi` are confirmed gaps for state-changing workflows. A misleading `found: 0` result can hide operator error before destructive or safety-critical work.

**Independent Test**: Run `cancel pi -b <missing>` and `delete pi -b <missing>` in human, machine, and non-interactive modes. Each command must fail with the shared missing visible process-definition diagnostic before process-instance search paging, dry-run planning, confirmation, or mutation.

**Acceptance Scenarios**:

1. **Given** no visible process definition matches the provided BPMN process ID in the effective tenant and version context, **When** an operator runs `cancel pi -b <missing>`, **Then** the command fails with `no visible process definition matches the provided selector: [<missing>]` before listing or mutating process instances.
2. **Given** no visible process definition matches the provided BPMN process ID in the effective tenant and version context, **When** an operator runs `delete pi -b <missing>`, **Then** the command fails with the same shared selector diagnostic before dry-run planning, confirmation, or deletion.
3. **Given** a visible process definition exists but has zero matching process instances for the requested runtime filters, **When** an operator runs `cancel pi -b <id>` or `delete pi -b <id>`, **Then** the existing valid empty-result behavior is preserved.

---

### User Story 2 - Validate incident searches by BPMN selector (Priority: P2)

As an operator investigating incidents by BPMN process ID, I need `get incident -b <id>` to reject a missing or invisible process-definition selector so incident search filters cannot hide a bad BPMN value behind an empty incident result.

**Why this priority**: The issue calls out `get incident -b` as needing an explicit decision. Because this flag means "incidents for this BPMN process definition", it should follow the same selector visibility rule as process-instance search.

**Independent Test**: Run `get incident -b <missing>` with human output and machine-oriented modes. The command must fail with the shared selector diagnostic before incident search paging. A visible definition with no matching incidents must still report a valid empty incident result.

**Acceptance Scenarios**:

1. **Given** no visible process definition matches the BPMN process ID, **When** an operator runs `get incident -b <missing>`, **Then** the command fails with the shared missing visible process-definition diagnostic before querying incident pages.
2. **Given** the BPMN process ID is visible and there are no matching incidents, **When** an operator runs `get incident -b <id>`, **Then** the command preserves the existing empty incident-list behavior.
3. **Given** output mode is `--json`, `--automation`, `--keys-only`, `--pi-keys-only`, or non-TTY, **When** selector validation fails, **Then** the command fails clearly without prompting for recovery output.

---

### User Story 3 - Audit direct process-definition selectors (Priority: P3)

As an operator listing or deleting process definitions directly, I need `get pd -b <id>` and `delete pd -b <id>` to have explicit, documented missing-selector behavior so direct process-definition commands do not drift from shared selector diagnostics.

**Why this priority**: These commands are direct process-definition operations rather than runtime-resource searches, so they need an explicit audit decision. Aligning their missing BPMN selector diagnostics keeps operator expectations consistent across the CLI.

**Independent Test**: Run `get pd -b <missing>` and `delete pd -b <missing>` with tenant and process-definition version selectors. Each command must either use the shared missing visible process-definition diagnostic or document a deliberately different not-found contract in help and tests.

**Acceptance Scenarios**:

1. **Given** no visible process definition matches the BPMN selector, **When** an operator runs `get pd -b <missing>`, **Then** the command provides an explicit missing process-definition selector outcome rather than an ambiguous empty result.
2. **Given** no visible process definition matches the BPMN selector, **When** an operator runs `delete pd -b <missing>`, **Then** the command fails before delete planning, confirmation, or mutation.
3. **Given** a visible process definition matches the BPMN selector and any applicable version or tenant filters, **When** an operator runs `get pd -b <id>` or `delete pd -b <id>`, **Then** existing successful selection, preview, and execution behavior is preserved.

---

### User Story 4 - Preserve pipelines and documented command contracts (Priority: P4)

As an automation user, I need selector validation to occur in the command that directly accepts `--bpmn-process-id`, while pipeline workflows continue to rely on the upstream selector command for validation.

**Why this priority**: The issue specifically distinguishes upstream pipeline validation from mutating commands that accept `-b` themselves. Documentation and command contract metadata must match the final behavior.

**Independent Test**: Verify key-pipeline workflows, machine output modes, help text, and generated documentation after the aligned commands are updated.

**Acceptance Scenarios**:

1. **Given** a pipeline such as `get pi -b <id> --keys-only | cancel pi -`, **When** the BPMN selector is missing, **Then** validation belongs to the upstream `get pi` command.
2. **Given** a mutating command directly receives `-b <missing>`, **When** it runs in automation-oriented mode, **Then** it fails before prompting and before mutation planning.
3. **Given** help or documentation describes aligned commands, **When** a user reads selector behavior, **Then** missing BPMN selectors are described as validation failures rather than simple empty searches.

### Edge Cases

- A BPMN process ID exists in a different tenant but is not visible in the effective tenant context.
- A BPMN process ID exists but the requested `--pd-version` or `--pd-version-tag` does not match any visible definition.
- Multiple runtime filters are provided and the process definition is visible, but the resulting process-instance or incident set is empty.
- Human interactive output can safely offer to list visible or matching process definitions, while machine and non-interactive modes must not prompt.
- Commands that accept stdin keys through `-` must not add BPMN selector validation unless they also directly receive `--bpmn-process-id`.
- Commands that do not expose `--bpmn-process-id` directly remain unchanged.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST audit every user-facing command that directly registers `--bpmn-process-id` and record whether the selector is validated, intentionally direct process-definition selection, or out of scope.
- **FR-002**: `get pi` and `run pi` MUST remain aligned with the existing visible process-definition selector validation behavior.
- **FR-003**: `cancel pi -b <missing>` MUST fail with the shared missing visible process-definition diagnostic before process-instance search paging, dry-run planning, confirmation, or mutation.
- **FR-004**: `delete pi -b <missing>` MUST fail with the shared missing visible process-definition diagnostic before process-instance search paging, dry-run planning, confirmation, or mutation.
- **FR-005**: `get incident -b <missing>` MUST validate the BPMN selector as a visible process-definition selector before incident search paging.
- **FR-006**: `get pd -b <missing>` and `delete pd -b <missing>` MUST be audited and aligned with an explicit missing-selector diagnostic unless a deliberate direct process-definition not-found contract is documented.
- **FR-007**: Selector validation MUST include applicable process-definition version, process-definition version tag, and effective tenant context.
- **FR-008**: Valid visible BPMN process IDs with zero matching process instances or incidents MUST preserve existing empty-result behavior.
- **FR-009**: Human interactive output MAY offer safe recovery listing of visible or matching process definitions when prompting is safe.
- **FR-010**: Non-interactive and machine-oriented modes, including `--json`, `--automation`, `--keys-only`, piped stdin/stdout, and related key-only outputs, MUST fail clearly without prompting when selector validation fails.
- **FR-011**: Commands that do not directly expose `--bpmn-process-id` MUST remain behaviorally unchanged unless this feature adds no new selector flags.
- **FR-012**: Help text, README guidance, and generated command documentation for aligned commands MUST describe missing BPMN selectors as validation failures where relevant.
- **FR-013**: Automated tests MUST cover missing-selector behavior for every aligned command and preserve valid-empty-result behavior when the process definition exists but no runtime resources match.
- **FR-014**: Pipeline behavior MUST keep selector validation in the upstream command when downstream commands receive keys through stdin instead of receiving `--bpmn-process-id` directly.

### Key Entities *(include if feature involves data)*

- **BPMN Process Definition Selector**: The user-provided BPMN process ID plus applicable process-definition version, version tag, and tenant context used to identify visible process definitions.
- **Operational Command**: A user-facing CLI command that searches, waits on, plans, or mutates runtime resources using a direct `--bpmn-process-id` selector.
- **Selector Validation Outcome**: The preflight result that distinguishes visible selector matches, missing or invisible selectors, and valid empty runtime-resource results.
- **Runtime Resource Result**: Process-instance or incident search results that remain empty only after the selector itself has been proven visible when validation applies.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of audited direct `--bpmn-process-id` commands have an explicit validation or direct-selector decision recorded in tests or documentation.
- **SC-002**: `cancel pi -b <missing>` and `delete pi -b <missing>` fail before any runtime-resource search or mutation planning in automated tests.
- **SC-003**: Every aligned missing-selector command emits the shared diagnostic text `no visible process definition matches the provided selector` or its plural equivalent.
- **SC-004**: Every aligned machine or non-interactive mode test confirms no recovery prompt is emitted after selector validation fails.
- **SC-005**: Existing valid empty-result scenarios continue to report empty process-instance or incident results when the BPMN selector is visible.
- **SC-006**: User-facing documentation and generated command reference match the implemented selector-validation behavior for all aligned commands.

## Assumptions

- The shared missing-selector diagnostic from issue #175 is the canonical user-facing wording for missing or invisible BPMN selectors.
- `get incident -b` is treated as an incident search scoped to a BPMN process definition, so a missing BPMN selector is invalid rather than a legitimate empty incident search.
- Direct process-definition commands should prefer the shared selector diagnostic for missing BPMN process IDs unless implementation review uncovers a stronger existing not-found contract that must be preserved and documented.
- The feature does not add `--bpmn-process-id` to commands that do not currently expose it directly.
- Existing confirmation, dry-run, automation, JSON, key-only, tenant, version, and version-tag semantics remain authoritative unless this specification explicitly changes selector validation timing.
