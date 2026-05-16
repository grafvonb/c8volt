# Feature Specification: Ops Purge All Process Definitions

**Feature Branch**: `208-purge-process-definitions`  
**Created**: 2026-05-16  
**Status**: Draft  
**Input**: GitHub issue #208 plus mandatory implementation context `specs/ralph-implementation-rules.md`

## Source Traceability

- **Issue Number**: 208
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/208
- **Issue Title**: feat(ops): add purge command for all process definitions
- **Command Surface**: `c8volt ops purge all-process-definitions`
- **Suggested Alias**: `c8volt ops purge all-pds`
- **Mandatory Implementation Context**: `specs/ralph-implementation-rules.md` MUST be read and applied during planning, task generation, and every Ralph implementation iteration. Ralph MUST NOT be launched unless the launch instructions include `--implementation-context specs/ralph-implementation-rules.md`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Register All Process Definitions Purge Command (Priority: P1)

As a Camunda operator, I want a discoverable `ops purge all-process-definitions` command and `all-pds` alias with the right selection and purge flags so I can preview the workflow safely before any remote action occurs.

**Why this priority**: Command shape, aliases, local validation, and contract metadata are the smallest safe foundation for every later behavior.

**Independent Test**: Can be tested with command help, command contract metadata, alias resolution, and unsupported flag checks without Camunda data.

**Acceptance Scenarios**:

1. **Given** `c8volt ops purge all-process-definitions --help`, **When** help is rendered, **Then** the command exposes supported process-definition selection flags and purge/delete workflow flags.
2. **Given** `c8volt ops purge all-pds --help`, **When** help is rendered, **Then** the alias behaves exactly like the full command.
3. **Given** unsupported display-only process-definition flags, **When** the command is invoked, **Then** those flags are not accepted by the purge command.
4. **Given** command metadata inspection, **When** contracts are evaluated, **Then** the command is state-changing and declares full automation support.

---

### User Story 2 - Discover And Freeze Candidate Process Definitions (Priority: P2)

As a Camunda operator, I want the purge command to discover process-definition versions with the same selection semantics as `get pd` and freeze the candidate keys before planning or confirmation.

**Why this priority**: Candidate discovery is the first remote behavior and must be deterministic before delete planning or mutation is introduced.

**Independent Test**: Can be tested by running dry-run discovery against fake process-definition responses and verifying all versions, filters, duplicate handling, and no-target behavior without delete requests.

**Acceptance Scenarios**:

1. **Given** no filters, **When** the operator runs `--dry-run`, **Then** every process-definition version visible to `get pd` is discovered as a candidate and no mutation occurs.
2. **Given** `--bpmn-process-id <id>`, **When** discovery runs, **Then** only versions for that BPMN process ID are considered.
3. **Given** `--pd-version` or `--pd-version-tag`, **When** discovery runs, **Then** only matching process-definition versions are considered.
4. **Given** `--latest`, **When** discovery runs, **Then** only latest matching definitions are considered and output makes clear that the scope is narrowed.
5. **Given** multiple discovery paths return the same process-definition key, **When** discovery freezes candidates, **Then** duplicate keys do not produce duplicate delete submissions.
6. **Given** no matching process definitions, **When** discovery completes, **Then** no delete request is submitted and the command reports zero candidates.

---

### User Story 3 - Build Delete Plan From Frozen Candidates (Priority: P3)

As a Camunda operator, I want discovered process-definition keys to flow through the same delete preflight source of truth as `delete pd` so active-instance impact and safety blockers are evaluated before mutation.

**Why this priority**: Delete planning is the safety gate between discovery and destructive execution.

**Independent Test**: Can be tested by feeding frozen candidate keys into delete-plan fixtures and verifying dedupe, impact analysis, active-instance blockers, force behavior, and dry-run output with no delete submission.

**Acceptance Scenarios**:

1. **Given** candidate process-definition keys, **When** dry-run planning executes, **Then** the command runs existing delete preflight and reports candidate count plus affected process-instance impact.
2. **Given** duplicate candidate keys, **When** planning executes, **Then** duplicates are represented separately from unique candidates and do not duplicate delete work.
3. **Given** unsafe active process-instance impact and no `--force`, **When** planning executes for a destructive run, **Then** the command fails before mutation like `delete pd`.
4. **Given** `--force`, **When** planning and execution are confirmed, **Then** cancel-before-delete and process-instance history cleanup semantics are delegated to the existing `delete pd` path.

---

### User Story 4 - Execute Confirmed Purge Through Delete PD (Priority: P4)

As a Camunda operator, I want confirmed destructive purges to reuse the deterministic `delete pd` workflow so force, wait, worker, fail-fast, and process-definition deletion behavior remain trusted and unchanged.

**Why this priority**: The command is not production-ready until the destructive path preserves the behavior operators already trust in `delete pd`.

**Independent Test**: Can be tested by running destructive command fixtures and verifying confirmation, mutation, `--force`, `--no-wait`, worker, no-worker-limit, and fail-fast behavior matches `delete pd`.

**Acceptance Scenarios**:

1. **Given** a confirmed purge, **When** deletion starts, **Then** the submitted process-definition deletes are exactly the frozen candidate keys that passed preflight.
2. **Given** `--no-wait`, **When** delete work is accepted, **Then** the command returns according to existing `delete pd` no-wait behavior.
3. **Given** worker and fail-fast flags, **When** the purge executes, **Then** the flags are passed to the delete path and preserve existing ordering, dedupe, and failure behavior.
4. **Given** a frozen candidate set, **When** destructive execution begins, **Then** no second discovery can expand the submitted delete scope.

---

### User Story 5 - Produce Compact Output, Reports, And Automation-Safe JSON (Priority: P5)

As an operator or automation author, I want compact human output, complete machine-readable JSON, and optional report files so the purge can be reviewed, audited, and scripted safely.

**Why this priority**: The workflow must align with the established ops rhythm from orphan cleanup, retention policy, and incident purge workflows.

**Independent Test**: Can be tested by running human, verbose, JSON, dry-run, confirmed, unconfirmed, and report-file cases and verifying output streams, report overwrite safety, and error classes.

**Acceptance Scenarios**:

1. **Given** normal human output, **When** the command previews or executes a purge, **Then** output follows discovery, delete plan, deletion, outcome, and report rhythm without printing full key lists unless `--verbose` is supplied.
2. **Given** `--verbose`, **When** the command reports results, **Then** it includes candidate process-definition keys, BPMN process IDs and versions where available, duplicate candidate keys, affected process-instance keys, and blocked keys where applicable.
3. **Given** `--automation --json`, **When** the command runs in a supported non-interactive path, **Then** stdout remains deterministic machine-readable JSON and supported prompts are implicitly accepted through `shouldImplicitlyConfirm(cmd)` without requiring `--auto-confirm`.
4. **Given** a report file request, **When** overwrite is not already allowed, **Then** overwrite safety is checked before discovery, preflight, or planning and existing files are preserved for dry-run, aborted, unconfirmed, or locally blocked runs.

---

### User Story 6 - Preserve Documentation And Regression Contracts (Priority: P6)

As a c8volt user, I want help text, generated docs, safe examples, and regression tests to reflect the all-process-definitions purge workflow without changing existing `get pd` or `delete pd` behavior.

**Why this priority**: Discoverability and docs complete the feature after behavior and machine contracts are stable.

**Independent Test**: Can be tested by inspecting generated CLI docs, README examples, command contract tests, and regression tests for existing `get pd` and `delete pd` behavior.

**Acceptance Scenarios**:

1. **Given** generated CLI docs and examples, **When** users read them, **Then** examples teach dry-run and report workflows before destructive execution examples.
2. **Given** existing `get pd` behavior, **When** regression tests run, **Then** process-definition selection and output behavior remains unchanged.
3. **Given** existing `delete pd` behavior, **When** regression tests run, **Then** delete planning, force, wait, and output behavior remains unchanged.

### Edge Cases

- `c8volt ops purge` is only a grouping command and MUST NOT perform cleanup directly.
- `--dry-run` MUST never mutate state and MUST report planned purge data.
- No matching process definitions MUST be a successful no-op with no delete request submitted.
- Candidate discovery MUST be frozen before delete preflight, confirmation, or mutation.
- Duplicate process-definition keys MUST be counted separately from unique candidate process definitions and MUST NOT duplicate delete submissions.
- `--latest` MUST be treated as explicit narrowing and MUST be visible in output/report context.
- Unsafe active process-instance impact MUST fail before mutation unless `--force` is supplied.
- Existing report files MUST be preserved during dry-run, aborted, unconfirmed, or locally blocked runs when overwrite is not already allowed.
- Static CLI shape errors MUST use invalid-input helpers and `exitcode.InvalidArgs`; runtime/local precondition failures after discovery or preflight MUST use `localPreconditionError` and `exitcode.Error`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add the purge target `c8volt ops purge all-process-definitions`.
- **FR-002**: The target command MUST provide the alias `all-pds` and MUST NOT add broad ambiguous aliases such as `purge-definitions` or `delete-all`.
- **FR-003**: The command MUST discover process definitions using the same effective selection behavior as `c8volt get pd`.
- **FR-004**: By default, the command MUST discover every process-definition version visible to `c8volt get pd`.
- **FR-005**: The command MUST support process-definition selection flags `--key`, `--bpmn-process-id`, `--pd-version`, `--pd-version-tag`, and `--latest`.
- **FR-006**: The command MUST NOT expose display/output-only flags such as `--xml` or `--stat`.
- **FR-007**: The command MUST freeze the candidate process-definition key set before delete preflight, confirmation, or mutation.
- **FR-008**: The delete phase MUST reuse the existing `delete pd` source of truth for key dedupe, impact analysis, active process-instance checks, `--force` cancel-before-delete behavior, process-instance history deletion, process-definition deletion, `--no-wait`, worker behavior, fail-fast behavior, and no-worker-limit behavior.
- **FR-009**: The command MUST submit no delete request when no process definitions match.
- **FR-010**: The command MUST preserve `delete pd` preflight semantics, including refusing unsafe active-instance impact unless `--force` is supplied.
- **FR-011**: The command MUST support `--dry-run`, `--no-wait`, `--workers`, `--no-worker-limit`, `--fail-fast`, `--force`, confirmation behavior, `--automation`, `--auto-confirm`, `--report-file`, and `--report-format` consistently with existing ops workflows.
- **FR-012**: The command MUST use `shouldImplicitlyConfirm(cmd)` for confirmation and prompt decisions so `--automation` implicitly accepts supported prompts for commands declaring full automation support.
- **FR-013**: `--auto-confirm` MUST remain a human/script convenience flag, not a required companion to `--automation`.
- **FR-014**: The command MUST check report-file overwrite safety before discovery, preflight, or planning when overwrite is not already allowed.
- **FR-015**: The command MUST NOT add an extra overwrite confirmation prompt.
- **FR-016**: The command MUST preserve existing report files for dry-run, aborted, unconfirmed, or locally blocked runs.
- **FR-017**: Human output MUST use candidate terminology, including `candidate process definitions`, `duplicate candidate process definitions`, `affected process instances`, and `submitted process-definition deletes`.
- **FR-018**: Normal human output MUST remain compact and suppress full key lists unless `--verbose` is supplied.
- **FR-019**: JSON output and report output MUST retain complete command metadata, flags, process-definition filters, discovery results, candidate set, delete preflight, deletion result, errors, notices, and final outcome.
- **FR-020**: Help examples MUST teach safe preview-first workflows and MUST NOT use a bare destructive automation example such as `c8volt ops purge all-process-definitions --automation --json`.
- **FR-021**: Existing `get pd` and `delete pd` behavior MUST remain unchanged.
- **FR-022**: Planning, task generation, and Ralph implementation instructions MUST include `specs/ralph-implementation-rules.md` as mandatory implementation context.

### Key Entities

- **Process Definition Selection**: User-supplied lookup criteria, including process-definition key, BPMN process ID, process-definition version, version tag, and latest-only narrowing.
- **Candidate Process Definition**: A process-definition version discovered by `get pd`-equivalent selection and frozen before delete planning.
- **Delete Plan**: The existing `delete pd` preflight result, including unique candidates, affected process-instance impact, active-instance blockers, force readiness, notices, and mutation readiness.
- **Purge Result**: The command result containing discovery, candidate set, delete plan, deletion submissions, confirmation status, notices, errors, and final outcome.
- **Purge Report**: Human, JSON, or report-file representation containing command metadata, filters, candidates, delete plan, deletion result, notices, errors, and final outcome.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Dry-run with no filters discovers all process-definition versions visible to `get pd`, runs delete preflight, reports planned purge data, and sends zero delete requests.
- **SC-002**: Filtered dry-runs by BPMN process ID, process-definition version, process-definition version tag, and latest-only scope include only matching process-definition versions.
- **SC-003**: Duplicate process-definition keys produce one unique delete target and no duplicate delete submission.
- **SC-004**: No-target discovery submits zero delete requests and exits with a successful no-op result.
- **SC-005**: Unsafe active process-instance impact without `--force` fails before mutation with the same safety behavior as `delete pd`.
- **SC-006**: `--automation --json` succeeds without `--auto-confirm` for supported non-interactive paths and emits deterministic JSON on stdout.
- **SC-007**: Markdown and JSON reports preserve complete discovery, delete-plan, deletion, notice, error, and final outcome data.
- **SC-008**: Existing `get pd` and `delete pd` regression tests continue to pass unchanged.

## Assumptions

- Operators understand that deleting all process-definition versions is destructive and should normally preview with `--dry-run`.
- The implementation will reuse shared ops workflow helpers wherever they already cover report validation, report writing, semantic notice filtering, delete planning, output rhythm, and confirmation behavior.
- The command is a CLI-only feature; no new external API endpoint or generated Camunda client change is expected unless required by the existing process-definition service boundary.
- Generated CLI documentation is refreshed from command metadata when the command surface changes.
- Ralph implementation must complete one work unit at a time and apply `specs/ralph-implementation-rules.md` in every iteration.
- Every commit subject for this feature must use Conventional Commits format and end with `#208`.

## Out Of Scope

- Interactive process-definition selection.
- Broad aliases such as `purge-definitions` or `delete-all`.
- Display-only process-definition output modes such as XML or stats on the purge command.
- Deleting process definitions that were not present in the frozen candidate set.
- Changing existing `get pd` or `delete pd` behavior.
- Hand-editing generated CLI documentation instead of regenerating it through the existing path.
