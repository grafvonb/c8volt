# Feature Specification: Ops Execute Smoke Test

**Feature Branch**: `188-ops-smoke-test`
**Created**: 2026-05-17
**Status**: Draft
**GitHub Issue**: [#188 feat(ops): add execute smoke-test command with audited cluster workflow report](https://github.com/grafvonb/c8volt/issues/188)
**Input**: GitHub issue #188 plus mandatory implementation context `specs/ralph-implementation-rules.md`
**Mandatory Implementation Context**: Planning, task generation, and every Ralph implementation iteration MUST read and apply `specs/ralph-implementation-rules.md`. Ralph MUST NOT be launched unless implementation instructions include `--implementation-context specs/ralph-implementation-rules.md`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Register Smoke Test Command Surface (Priority: P1)

As a c8volt operator, I can discover `c8volt ops execute smoke-test` with the expected flags and command metadata before any cluster workflow is executed.

**Why this priority**: Command registration, grouping behavior, validation, and metadata are the safe foundation for every later workflow slice.

**Independent Test**: Run help, command contract, alias/flag, invalid flag, and grouping-command tests without a Camunda runtime.

**Acceptance Scenarios**:

1. **Given** the ops command family exists, **When** the user runs `c8volt ops execute --help`, **Then** it behaves as a grouping command and does not run a smoke test directly.
2. **Given** `c8volt ops execute smoke-test --help`, **When** help is rendered, **Then** it exposes count, worker, cleanup, automation, dry-run, and report flags.
3. **Given** command metadata inspection, **When** contracts are evaluated, **Then** `smoke-test` is marked state-changing and automation-compatible only when runtime behavior satisfies the automation contract.
4. **Given** invalid count or report format flags, **When** command validation runs, **Then** the command fails before remote clients or mutation are created.

---

### User Story 2 - Dry-Run Smoke Test Planning (Priority: P2)

As an operator, I can run `c8volt ops execute smoke-test --dry-run` to validate local configuration, connectivity, fixture availability, concurrency settings, cleanup intent, and report options without mutation.

**Why this priority**: A mutation-free preview gives operators a safe way to verify profiles and gives implementation a stable planning model before deployment or cleanup.

**Independent Test**: Run dry-run fixtures and subprocess tests verifying planned steps, read-only behavior, JSON output, and optional dry-run report generation.

**Acceptance Scenarios**:

1. **Given** a valid profile, **When** the command runs with `--dry-run`, **Then** it validates local configuration, may perform read-only connectivity validation, verifies the version-matched embedded fixture, reports planned steps, and performs no mutation.
2. **Given** `--dry-run --report-file smoke-test.md`, **When** dry-run succeeds or fails after planning starts, **Then** the report is written and clearly marks `dryRun: true`.
3. **Given** an unsupported or missing embedded fixture for the configured Camunda version, **When** dry-run runs, **Then** the command fails before mutation with a clear message.
4. **Given** `--dry-run --json`, **When** output is rendered, **Then** stdout uses the existing deterministic command result envelope.

---

### User Story 3 - Select And Deploy Version-Matched Fixture (Priority: P3)

As an operator, I want the smoke test to select the embedded multiple-subprocess BPMN fixture for my configured Camunda version and deploy it through existing resource behavior.

**Why this priority**: Deployment is the first mutation and must be version-aware, auditable, and implemented through existing resource services rather than shell composition.

**Independent Test**: Use fake service/client fixtures to verify fixture selection, missing fixture failure, deployment call shape, tenant propagation, and report fields.

**Acceptance Scenarios**:

1. **Given** Camunda 8.7, **When** fixture selection runs, **Then** it selects `C87_MultipleSubProcessesParentProcess`.
2. **Given** Camunda 8.8, **When** fixture selection runs, **Then** it selects `C88_MultipleSubProcessesParentProcess`.
3. **Given** Camunda 8.9, **When** fixture selection runs, **Then** it selects `C89_MultipleSubProcessesParentProcess`.
4. **Given** deployment succeeds, **When** results are reported, **Then** the report includes fixture file, BPMN process ID, deployed process-definition key when available, deployed version when available, and tenant id when available.

---

### User Story 4 - Start And Walk Created Instances (Priority: P4)

As an operator, I want the smoke test to start one or more process instances from the deployed definition and walk each created process-instance family through existing traversal behavior.

**Why this priority**: The core cluster proof is that deployment, run, and traversal work end to end for the selected fixture.

**Independent Test**: Run count, shorthand, worker, fail-fast, and traversal tests against fake process services while verifying created keys and per-instance statuses.

**Acceptance Scenarios**:

1. **Given** no count flag, **When** the smoke test runs, **Then** it starts exactly one process instance.
2. **Given** `--count 5`, **When** the smoke test runs, **Then** it starts five process instances and walks each created instance.
3. **Given** `-n 5`, **When** the smoke test runs, **Then** it behaves the same as `--count 5`.
4. **Given** the deployment response includes a process-definition key, **When** process instances are started, **Then** the workflow prefers that deployed key over latest BPMN ID lookup where supported.
5. **Given** worker, no-worker-limit, or fail-fast flags, **When** count is greater than one, **Then** existing process-instance creation concurrency behavior is reused.

---

### User Story 5 - Cleanup Created Resources Safely (Priority: P5)

As an operator, I want the default smoke test to clean up the process instances and process definition it created without deleting unrelated process instances or unsafe process definitions.

**Why this priority**: Cleanup makes the smoke test production-friendly, but it must preserve existing delete planning and process-definition safety behavior.

**Independent Test**: Use fake process-instance and process-definition services to verify cleanup planning, confirmation, no-wait behavior, unrelated instance blockers, and retained resources.

**Acceptance Scenarios**:

1. **Given** cleanup is enabled, **When** created process instances exist, **Then** the command deletes them through existing `delete pi` planning and deletion behavior.
2. **Given** no unrelated process instances exist for the deployed definition or BPMN process ID, **When** process-definition cleanup runs, **Then** the deployed smoke-test definition is deleted through existing process-definition deletion behavior.
3. **Given** unrelated process instances exist for the deployed definition or BPMN process ID, **When** process-definition cleanup runs, **Then** process-definition deletion is skipped and reported as blocked or not applicable.
4. **Given** `--no-wait`, **When** cleanup requests are submitted, **Then** the command preserves existing no-wait behavior for lower-level cleanup paths.

---

### User Story 6 - Support No-Cleanup Retention Mode (Priority: P6)

As an operator, I can run `c8volt ops execute smoke-test --no-cleanup` to leave created process instances and the deployed process definition in place for inspection.

**Why this priority**: Operators need an inspection mode that intentionally stops before destructive cleanup and clearly identifies retained resources.

**Independent Test**: Run no-cleanup fixtures verifying no delete calls, retained key reporting, JSON/report completeness, and automation behavior without cleanup confirmation.

**Acceptance Scenarios**:

1. **Given** `--no-cleanup`, **When** deployment, run, and traversal finish, **Then** no process-instance or process-definition delete request is submitted.
2. **Given** `--no-cleanup`, **When** human output and reports are rendered, **Then** they clearly list created process-instance keys and deployed process-definition key or BPMN process ID for later inspection or cleanup.
3. **Given** `--automation --no-cleanup`, **When** the smoke test runs, **Then** the command may proceed without destructive cleanup confirmation.

---

### User Story 7 - Produce Audit Reports And Stable Output (Priority: P7)

As an operator or automation author, I want compact human output, deterministic JSON, and optional Markdown or JSON audit reports describing every smoke-test step and final outcome.

**Why this priority**: The workflow exists to prove operational readiness, so the audit trail is part of the product, not decoration.

**Independent Test**: Run human, verbose, JSON, Markdown report, JSON report, failure-after-planning, and report-format inference tests.

**Acceptance Scenarios**:

1. **Given** normal human output, **When** the smoke test runs, **Then** output shows connectivity validation, deployment, process-instance creation, traversal, cleanup, report writing, and final outcome.
2. **Given** `--report-file smoke-test.md`, **When** the command succeeds or fails after planning starts, **Then** it writes a readable Markdown report with timestamps, duration, created keys, cleanup status, errors, and final outcome.
3. **Given** `--report-file smoke-test.json --report-format json`, **When** the command finishes, **Then** it writes a structured JSON report using a stable model.
4. **Given** `--report-format` is omitted, **When** the report path ends in `.json`, `.md`, or `.markdown`, **Then** the format is inferred accordingly; otherwise Markdown is used.
5. **Given** `--automation --json`, **When** the command runs, **Then** stdout remains deterministic machine-readable JSON and progress or logs stay off stdout.

---

### User Story 8 - Preserve Documentation And Existing Behavior (Priority: P8)

As a c8volt maintainer, I want generated docs, examples, and regression tests to cover the smoke-test workflow while preserving existing `config test-connection`, `embed deploy`, `run pi`, `walk pi`, `delete pi`, and `delete pd` behavior.

**Why this priority**: The feature is user-facing and composes existing capabilities; docs and regression checks complete the work after the behavior is stable.

**Independent Test**: Regenerate CLI docs and run targeted regression tests for existing commands and the new smoke-test examples.

**Acceptance Scenarios**:

1. **Given** generated CLI docs exist, **When** command metadata changes, **Then** docs are regenerated through `make docs-content`.
2. **Given** existing lower-level commands, **When** regression tests run, **Then** their behavior remains unchanged.
3. **Given** documentation examples, **When** users read them, **Then** examples include dry-run, count, no-cleanup, automation JSON, and report-file flows.

### Edge Cases

- `ops execute` must remain a grouping command and must not run smoke-test behavior directly.
- Matching embedded fixture absence must fail before mutation.
- `--count` must reject zero, negative, and non-integer values before creating clients or mutating remote state.
- Dry-run must not deploy, start instances, walk newly created instances, delete process instances, or delete process definitions.
- Dry-run must not create a normal audit report unless `--report-file` is supplied and the report clearly marks dry-run mode.
- `--no-cleanup` must never delete created process instances or process definitions.
- Cleanup must not delete process instances or process definitions that were not created by this smoke-test run, except where existing `delete pi` behavior expands a created process-instance key to its root or family scope.
- Process-definition cleanup must be skipped and reported when unrelated instances exist for the deployed definition or BPMN process ID.
- Existing report files must follow the shared ops overwrite safety rules introduced by related ops workflows.
- Expected workflow notices should be compact in human output while remaining complete in JSON and reports.
- Runtime/local precondition failures after planning or preflight should use local precondition error classification; static CLI shape errors should use invalid-input helpers.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add `c8volt ops execute smoke-test` under the existing `ops execute` command group.
- **FR-002**: `c8volt ops execute` MUST remain a grouping command and MUST NOT execute a smoke test directly.
- **FR-003**: The smoke-test command MUST support `--count <n>` and `-n <n>` with a default count of `1`.
- **FR-004**: The smoke-test command MUST validate `--count` as a positive integer before remote clients or mutation are created.
- **FR-005**: The smoke-test command MUST support `--workers <n>`, `--no-worker-limit`, and `--fail-fast` by reusing existing process-instance creation concurrency behavior when count is greater than one.
- **FR-006**: The smoke-test command MUST support `--dry-run`, `--no-cleanup`, `--automation`, `--auto-confirm`, `--no-wait`, `--report-file <path>`, and `--report-format markdown|json`.
- **FR-007**: Dry-run MUST perform read-only validation and planning only, including local configuration validation, optional read-only connectivity validation, fixture existence checks, flag validation, cleanup intent, and report path/format planning.
- **FR-008**: Dry-run MUST NOT deploy, start instances, walk newly created instances, delete process instances, or delete process definitions.
- **FR-009**: The workflow MUST validate connectivity using the same effective behavior as `config test-connection`.
- **FR-010**: The workflow MUST select the embedded multiple-subprocess fixture matching the configured Camunda version: 8.7 uses `C87_MultipleSubProcessesParentProcess`, 8.8 uses `C88_MultipleSubProcessesParentProcess`, and 8.9 uses `C89_MultipleSubProcessesParentProcess`.
- **FR-011**: If the matching embedded fixture is unavailable, the command MUST fail before mutation with a clear message.
- **FR-012**: The workflow MUST deploy the selected embedded fixture through existing resource service or facade behavior, not shell composition.
- **FR-013**: The workflow MUST start one or more process instances through existing process-instance run behavior.
- **FR-014**: When the deployment response provides a process-definition key, the workflow MUST prefer starting by that deployed key where supported instead of relying on latest BPMN ID lookup.
- **FR-015**: The workflow MUST walk each created process-instance family through existing traversal behavior.
- **FR-016**: Unless `--no-cleanup` is supplied, the workflow MUST delete created process instances through existing `delete pi` planning and deletion behavior.
- **FR-017**: Unless `--no-cleanup` is supplied, the workflow MUST delete the deployed smoke-test process definition only when no unrelated process instances exist for that deployed definition or BPMN process ID.
- **FR-018**: The workflow MUST NOT delete unrelated process instances or process definitions, except for scope expansion already owned by existing `delete pi` behavior from a created process-instance key.
- **FR-019**: `--automation` MUST be non-interactive and MUST use the existing automation confirmation contract for supported state-changing ops commands.
- **FR-020**: `--automation --json` MUST keep stdout deterministic and reserve progress/log output for stderr or suppress it according to existing patterns.
- **FR-021**: The workflow MUST route ops workflow logs through the configured c8volt logger and MUST NOT use `slog.Default()` in ops orchestration paths.
- **FR-022**: The workflow MUST reuse the shared ops report helpers for report-file validation, report-format inference, report-file writing, and overwrite safety.
- **FR-023**: The workflow MUST check report-file overwrite safety before preflight, planning, or discovery when overwrite is not already allowed.
- **FR-024**: Existing report files MUST be preserved for dry-run, aborted, unconfirmed, or locally blocked runs unless overwrite is already allowed by confirmation state.
- **FR-025**: The report MUST be created from a stable structured model before rendering Markdown or JSON.
- **FR-026**: The report MUST be written when the command succeeds or fails after planning starts, as long as `--report-file` is supplied.
- **FR-027**: The report model MUST include schema/version, command name, start time, finish time, duration, dry-run, c8volt version when available, configured Camunda version, safe profile/config identity, tenant id when available, fixture file, BPMN process ID, deployment status, deployed process-definition key, deployed process-definition version when available, requested count, created count, created process-instance keys, per-instance run status, per-instance walk status, traversal summaries, cleanup flags, process-instance cleanup status, process-definition cleanup eligibility, process-definition cleanup status, auto-confirm, automation, no-wait, errors, and final outcome.
- **FR-028**: Final outcome MUST be one of `planned`, `passed`, `passed_cleanup_skipped`, `partially_failed`, or `failed`.
- **FR-029**: JSON output MUST use the existing shared command result envelope where applicable and include structured per-step results.
- **FR-030**: JSON output MUST distinguish `planned`, `skipped`, `submitted`, `confirmed`, `blocked`, `confirmation_failed`, and `failed` step states where applicable.
- **FR-031**: Human output MUST clearly show created process-instance keys and cleanup-skipped or cleanup-blocked states.
- **FR-032**: Markdown reports MUST be readable by operators and MUST NOT dump unrelated process variables or sensitive data.
- **FR-033**: Existing `config test-connection`, `embed deploy`, `run pi`, `walk pi`, `delete pi`, and `delete pd` behavior MUST remain unchanged.
- **FR-034**: Downstream Ralph launch instructions MUST include `--implementation-context specs/ralph-implementation-rules.md`.

### Key Entities

- **Smoke Test Plan**: The read-only plan for connectivity validation, fixture selection, deployment, instance creation, traversal, cleanup, and report output.
- **Embedded Smoke-Test Fixture**: The version-matched multiple-subprocess BPMN fixture selected for the configured Camunda version.
- **Deployment Result**: The deployed fixture metadata, including BPMN process ID, process-definition key, process-definition version, tenant id, and deployment status.
- **Smoke-Test Process Instance**: A process instance created by the smoke-test run and eligible for traversal and cleanup.
- **Traversal Summary**: Per-created-instance walk result containing family traversal status and summary counts without dumping unrelated sensitive data.
- **Cleanup Eligibility**: The safety decision for process-instance and process-definition cleanup, including unrelated-instance blockers.
- **Smoke-Test Audit Report**: The stable structured model rendered to human, JSON, Markdown report, or JSON report output.
- **Workflow Step Result**: A per-step status entry using constrained states such as planned, skipped, submitted, confirmed, blocked, confirmation_failed, or failed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `c8volt ops execute smoke-test --dry-run` performs zero mutation requests and reports the planned connectivity, fixture, deployment, run, traversal, cleanup, and report steps.
- **SC-002**: `c8volt ops execute smoke-test` validates connectivity, deploys the matching fixture, starts one process instance, walks its family, cleans up the created instance, conditionally deletes the process definition, and reports success.
- **SC-003**: `--count 5` and `-n 5` each create five process instances and walk each created instance.
- **SC-004**: `--no-cleanup` creates and walks instances, submits no cleanup requests, and reports retained process-instance keys plus deployed definition metadata.
- **SC-005**: If unrelated process instances exist for the deployed definition or BPMN process ID, process-definition cleanup is skipped and reported as blocked or not applicable.
- **SC-006**: `--automation --json` runs through supported non-interactive paths with deterministic JSON on stdout and no interactive prompts.
- **SC-007**: `--report-file smoke-test.md` writes a Markdown report with each smoke-test step, timestamps, duration, created keys, cleanup status, errors, and final outcome.
- **SC-008**: `--report-file smoke-test.json --report-format json` writes a structured JSON report suitable for automation.
- **SC-009**: Generated CLI documentation and examples include dry-run, count, no-cleanup, automation JSON, and report-file workflows.
- **SC-010**: Existing lower-level command tests continue to pass without behavior changes caused by the smoke-test workflow.

## Assumptions

- The existing ops command foundation, report helpers, overwrite safety conventions, output rhythm, and automation confirmation contract from related ops work are available or will be reused when present.
- The implementation should add missing primitive capabilities to the correct existing resource or process facade/service rather than implementing resource-specific logic in the ops command or ops facade.
- Camunda v2 API behavior remains the preferred path for new operations, with version-specific differences handled in the appropriate service packages.
- The workflow is CLI-only and does not require new generated Camunda client endpoints unless nearby service inspection proves a primitive is missing.
- Every commit subject for this feature must use Conventional Commits format and end with `#188`.

## Out of Scope

- Shelling out to existing c8volt CLI commands to implement the workflow.
- Adding new BPMN fixtures beyond the existing multiple-subprocess embedded fixture.
- Running arbitrary user-selected BPMN files.
- Deleting unrelated process instances.
- Force-deleting process definitions when unrelated process instances exist.
- Adding Camunda batch operation usage.
- Hand-editing generated CLI docs instead of regenerating them through the existing docs path.
