# Feature Specification: Tasklist V1 Fallback For Task-Key Process-Instance Lookup

**Feature Branch**: `162-tasklist-v1-fallback`  
**Created**: 2026-05-03  
**Status**: Draft  
**Input**: User description: "GitHub issue #162: feat(get pi): fall back to Tasklist V1 for user-task process-instance lookup"

## GitHub Issue Traceability

- **Issue Number**: 162
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/162
- **Issue Title**: feat(get pi): fall back to Tasklist V1 for user-task process-instance lookup
- **Related Issue**: #152 shipped the original v2-only `--has-user-tasks` lookup and is superseded only for the fallback behavior described here.

## Clarifications

### Session 2026-05-03

- No critical ambiguities detected worth formal clarification. The GitHub issue defines the affected command, supported versions, primary-vs-fallback lookup order, fallback eligibility, error handling, documentation updates, and required test coverage.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Resolve Legacy Task Keys Through Fallback (Priority: P1)

As a CLI operator who has a legacy job-worker-based user task key, I want `get process-instance` / `get pi --has-user-tasks=<task-key>` to find the owning process instance when the primary user-task lookup cannot see the task, so that I can inspect the process instance without a manual Tasklist lookup.

**Why this priority**: This is the new value requested by the issue and closes the gap left by the v2-only implementation for legacy user tasks.

**Independent Test**: Run `get pi --has-user-tasks=<task-key>` on Camunda 8.8 and 8.9 fixtures where the primary user-task lookup returns no match and the fallback lookup returns a task with a known process-instance key, then verify the rendered result matches direct keyed process-instance lookup for that key.

**Acceptance Scenarios**:

1. **Given** a Camunda 8.9 legacy user task is absent from the primary user-task lookup but present in the fallback task lookup with process instance `P`, **When** the operator runs `c8volt get pi --has-user-tasks=<task-key>`, **Then** the command returns the same human-readable process-instance output as `c8volt get pi --key=P`.
2. **Given** a Camunda 8.8 legacy user task is absent from the primary user-task lookup but present in the fallback task lookup with process instance `P`, **When** the operator runs `c8volt get process-instance --has-user-tasks=<task-key>`, **Then** the command returns the owning process instance using the existing single-instance lookup behavior.
3. **Given** the fallback resolves a task key and the operator requests JSON output, **When** `c8volt get pi --has-user-tasks=<task-key> --json` runs, **Then** the JSON shape matches direct keyed process-instance lookup for the resolved process instance.

---

### User Story 2 - Preserve Primary Lookup As First Choice (Priority: P2)

As a CLI operator using modern Camunda user tasks, I want existing v2 task-key lookup behavior to remain the first and fastest path so that the fallback does not change successful current workflows.

**Why this priority**: The fallback is transitional compatibility, not a replacement for the primary supported user-task lookup path.

**Independent Test**: Run `get pi --has-user-tasks=<task-key>` against 8.8 and 8.9 fixtures where the primary lookup resolves the task, then verify the fallback endpoint is not called and output remains unchanged.

**Acceptance Scenarios**:

1. **Given** a Camunda 8.9 user task is visible to the primary user-task lookup, **When** the operator runs `c8volt get pi --has-user-tasks=<task-key>`, **Then** the command resolves through the primary lookup and does not call the fallback.
2. **Given** a Camunda 8.8 user task is visible to the primary user-task lookup, **When** the operator runs `c8volt get pi --has-user-tasks=<task-key>`, **Then** the command resolves through the primary lookup and does not call the fallback.
3. **Given** multiple `--has-user-tasks` values include both primary-resolvable and fallback-resolvable tasks, **When** lookup runs, **Then** each task uses the first successful path for that task and the final output follows existing multi-key rendering behavior.

---

### User Story 3 - Fail Clearly When Neither Lookup Resolves (Priority: P3)

As a CLI user or script author, I want absent task keys and fallback failures to produce clear, consistent errors so that automation can distinguish not found from configuration or service failures.

**Why this priority**: The fallback must not hide real operational failures or make scripts interpret auth, server, or malformed responses as missing tasks.

**Independent Test**: Run task-key lookup cases where both lookup paths miss, where the fallback task has no process-instance key, and where fallback configuration or service errors occur, then verify each outcome matches the expected error class.

**Acceptance Scenarios**:

1. **Given** a task key is absent from both lookup paths, **When** the operator runs `c8volt get pi --has-user-tasks=<task-key>`, **Then** the command returns the existing not-found style error.
2. **Given** the fallback returns a task without a usable process-instance key, **When** lookup runs, **Then** the command fails with a clear resolution error instead of rendering incomplete process-instance data.
3. **Given** the fallback returns an authentication, authorization, configuration, malformed response, network, or server failure, **When** lookup runs, **Then** the command surfaces that failure and does not collapse it into not found.

---

### User Story 4 - Keep Version And Documentation Contract Accurate (Priority: P4)

As a CLI user reading help or documentation, I want the task-key lookup behavior to explain the v2-first strategy, fallback support, deprecation caveat, and version limits so that I can choose the command with correct expectations.

**Why this priority**: Existing documentation says there is no Tasklist fallback, and that user-facing contract must be updated with the behavior change.

**Independent Test**: Inspect help text, README examples, and generated CLI docs to verify fallback behavior and version limits are accurately documented.

**Acceptance Scenarios**:

1. **Given** a user views help for `get process-instance` or `get pi`, **When** `--has-user-tasks` is listed, **Then** the help text describes v2-first lookup with Tasklist V1 fallback for 8.8 and 8.9.
2. **Given** generated CLI docs are refreshed, **When** the affected command pages are inspected, **Then** they no longer claim that no Tasklist fallback exists.
3. **Given** a user reads user-facing documentation, **When** task-key lookup is described, **Then** it states that the fallback exists for legacy user-task compatibility and depends on deprecated Tasklist V1 behavior.
4. **Given** the active Camunda version is 8.7, **When** the operator runs `c8volt get pi --has-user-tasks=<task-key>`, **Then** the command still fails with the existing explicit unsupported-version error before attempting either lookup path.

### Edge Cases

- A task found by the primary lookup must not trigger fallback lookup.
- A task missing from the primary lookup but found by fallback lookup must continue through existing process-instance keyed lookup and rendering.
- A task missing from both lookup paths must return the same not-found style outcome expected by current task-key lookup.
- A fallback task without a usable process-instance key must return a clear resolution error.
- Fallback authentication, authorization, configuration, malformed response, network, and server errors must not be masked as not found.
- Camunda 8.7 must remain unsupported for `--has-user-tasks` and must not attempt fallback lookup.
- Existing selector conflicts for `--has-user-tasks` with `--key`, stdin key input, search filters, `--total`, or `--limit` must remain unchanged.
- Existing render flags valid for direct single process-instance lookup must continue to behave the same after fallback resolution.
- Tenant visibility must not be broadened beyond the configured task and process-instance lookup context.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: On Camunda 8.9, the system MUST attempt primary user-task lookup before fallback lookup for every `--has-user-tasks` task key.
- **FR-002**: On Camunda 8.8, the system MUST attempt primary user-task lookup before fallback lookup for every `--has-user-tasks` task key.
- **FR-003**: The system MUST call the Tasklist V1 fallback only when the primary user-task lookup returns a not-found or empty-result outcome for that task key.
- **FR-004**: The system MUST NOT call the Tasklist V1 fallback when the primary user-task lookup successfully resolves a task.
- **FR-005**: The system MUST NOT treat primary lookup authentication, authorization, configuration, malformed response, network, or server failures as fallback-eligible not-found outcomes.
- **FR-006**: The system MUST resolve the owning process-instance key from a fallback task result when the fallback succeeds.
- **FR-007**: After resolving the owning process-instance key through fallback, the system MUST reuse the existing process-instance lookup and rendering path.
- **FR-008**: Human-readable fallback output MUST match direct keyed process-instance lookup as closely as possible for the resolved key.
- **FR-009**: JSON fallback output MUST match the existing JSON shape for direct keyed process-instance lookup.
- **FR-010**: If both primary and fallback lookups miss, the system MUST return a clear not-found style error.
- **FR-011**: If fallback lookup returns a task without a usable process-instance key, the system MUST return a clear resolution error.
- **FR-012**: Fallback authentication, authorization, configuration, malformed response, network, and server failures MUST be surfaced without being converted to not found.
- **FR-013**: Camunda 8.7 MUST remain explicitly unsupported for `--has-user-tasks`.
- **FR-014**: Existing `--has-user-tasks` selector conflict validation MUST remain unchanged.
- **FR-015**: Repeated `--has-user-tasks` values MUST resolve independently, allowing one task to resolve through primary lookup and another through fallback in the same command.
- **FR-016**: Tenant-aware behavior MUST remain consistent with the configured task and process-instance lookup context.
- **FR-017**: Help text MUST describe primary lookup followed by Tasklist V1 fallback for Camunda 8.8 and 8.9.
- **FR-018**: User-facing documentation MUST describe the fallback as legacy compatibility for deprecated Tasklist V1 behavior.
- **FR-019**: Generated CLI documentation MUST be refreshed when command help text changes.
- **FR-020**: Automated tests MUST cover primary success without fallback on Camunda 8.9.
- **FR-021**: Automated tests MUST cover primary success without fallback on Camunda 8.8.
- **FR-022**: Automated tests MUST cover primary miss followed by fallback success on Camunda 8.9.
- **FR-023**: Automated tests MUST cover primary miss followed by fallback success on Camunda 8.8.
- **FR-024**: Automated tests MUST cover both-lookups-missing not-found behavior.
- **FR-025**: Automated tests MUST cover fallback task results without a usable process-instance key.
- **FR-026**: Automated tests MUST cover fallback auth, config, or server failures being surfaced distinctly from not found.
- **FR-027**: Automated tests MUST cover Camunda 8.7 remaining unsupported without fallback.
- **FR-028**: Automated tests MUST cover updated help text or user-facing command contract changes where relevant.

### Key Entities *(include if feature involves data)*

- **Task-Key Lookup Request**: A process-instance lookup invocation that starts from one or more `--has-user-tasks` values rather than a process-instance key.
- **Primary User Task Lookup**: The current Camunda v2 user-task lookup path for modern Camunda user tasks.
- **Fallback Task Lookup**: The Tasklist V1 lookup path used only after the primary lookup cannot find a task key.
- **Fallback Task Result**: A task result from the fallback path that can provide the owning process-instance key.
- **Owning Process Instance Key**: The process-instance identifier extracted from either lookup path and passed into existing process-instance lookup behavior.
- **Fallback-Eligible Miss**: A primary lookup outcome that means the task key was not found or not visible, and therefore may be retried through fallback.
- **Unsupported Version Error**: The explicit failure returned when `--has-user-tasks` is used against Camunda 8.7.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show Camunda 8.9 primary task lookup success does not call fallback lookup.
- **SC-002**: Automated tests show Camunda 8.8 primary task lookup success does not call fallback lookup.
- **SC-003**: Automated tests show Camunda 8.9 primary miss followed by fallback success returns the owning process instance.
- **SC-004**: Automated tests show Camunda 8.8 primary miss followed by fallback success returns the owning process instance.
- **SC-005**: Automated tests show a task key absent from both lookup paths returns the existing not-found style error.
- **SC-006**: Automated tests show fallback task results without usable process-instance keys fail clearly.
- **SC-007**: Automated tests show fallback auth, config, malformed response, network, or server failures are not reported as not found.
- **SC-008**: Automated tests show Camunda 8.7 still fails with an explicit unsupported-version message before fallback is attempted.
- **SC-009**: Help output and generated docs describe v2-first lookup, Tasklist V1 fallback, 8.8/8.9 support, and 8.7 unsupported behavior.
- **SC-010**: Repository validation passes with the closest relevant targeted tests and the broader project test command required by the constitution.

## Assumptions

- The affected command surface remains `get process-instance` and its `get pi` alias with the existing `--has-user-tasks` flag.
- The primary lookup path from #152 remains the preferred path for modern Camunda user tasks.
- Tasklist V1 fallback is transitional compatibility for legacy job-worker-based user tasks and should not become the primary lookup mechanism.
- Existing configuration already distinguishes the Camunda API and Tasklist API endpoints sufficiently for the fallback path.
- Existing authentication wiring can supply credentials for Tasklist API calls when the profile is configured for them.
- Existing process-instance lookup remains the source of truth for rendering, process-instance not-found behavior, and process-instance API errors after task ownership is resolved.
- Documentation generated from command metadata should be regenerated rather than hand-edited when command metadata changes.
