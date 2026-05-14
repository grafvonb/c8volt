# Feature Specification: Ops Purge Orphan Process Instances

**Feature Branch**: `186-ops-orphan-cleanup`  
**Created**: 2026-05-11  
**Status**: Draft  
**Input**: GitHub issue #186, updated to use `ops purge` for destructive cleanup workflows  

## Source Traceability

- **Issue Number**: 186
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/186
- **Original Issue Title**: feat(ops): add execute orphan-cleanup command with audited deletion report
- **Corrected Command Surface**: `c8volt ops purge orphan-process-instances`
- **Foundation Dependency**: issue #197 already created the `ops`, `ops execute`, `ops repair`, and shared workflow contract foundation. This feature adds the missing `ops purge` grouping command and the orphan process-instance purge workflow.
- **Mandatory Implementation Context**: `specs/ralph-implementation-rules.md` MUST be read and applied during planning, task generation, and every Ralph implementation iteration. Ralph MUST NOT be launched unless the launch instructions include `--implementation-context specs/ralph-implementation-rules.md`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Preview Orphan Cleanup Safely (Priority: P1)

As a Camunda operator, I want to run `c8volt ops purge orphan-process-instances --dry-run` so I can see exactly which orphan child process instances would be deleted before any mutation occurs.

**Why this priority**: Dry-run planning is the safest MVP because it proves discovery, filtering, output, and deletion-plan validation without changing remote state.

**Independent Test**: Can be tested by running the command against a fake Camunda server with orphan child instances and verifying the output reports the selected keys while no delete request is sent.

**Acceptance Scenarios**:

1. **Given** orphan child process instances exist, **When** the user runs `c8volt ops purge orphan-process-instances --dry-run`, **Then** c8volt discovers orphan child keys, validates the deletion plan, reports the planned purge, and does not delete anything.
2. **Given** no orphan child process instances exist, **When** the user runs `c8volt ops purge orphan-process-instances --dry-run`, **Then** c8volt reports that no purge targets were found and exits successfully without deletion.
3. **Given** compatible process-instance selection filters are supplied, **When** the user runs a dry run, **Then** discovery applies those filters together with orphan-child-only selection.

---

### User Story 2 - Run Confirmed Orphan Purge (Priority: P2)

As a Camunda operator, I want to run `c8volt ops purge orphan-process-instances --auto-confirm` so the discovered orphan child process instances are deleted through the same safe deletion behavior used by process-instance deletion.

**Why this priority**: This provides the operational value after the plan can be previewed and validated.

**Independent Test**: Can be tested by running against a fake Camunda server, verifying the command deletes exactly the keys discovered at command start, and confirming it does not continue discovering new orphan instances.

**Acceptance Scenarios**:

1. **Given** orphan child process instances exist, **When** the user runs `c8volt ops purge orphan-process-instances --auto-confirm`, **Then** c8volt discovers orphan child keys and deletes exactly those discovered keys.
2. **Given** the discovered orphan set changes after discovery, **When** deletion starts, **Then** only the originally discovered keys are deletion targets.
3. **Given** no orphan child process instances exist, **When** the command runs with `--auto-confirm`, **Then** c8volt exits successfully without attempting deletion.

---

### User Story 3 - Run Cleanup In Automation (Priority: P3)

As an automation owner, I want `c8volt ops purge orphan-process-instances --automation --json --auto-confirm` to run non-interactively with deterministic machine-readable output.

**Why this priority**: Automation safety is required for scheduled cleanup, but it depends on the planned and confirmed cleanup behavior from earlier stories.

**Independent Test**: Can be tested by executing the command in automation mode and verifying stdout contains only deterministic JSON while destructive execution requires `--auto-confirm`.

**Acceptance Scenarios**:

1. **Given** `--automation --json --auto-confirm` is supplied, **When** cleanup runs, **Then** stdout is deterministic JSON and interactive confirmation is not required.
2. **Given** `--automation` is supplied without `--auto-confirm`, **When** deletion would be performed, **Then** the command fails before mutation with a clear destructive-confirmation message.
3. **Given** automation mode is enabled, **When** progress or diagnostic output is produced, **Then** it does not pollute deterministic stdout.

---

### User Story 4 - Produce Audit Reports (Priority: P4)

As an operator or audit reviewer, I want `--report-file` and `--report-format` to create a stable audit report for dry-run, successful, and failed cleanup attempts.

**Why this priority**: Auditable reporting is valuable after the command behavior is reliable and automation-safe.

**Independent Test**: Can be tested by requesting Markdown and JSON reports and verifying both are generated from the same structured cleanup result model with timestamps, selected keys, statuses, errors, and final outcome.

**Acceptance Scenarios**:

1. **Given** `--report-file orphan-purge.md` is supplied, **When** cleanup finishes successfully or fails after discovery, **Then** c8volt writes a Markdown report containing discovery, selected keys, deletion statuses, timestamps, and final outcome.
2. **Given** `--report-file orphan-purge.json --report-format json` is supplied, **When** cleanup finishes, **Then** c8volt writes structured JSON suitable for audit automation.
3. **Given** `--report-format` is omitted, **When** the report file ends with `.json`, `.md`, or `.markdown`, **Then** c8volt infers JSON or Markdown format from the extension; otherwise it defaults to Markdown.
4. **Given** `--dry-run --report-file` is supplied, **When** the report is written, **Then** the report clearly marks `dryRun: true`.

### Edge Cases

- `c8volt ops purge` is only a grouping command and MUST NOT perform cleanup directly.
- `--dry-run` MUST never require confirmation and MUST never mutate state.
- `--automation` without `--auto-confirm` MUST fail before mutation when cleanup targets exist.
- No orphan child process instances MUST be a successful no-op with clear human and JSON output.
- Selection filters MUST narrow only the orphan-child discovery set and MUST NOT turn the command into a general process-instance delete command.
- Cleanup MUST operate on the orphan set discovered at command start and MUST NOT keep discovering newly orphaned process instances.
- A requested audit report MUST be written even when cleanup fails after discovery, as long as enough report context is available.
- Existing `get pi --orphan-children-only --keys-only` and `delete pi --key` behavior MUST remain unchanged.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add an `ops purge` command group under the existing `ops` foundation from issue #197.
- **FR-002**: The system MUST add `ops purge orphan-process-instances` as the orphan process-instance purge workflow.
- **FR-003**: `ops purge` itself MUST be a grouping command and MUST NOT perform cleanup directly.
- **FR-004**: `ops purge orphan-process-instances` MUST discover orphan child process instances with the same effective selection semantics as `get pi --orphan-children-only --keys-only`.
- **FR-005**: The command MUST support relevant existing process-instance selection filters that are compatible with orphan-child discovery, and those filters MUST narrow the discovered orphan set.
- **FR-006**: The command MUST build and validate a deletion plan from the discovered keys before any mutation.
- **FR-007**: When `--dry-run` is supplied, the command MUST report the planned cleanup and stop before deletion.
- **FR-008**: When deletion is allowed, the command MUST delete exactly the process-instance keys discovered at command start.
- **FR-009**: The command MUST reuse existing process-instance service discovery and deletion behavior rather than shelling out to c8volt subcommands or duplicating resource-specific logic.
- **FR-010**: Destructive execution MUST reuse the existing destructive confirmation behavior used by process-instance deletion.
- **FR-011**: The command MUST support `--auto-confirm`.
- **FR-012**: The command MUST support `--automation`.
- **FR-013**: `--automation` MUST be non-interactive.
- **FR-014**: `--automation` without `--auto-confirm` MUST fail before mutation when deletion would be performed.
- **FR-015**: Human output MUST show discovery, deletion plan, deletion execution when applicable, and final outcome.
- **FR-016**: JSON output MUST use the existing shared command result envelope where applicable.
- **FR-017**: JSON output MUST include structured per-step results and distinguish `planned`, `skipped`, `submitted`, `confirmed`, `confirmation_failed`, and `failed` steps where applicable.
- **FR-018**: The command MUST be marked state-changing in the command contract.
- **FR-019**: The command MUST be marked automation-compatible only when runtime behavior satisfies the existing automation contract.
- **FR-020**: `--automation --json` MUST keep stdout deterministic and route progress or diagnostics according to existing repository patterns.
- **FR-021**: The command MUST support `--report-file <path>`.
- **FR-022**: The command MUST support `--report-format markdown|json`.
- **FR-023**: If `--report-format` is omitted, report format MUST be inferred from `.json`, `.md`, or `.markdown` extensions when possible and default to Markdown otherwise.
- **FR-024**: Audit reports MUST be created from a stable structured report model before rendering Markdown or JSON.
- **FR-025**: Requested reports MUST include schema/version, command name, started timestamp, finished timestamp, duration, dry-run flag, c8volt version when available, configured Camunda version, safe profile/config identity, selection filters, orphan discovery status, discovered count, discovered keys, delete requested flag, auto-confirm flag, automation flag, per-key or per-batch delete status, errors, and final outcome.
- **FR-026**: Final audit outcome MUST be one of `planned`, `deleted`, `partially_failed`, or `failed`.
- **FR-027**: The implementation MUST preserve existing behavior for `get pi --orphan-children-only --keys-only` and `delete pi --key`.
- **FR-028**: Planning, tasks, and Ralph implementation instructions MUST include `specs/ralph-implementation-rules.md` as mandatory implementation context.

### Key Entities *(include if feature involves data)*

- **Orphan Purge Request**: User-supplied purge intent, including dry-run, auto-confirm, automation, output, selection filters, report path, and report format.
- **Orphan Discovery Result**: The discovered orphan child process-instance keys, selection filters used, count, discovery status, and discovery errors.
- **Deletion Plan**: The immutable set of process-instance keys selected at command start, validation status, and confirmation requirement.
- **Deletion Result**: Per-key or per-batch deletion status, confirmation status, errors, and final deletion outcome.
- **Audit Report**: Stable structured record of the command run, rendered as Markdown or JSON when requested.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Dry-run execution reports discovered orphan child keys and sends zero delete requests in automated command tests.
- **SC-002**: Confirmed execution deletes exactly the discovered keys and does not delete keys discovered only after the initial discovery step.
- **SC-003**: Automation mode without `--auto-confirm` fails before mutation whenever deletion targets exist.
- **SC-004**: `--automation --json --auto-confirm` produces deterministic JSON stdout that includes structured step results and no interactive prompt.
- **SC-005**: Markdown and JSON audit reports can be generated for successful cleanup and for failures that occur after discovery.
- **SC-006**: Existing process-instance orphan discovery and delete command tests continue to pass unchanged.

## Assumptions

- Operators already understand the meaning and risk of orphan child process instances.
- Existing process-instance service APIs either already expose the required primitives or will be extended at the service boundary rather than inside command code.
- Compatible selection filters are limited to filters already valid for orphan-child discovery.
- Generated CLI documentation is refreshed from command metadata when the command surface changes.
- Ralph implementation must complete one work unit at a time and apply `specs/ralph-implementation-rules.md` in every iteration.

## Out Of Scope

- Interactive orphan selection.
- General process-instance deletion beyond orphan child process instances.
- Searching unrelated process instances by arbitrary filters that are not compatible with orphan-child discovery.
- Chasing newly created orphan children after the initial discovery.
- Adding new deletion primitives unless required by the process-instance service boundary.
- Hand-editing generated CLI documentation instead of regenerating it through the existing path.
