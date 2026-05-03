# Feature Specification: Resolve Process Instance From User Task Key

**Feature Branch**: `152-task-key-pi-lookup`  
**Created**: 2026-04-30  
**Status**: Draft  
**Input**: User description: "GitHub issue #152: feat(get pi): resolve process instance from user task key"

## GitHub Issue Traceability

- **Issue Number**: 152
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/152
- **Issue Title**: feat(get pi): resolve process instance from user task key

## Clarifications

### Session 2026-04-30

- No critical ambiguities detected worth formal clarification. The GitHub issue defines the affected command, supported versions, forbidden API fallbacks, selector conflicts, rendering expectations, documentation updates, and required test coverage.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Lookup Process Instance By User Task Key (Priority: P1)

As a CLI operator who has one or more Camunda user task keys, I want `get process-instance` / `get pi` to resolve and display the owning process instances so that I can inspect the instances without doing manual lookups first.

**Why this priority**: This is the primary value of the feature and gives operators a direct workflow from task tooling to process-instance inspection.

**Independent Test**: Run `get pi --has-user-tasks=<task-key>` and repeated `--has-user-tasks` invocations against 8.8 and 8.9 fixtures where each user task references a known process instance, then verify output matches direct keyed process-instance lookup for the resolved keys.

**Acceptance Scenarios**:

1. **Given** a Camunda 8.9 user task belongs to process instance `P`, **When** the operator runs `c8volt get pi --has-user-tasks=<task-key>`, **Then** the command returns the same human-readable process-instance output as `c8volt get pi --key=P`.
2. **Given** a Camunda 8.8 user task belongs to process instance `P`, **When** the operator runs `c8volt get process-instance --has-user-tasks=<task-key>`, **Then** the command returns the owning process instance using the existing single-instance lookup behavior.
3. **Given** the operator requests JSON output, **When** `c8volt get pi --has-user-tasks=<task-key> --json` resolves the task, **Then** the JSON shape matches direct keyed process-instance lookup for the resolved process instance.

---

### User Story 2 - Reject Unsupported Or Ambiguous Task-Key Requests (Priority: P2)

As a CLI user or script author, I want unsupported versions and conflicting selectors to fail clearly so that has-user-tasks lookup never silently falls back to the wrong API or combines incompatible lookup modes.

**Why this priority**: The issue explicitly forbids Tasklist and Operate fallbacks and defines `--has-user-tasks` as a lookup selector, so safe rejection behavior is required before the feature can be trusted.

**Independent Test**: Run the command on 8.7 and with every invalid selector combination, then verify each invocation fails before process-instance lookup or fallback API usage occurs.

**Acceptance Scenarios**:

1. **Given** the active Camunda version is 8.7, **When** the operator runs `c8volt get pi --has-user-tasks=<task-key>`, **Then** the command fails with an explicit unsupported-version error.
2. **Given** the operator supplies `--has-user-tasks` and `--key`, **When** validation runs, **Then** the command rejects the invocation as a selector conflict.
3. **Given** the operator supplies `--has-user-tasks` and stdin key input via `-`, **When** validation runs, **Then** the command rejects the invocation as a selector conflict.
4. **Given** the operator supplies `--has-user-tasks` with search filters, `--total`, or `--limit`, **When** validation runs, **Then** the command rejects the invocation because has-user-tasks lookup is not a search mode.

---

### User Story 3 - Preserve Existing Single Lookup Rendering Options (Priority: P3)

As a CLI operator, I want has-user-tasks lookup to work with output flags already valid for direct process-instance lookup so that user-task-key based inspection feels like the same command after resolution.

**Why this priority**: The feature should add a new selector without creating a parallel output model or surprising differences from keyed process-instance lookup.

**Independent Test**: Run has-user-tasks lookup with `--json` and any existing single-lookup render flags supported by direct keyed lookup, then compare behavior to the resolved `--key` invocation.

**Acceptance Scenarios**:

1. **Given** direct keyed lookup supports `--keys-only`, **When** the operator runs `get pi --has-user-tasks=<task-key> --keys-only`, **Then** output follows the same keys-only behavior for the resolved process instance.
2. **Given** direct keyed lookup includes age output, **When** the operator runs `get pi --has-user-tasks=<task-key>`, **Then** output follows the same age-rendering behavior for the resolved process instance.
3. **Given** a resolved user task lacks a usable owning process-instance key, **When** has-user-tasks lookup runs, **Then** the command fails with a clear resolution error instead of rendering incomplete process-instance data.

---

### User Story 4 - Discover Task-Key Lookup In Help And Docs (Priority: P4)

As a CLI user reading command help or documentation, I want concrete has-user-tasks examples and version limits so that I can use the feature without source inspection.

**Why this priority**: This is a user-facing command contract change and the project constitution requires documentation to match behavior.

**Independent Test**: Inspect help text, README examples, and generated CLI docs to verify has-user-tasks usage, selector conflicts, and 8.7 unsupported behavior are accurately documented.

**Acceptance Scenarios**:

1. **Given** a user views help for `get process-instance` or `get pi`, **When** `--has-user-tasks` is listed, **Then** the help text describes it as a user task key selector.
2. **Given** a user reads examples, **When** has-user-tasks lookup is documented, **Then** examples include human and JSON lookup forms.
3. **Given** generated CLI docs are refreshed, **When** the affected command pages are inspected, **Then** they include the new flag and examples without introducing Tasklist or Operate fallback guidance.

### Edge Cases

- Camunda 8.7 must fail explicitly when `--has-user-tasks` is present before any fallback API call is attempted.
- A missing user task must return a clear not-found style error consistent with existing command behavior.
- A user task response without a usable owning process-instance key must return a clear resolution error.
- After task ownership is resolved, process-instance not-found and process-instance API errors must behave like direct keyed lookup for the resolved process-instance key.
- `--has-user-tasks` must be mutually exclusive with `--key`, stdin key input via `-`, process-instance search filters, `--total`, and `--limit`.
- `--has-user-tasks` must remain compatible with output and rendering flags already valid for direct single process-instance lookup.
- Tenant-aware behavior must remain consistent with existing process-instance lookup flows.
- No implementation path may use Tasklist or Operate to satisfy has-user-tasks lookup.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add `--has-user-tasks` to `get process-instance` and `get pi`.
- **FR-002**: `--has-user-tasks` MUST identify one or more Camunda user task keys whose owning process instances should be fetched.
- **FR-003**: On Camunda 8.9, the system MUST resolve user tasks through tenant-aware native Camunda v2 user-task search available to the generated client.
- **FR-004**: On Camunda 8.8, the system MUST resolve user tasks through tenant-aware native Camunda v2 user-task search available to the generated client.
- **FR-005**: On Camunda 8.7, the system MUST fail explicitly as unsupported whenever `--has-user-tasks` is used.
- **FR-006**: The system MUST NOT use Tasklist API calls to resolve has-user-tasks lookup.
- **FR-007**: The system MUST NOT use Operate API calls to resolve has-user-tasks lookup.
- **FR-008**: The system MUST extract the owning process-instance key from the resolved user task result.
- **FR-009**: After resolving the owning process-instance key, the system MUST reuse the existing process-instance lookup and rendering path.
- **FR-010**: Human-readable output for has-user-tasks lookup MUST match direct keyed process-instance lookup as closely as possible for the resolved key.
- **FR-011**: JSON output for has-user-tasks lookup MUST match the existing JSON shape for direct keyed process-instance lookup.
- **FR-012**: Existing process-instance not-found and process-instance API error behavior MUST be preserved after the process-instance key is resolved.
- **FR-013**: Missing user tasks MUST produce a clear not-found style error.
- **FR-014**: User task responses without a usable process-instance key MUST produce a clear resolution error.
- **FR-015**: `--has-user-tasks` MUST be mutually exclusive with `--key`.
- **FR-016**: `--has-user-tasks` MUST be mutually exclusive with stdin key input via `-`.
- **FR-017**: `--has-user-tasks` MUST be mutually exclusive with process-instance search filters.
- **FR-018**: `--has-user-tasks` MUST be mutually exclusive with `--total`.
- **FR-019**: `--has-user-tasks` MUST be mutually exclusive with `--limit`.
- **FR-020**: `--has-user-tasks` MUST work with `--json` and other output flags already valid for direct single process-instance lookup.
- **FR-021**: Tenant-aware behavior for has-user-tasks lookup MUST remain consistent with existing process-instance lookup flows, including the user-task resolution step.
- **FR-021a**: Repeating `--has-user-tasks` MUST resolve each user task key and render the resulting process instances through the existing keyed process-instance lookup behavior.
- **FR-022**: Help text and generated CLI documentation MUST include concrete has-user-tasks examples for human and JSON output.
- **FR-023**: README or user-facing documentation MUST describe has-user-tasks lookup, version support, and forbidden Tasklist/Operate fallback behavior.
- **FR-024**: Automated tests MUST cover successful user-task resolution on Camunda 8.9.
- **FR-025**: Automated tests MUST cover successful user-task resolution on Camunda 8.8.
- **FR-026**: Automated tests MUST cover the Camunda 8.7 unsupported error path.
- **FR-027**: Automated tests MUST cover missing user task handling.
- **FR-028**: Automated tests MUST cover user task responses without a usable process-instance key.
- **FR-029**: Automated tests MUST cover `--has-user-tasks` with `--json`.
- **FR-030**: Automated tests MUST cover `--has-user-tasks` conflicts with `--key`, stdin key input, search filters, `--total`, and `--limit`.
- **FR-031**: Automated tests MUST cover generated help text or user-facing command contract changes where relevant.

### Key Entities *(include if feature involves data)*

- **User Task Key**: A user-provided Camunda user task identifier used as a lookup selector for `get process-instance` / `get pi`.
- **Resolved User Task**: The native Camunda v2 user-task lookup result that contains the owning process-instance key.
- **Owning Process Instance Key**: The process-instance identifier extracted from the resolved user task and passed into the existing process-instance lookup path.
- **Task-Key Lookup Request**: A process-instance lookup invocation that starts from one or more `--has-user-tasks` values rather than `--key`.
- **Unsupported Version Error**: The explicit failure returned when `--has-user-tasks` is used against Camunda 8.7.
- **Selector Conflict**: Any invocation that combines `--has-user-tasks` with another lookup or search selector.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show `get pi --has-user-tasks=<task-key>` on Camunda 8.9 returns the owning process instance.
- **SC-002**: Automated tests show `get pi --has-user-tasks=<task-key>` on Camunda 8.8 returns the owning process instance.
- **SC-003**: Automated tests show `get pi --has-user-tasks=<task-key>` on Camunda 8.7 fails with an explicit unsupported-version message.
- **SC-004**: Automated tests show has-user-tasks lookup calls no Tasklist or Operate fallback path.
- **SC-005**: Automated tests show task-key human output matches direct keyed process-instance lookup for the resolved key.
- **SC-006**: Automated tests show task-key JSON output matches direct keyed process-instance lookup for the resolved key.
- **SC-007**: Automated tests show missing user tasks and task responses without usable process-instance keys fail clearly.
- **SC-008**: Automated tests show `--has-user-tasks` conflicts fail for `--key`, stdin key input, search filters, `--total`, and `--limit`.
- **SC-009**: Help output, README examples, and generated CLI docs document has-user-tasks lookup, JSON usage, version support, and unsupported fallbacks.
- **SC-010**: Repository validation passes with the closest relevant targeted tests and the broader project test command required by the constitution.

## Assumptions

- The affected command surface is `get process-instance` and its `get pi` alias.
- Camunda 8.8 and 8.9 generated clients expose native user-task search in a way that can return the owning process-instance key while filtering by tenant.
- Camunda 8.7 cannot support `--has-user-tasks` through the allowed native API path and must not be patched through Tasklist or Operate.
- Existing single process-instance lookup remains the source of truth for rendering, tenant handling, not-found behavior, and API error behavior after resolution.
- Existing command validation patterns are sufficient for mutual-exclusion failures as long as the resulting errors are clear and test-covered.
- Documentation generated from Cobra command metadata should be regenerated rather than edited by hand when command metadata changes.
