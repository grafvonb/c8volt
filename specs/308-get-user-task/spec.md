# Feature Specification: Get User Tasks by Key or Search

**Feature Branch**: `codex/308-get-user-task`

**Created**: 2026-09-13

**Status**: Draft

**Input**: [GitHub issue #308 — feat(cmd): add get user-task with keyed lookup and paginated search](https://github.com/grafvonb/c8volt/issues/308)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect Known User Tasks (Priority: P1)

As an operator with one or more user-task keys, I want to retrieve those tasks directly so I can inspect their state, assignment, owning process instance, and tenant without searching manually.

**Why this priority**: Direct inspection is the smallest useful workflow and enables both troubleshooting and composition with existing commands.

**Independent Test**: Retrieve known tasks using each supported key input form on each supported version, then request a missing key and verify strict failure.

**Acceptance Scenarios**:

1. **Given** a visible task, **When** the operator runs `c8volt get user-task --key <key>` or uses `user-tasks`, `ut`, or `uts`, **Then** each invocation returns the same task and its identifying details.
2. **Given** several visible tasks, **When** keys are supplied with repeated or comma-separated `--key`/`-k` values, stdin, or stdin with the optional `-` argument, **Then** the command follows `get pi` input, merging, ordering, and duplicate-handling conventions and returns the requested tasks.
3. **Given** a key that does not exist, including one among otherwise valid keys, **When** direct lookup runs, **Then** the command reports the established not-found error and does not represent an incomplete lookup as successful.
4. **Given** an explicit key for a task outside the selected discovery tenant that the caller is authorized to read, **When** lookup runs with `--tenant`, **Then** backend authorization determines access and the returned task retains its actual tenant metadata.
5. **Given** explicit keys and a search filter, `--limit`, or `--total`, **When** the command validates the request, **Then** it rejects the conflicting combination using the established invalid-input contract.

---

### User Story 2 - Discover and Count Matching User Tasks (Priority: P1)

As an operator investigating a workload, I want to search user tasks by process, task state, assignment, candidate, and tenant, then retrieve a bounded set or count all matches so I can understand the work requiring attention.

**Why this priority**: Discovery makes the command useful when operators do not already know task keys; reliable limits and counts support workload assessment.

**Independent Test**: Search a known collection with individually tested and combined filters, multiple pages, sparse intermediate pages, capped reported totals, and selected tenants. Compare returned tasks and counts with the known matching collection.

**Acceptance Scenarios**:

1. **Given** tasks differing in process-instance key, process-definition key, BPMN process ID, element ID, state, assignee, candidate user, and candidate group, **When** each corresponding filter is used alone or in combination, **Then** only tasks matching all supplied criteria are returned, with exact matching for assignee and candidate values.
2. **Given** a supported task state, **When** its name is supplied in different letter cases, **Then** the same state is selected; omitting `--state` or supplying `all` imposes no state restriction. `assigned` is rejected because assignment is an assignee property.
3. **Given** more matches than fit in one page, **When** the operator sets a batch size and a result limit, **Then** the batch size controls each retrieval page and the total number of returned tasks never exceeds the limit.
4. **Given** a sparse or empty intermediate page with continuation available, **When** discovery proceeds, **Then** later matches are still retrieved until the limit, an eligible operator stop, or authoritative completion is reached.
5. **Given** a selected tenant, **When** search or counting runs, **Then** only matching tasks in the effective discovery tenant scope contribute to results or the count.
6. **Given** an exact or capped reported total, **When** `--total` runs, **Then** it emits only the exact matching count, continuing discovery when needed to resolve a capped total. Zero matches produce `0` followed by a newline.
7. **Given** no matches after completed discovery, **When** ordinary search runs, **Then** it succeeds with the selected mode's empty-result representation and does not ask a paging question.
8. **Given** malformed keys, an invalid state, an invalid batch size or explicit limit, or `--total` combined with `--limit`, `--json`, or `--keys-only`, **When** the command validates input, **Then** it rejects the request with established errors before task retrieval.

---

### User Story 3 - Use Results Interactively and in Automation (Priority: P2)

As an operator or automation author, I want predictable task output and paging behavior so I can inspect tasks at a terminal or pass results to another tool without control text corrupting them.

**Why this priority**: Consistent output and interaction make the lookup and discovery workflows safe to compose with existing operational scripts.

**Independent Test**: Exercise command execution with nonempty and empty results across human, JSON, keys-only, quiet, and automation modes and supported combinations. Exercise paging with real terminal stdin while capturing stdout and stderr separately.

**Acceptance Scenarios**:

1. **Given** returned tasks, including unassigned tasks and tasks without a display name, **When** human output is selected, **Then** compact rows show task key, tenant, element ID, state, optional labelled name, then BPMN process ID and labelled process-instance, element-instance, and process-definition keys, with assignee always last and shown as `assignee:<unassigned>` when empty, using established get-command ordering and omission conventions.
2. **Given** JSON output, including quiet combined with JSON, **When** a lookup or search succeeds, **Then** stdout contains exactly one successful shared command envelope with stable task fields, and no prompt or diagnostic text. Empty search preserves the established empty collection and omission conventions.
3. **Given** keys-only output, including quiet combined with keys-only, **When** tasks are returned, **Then** stdout contains exactly one task key per line; an empty search produces zero bytes.
4. **Given** quiet human output or automation mode, **When** the command executes, **Then** established result suppression and unattended execution rules apply; explicitly requested machine output remains available and automation requires no interactive response.
5. **Given** paging is eligible under existing get-command rules, **When** another page is offered, **Then** the plain prompt uses the configured or inherited stderr destination, while stdout contains only results. Accepting, declining, and end-of-input follow the established paging behavior.
6. **Given** terminal stdin with redirected or piped stdout, **When** paging eligibility is evaluated, **Then** the established get-command eligibility rules still apply; terminal stdin alone does not change them.
7. **Given** Camunda 8.8, 8.9, or 8.10, **When** the command reads tasks, **Then** the supported workflows work; Camunda 8.7 produces a clear unsupported-version error.
8. **Given** a backend failure, denied access, or failure after an earlier page, **When** the command cannot complete the requested read, **Then** it follows established error and exit behavior and does not report successful empty or complete results.
9. **Given** an existing `get pi --has-user-tasks` workflow, **When** the new command is introduced, **Then** existing process-instance resolution, tenant scoping, and Tasklist fallback behavior remain unchanged.

### Edge Cases

- Duplicate keys, whitespace, malformed stdin, empty stdin, and the optional `-` follow existing `get pi` conventions; extra positional arguments are rejected.
- An explicit key remains an authorized direct lookup even when discovery tenant settings differ; authorization failures are not converted into empty results.
- A task may be unassigned or lack a display name; these conditions do not prevent rendering its identity and owning process instance.
- A result limit can fall within a page or exceed the number of available matches. Omitting the limit allows continuation through all matches under existing paging rules.
- Sparse intermediate pages and capped totals cannot prove that discovery is complete. Retrieval failures and operator stops cannot be presented as an exact full count.
- Empty search is distinct from a missing explicit key. Empty output adds neither synthetic tasks nor extra requests for rendering.
- JSON takes precedence over keys-only when both output flags are supplied, following current shared output selection. Total-mode conflicts are rejected rather than resolved by output precedence.
- Prompts remain separate from keys-only results even with terminal stdin and an overridden or inherited stderr writer.
- Task states are validated for the selected supported version; assignment does not introduce a new lifecycle state.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provide the read-only `c8volt get user-task` command with aliases `user-tasks`, `ut`, and `uts` and the existing get-command help and capability conventions.
- **FR-002**: Accept repeated and comma-separated `--key`/`-k`, stdin keys, and optional `-` with the same parsing, validation, merging, and duplicate behavior as `get pi`.
- **FR-003**: Direct lookup MUST be strict: any missing requested task produces a not-found error. Explicit keys MUST use backend authorization without discovery tenant filtering and MUST preserve returned tenant metadata.
- **FR-004**: Search MUST support `--pi-key`, `--pd-key`, `--bpmn-process-id`/`-b`, `--element-id`, `--state`/`-s`, `--assignee`, `--candidate-user`, and `--candidate-group`. Supplied criteria MUST be combined and applied by the backend; assignment and candidate filters MUST use exact matching.
- **FR-005**: State filtering MUST be case-insensitive, default to `all`, and accept only states supported by the selected version's task contract. `all` means no state restriction; `assigned` MUST NOT be accepted as a state.
- **FR-006**: Inherited tenant settings and `--tenant` MUST scope search and counting according to existing discovery rules.
- **FR-007**: Support `--batch-size`/`-n` for page size and `--limit`/`-l` for the maximum tasks returned across all pages. Validate bounds using existing get-command conventions; an explicitly supplied limit MUST be positive.
- **FR-008**: Discovery MUST continue across sparse pages when continuation remains available, respecting result limits and established interactive stop behavior. Rendering MUST NOT add discovery requests.
- **FR-009**: `--total` MUST return the exact matching count as a numeric line, including zero, paging when reported totals are capped. It MUST NOT report a partial count as exact.
- **FR-010**: Reject explicit keys from any supported source combined with search filters, `--limit`, or `--total`. Reject `--total` with `--limit`, `--json`, or `--keys-only`, following the existing `get pi` count-mode convention. Tenant selection remains governed by FR-003 rather than treated as a keyed-mode conflict.
- **FR-011**: Human output MUST follow get-command ordering: task key, tenant, element ID, state, optional `name:` details, then BPMN process ID and related `pi:`/`ei:`/`pd:` keys. Assignee MUST be the last human field and show `assignee:<unassigned>` when empty. Display name MUST NOT replace element ID; empty optional fields are omitted. Default output MUST avoid per-request and per-page diagnostic details.
- **FR-012**: JSON lookup and search output MUST use stable public task fields in exactly one shared command envelope. Keys-only MUST print one task key per line and no other text. Shared output precedence MUST be preserved.
- **FR-013**: A completed empty search MUST succeed through mode-aware rendering: one successful JSON envelope with the established empty payload conventions, zero-byte keys-only output, and the established human empty-result style. Quiet MUST suppress informational human summaries without suppressing explicitly requested machine results.
- **FR-014**: Reuse inherited verbosity, output, quiet, and automation behavior. Paging eligibility, prompt wording, default answers, EOF handling, and stop behavior MUST match existing get commands. All interactive control text MUST use configured or inherited stderr; stdout MUST contain only command results.
- **FR-015**: Support task reads on Camunda 8.8, 8.9, and 8.10 and return a clear unsupported-version error on 8.7. Preserve established invalid-input, not-found, authorization, backend-failure, and exit-code conventions.
- **FR-016**: Preserve all existing `get pi --has-user-tasks` resolution, tenant scoping, and Tasklist fallback behavior. The new task-read workflow MUST use native task reads without expanding that fallback to this command.
- **FR-017**: Acceptance validation MUST exercise command execution for key forms, filters, conflicts, tenants, paging, limits, exact totals, all output modes and supported combinations, empty results, errors, and supported versions. Interactive checks MUST use real terminal stdin and separately captured stdout/stderr, including configured and inherited stderr destinations. Existing task-to-process resolution MUST retain regression coverage.
- **FR-018**: Help, aliases, command metadata, examples, README, and generated CLI documentation MUST describe the supported behavior and limitations consistently, including input forms, version support, tenant handling, output, paging, and count-mode conflicts.
- **FR-019**: The feature MUST remain limited to lookup, search, and count. Task mutations, variables, forms, audit logs, date filters, custom sorting, and watch mode are excluded.

### Key Entities

- **User task**: A human-work item identified by a task key, with lifecycle state, name or BPMN element ID, optional assignee, candidate users/groups, owning process-instance and process-definition identifiers, BPMN process ID, and tenant metadata.
- **Task selection**: Either a set of explicit task keys or combined discovery criteria with an effective tenant scope; explicit-key and filtered-search selection are mutually exclusive.
- **Task result collection**: The tasks returned for a lookup or search, constrained by the requested result limit and carrying the established output contract.
- **Matching count**: The exact number of tasks satisfying discovery criteria, distinguished from a capped reported total or an incomplete traversal.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All four command names and every supported key input form retrieve the expected tasks in acceptance checks; every missing-key case reports failure rather than incomplete success.
- **SC-002**: Across known test collections, 100% of returned tasks satisfy the selected filters and tenant scope, zero expected matches are lost to sparse pages during completed discovery, and zero result sets exceed the requested limit.
- **SC-003**: Every completed count equals the known matching population, including zero matches and populations larger than the reported total cap.
- **SC-004**: All output-mode acceptance cases produce the expected result content: exactly one successful JSON envelope, one key per keys-only line, zero keys-only bytes for empty results, and zero control-text contamination on stdout.
- **SC-005**: All eligible paging cases allow continuation or stopping with the established interaction behavior, and all automation cases complete without interactive input.
- **SC-006**: The full acceptance matrix passes for all three supported versions; all 8.7 cases clearly report unsupported operation, and existing task-to-process lookup scenarios retain their outcomes.
- **SC-007**: Every documented command example corresponds to a supported, validated workflow, allowing operators to perform keyed inspection, filtered discovery, and exact counting directly from help.

## Assumptions

- Issue #308 is authoritative. Its architecture, package ownership, reuse, generated-client, formatting, and validation instructions remain binding planning constraints in the linked issue; this specification states the user-visible contract.
- Existing `get process-instance` and `get element` behavior provides defaults for unspecified formatting, paging eligibility, limits, empty-result wording, error categories, and inherited flags. Count-mode conflicts follow `get pi`, including rejecting `--total` with JSON, keys-only, or a limit.
- Existing authentication and permissions are reused. This feature introduces no new permissions or tenant policy.
- With neither explicit keys nor search filters, the command lists tasks within the effective discovery tenant scope under the normal paging rules.
- Exact-count acceptance uses a stable collection. No new snapshot-consistency guarantee for tasks changing during discovery is introduced.
- Native task reads depend on the selected supported Camunda version. Tasklist-only task discovery is outside this command's scope; the existing process-instance lookup fallback remains intact.
- Planning will retain the issue's required command execution and real-terminal checks, documentation regeneration with `make docs-content`, formatting of touched Go files, targeted tests, and validation proportional to the changed behavior under constitution v2.0.0. The integrated feature warrants `make test` because it changes shared service/facade contracts and concurrent bulk execution; documentation-only changes and adequately covered focused slices do not require the full suite.
