# Feature Specification: get incident process-instance key output

**Feature Branch**: `198-pi-keys-only`  
**Created**: 2026-05-10  
**Status**: Draft  
**Input**: GitHub issue [#198](https://github.com/grafvonb/c8volt/issues/198) - `feat(get): add --pi-keys-only output for get incident pipelines`

## GitHub Issue Traceability

- **Issue Number**: #198
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/198
- **Issue Title**: feat(get): add --pi-keys-only output for get incident pipelines

## Clarifications

### Session 2026-05-10

- Q: How should `--pi-keys-only` handle incident items without `processInstanceKey`? -> A: Skip incident items without `processInstanceKey` and continue rendering the rest.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Pipe incident matches into process-instance commands (Priority: P1)

As a Camunda operator, I want `get incident` to emit process instance keys for matching incidents so that I can pipe incident selections into process-instance commands without manually extracting fields.

**Why this priority**: This is the core pipeline workflow requested by the issue and provides the smallest useful increment.

**Independent Test**: Can be fully tested by running `get incident` with `--pi-keys-only` in keyed and search modes and verifying only process instance keys are printed.

**Acceptance Scenarios**:

1. **Given** a known incident with process instance key `2251799813711967`, **When** the user runs `c8volt get incident --key <incident-key> --pi-keys-only`, **Then** the command prints `2251799813711967` as the only output line.
2. **Given** active incident search results, **When** the user runs `c8volt get incident --state active --pi-keys-only`, **Then** each matching incident contributes one output line containing its process instance key.
3. **Given** two matching incidents on the same process instance, **When** the user runs `c8volt get incident --state active --pi-keys-only`, **Then** the same process instance key appears twice.
4. **Given** a paged incident search, **When** `--pi-keys-only` output is rendered page by page, **Then** every rendered row is a process instance key and no human row or summary footer is emitted.

---

### User Story 2 - Avoid ambiguous output mode combinations (Priority: P2)

As a script author, I want `--pi-keys-only` to reject incompatible output modifiers so that command output remains predictable for pipelines.

**Why this priority**: Pipeline output must stay machine-safe; ambiguous render modes can break downstream destructive commands.

**Independent Test**: Can be tested by combining `--pi-keys-only` with each incompatible flag and verifying a local validation error occurs before remote calls.

**Acceptance Scenarios**:

1. **Given** a user requests JSON output, **When** they also pass `--pi-keys-only`, **Then** the command fails with a mutual-exclusion error.
2. **Given** a user requests incident keys with `--keys-only`, **When** they also pass `--pi-keys-only`, **Then** the command fails with a mutual-exclusion error.
3. **Given** a user requests totals or message formatting options, **When** they also pass `--pi-keys-only`, **Then** the command fails with a mutual-exclusion error.

---

### User Story 3 - Keep docs and command metadata aligned (Priority: P3)

As an operator reading help or generated documentation, I want `--pi-keys-only` to be documented alongside existing incident output modes so that I can discover the pipeline workflow safely.

**Why this priority**: The command is intended for shell composition, so help text and reference docs need to show the distinction between incident-key and process-instance-key output.

**Independent Test**: Can be tested by checking command help, command capability metadata, generated CLI docs, and user-facing examples for the new flag and pipeline example.

**Acceptance Scenarios**:

1. **Given** the user views `c8volt get incident --help`, **When** help is rendered, **Then** it describes `--pi-keys-only` as process-instance-key output and distinguishes it from `--keys-only`.
2. **Given** generated command documentation is refreshed, **When** the incident reference is inspected, **Then** it includes the new flag and an incident-to-process-instance pipeline example.

---

### User Story 4 - Normalize delete process-instance duplicate stdin handling (Priority: P4)

As a maintainer, I want `delete pi` to dedupe merged key inputs at the command boundary like `cancel pi` so that duplicate stdin keys do not inflate command-local planning counts.

**Why this priority**: This is a small cleanup adjacent to pipeline safety, but it must not distract from the `--pi-keys-only` feature.

**Independent Test**: Can be tested by providing duplicate stdin keys to `delete pi` and verifying the command-local planning path receives unique keys.

**Acceptance Scenarios**:

1. **Given** duplicate process instance keys on stdin, **When** the user runs `c8volt delete pi -`, **Then** duplicate key input is accepted and the delete planning path receives each process instance key once.
2. **Given** duplicate process instance keys are emitted by `get incident --pi-keys-only`, **When** those keys are piped to `delete pi`, **Then** `delete pi` may dedupe before deletion while `get incident --pi-keys-only` continues to preserve duplicate output.

### Edge Cases

- Incident records without a process instance key must not produce malformed pipeline output; `--pi-keys-only` should skip those incident items and continue rendering remaining process instance keys.
- `--pi-keys-only` must work in both keyed lookup and search/list modes.
- Duplicate process instance keys are valid output for `--pi-keys-only` v1 and must not be removed by the incident command.
- Local validation for incompatible flags must happen before remote incident lookup or search.
- Existing `--keys-only` output must continue to emit incident keys.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `get incident`, `get incidents`, and `get inc` MUST support a command-local `--pi-keys-only` flag.
- **FR-002**: With `--pi-keys-only`, the command MUST print one process instance key per incident item.
- **FR-003**: With `--pi-keys-only`, duplicate process instance keys MUST be preserved in incident output.
- **FR-004**: `--pi-keys-only` MUST support direct keyed incident lookup.
- **FR-005**: `--pi-keys-only` MUST support incident search/list mode, including paged and incremental rendering paths.
- **FR-006**: `--pi-keys-only` MUST skip incident items without a process instance key and continue rendering remaining process instance keys.
- **FR-007**: `--pi-keys-only` MUST preserve existing incident filtering, paging, limit, automation, stdin, and error behavior except for the selected output field and missing-process-instance-key skip behavior.
- **FR-008**: `--pi-keys-only` MUST be mutually exclusive with `--keys-only`, `--json`, `--total`, `--error-message-limit`, and `--with-no-error-message`.
- **FR-009**: Existing `--keys-only` behavior MUST remain unchanged and continue emitting incident keys.
- **FR-010**: Help, command metadata, generated CLI docs, and user-facing examples MUST describe the new flag and its intended process-instance pipeline use.
- **FR-011**: If safe and local, `delete pi` SHOULD dedupe merged flag and stdin keys at the command boundary, matching `cancel pi` behavior.
- **FR-012**: The `delete pi` dedupe cleanup MUST NOT alter the `get incident --pi-keys-only` duplicate-output contract.

### Key Entities *(include if feature involves data)*

- **Incident Result**: A matched incident row containing both an incident key and its associated process instance key.
- **Pipeline Key Output**: A line-oriented machine output mode where each line is a key intended for downstream commands.
- **Process Instance Target Key**: The process instance key emitted from an incident result and accepted by process-instance commands.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can pipe `get incident --pi-keys-only` into a process-instance command without manual field extraction.
- **SC-002**: Keyed lookup, unpaged search, and paged search each emit process instance keys under `--pi-keys-only`.
- **SC-003**: Duplicate process instance keys from multiple matching incidents are preserved in `--pi-keys-only` output.
- **SC-004**: Incompatible output-mode combinations fail locally with clear mutual-exclusion diagnostics.
- **SC-005**: Existing incident-key pipelines using `--keys-only` continue to pass unchanged.
- **SC-006**: Documentation and command metadata expose the new flag and at least one process-instance pipeline example.

## Assumptions

- `get incident` results already carry a process instance key for the incident records this command renders.
- Downstream process-instance commands may dedupe stdin targets independently; `get incident --pi-keys-only` is responsible only for faithful incident-row output.
- The delete process-instance cleanup is limited to immediate command-boundary deduplication when that can be done without broader behavioral changes.
