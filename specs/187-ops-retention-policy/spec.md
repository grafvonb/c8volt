# Feature Specification: Ops Retention Policy Execution

**Feature Branch**: `187-ops-retention-policy`
**Created**: 2026-05-14
**Status**: Draft
**Input**: GitHub issue [#187](https://github.com/grafvonb/c8volt/issues/187) - `feat(ops): add execute retention-policy command with audited process-instance deletion report`

## Source Issue

- **Issue Number**: 187
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/187
- **Issue Title**: feat(ops): add execute retention-policy command with audited process-instance deletion report

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Register Retention Policy Command (Priority: P1)

As a Camunda operator, I want `c8volt ops execute retention-policy` to exist under the ops execute group with clear validation so retention cleanup has a discoverable, safe entry point.

**Why this priority**: Command registration, help, and local validation are the smallest independent slice and create the surface all later workflow behavior depends on.

**Independent Test**: Can be tested by inspecting command help and running validation-only command invocations without requiring Camunda data.

**Acceptance Scenarios**:

1. **Given** the c8volt CLI is installed, **When** the user runs `c8volt ops execute --help`, **Then** `retention-policy` is listed as an execute subcommand and `ops execute` itself does not perform cleanup.
2. **Given** the user omits `--retention-days`, **When** `c8volt ops execute retention-policy` runs, **Then** the command fails locally with an invalid-input style error before remote calls.
3. **Given** the user supplies a negative or non-integer retention value, **When** the command runs, **Then** validation fails locally with the existing invalid-argument exit behavior.

---

### User Story 2 - Discover Retention Seeds (Priority: P2)

As an operator, I want retention cleanup to discover finished process instances whose end date is older than the requested retention age so the cleanup seed set is explicit and repeatable.

**Why this priority**: Discovery establishes the immutable seed set and maps the new command to existing process-instance age selection semantics.

**Independent Test**: Can be tested by running the command in dry-run mode against fixtures or service fakes and verifying the discovered seed keys and boundary information.

**Acceptance Scenarios**:

1. **Given** process instances with end dates older and newer than the requested age, **When** the user runs `retention-policy --retention-days 90 --dry-run`, **Then** only instances matching the existing `--end-date-older-days 90` semantics are included as retention seeds.
2. **Given** process instances without an end date, **When** retention discovery runs, **Then** those instances are excluded by the existing end-date filter behavior.
3. **Given** discovery completes, **When** output or report data is produced, **Then** it includes the requested retention day count and the derived end-date boundary when available.

---

### User Story 3 - Apply Compatible Selection Filters (Priority: P3)

As an operator, I want to narrow retention discovery by existing process-instance filters so I can clean up a bounded process population without changing the retention age requirement.

**Why this priority**: Filter support is independently testable and prevents retention cleanup from being all-or-nothing.

**Independent Test**: Can be tested by applying each supported filter with dry-run discovery and verifying that all filters are combined with the retention age filter.

**Acceptance Scenarios**:

1. **Given** process instances across different BPMN IDs, process-definition keys, versions, states, parents, and incident states, **When** compatible filters are supplied, **Then** discovery applies them together with `--retention-days`.
2. **Given** the user supplies `--limit` or `--batch-size`, **When** discovery runs, **Then** the command follows existing process-instance search limit and paging behavior.
3. **Given** the user attempts to supply an explicit process-instance key selector, **When** the command validates flags, **Then** the command rejects that selector because retention policy discovers eligible keys rather than accepting explicit keys.

---

### User Story 4 - Build And Validate Delete Plan (Priority: P4)

As an operator, I want retention seeds to be expanded through the existing delete planning behavior so root resolution, descendant traversal, duplicates, and unsafe states are handled exactly like `delete pi`.

**Why this priority**: Delete planning is the safety boundary before any mutation and must reuse existing c8volt behavior.

**Independent Test**: Can be tested by feeding discovered seed keys into planning fakes and verifying root keys, affected family keys, duplicate handling, missing ancestor notices, and non-final state blocking.

**Acceptance Scenarios**:

1. **Given** discovered seed keys include child process instances, **When** planning runs, **Then** c8volt resolves roots, traverses descendants, and reports affected process-instance scope using the existing delete plan behavior.
2. **Given** duplicate roots are reached from multiple retention seed keys, **When** planning completes, **Then** duplicates are reported and each resolved root is submitted at most once.
3. **Given** affected process instances include non-final states, **When** deletion is not explicitly allowed by existing delete controls, **Then** the command refuses mutation before deletion and reports blocked keys and states.

---

### User Story 5 - Support Dry Run Output (Priority: P5)

As an operator, I want dry-run retention cleanup to show the planned discovery and delete scope without changing remote state so I can review impact before confirming.

**Why this priority**: Dry run is the primary safe operator workflow and proves discovery plus planning before destructive execution.

**Independent Test**: Can be tested by running dry-run in human and JSON output modes and asserting no delete or cancel calls are submitted.

**Acceptance Scenarios**:

1. **Given** matching retention seeds exist, **When** the user runs with `--dry-run`, **Then** c8volt reports retention days, boundary, filters, seed count, root count, affected count, duplicate handling, non-final count, missing ancestor notices, confirmation requirement, and final outcome `planned`.
2. **Given** no retention-eligible instances exist, **When** dry-run runs, **Then** c8volt reports no cleanup targets and exits successfully without deletion.
3. **Given** `--dry-run --json`, **When** output is rendered, **Then** stdout contains deterministic structured data for discovery, planning, skipped deletion, and outcome.

---

### User Story 6 - Execute Confirmed Deletion (Priority: P6)

As an operator, I want confirmed retention cleanup to reuse existing c8volt deletion controls so destructive execution follows established confirmation, cancellation, waiting, and concurrency behavior.

**Why this priority**: Mutation must come after command validation, discovery, planning, and dry-run safety are available.

**Independent Test**: Can be tested by confirming deletion through service fakes and verifying existing delete execution controls and per-key or per-batch statuses.

**Acceptance Scenarios**:

1. **Given** deletion is allowed and the user confirms, **When** cleanup executes, **Then** c8volt submits deletion through the existing process-instance deletion service with existing worker and wait behavior.
2. **Given** `--auto-confirm` is supplied, **When** deletion is allowed, **Then** the command proceeds without an interactive prompt.
3. **Given** `--automation --json` is supplied for the supported state-changing command, **When** deletion is allowed, **Then** automation mode implicitly accepts supported prompts and stdout remains deterministic JSON without requiring `--auto-confirm`.
4. **Given** deletion is blocked by local preconditions or unsafe workflow state after planning, **When** execution reaches that point, **Then** c8volt fails with the existing local-precondition error class and does not mutate state.

---

### User Story 7 - Write Audit Reports (Priority: P7)

As an operator or automation owner, I want optional Markdown or JSON audit reports so retention cleanup can be reviewed after planned, successful, partial, or failed runs.

**Why this priority**: Reports are a separate operator artifact and can be delivered after core execution paths while preserving the structured workflow model.

**Independent Test**: Can be tested by requesting Markdown and JSON report files for dry-run, success, no-target, and post-discovery failure paths.

**Acceptance Scenarios**:

1. **Given** `--report-file retention-report.md`, **When** cleanup finishes or fails after discovery, **Then** c8volt writes a readable Markdown report containing discovery, planning, deletion statuses, timestamps, and final outcome.
2. **Given** `--report-file retention-report.json --report-format json`, **When** cleanup finishes, **Then** c8volt writes a structured JSON report suitable for audit automation.
3. **Given** an existing report file and overwrite is not confirmed or otherwise non-interactively accepted, **When** the command starts, **Then** c8volt fails fast before preflight or discovery and preserves the existing report.
4. **Given** a report is written in human output mode, **When** final output is rendered, **Then** c8volt prints a compact `report: written <path>` line.

---

### User Story 8 - Preserve Existing Contracts (Priority: P8)

As a c8volt maintainer, I want the retention policy workflow to compose existing services and documentation generation paths so current `get pi`, `delete pi`, and ops workflows do not regress.

**Why this priority**: Contract preservation is cross-cutting and should finish the feature after the new workflow behavior is in place.

**Independent Test**: Can be tested through regression tests for existing get/delete commands, generated command docs, and command contract tests.

**Acceptance Scenarios**:

1. **Given** existing `get pi --end-date-older-days` behavior, **When** this feature is added, **Then** the existing command output and filtering behavior remain unchanged.
2. **Given** existing `delete pi --keys` behavior, **When** this feature is added, **Then** delete hierarchy planning, cancellation controls, waiting, and worker behavior remain unchanged.
3. **Given** CLI documentation is generated from source metadata, **When** the new command is documented, **Then** generated docs are refreshed through the repository generation path rather than hand-edited.

### Edge Cases

- `ops execute` is only a grouping command and never performs retention cleanup directly.
- `--retention-days 0` is valid and uses existing non-negative relative end-date semantics.
- Explicit process-instance key selection is rejected for retention policy; the command owns discovery.
- Selection filters narrow retention seeds but never replace the retention age requirement.
- The discovered retention seed set is frozen for one command execution; the workflow does not chase newly eligible instances.
- Process instances without `endDate` are excluded by existing end-date filtering behavior.
- No matching retention seeds is a successful no-op outcome.
- Duplicate roots reached through multiple seeds are reported and deleted at most once.
- Missing ancestors and partial traversal warnings are preserved in JSON/report data even when human output is compact.
- Non-final affected process instances block mutation unless existing delete controls explicitly allow cancellation.
- `--dry-run` never deletes or cancels process instances and never requires confirmation.
- `--automation --json` must keep stdout deterministic; logs and progress belong on stderr or are suppressed according to existing patterns.
- Existing report files are preserved for dry-run, aborted, unconfirmed, or locally blocked runs.
- Existing report-file overwrite safety is checked before preflight, planning, or discovery when overwrite is not already allowed.
- Markdown reports avoid unrelated process variables and sensitive values.
- JSON reports follow existing machine-output conventions and avoid exposing sensitive values beyond established c8volt behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST add `c8volt ops execute retention-policy` under the `ops execute` command group.
- **FR-002**: `c8volt ops execute` MUST remain a grouping command and MUST NOT perform cleanup directly.
- **FR-003**: `retention-policy` MUST require `--retention-days <days>`.
- **FR-004**: `--retention-days` MUST accept non-negative integers and reject missing, negative, and non-integer values with existing invalid-input behavior.
- **FR-005**: Retention discovery MUST use the same effective semantics as `c8volt get pi --end-date-older-days <days> --keys-only`.
- **FR-006**: Retention discovery MUST exclude process instances without `endDate` according to existing end-date filter behavior.
- **FR-007**: Discovery MUST freeze the eligible retention seed key set for a single command execution and MUST NOT continue discovering newly eligible instances forever.
- **FR-008**: Discovery MUST support compatible existing process-instance filters: `--bpmn-process-id`, `--pd-key`, `--pd-version`, `--pd-version-tag`, `--state`, `--parent-key`, `--roots-only`, `--children-only`, `--incidents-only`, `--no-incidents-only`, `--limit`, and `--batch-size`.
- **FR-009**: Selection filters MUST narrow the retention seed set while retaining the required retention age filter.
- **FR-010**: The command MUST reject explicit `--key` process-instance selection for retention policy.
- **FR-011**: The ops command facade MUST orchestrate discovery, delete planning, deletion execution, and report aggregation without owning resource-specific process-instance API logic.
- **FR-012**: Missing primitive capabilities needed by the workflow MUST be added to existing process-instance services or existing process facades instead of ad hoc ops logic.
- **FR-013**: Delete planning MUST reuse existing c8volt process-instance delete planning behavior.
- **FR-014**: Delete planning output MUST distinguish retention seed keys, resolved root keys, affected process-instance family keys, and duplicate keys removed by planning.
- **FR-015**: Delete planning MUST report final-state selected instances, non-final affected instances, missing ancestors, and traversal warnings.
- **FR-016**: If affected process instances are not in a final state, mutation MUST be refused unless existing delete controls explicitly allow cancellation.
- **FR-017**: `--dry-run` MUST perform discovery, delete planning, and validation without deleting or cancelling process instances.
- **FR-018**: Dry-run human output MUST show the planned retention days, derived boundary when available, selected filters, counts for seeds/roots/affected instances, duplicate handling, non-final count, missing ancestor notices, confirmation requirement, and report path/format when requested.
- **FR-019**: Dry-run JSON output MUST include structured discovery, planning, skipped deletion, and final outcome data.
- **FR-020**: Human output MUST keep the compact #186 ops rhythm: retention discovery, delete plan, deletion, outcome, report.
- **FR-021**: Human output MUST show detailed key lists only in verbose output unless keys are the primary output of the selected mode.
- **FR-022**: JSON output MUST distinguish `planned`, `skipped`, `submitted`, `confirmed`, `confirmation_failed`, and `failed` steps where applicable.
- **FR-023**: The command MUST be marked state-changing.
- **FR-024**: The command MUST be marked automation-compatible only if runtime behavior satisfies the existing automation contract.
- **FR-025**: `--automation` MUST be the canonical non-interactive mode for agents and scripts.
- **FR-026**: For supported state-changing ops commands, `--automation` MUST implicitly accept supported prompts through the existing prompt decision helper and MUST NOT require `--auto-confirm` in addition.
- **FR-027**: `--auto-confirm` MUST remain supported as a human or script convenience flag.
- **FR-028**: `--dry-run` MUST never require confirmation and MUST never mutate state.
- **FR-029**: `--automation --json` MUST keep stdout deterministic machine-readable JSON; progress or logs MUST be sent to stderr or suppressed according to existing patterns.
- **FR-030**: Deletion execution MUST reuse existing process-instance deletion service behavior, including concurrency, wait, no-wait, state-check, fail-fast, and cancellation controls.
- **FR-031**: The command MUST support applicable execution controls: `--workers`, `--no-worker-limit`, `--fail-fast`, `--no-wait`, `--no-state-check`, and `--force` only when it preserves existing `delete pi` semantics exactly.
- **FR-032**: If `--force` is supported, output and reports MUST clearly state that active or non-final affected instances may be cancelled before deletion according to existing delete behavior.
- **FR-033**: The workflow MUST NOT use Camunda native retention policies.
- **FR-034**: The workflow MUST NOT use Camunda batch deletion APIs.
- **FR-035**: The workflow MUST NOT use shell command composition.
- **FR-036**: The workflow MUST NOT introduce a parallel hierarchy traversal or deletion implementation.
- **FR-037**: `--report-file <path>` MUST be supported.
- **FR-038**: `--report-format markdown|json` MUST be supported.
- **FR-039**: If report format is omitted, the command MUST infer `.json` as JSON, `.md` or `.markdown` as Markdown, and otherwise default to Markdown.
- **FR-040**: Report-file validation, format inference, overwrite safety, and file writing MUST reuse shared ops report helpers from #186.
- **FR-041**: Report-file overwrite safety MUST run before preflight, planning, or discovery when overwrite is not already allowed.
- **FR-042**: Existing report files MUST be preserved for dry-run, aborted, unconfirmed, or locally blocked runs.
- **FR-043**: The command MUST NOT add an extra overwrite confirmation prompt; existing command confirmation or non-interactive confirmation is the overwrite boundary.
- **FR-044**: Reports MUST be created from a stable structured model before rendering to Markdown or JSON.
- **FR-045**: If `--report-file` is supplied, the command MUST write the report even when cleanup fails after discovery.
- **FR-046**: Reports MUST include schema/version, command name, timestamps, duration, dry-run flag, c8volt version when available, configured Camunda version, safe profile/config identity, tenant when available, retention days, derived boundary when available, filters, discovery status, seed/root/affected counts and keys, duplicate handling, final and non-final state details, missing ancestors, traversal warnings, confirmation flags, execution controls, per-key or per-batch delete status, errors, and final outcome.
- **FR-047**: Final outcome MUST be one of `planned`, `deleted`, `partially_failed`, or `failed`.
- **FR-048**: After a report is written in human output, the command MUST print a compact `report: written <path>` line.
- **FR-049**: Ops workflow logs MUST use the configured c8volt logger and MUST NOT use `slog.Default()` inside CLI ops orchestration paths.
- **FR-050**: Static CLI shape errors MUST use existing invalid-input helpers and exit with `exitcode.InvalidArgs`.
- **FR-051**: Runtime or local precondition failures discovered after planning or preflight MUST use `localPreconditionError` and exit with `exitcode.Error`.
- **FR-052**: User-facing help, generated CLI documentation, and examples MUST be updated through the repository's existing generation path.
- **FR-053**: Existing `get pi --end-date-older-days`, `delete pi --keys`, and delete hierarchy planning behavior MUST remain unchanged.

### Key Entities *(include if feature involves data)*

- **Retention Policy Request**: The validated command intent, including retention days, selection filters, dry-run flag, confirmation mode, automation mode, execution controls, output mode, and report settings.
- **Retention Discovery Result**: The frozen set of process-instance seed keys matching the retention age and selection filters, plus retention boundary, discovery status, and notices.
- **Delete Plan Summary**: The existing process-instance delete plan projected for retention cleanup, including seed keys, resolved roots, affected family keys, duplicates, final-state instances, non-final affected instances, missing ancestors, and traversal warnings.
- **Retention Execution Result**: The deletion step result, including submitted keys or batches, confirmation status, wait status, per-key or per-batch outcomes, errors, and final outcome.
- **Retention Audit Report**: A structured report model rendered to Markdown or JSON containing the full discovery, planning, deletion, and outcome record for audit review.
- **Ops Workflow Notice**: A semantic notice emitted by discovery, planning, or deletion that can be included fully in JSON/report data while being filtered for compact human output.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can run `c8volt ops execute retention-policy --retention-days 90 --dry-run` and see discovery and delete-plan counts without any mutation.
- **SC-002**: Retention discovery returns the same eligible seed set as existing `get pi --end-date-older-days <days> --keys-only` semantics for equivalent filters.
- **SC-003**: Compatible selection filters combine with the retention age filter and cannot replace it.
- **SC-004**: Child seed keys, duplicate roots, missing ancestors, and non-final affected instances are reported through the existing delete planning behavior.
- **SC-005**: Confirmed deletion submits each resolved root at most once and reuses existing process-instance deletion controls.
- **SC-006**: `--automation --json` works for this supported state-changing ops command without requiring `--auto-confirm` and produces deterministic stdout.
- **SC-007**: Existing report files are preserved for dry-run, aborted, unconfirmed, or locally blocked runs.
- **SC-008**: Markdown and JSON reports include discovery, planning, execution status, timestamps, errors, and final outcome for successful and post-discovery failure paths.
- **SC-009**: Human output stays compact and follows the #186 ops rhythm while JSON/report data remains complete.
- **SC-010**: Regression tests show existing `get pi --end-date-older-days`, `delete pi --keys`, and delete hierarchy planning behavior remain unchanged.

## Assumptions

- Operators already have c8volt configuration and Camunda permissions needed to search and delete process instances for the selected tenant context.
- Existing process-instance date parsing, relative day, tenant, paging, and limit conventions are reused.
- Existing delete planning behavior remains the source of truth for root resolution, descendant traversal, duplicate removal, missing ancestor handling, and non-final state blocking.
- Existing command confirmation, automation, JSON envelope, logger, and error classification helpers are available to ops workflows.
- Shared ops report helpers introduced for #186 are available or will be completed before this feature depends on them.
- Documentation for CLI commands is generated from source metadata and should be regenerated after command changes.
