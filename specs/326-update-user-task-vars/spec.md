# Feature Specification: Update User-Task Variables

**Feature Branch**: `codex/326-update-user-task-vars`

**Created**: 2026-09-29

**Status**: Draft

**Input**: [GitHub issue #326](https://github.com/grafvonb/c8volt/issues/326) — Add `update user-task` variable updates consistent with `update process-instance`. Preserve the existing command grammar and conventions; keep the feature small.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Preview and Update One Task's Variables (Priority: P1)

As an operator, I want to update the variables visible through `get ut --with-vars`, using the same workflow as `update pi`, so I can correct task data without learning another command style.

**Why this priority**: Correct scope selection and a trustworthy preview are the core value of the command.

**Independent Test**: Select a task with local and inherited variables, preview a payload containing changes and a new name, confirm it, and inspect the resulting values and scopes.

**Acceptance Scenarios**:

1. **Given** a task with local and inherited variables, **When** I run `update ut --key <key> --vars '<object>' --dry-run`, **Then** the preview distinguishes additions, changes, unchanged requested values, and untouched values using the PI command's conventions, identifies the intended scopes including inherited targets, and performs no mutation or confirmation prompt.
2. **Given** an existing effective variable, **When** I confirm a changed value, **Then** the value changes at its returned scope, without creating a task-local shadow or propagating beyond that scope; unrelated values remain unchanged.
3. **Given** a requested name absent from the task's effective variables, **When** I confirm the update, **Then** it is created in the task's local element-instance scope and is visible through `get ut --with-vars`.
4. **Given** requested values already equal the complete effective values, **When** I execute the command, **Then** it reports a successful no-op without mutation or confirmation, including when the matching values are inherited.
5. **Given** an accepted mutation, **When** ordinary execution finishes, **Then** success is reported only after the requested values are confirmed at their intended scopes; inability to confirm remains an explicit failure distinct from submission failure.
6. **Given** an interactive plan, **When** I decline confirmation or reach EOF, **Then** the command preserves the existing PI abort behavior and submits no mutation.

---

### User Story 2 - Update Multiple Explicit Tasks (Priority: P2)

As an operator, I want to apply one variable payload to several task keys from flags or a pipeline, with the same execution controls as `update pi`.

**Why this priority**: Bulk input is already part of the established update workflow and must remain predictable when tasks share inherited variables.

**Independent Test**: Supply repeated and piped keys for tasks sharing one inherited variable, execute one payload, and verify unique writes, input ordering, and truthful per-task outcomes.

**Acceptance Scenarios**:

1. **Given** repeated or comma-separated `--key` values and newline-separated stdin keys selected with `-`, **When** I run the command, **Then** unique tasks retain first-input order and receive the same payload using the existing PI input rules.
2. **Given** selected tasks inheriting the same variable from the same scope, **When** they request the same change, **Then** that scope/name has one planned mutation target and each affected task has an accurate outcome; variables with the same name at different scopes remain separate targets.
3. **Given** a payload from `--vars-file`, **When** I use the same content through `--vars`, **Then** planning and execution are equivalent.
4. **Given** several planned updates and a mutation failure, **When** execution proceeds with the established worker and fail-fast controls, **Then** completed work and failures remain distinguishable, unscheduled work is not reported as successful, and shared targets do not produce contradictory task outcomes.
5. **Given** a selected task outside the configured discovery tenant, **When** explicit-key backend authorization permits access, **Then** the command uses the actual task and variable tenant context; denied access remains an error.

---

### User Story 3 - Use Existing Automation and Output Conventions (Priority: P2)

As an automation author, I want the task update command to behave like `update pi` so existing scripts need only resource-specific substitutions.

**Why this priority**: Output and confirmation consistency prevent scripts from confusing a preview, an accepted write, and a confirmed result.

**Independent Test**: Exercise dry-run, no-op, accepted, confirmed, and failed execution with the established output and unattended flags, capturing stdout and stderr separately.

**Acceptance Scenarios**:

1. **Given** `--automation` or `--auto-confirm`, **When** updates are required, **Then** execution follows the PI unattended confirmation rules without prompting.
2. **Given** `--no-wait`, **When** writes are accepted, **Then** the result distinguishes submission from confirmation; mutation failures remain failures and a no-op never claims submission.
3. **Given** JSON output in a permitted PI-equivalent flag combination, **When** execution returns a preview, no-op, result, or error, **Then** stdout contains exactly one shared envelope followed by EOF, with no human output mixed in.
4. **Given** effective keys-only output, **When** results exist, **Then** stdout contains only user-task keys, one per line; successful no-work output contains zero bytes. Quiet mode suppresses human summaries while preserving explicitly requested machine output.
5. **Given** real terminal stdin and redirected stdout, **When** confirmation is eligible under the existing PI rules, **Then** the unchanged confirmation interaction uses configured or inherited stderr and leaves stdout uncontaminated.
6. **Given** Camunda 8.8, 8.9, or 8.10, **When** I execute a supported update, **Then** the same command contract applies; Camunda 8.7 reports an explicit unsupported error without mutation.

### Edge Cases

- Missing keys or empty stdin remain the established PI input error; they must not become an unrestricted search. This differs from selected tasks whose completed plan requires no changes.
- Missing, conflicting, unreadable, malformed, or non-object payload sources fail before remote work where locally determinable. An empty object produces no changes for valid selected tasks. JSON null as a property value sets that value; it does not delete a variable.
- Variable discovery must be complete before treating a name as absent. Empty intermediate pages with remaining results do not establish absence.
- Backend-truncated values must not establish equality or successful confirmation. Retrieval or scope-resolution failure must not be converted to an addition or successful no-op.
- A task can disappear, complete, or lose authorization between planning and execution. Report the resulting failure without silently choosing another scope.
- An inherited variable may affect tasks outside the selected keys. The preview must make its inherited scope visible without expanding task selection or introducing extra discovery.
- Several targets can succeed before another fails. Preserve accepted work and report partial outcomes; no rollback or atomic transaction is promised.
- A variable can change after planning. Preserve the planned target scopes and existing update semantics; this feature adds no concurrency-locking or snapshot guarantee.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provide `c8volt update user-task` with established user-task aliases `ut`, `uts`, and `user-tasks`. Preserve the PI command's verb/resource grammar, input conventions, wording rhythm, and help style.
- **FR-002**: Accept explicit keys through repeated/comma-separated `--key` and stdin using the PI command's existing selection and validation rules. Deduplicate keys in first-input order. Do not add search-based selection or implicit selection of all tasks.
- **FR-003**: Require exactly one of `--vars` and `--vars-file`, containing a JSON object, and apply the same payload to each selected task. Preserve PI payload validation and value semantics.
- **FR-004**: Determine existing names and scopes from the same complete effective-variable view as `get ut --with-vars`. Update existing names at their returned scope, create absent names locally on the task, and leave unmentioned names untouched. Writes MUST remain confined to the resolved target scopes.
- **FR-005**: Produce a plan distinguishing additions, changes, unchanged requested values, and untouched values. Show inherited target scope where needed to understand the effect. Equality MUST use complete values; inherited equal values MUST remain unchanged rather than create local overrides.
- **FR-006**: Deduplicate planned changes by scope and variable name across selected tasks. Preserve task associations and accurate per-task outcomes, with consistent counts between preview, confirmation, and execution. Execute only planned changes without silently changing the confirmed target scopes.
- **FR-007**: Match PI dry-run and confirmation conventions, including `--auto-confirm` and `--automation`. Dry-run, successful no-op, and abort paths MUST issue zero mutations; dry-run and no-op MUST not prompt. Interactive control text MUST use configured stderr and retain existing prompt eligibility, defaults, and EOF behavior.
- **FR-008**: Confirm requested values at their intended scopes through the existing task variable lookup before claiming confirmed success. Preserve established waiting controls. `--no-wait` skips confirmation and reports accepted submission truthfully, without hiding mutation errors.
- **FR-009**: Support the PI worker-limit and fail-fast controls, deterministic result ordering, cancellation, and established error classifications. Preserve meaningful partial outcomes; never label failed, unconfirmed, or unscheduled work as confirmed success.
- **FR-010**: Follow existing human, JSON, keys-only, quiet, and verbose conventions and output precedence. JSON MUST emit exactly one shared envelope, including successful no-ops and errors. Effective keys-only output MUST contain only task keys and zero bytes for no work. Quiet MUST preserve explicitly requested machine results. Retain the PI command's JSON confirmation and JSON/verbose compatibility rules.
- **FR-011**: Use explicit-key authorization without discovery-tenant filtering and report actual tenant context using existing conventions. Authorization and lookup failures MUST remain failures.
- **FR-012**: Support Camunda 8.8, 8.9, and 8.10 with equivalent observable behavior; Camunda 8.7 MUST fail explicitly as unsupported without mutation.
- **FR-013**: Keep scope resolution and behavior consistent with existing task reads and preserve existing PI behavior. The feature MUST follow repository governance and reuse existing conventions without introducing extra capabilities or unrelated refactoring.
- **FR-014**: Keep command discovery metadata, help, examples, README guidance, and generated CLI documentation aligned with the delivered behavior, including inherited-scope effects, local creation, supported versions, and no-wait meaning.

### Key Entities *(include if feature involves data)*

- **Selected user task**: An explicitly requested task, its local element scope, actual tenant, and position in the unique input order.
- **Effective variable**: A name, received value, actual scope, tenant context, and completeness information visible for a selected task.
- **Variable update plan**: Requested values classified as additions, changes, unchanged, or untouched, with resolved target scopes and associated tasks.
- **Update outcome**: Per-task facts about submission, confirmation, failure, and unfinished work, including effects of shared variable targets.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All acceptance cases for local changes, inherited changes, missing names, and untouched values produce the intended scope/value results, with zero unintended scope changes.
- **SC-002**: Operators can perform inline, file-based, and piped-key updates using the existing PI command pattern, changing only the resource name, keys, and payload; no new scope flags or command hierarchy are required.
- **SC-003**: Every tested dry-run, successful no-op, and abort submits zero mutations; every shared scope/name has one logical mutation target regardless of the number of selected tasks.
- **SC-004**: Every tested machine-output combination meets the one-envelope or one-key-per-line contract; no-work keys output is zero bytes and interactive text contributes zero bytes to stdout.
- **SC-005**: Every tested failure and no-wait outcome distinguishes accepted work from confirmed work, with no false confirmed-success result, across all three supported versions.

## Assumptions

- The issue and the user's scope decisions are authoritative: existing effective variables retain their actual scopes; absent names are created locally. No clarification is outstanding on this distinction.
- The compatibility baseline is `update pi`, subject to the stricter current output and architecture requirements in [AGENTS.md](../../AGENTS.md), [Ralph implementation rules](../ralph-implementation-rules.md), and the project constitution. Existing deviations are not precedent for reproducing them.
- Existing authentication, configuration, retry, waiting, tenant, and output facilities remain the baseline. This feature adds no new controls in those areas.
- Existing user-task lookup and complete effective-variable retrieval are prerequisites; variable values and scope metadata may change concurrently.
- Scope is limited to variable updates for explicit task keys. Task attributes, assignment, completion, search-based mutation selection, new scope-selection flags, unrelated refactoring, and additional capabilities are excluded.
- Implementation planning must retain the issue's architecture, coverage, and documentation obligations. This specification defines observable behavior; package design, validation commands, and delivery tasks belong in the plan and tasks.
