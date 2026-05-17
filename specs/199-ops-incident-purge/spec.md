# Feature Specification: Ops Purge Process Instances With Incidents

**Feature Branch**: `199-ops-incident-purge`
**Created**: 2026-05-16
**Status**: Draft
**GitHub Issue**: [#199 feat(ops): add purge command for process instances with filtered incidents](https://github.com/grafvonb/c8volt/issues/199)
**Input**: GitHub issue #199 plus mandatory implementation context `specs/ralph-implementation-rules.md`

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Register Incident Purge Command (Priority: P1)

As an operator, I want a discoverable `ops purge process-instances-with-incidents` command and `pi-with-incidents` alias with the correct incident-selection flags so I can identify and validate the workflow before any remote action occurs.

**Why this priority**: Command shape, aliases, local validation, and contract metadata are the smallest safe foundation for every later behavior.

**Independent Test**: Can be tested with command help, command contract metadata, alias resolution, and invalid flag checks without Camunda data.

**Acceptance Scenarios**:

1. **Given** `c8volt ops purge process-instances-with-incidents --help`, **When** help is rendered, **Then** the command exposes supported incident selection flags and purge/delete workflow flags.
2. **Given** `c8volt ops purge pi-with-incidents --help`, **When** help is rendered, **Then** the alias behaves exactly like the full command.
3. **Given** command metadata inspection, **When** contracts are evaluated, **Then** the command is state-changing and declares full automation support.
4. **Given** unsupported display-only incident flags, **When** the command is invoked, **Then** those flags are not accepted by the incident purge command.

---

### User Story 2 - Discover And Freeze Candidate Process Instances (Priority: P2)

As an operator, I want incident filters to discover candidate incidents and freeze unique candidate process-instance keys so dry-run and destructive runs operate on a stable target set.

**Why this priority**: Candidate discovery is the first remote behavior and must be deterministic before delete planning or mutation is introduced.

**Independent Test**: Can be tested by running dry-run discovery against fake incident responses and verifying candidate incidents, unique candidate process-instance keys, duplicates, no-target behavior, and `--limit` handling without delete requests.

**Acceptance Scenarios**:

1. **Given** matching active incidents, **When** the operator runs the command with `--dry-run`, **Then** the command searches incidents and extracts candidate process-instance keys without mutation.
2. **Given** multiple incidents for the same process instance, **When** discovery runs, **Then** duplicate process-instance keys are represented separately from unique candidate process-instance keys.
3. **Given** no incidents matching the selection criteria, **When** discovery runs, **Then** no delete request is submitted and the output reports zero candidate incidents and zero candidate process instances.
4. **Given** `--limit 5`, **When** more than five incidents match the filters, **Then** only the first five matching incidents are considered for candidate process-instance extraction.
5. **Given** `--key <incident-key>`, **When** discovery runs, **Then** `--key` means incident key, not process-instance key.

---

### User Story 3 - Build Incident Purge Delete Plan (Priority: P3)

As an operator, I want candidate process instances expanded through the existing process-instance delete plan so I can see roots, affected families, duplicate handling, and safety blockers before mutation.

**Why this priority**: Delete planning is the safety gate between discovery and destructive execution.

**Independent Test**: Can be tested by feeding frozen candidate keys into delete-plan fixtures and verifying root resolution, affected keys, duplicate handling, and non-final blockers with no delete submission.

**Acceptance Scenarios**:

1. **Given** candidate process-instance keys, **When** dry-run planning executes, **Then** the command runs the existing delete preflight/plan and reports candidate, root, and affected counts.
2. **Given** duplicate candidate process-instance keys, **When** planning executes, **Then** duplicates do not produce duplicate delete-plan roots.
3. **Given** non-final affected process instances and no `--force`, **When** planning executes for a destructive run, **Then** the command fails before mutation like `delete pi`.
4. **Given** expected planning notices, **When** human output is compact, **Then** notices are filtered semantically while JSON/report data remains complete.

---

### User Story 4 - Execute Confirmed Purge Through Delete Plan (Priority: P4)

As an operator, I want confirmed destructive purges to reuse the same deterministic delete planning and deletion source of truth as `delete pi` so incident-based cleanup preserves existing force, wait, and safety behavior.

**Why this priority**: The command is not production-ready until the destructive path preserves the behavior operators already trust in `delete pi`.

**Independent Test**: Can be tested by running destructive command fixtures with final and non-final affected process instances and verifying confirmation, mutation, `--force`, and `--no-wait` behavior matches `delete pi`.

**Acceptance Scenarios**:

1. **Given** non-final affected process instances and `--force`, **When** the operator confirms the purge, **Then** the command uses existing cancel-before-delete semantics before delete submission.
2. **Given** `--no-wait`, **When** delete requests are accepted, **Then** the command returns according to existing `delete pi` no-wait behavior.
3. **Given** worker and fail-fast flags, **When** the purge executes, **Then** the flags are passed to the delete path and preserve existing ordering, dedupe, and failure behavior.
4. **Given** a frozen candidate set, **When** destructive execution begins, **Then** no second incident discovery can expand the submitted delete scope.

---

### User Story 5 - Produce Compact Output, Complete Reports, And Automation-Safe JSON (Priority: P5)

As an operator or automation author, I want compact human output, complete machine-readable JSON, and optional report files so the purge can be reviewed, audited, and scripted safely.

**Why this priority**: The workflow must align with the established #186/#187 ops rhythm and remain reliable for agents, CI, and production runbooks.

**Independent Test**: Can be tested by running human, verbose, JSON, dry-run, confirmed, unconfirmed, and report-file cases and verifying output streams, report overwrite safety, and error classes.

**Acceptance Scenarios**:

1. **Given** normal human output, **When** the command previews or executes a purge, **Then** output follows the discovery/plan/action/outcome/report rhythm and does not print full key lists unless `--verbose` is supplied.
2. **Given** `--verbose`, **When** the command reports results, **Then** it includes incident keys, candidate process-instance keys, resolved root keys, affected process-instance keys, and skipped or blocked keys where applicable.
3. **Given** `--automation --json`, **When** the command runs in a supported non-interactive path, **Then** stdout remains deterministic machine-readable JSON and supported prompts are implicitly accepted through `shouldImplicitlyConfirm(cmd)` without requiring `--auto-confirm`.
4. **Given** an existing report file and overwrite is not explicitly allowed, **When** the command is dry-run, aborted, unconfirmed, or locally blocked, **Then** the existing report file is preserved and the failure uses local precondition error classification when discovered after planning/preflight.

---

### User Story 6 - Preserve Documentation And Regression Contracts (Priority: P6)

As a c8volt user, I want help text, generated docs, README examples, and regression tests to reflect the incident purge workflow without changing existing `get incident` or `delete pi` behavior.

**Why this priority**: Discoverability and docs complete the feature after the behavior and machine contracts are stable.

**Independent Test**: Can be tested by inspecting generated CLI docs, README examples, command contract tests, and regression tests for existing `get incident` and `delete pi` behavior.

**Acceptance Scenarios**:

1. **Given** generated CLI docs and README examples, **When** users read them, **Then** examples teach dry-run and report workflows before destructive execution examples.
2. **Given** existing `get incident` behavior, **When** regression tests run, **Then** incident search and output behavior remains unchanged.
3. **Given** existing `delete pi` behavior, **When** regression tests run, **Then** delete planning, force, wait, and output behavior remains unchanged.

### Edge Cases

- No matching incidents must short-circuit before delete submission while still producing valid human, JSON, and report output.
- Incidents missing a usable process-instance key must not produce invalid delete requests; such records must be represented in structured notices or errors.
- Duplicate incidents for the same process instance must be counted separately from unique candidate process instances.
- `--key` means incident key, not process-instance key.
- `--limit` applies to matching incidents before candidate process-instance dedupe.
- Existing report files must not be overwritten during dry-run, aborted, unconfirmed, or locally blocked runs.
- Expected planning notices must be filtered semantically for compact human output while remaining complete in JSON and reports.
- Static CLI shape errors must use invalid-input helpers and `exitcode.InvalidArgs`; runtime/local precondition failures must use `localPreconditionError` and `exitcode.Error`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add an `ops purge` command group when not already present and add the target command `c8volt ops purge process-instances-with-incidents`.
- **FR-002**: The target command MUST provide the alias `pi-with-incidents` and MUST NOT add the alias `incident-pis`.
- **FR-003**: The command MUST support incident selection flags `--key`, `--state`, `--error-type`, `--error-message`, `--bpmn-process-id`, `--pd-key`, `--pi-key`, `--root-key`, `--flow-node-id`, `--fni-key`, `--creation-time-after`, `--creation-time-before`, `--batch-size`, and `--limit`.
- **FR-004**: The command MUST NOT expose display-only incident flags such as `--pi-keys-only`, `--total`, `--error-message-limit`, or `--with-no-error-message`.
- **FR-005**: The command MUST treat matching incidents as discovery candidates, extract candidate process-instance keys, and freeze the candidate process-instance set before delete planning.
- **FR-006**: The delete phase MUST reuse the existing `delete pi` delete-plan source of truth for key dedupe, root resolution, descendant/family expansion, non-final affected process-instance checks, `--force`, `--no-wait`, worker behavior, fail-fast behavior, and no-worker-limit behavior.
- **FR-007**: The command MUST submit no delete request when no incidents match.
- **FR-008**: The command MUST preserve `delete pi` preflight semantics, including refusing non-final affected instances unless `--force` is supplied.
- **FR-009**: The command MUST support `--dry-run`, `--no-wait`, `--workers`, `--no-worker-limit`, `--fail-fast`, confirmation behavior, `--automation`, `--auto-confirm`, `--report-file`, and `--report-format` consistently with existing ops cleanup workflows.
- **FR-010**: The command MUST use `shouldImplicitlyConfirm(cmd)` for confirmation and prompt decisions so `--automation` implicitly accepts supported prompts for commands declaring full automation support.
- **FR-011**: The command MUST keep `--auto-confirm` as a human/script convenience flag, not as a required companion to `--automation`.
- **FR-012**: The command MUST route ops workflow logs through the configured c8volt logger and MUST NOT use `slog.Default()` inside CLI ops orchestration paths.
- **FR-013**: The command MUST check report-file overwrite safety before discovery, preflight, or planning when overwrite is not already allowed.
- **FR-014**: The command MUST NOT add an extra overwrite confirmation prompt; existing command confirmation and `--auto-confirm` are the overwrite boundary.
- **FR-015**: The command MUST preserve existing report files for dry-run, aborted, unconfirmed, or locally blocked runs.
- **FR-016**: The command MUST write complete JSON/report data including command metadata, dry-run, automation, auto-confirm, no-wait flags, incident selection filters, incident discovery result, candidate process-instance set, delete plan, deletion result, errors, and notices.
- **FR-017**: Human output MUST use candidate terminology, including `candidate incidents`, `candidate process instances`, `duplicate candidate process instances`, `resolved root keys`, and `affected process-instance keys`.
- **FR-018**: Human output MUST remain compact and suppress full key lists unless `--verbose` is supplied.
- **FR-019**: JSON/report output MUST remain complete even when human output suppresses expected notices or key lists.
- **FR-020**: Help examples MUST teach safe automation and preview-first workflows and MUST NOT use a bare destructive automation example such as `c8volt ops purge process-instances-with-incidents --automation --json`.
- **FR-021**: Existing `get incident` and `delete pi` behavior MUST remain unchanged.
- **FR-022**: Downstream Ralph launch instructions MUST include `--implementation-context specs/ralph-implementation-rules.md`.

### Key Entities

- **Incident Selection**: User-supplied incident lookup criteria, including incident key, state, error filters, process identifiers, flow-node identifiers, creation-time bounds, batch size, and limit.
- **Candidate Incident**: A matched incident returned by discovery before candidate process-instance extraction.
- **Candidate Process Instance**: A unique process-instance key extracted from candidate incidents and frozen before delete planning.
- **Delete Plan**: The existing `delete pi` plan result, including candidate count, resolved root keys, affected process-instance keys, non-final affected instances, expected notices, and mutation readiness.
- **Purge Report**: Human, JSON, or report-file representation containing command metadata, discovery, candidate set, delete plan, deletion result, notices, and errors.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Dry-run with matching incidents produces zero delete requests and reports candidate incident, candidate process-instance, root, and affected process-instance counts.
- **SC-002**: Duplicate incidents for one process instance produce one candidate process-instance entry and no duplicate delete submission.
- **SC-003**: Non-final affected process instances without `--force` fail before mutation with the same safety behavior as `delete pi`.
- **SC-004**: `--automation --json` succeeds without `--auto-confirm` for supported non-interactive paths and emits deterministic JSON on stdout.
- **SC-005**: Report overwrite protection preserves existing report files for dry-run, aborted, unconfirmed, and locally blocked runs.
- **SC-006**: Generated CLI documentation and README examples include the new command and safe preview-first usage.

## Assumptions

- The implementation will reuse the #186/#187 ops workflow helpers wherever they already cover report validation, report writing, semantic notice filtering, delete planning, and output rhythm.
- The command is a CLI-only feature; no new external API endpoint or generated Camunda client change is expected.
- Camunda v2 incident lookup behavior already exposed by `get incident` remains the source of incident discovery semantics.
- The current repository branch and feature directory identity are `199-ops-incident-purge`, matching GitHub issue #199.
- Every commit subject for this feature must use Conventional Commits format and end with `#199`.
