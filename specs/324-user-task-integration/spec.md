# Feature Specification: User-Task Integration Coverage

**Feature Branch**: `codex/324-user-task-integration`

**Created**: 2026-09-23

**Status**: Draft

**Input**: [GitHub issue #324](https://github.com/grafvonb/c8volt/issues/324): Extend user-task integration coverage using existing versioned embedded models. Validate variable filtering (#310) and effective-variable display (#309) through the existing integration suite without changing production behavior or BPMN models.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Trust Local Filtering and Effective Display (Priority: P1)

As an operator validating a release, I want real-cluster evidence that task-local filtering selects the right tasks and that variable display independently shows their effective values, so I can trust searches and inspections during operations.

**Why this priority**: Request-level tests alone cannot establish actual backend scope selection and shadowing.

**Independent Test**: Run the baseline get integration slice against one supported profile using its existing `SimpleUserTask` and `SimpleUserTaskWithIncident` models. Compare observed task identities and variable scopes with known seeded values.

**Acceptance Scenarios**:

1. **Given** a task created from `SimpleUserTaskWithIncident` with `hasIncident=false`, process-scope `incident=99`, a process-only string, a longer structured value, and the run marker, **When** its task and variables become observable, **Then** its local `incident` is `1`, `--var 'incident=1'` and `--var-exists incident` select it, and `--var 'incident=99'` excludes it.
2. **Given** a process-only string visible to an active task, **When** searching for that value with a task-local filter, **Then** the task is excluded while effective-variable display still exposes that process-scope value. The ordinary `SimpleUserTask` supplies a case without the incident model's local mapping.
3. **Given** a matching local filter, **When** adding `--with-vars`, **Then** the selected task keys remain identical, effective `incident=1` shadows process `incident=99`, and the process-only variables remain visible with their expected scope metadata.
4. **Given** the existing model's known local value, **When** exercising a compact operator table, **Then** equality, inequality, explicit existence, membership and basic wildcard cases select the expected task identities under the established filter grammar and backend contract.
5. **Given** a long structured value in effective display, **When** requesting human shortening with `--var-value-limit`, **Then** the existing shortening contract is preserved and JSON retains the full received value and truncation metadata.

---

### User Story 2 - Verify Variables in Called Processes (Priority: P2)

As an operator validating workflows with call activities, I want the same scope guarantees for a task in a called process, so I do not confuse caller, child-process, and task-local values.

**Why this priority**: Parent-child workflows introduce ownership boundaries not covered by a standalone task.

**Independent Test**: Deploy matching versions of `SimpleParentWithIncidentSubprocess` and `SimpleUserTaskWithIncident`, start the parent with the same seed values as Story 1, discover the child through existing commands, and inspect its task.

**Acceptance Scenarios**:

1. **Given** a parent and child definition from the selected profile's model family, **When** the parent starts and the child is discovered, **Then** the child PI key and task ownership are recorded and subsequent task queries target that child PI.
2. **Given** copied child-process `incident=99` and task-local `incident=1`, **When** filtering and displaying the child task, **Then** local `1` matches, process `99` does not, and effective display returns the shadowing local value plus the expected copied process variables.
3. **Given** caller variables supplied at start, **When** evaluating child results, **Then** assertions use the copied child values and do not assume subsequent caller updates automatically propagate into an active child.

---

### User Story 3 - Trust Bounded Searches Across Versions (Priority: P2)

As a release operator, I want filtered paging and counts verified using existing integration entry points, with honest version outcomes, so a successful run means the selected version's operational behavior was exercised.

**Why this priority**: Missing pages, inconsistent counts, or silently skipped versions can undermine scripts even when a single-task search succeeds.

**Independent Test**: Run the get volume slice with at least three matching tasks, a batch size of one, and a limit of two, then compare scoped keys, JSON results, totals, and unmatched output.

**Acceptance Scenarios**:

1. **Given** at least three suite-owned matching tasks, **When** traversing with small batches, **Then** all expected task keys appear exactly once, and unrestricted JSON, keys-only output, and total agree on the scoped set.
2. **Given** more matches than the requested limit, **When** requesting a limit of two, **Then** exactly two distinct matching tasks are returned without requiring an unspecified ordering.
3. **Given** an observed seeded dataset, **When** applying an unmatched filter, **Then** existing empty human and JSON output is preserved and keys-only output contains zero bytes.
4. **Given** selected 8.8, 8.9, or 8.10 profiles, **When** running either get slice, **Then** the corresponding C88, C89, or C810 models are used and the applicable baseline or volume assertions execute for each selected supported version.
5. **Given** a selected 8.7 profile, **When** checking native user-task search and effective-variable display, **Then** the established unsupported behavior is asserted instead of attempting supported-version scenarios.
6. **Given** an unavailable version or missing prerequisite, **When** reporting the run, **Then** it is explicitly distinguished from verified coverage; failed required setup is not silently converted into a pass.

### Edge Cases

- Delayed task or variable visibility: wait within a bounded readiness period before negative assertions; report failure when required observable state does not arrive.
- Dirty clusters and retained prior runs: scope results to suite-owned data and known keys rather than global counts or model names alone.
- Definition selection: deploy matching parent and child models, and verify the discovered child's identity instead of assuming an unrelated deployed definition was used.
- Missing profiles or connection/version mismatches: retain existing readiness behavior and explicit outcomes; no false claim that all supported versions ran.
- Structured or backend-truncated variables: compare received values and truncation metadata without claiming the backend supplied omitted content.
- Fixed local value: missing/null and complex string-pattern cases unsupported by the existing models remain outside this live dataset; do not manufacture new models or silently claim those cases passed.
- Optional reuse of `MultipleSubProcessesParent` must reflect its actual active leaf tasks; its own later task and the second task of `DoubleUserTask` are not initially active.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Coverage MUST exercise actual CLI commands against selected disposable profiles for features #309 and #310, using existing authentication, profile selection, and version readiness checks.
- **FR-002**: Each supported profile MUST use its corresponding existing C88, C89, or C810 embedded models. No model identifier rewriting, cross-version definitions, new models, or model modifications are permitted. C87 coverage MUST assert unsupported native user-task behavior.
- **FR-003**: Baseline coverage MUST prove local matching, process-only exclusion, shadowing, display independence, and display shortening as specified in Story 1, including a compact table of five operators ($eq, $neq, $exists, $in, $like) using available local values.
- **FR-004**: Called-process coverage MUST use the matching incident parent/child models and discovered child PI keys, proving local and effective scopes without assuming live caller-variable inheritance.
- **FR-005**: Volume coverage MUST prove complete filtered traversal, duplicate-free identities, bounded results, consistent counts, and established empty output using multiple matching tasks.
- **FR-006**: Seeded scenarios MUST preserve run markers and existing defaults while allowing scenario-specific starting variables. Setup and discovery MUST use existing commands, without direct API setup.
- **FR-007**: Assertions MUST use suite-owned keys and bounded queries and establish expected task/variable visibility before checking exclusions. Required setup failures MUST remain visible failures.
- **FR-008**: Existing evidence MUST record command execution, selected keys, relevant variable scopes, and the tested version. Passed, failed, skipped, unavailable, and expected-unsupported outcomes MUST be distinguishable without claiming unexecuted coverage.
- **FR-009**: New baseline and volume scenarios MUST be selected by `integration-cli-get` and `integration-cli-get-volume`, respectively, and consequently by `integration-test-all`; no new runner or issue-specific target is required.
- **FR-010**: The user-task coverage manifest MUST include `var`, `var-exists`, `var-like`, `with-vars`, and `var-value-limit`, aligned with executable coverage.
- **FR-011**: Changes MUST reuse the existing integration suite and helpers and preserve unrelated scenario behavior. Production changes, new dependencies, broad runner improvements, and a complete live operator/output permutation matrix are out of scope.

### Key Entities

- **Selected profile**: Existing operator configuration identifying the target cluster and expected version.
- **Versioned model family**: Existing matching embedded process definitions, including required called-process dependencies.
- **Seeded scenario**: Run-owned process instances, discovered tasks, starting values, expected variable scopes, and known matching identities.
- **Execution evidence**: Observable command results and version-specific outcomes sufficient to distinguish verified behavior from unexecuted coverage.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every selected, ready supported version, all required baseline assertions return exactly the expected task identities and effective values, with zero process-only false matches.
- **SC-002**: Every called-process scenario identifies the expected child ownership and demonstrates the same local-versus-process distinction as the standalone scenario.
- **SC-003**: A volume dataset of at least three matches is retrieved without missing or duplicated tasks; unrestricted result modes agree on the count and a limit of two returns exactly two matches.
- **SC-004**: Every unmatched keys-only case emits zero bytes, and variable display changes zero selected task identities.
- **SC-005**: Every selected version has an explicit result; zero skipped or unavailable scenarios are reported as verified, and 8.7 is recorded as expected unsupported when that assertion passes.
- **SC-006**: Existing baseline, volume, and aggregate entry points include the new scenarios without additional operator setup conventions, new model artifacts, or product behavior changes.

## Assumptions

- Issue #324 is authoritative; the prior discussion explains its rationale without expanding scope.
- Features #309 and #310 are available in the tested binary. Selected clusters are disposable and accessible through the existing default-local configuration.
- The supported feature versions are 8.8, 8.9, and 8.10. Actual execution evidence depends on available selected profiles; availability is never inferred merely from model files or adapter support.
- `SimpleUserTaskWithIncident` is started with `hasIncident=false`; its existing mapping is expected to create local `incident=1`. Setup verifies this state rather than relying solely on model inspection.
- Existing models constrain the live dataset. Rich string escaping, arbitrary task-local nulls, and exhaustive grammar/output combinations remain covered by focused non-live tests.
- `SimpleParent` and `MultipleSubProcessesParent` may be reused where existing discovery coverage helps; adding redundant topology permutations is not required.
- This specification defines test outcomes and scope. Detailed helper changes and test organization belong in the implementation plan. Broader aggregate-runner corrections remain separate work.

## Approved coverage adjustment

The user requested removal of both live `$notIn` cases for all Camunda versions after the C89 HTTP 500. Similar behavior on other versions is a working assumption, not verified evidence. Live `$notIn` semantics are outside this suite; existing unit request-encoding tests remain unchanged. Other-version availability gaps remain explicit.
