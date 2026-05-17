# Feature Specification: Ops Repair Workflows

**Feature Branch**: `183-ops-repair-workflows`
**Created**: 2026-05-17
**Status**: Draft
**Input**: GitHub issue [grafvonb/c8volt#183](https://github.com/grafvonb/c8volt/issues/183) - "feat(ops): add bulk repair workflows with filtered discovery and audited reports"

## Issue Traceability

- **Issue Number**: 183
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/183
- **Issue Title**: feat(ops): add bulk repair workflows with filtered discovery and audited reports

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Repair Explicit Incidents (Priority: P1)

As a Camunda operator, I can run `c8volt ops repair incident` against one or more explicit incident keys, including keys read from stdin, so that known production incidents can be remediated through a repeatable workflow.

**Why this priority**: Explicit incident repair is the smallest useful operational workflow and establishes job-backed versus non-job incident handling.

**Independent Test**: Can be tested with fake active incident records for one job-backed incident and one non-job incident, verifying that the command repairs only the requested incident keys and reports each step outcome.

**Acceptance Scenarios**:

1. **Given** a single active job-backed incident, **When** the operator runs `c8volt ops repair incident --key <incident-key>`, **Then** retries default to `1`, the incident is resolved, clearance is confirmed, and the result reports success.
2. **Given** a single active non-job incident, **When** the operator runs `c8volt ops repair incident --key <incident-key>`, **Then** job retry and timeout steps are marked `not_applicable`, the incident is resolved, clearance is confirmed, and the result clearly states that no related job was present.
3. **Given** multiple incident keys from repeated `--key` flags or stdin `-`, **When** the command runs, **Then** the repair plan freezes those incident keys before mutation and repairs only that frozen set.

---

### User Story 2 - Discover Incidents With Filters (Priority: P2)

As a Camunda operator, I can run `c8volt ops repair incident` with native incident search filters, so that I can repair a bounded operational set without manually collecting every incident key.

**Why this priority**: Filtered discovery is the core bulk workflow and must be safe before broader process-instance driven workflows build on it.

**Independent Test**: Can be tested by running the incident repair command with incident filters against a fake server and verifying frozen discovery, dry-run behavior, and no expansion to newly created incidents.

**Acceptance Scenarios**:

1. **Given** incident search filters such as `--state active --error-type io_mapping_error --limit 25`, **When** the operator runs incident repair, **Then** the command discovers matching incidents, freezes the incident keys, and repairs only that set.
2. **Given** both explicit keys and search filters, **When** the operator runs incident repair, **Then** the command rejects the request before remote mutation because keyed mode and search-filter mode are mutually exclusive.
3. **Given** new incidents appear after discovery, **When** a bulk repair is already running, **Then** those new incidents may appear only as post-repair context and must not expand the mutation set.

---

### User Story 3 - Repair Incidents From Process Instances (Priority: P3)

As a Camunda operator, I can run `c8volt ops repair process-instance` against explicit process-instance keys, stdin keys, or incident-bearing process-instance filters, so that I can repair all active incidents associated with selected process instances.

**Why this priority**: Process-instance selection is the second repair entry point and depends on reliable incident discovery and frozen target semantics.

**Independent Test**: Can be tested by selecting process instances with active incidents and verifying that the command freezes selected process instances and deduped incident keys before repair.

**Acceptance Scenarios**:

1. **Given** an explicit process-instance key with active incidents, **When** the operator runs `c8volt ops repair process-instance --key <process-instance-key>`, **Then** the command discovers active incidents for that process instance, freezes the deduped incident set, and repairs those incidents.
2. **Given** process-instance search filters, **When** the operator runs process-instance repair, **Then** the command requires an incident-bearing selector such as `--incidents-only` or `--direct-incidents-only` before scanning and repair.
3. **Given** a process-instance selection that produces duplicate incident references, **When** repair starts, **Then** each incident is repaired once and output preserves deterministic reporting.

---

### User Story 4 - Apply Shared Variable Updates Safely (Priority: P4)

As a Camunda operator, I can provide `--vars` or `--vars-file` during repair, so that required process-instance variables are updated once per unique scope before dependent incidents are resolved.

**Why this priority**: Variable repair is commonly needed for mapping and expression incidents and must gate resolution when updates fail.

**Independent Test**: Can be tested with multiple incidents sharing one process-instance variable scope, verifying deduped variable mutation, normalized value confirmation, and blocked dependent resolution on update failure.

**Acceptance Scenarios**:

1. **Given** multiple incidents with the same process-instance variable scope, **When** the operator supplies `--vars <json-object>`, **Then** the variable update is applied once for that scope and only requested variable names are confirmed.
2. **Given** a variable update fails for a scope, **When** incidents depend on that scope, **Then** those incidents are not resolved and their results explain that repair was blocked by the variable update failure.
3. **Given** `--vars-file <path>` is supplied, **When** the file is parsed, **Then** parsing and validation match existing process-instance variable update behavior.

---

### User Story 5 - Preview Repair Without Mutation (Priority: P5)

As a Camunda operator, I can run either repair target with `--dry-run`, so that I can inspect the exact frozen target set, applicability decisions, and requested mutations before changing a cluster.

**Why this priority**: Dry-run keeps bulk operational remediation safe and supports review before mutation.

**Independent Test**: Can be tested by verifying discovery calls occur while variable, job, and incident mutation calls do not occur, and by validating the planned output fields.

**Acceptance Scenarios**:

1. **Given** `--dry-run`, **When** incident repair runs, **Then** the command performs discovery and validation without mutating variables, jobs, or incidents.
2. **Given** `--dry-run` with mixed job-backed and non-job incidents, **When** output is rendered, **Then** the output shows job keys where present and `not_applicable` job steps where absent.
3. **Given** `--dry-run` with report options, **When** the command completes, **Then** the planned report path and format are included without implying mutations occurred.

---

### User Story 6 - Produce Audited Repair Reports (Priority: P6)

As a Camunda operator, I can request a structured repair report, so that successful and failed repair attempts are auditable after discovery.

**Why this priority**: Audit reports make operational remediation traceable and useful for post-incident review.

**Independent Test**: Can be tested by requesting Markdown and JSON reports and validating that the structured report contains discovery, frozen targets, per-step statuses, notices, errors, and final outcome.

**Acceptance Scenarios**:

1. **Given** `--report-file <path> --report-format json`, **When** repair completes after discovery, **Then** a structured JSON report is written with schema version, command metadata, timestamps, target sets, step statuses, errors, notices, and final outcome.
2. **Given** `--report-format markdown`, **When** a report is written, **Then** the Markdown rendering is derived from the structured report model and includes the same operational facts.
3. **Given** discovery succeeds but later repair fails, **When** report options are set, **Then** the report is still written with `partially_failed` or `failed` outcome and the captured errors.

---

### Edge Cases

- Mixed bulk sets containing job-backed and non-job incidents must not fail merely because some incidents have no job key.
- `--retries 0` skips retry restoration but still permits other requested repair steps and incident resolution.
- `--job-timeout <duration>` applies only to incidents with related job keys and is `not_applicable` for non-job incidents.
- Explicit job flags in mixed repairs apply only where a related job exists.
- Keyed mode and search-filter mode are mutually exclusive for both repair targets.
- Process-instance search mode must require incident-bearing selectors until a future requirement explicitly allows scanning process instances without incidents.
- Bulk repair must not chase newly created incidents after the original frozen target set is built.
- Automation JSON output must remain deterministic and must not require interactive confirmation for automation-supported paths.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add `c8volt ops repair incident` and `c8volt ops repair process-instance` as concrete repair targets while keeping `c8volt ops repair` as a grouping command without an ambiguous top-level `--key`.
- **FR-002**: Incident repair MUST accept repeated `--key <incident-key>`, stdin keys with `-`, and native incident search filters from `get incident`, including state, error, process-instance, process-definition, flow-node, element-instance, creation-time, batch-size, and limit filters.
- **FR-003**: Process-instance repair MUST accept repeated `--key <process-instance-key>`, stdin keys with `-`, and native process-instance search filters from `get pi`, including process-definition, state, hierarchy, incident selector, date, batch-size, and limit filters.
- **FR-004**: For both repair targets, keyed mode and search-filter mode MUST be mutually exclusive and rejected before remote mutation.
- **FR-005**: Process-instance search repair MUST require an incident-bearing selector such as `--incidents-only` or `--direct-incidents-only`.
- **FR-006**: Bulk repair MUST freeze discovered targets before mutation, including incident keys, process-instance keys, job keys when present, variable scopes, original incident state, and original error context.
- **FR-007**: Repair MUST support job-backed and non-job incidents in the same operational pass.
- **FR-008**: For each discovered incident, the repair workflow MUST discover context, build a repair plan, optionally update variables, optionally update related job retry and timeout settings, resolve the incident, confirm the incident cleared, and include post-repair context in output and reports.
- **FR-009**: Job repair steps MUST be incident-local: related job retry and timeout requests apply only when a related job key exists and are reported as `not_applicable` when no related job is present.
- **FR-010**: Job-backed incidents MUST default requested retries to `1`, while `--retries 0` MUST skip retry restoration.
- **FR-011**: `--job-timeout <duration>` MUST update timeout only for incidents with related job keys.
- **FR-012**: `--vars <json-object>` and `--vars-file <path>` MUST parse and validate variables consistently with existing process-instance variable updates.
- **FR-013**: Initial variable updates MUST use the process-instance key as the variable update scope and MUST state that scope in help text, JSON output, and audit reports.
- **FR-014**: Bulk variable updates MUST apply the same payload once per unique variable scope, confirm only requested variable names, and compare normalized JSON values rather than raw formatting.
- **FR-015**: If a variable update fails for a scope, incidents dependent on that scope MUST NOT be resolved unless a future override flag is introduced.
- **FR-016**: `--dry-run` MUST perform discovery and validation while skipping all variable, job, and incident mutations.
- **FR-017**: Dry-run output MUST show discovery filters or input keys, frozen incident keys, process-instance keys, variable scopes and names, job keys where present, job repair applicability, retry and timeout requests, incident resolution targets, and report path/format when requested.
- **FR-018**: Repair commands MUST support applicable bulk controls: `--workers`, `--fail-fast`, `--no-worker-limit`, `--batch-size`, `--limit`, `--dry-run`, `--auto-confirm`, `--automation`, `--report-file`, and `--report-format`.
- **FR-019**: `--automation --json` MUST be deterministic and MUST NOT require `--auto-confirm` for commands marked with full automation support.
- **FR-020**: `--report-file <path>` and `--report-format markdown|json` MUST produce structured audit reports using shared ops report behavior for format inference, path validation, overwrite behavior, and writing.
- **FR-021**: Reports MUST include schema/version, command name, timestamps, duration, dry-run flag, c8volt version, configured Camunda version, safe profile identity, discovery mode, discovery filters, input keys, frozen targets, job applicability, original incident context, requested variables, variable update status, retry status, timeout status, resolution status, confirmation status, remaining incident summary, skipped/not-applicable steps, notices, errors, and final outcome.
- **FR-022**: Final repair outcomes MUST include at least `planned`, `repaired`, `partially_failed`, and `failed`.
- **FR-023**: Repair step statuses MUST include at least `planned`, `skipped`, `not_applicable`, `submitted`, `confirmed`, `confirmation_failed`, `blocked`, and `failed`.
- **FR-024**: Existing `get incident`, `get pi`, `update pi`, `update job`, `resolve incident`, and `resolve pi` behavior MUST remain unchanged.
- **FR-025**: Repair orchestration MUST preserve the repository layering requirement `cmd -> c8volt/ops facade -> internal/services/ops workflow -> resource services -> generated Camunda clients`.
- **FR-026**: Planning, task generation, and every Ralph implementation iteration for this feature MUST apply `specs/ralph-implementation-rules.md`; Ralph MUST NOT be launched unless that implementation context is included in the launcher instructions.

### Key Entities

- **Repair Command Target**: The selected repair entry point, either incident repair or process-instance repair, including discovery mode, filters, input keys, and bulk controls.
- **Frozen Repair Set**: The immutable set of incident keys, process-instance keys, related job keys, variable scopes, and original incident context discovered before mutation.
- **Repair Plan**: The per-incident plan describing variable update needs, job repair applicability, requested retries, requested timeout, resolution target, confirmation expectation, and dry-run status.
- **Repair Result**: The per-incident and aggregate outcome containing step statuses, errors, notices, post-repair context, and final result.
- **Variable Scope Update**: A deduped process-instance variable mutation request and confirmation result for a unique variable scope.
- **Audit Report**: The structured report model and Markdown or JSON rendering for a repair attempt after discovery.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A single active job-backed incident can be repaired with `ops repair incident --key <key>` and reports confirmed success without changing existing lower-level command behavior.
- **SC-002**: A single active non-job incident can be repaired with job steps reported as `not_applicable`, and the absence of a related job key does not fail the workflow.
- **SC-003**: A mixed bulk incident repair containing job-backed and non-job incidents completes one deterministic report where non-job job steps are `not_applicable` and job-backed steps reflect requested retry or timeout behavior.
- **SC-004**: Search-mode repair freezes the discovered set before mutation and never repairs incidents discovered only after that freeze point.
- **SC-005**: `--dry-run` performs zero mutation calls while still returning the same frozen target and applicability information needed to review the repair.
- **SC-006**: Variable repair updates each unique process-instance scope at most once per command run and blocks dependent incident resolution on failed variable updates.
- **SC-007**: Markdown and JSON audit reports can be generated for successful repairs, partial failures, and failures after discovery, with the required target, step, notice, error, and outcome fields.
- **SC-008**: Relevant command tests cover human output, JSON output, keys-only or deterministic machine output where supported, invalid flag combinations, dry-run, automation mode, keyed mode, search mode, mixed job applicability, variable failure blocking, and report generation.

## Assumptions

- Operators already have valid c8volt configuration, authentication, and access to the target Camunda cluster.
- Existing discovery and mutation primitives from incident, process-instance, job, and ops services should be reused and extended rather than bypassed.
- Initial variable repair scope is the process-instance key; future finer-grained scopes are out of scope for this issue.
- Interactive repair prompts beyond existing confirmation behavior are out of scope.
- Force-resolve, continue-on-error overrides, creating incidents, failing jobs, completing jobs, throwing BPMN errors, and updating retry backoff are out of scope.
- Generated Camunda clients must not be called directly from command or ops orchestration code.
- Every commit subject created for this issue must use Conventional Commits format and end with `#183`.
