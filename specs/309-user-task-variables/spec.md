# Feature Specification: Display Effective User-Task Variables

**Feature Branch**: `codex/309-user-task-variables`

**Created**: 2026-09-15

**Status**: Draft

**Input**: [GitHub issue #309 — feat(cmd): add variable display to get user-task](https://github.com/grafvonb/c8volt/issues/309)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect Variables for Known Tasks (Priority: P1)

As an operator investigating a user task, I want to see the variables visible to that task alongside its details so I can understand the task's effective business context without manually resolving overlapping scopes.

**Why this priority**: Direct task inspection is the smallest useful workflow and establishes trustworthy variable visibility.

**Independent Test**: Retrieve known tasks with variables using single keys, multiple keys, and stdin. Compare each task's variables with its known effective values, including values spread across multiple pages and names shared by multiple scopes.

**Acceptance Scenarios**:

1. **Given** a visible task with variables, **When** the operator runs `c8volt get ut --key <key> --with-vars`, **Then** the task retains its ordinary details and its effective variables appear beneath it as nested name/value lines.
2. **Given** several visible tasks, **When** repeated or comma-separated keys or stdin keys are supplied with `--with-vars`, **Then** each returned task has its own variables, preserving the base command's input merging, ordering, and duplicate-key behavior.
3. **Given** the same variable name exists in several scopes, **When** the task's variables are displayed, **Then** that name appears once with the effective value selected by the backend, rather than every scope's copy.
4. **Given** more variables than fit in one retrieval page, **When** inspection completes, **Then** every effective variable is included, including variables after a sparse page with continuation available.
5. **Given** a task with no effective variables, **When** it is inspected with `--with-vars`, **Then** it remains a successful task result with the established empty-variable representation and no invented variable rows.
6. **Given** denied access, a disappeared task, or a variable retrieval failure on any page or task, **When** inspection cannot complete, **Then** the command reports the existing error and exit behavior and does not represent failed retrieval as an empty variable list or complete success.

---

### User Story 2 - Inspect Variables for a Bounded Search (Priority: P1)

As an operator reviewing a workload, I want variables for the tasks I choose to inspect so that limits, tenant selection, and interactive paging continue to control the work performed.

**Why this priority**: Search is the primary way to inspect tasks when their keys are unknown, and unnecessary retrieval makes bounded inspection unpredictable.

**Independent Test**: Search a known collection with filters, tenants, limits, and interactive paging. Verify variable retrieval occurs only for tasks included in the command result, including when the limit falls inside a page or the operator stops paging.

**Acceptance Scenarios**:

1. **Given** more than ten tasks assigned to Alice, **When** `c8volt get ut --assignee alice --limit 10 --with-vars` runs, **Then** at most ten matching tasks are returned with variables and no excluded task has its variables retrieved.
2. **Given** eligible interactive paging, **When** an operator continues or declines the next page, **Then** the existing paging behavior is preserved and variables are retrieved only for tasks included before completion or the accepted stop.
3. **Given** sparse task pages with continuation available, **When** search runs with `--with-vars`, **Then** discovery continues under the existing rules and an empty intermediate page causes no variable retrieval.
4. **Given** completed discovery with no matches, **When** the command runs with `--with-vars`, **Then** it makes zero variable requests, renders the established mode-specific empty result, and asks no paging question.
5. **Given** selected discovery tenants or authorized explicit keys outside that discovery scope, **When** variables are requested, **Then** base task-selection and authorization rules remain unchanged and each variable remains associated with its actual task and tenant context.
6. **Given** terminal stdin and eligible paging, **When** human or machine results are written to stdout, **Then** prompts use configured or inherited stderr and retain existing wording, defaults, EOF handling, and eligibility with redirected stdout; unattended execution requires no interactive response.

---

### User Story 3 - Read Predictable Human and Machine Results (Priority: P2)

As an operator or automation author, I want familiar variable formatting and explicit truncation information so I can distinguish readable previews from complete values without breaking existing scripts.

**Why this priority**: Output compatibility lets operators adopt the feature while preserving reliable automation and accurate interpretation of values.

**Independent Test**: Compare user-task variable output with established `get pi --with-vars` conventions using Unicode, structured values, long values, backend-truncated values, and empty collections across supported output modes.

**Acceptance Scenarios**:

1. **Given** variables containing structured values and Unicode characters, **When** human output uses `--var-value-limit 120`, **Then** formatting follows `get pi`, values longer than 120 characters are shortened without splitting a character, and client-side truncation is explicitly marked. Values at or below the limit are not marked as shortened.
2. **Given** `--with-vars` with no explicit value limit or with `--var-value-limit 0`, **When** human output renders, **Then** no display-side shortening occurs. Backend shortening is avoided where supported; any remaining backend-truncated value is explicitly identified as incomplete.
3. **Given** JSON output with `--with-vars`, including quiet combined with JSON, **When** retrieval succeeds, **Then** stdout contains exactly one shared successful envelope with task-plus-variable entries following the existing enrichment structure, received values unaffected by the human display limit, and backend truncation information preserved.
4. **Given** an explicitly negative value limit or an explicitly supplied limit without `--with-vars`, **When** input is validated, **Then** the command rejects it before retrieval using existing flag-error conventions; explicit zero also requires `--with-vars`.
5. **Given** ordinary execution without `--with-vars`, **When** results render, **Then** output is unchanged and there are zero variable requests.
6. **Given** effective keys-only or count output with `--with-vars`, **When** a valid command runs, **Then** there are zero variable requests and output retains exactly one task key per line or the existing numeric count line. Empty keys-only output contains zero bytes; zero count is `0` followed by a newline.
7. **Given** quiet human output or combined output flags, **When** the command runs, **Then** existing suppression, output precedence, and count-mode validation apply; quiet does not suppress explicitly requested JSON or keys-only results.
8. **Given** a supported backend version, **When** variables are requested, **Then** these workflows have the same observable contract across supported versions; unsupported versions retain a clear unsupported-operation error.

### Edge Cases

- An empty task collection differs from a returned task whose variable collection is empty; neither invents variable data or masks an error.
- Effective names shared across scopes appear once per task; the same name on different tasks may legitimately have different values.
- A task or variable set may change during retrieval. Failures remain failures, and this feature adds no snapshot-consistency promise.
- Variable pagination completes independently of task result limits. A task limit must not become a variable limit.
- Empty intermediate pages do not prove completion when continuation remains available.
- A value may be shortened by the backend, by the human display limit, or by both; the output distinguishes those cases. Unlimited display must not imply that a backend-shortened value is complete.
- Unicode characters, object and array values, empty strings, and null values follow existing variable display semantics.
- JSON takes precedence over keys-only under existing selection rules. Variable retrieval is skipped for the effective keys-only mode, not merely because a lower-precedence flag is present.
- A failure after earlier human output has been emitted must still produce the established failure outcome rather than a successful completion claim.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `get user-task` and its existing aliases MUST support opt-in `--with-vars` for explicit single and multiple keys, stdin keys, and list/search execution.
- **FR-002**: Each returned task MUST display each effective variable name once, with the value selected by the backend for that task. Displaying duplicate scope values or independently choosing a winning scope is excluded.
- **FR-003**: Successful variable retrieval MUST include all pages for each included task, continuing through sparse pages when continuation exists. Task limits MUST NOT truncate a task's variable collection.
- **FR-004**: Only tasks included in the command result may have variables retrieved. Existing task filters, limits, tenant scope, key authorization, ordering, and interactive paging decisions MUST remain unchanged.
- **FR-005**: There MUST be zero variable requests without `--with-vars`, for empty task results, or for effective keys-only or count output. These modes MUST retain their existing results and validation rules.
- **FR-006**: Human output MUST preserve ordinary task rows and summaries and nest variables using the existing `get pi --with-vars` ordering, name/value formatting, structured-value compaction, and truncation labels.
- **FR-007**: `--var-value-limit` MUST default to `0`, meaning unlimited human display. Positive values MUST limit displayed values by Unicode characters using existing ellipsis and truncation-marker conventions; the marker itself is outside the character limit. Negative values and any explicit use without `--with-vars` MUST be rejected before retrieval.
- **FR-008**: JSON MUST use exactly one shared command envelope and the established task-plus-variables enrichment shape, preserving received values and available variable identity, scope, tenant, and backend truncation information. Human display limits MUST NOT shorten JSON values. Existing omission and null-versus-empty-array conventions MUST be preserved rather than inventing a new empty representation.
- **FR-009**: Retrieval MUST avoid backend value shortening where supported and explicitly report any remaining backend truncation in human and JSON results. Unlimited display MUST never silently present a shortened value as complete.
- **FR-010**: Tasks without variables MUST remain successful results. Variable retrieval failures, including failures on later pages, MUST propagate through existing error and exit conventions and MUST NOT be substituted with empty variables or complete-success claims.
- **FR-011**: Default output, empty-result rendering, quiet behavior, output precedence, count output, automation behavior, and paging contracts MUST be preserved. Stdout MUST contain only selected results; interactive control text MUST use configured or inherited stderr.
- **FR-012**: Variable display MUST support the base command's supported Camunda versions, 8.8, 8.9, and 8.10, while preserving the clear unsupported-version behavior for 8.7.
- **FR-013**: User-facing help, command metadata, examples, README, and generated CLI documentation MUST consistently describe both flags, default and unlimited limits, human-versus-JSON behavior, and excluded output modes.
- **FR-014**: Scope MUST remain read-only variable display. Variable filtering (`--var`, `--var-exists`, `--var-like`), variable mutation, every-scope duplicate display, forms, audit logs, and task mutations are excluded.

### Key Entities

- **User task**: An existing task result with its identity, assignment, process, and tenant information, optionally accompanied by effective variables.
- **Effective variable**: A name and value visible to a particular task after backend scope resolution, with available identity and scope context and an indication of backend truncation.
- **Enriched task collection**: The selected task results paired with their complete retrieved variable collections, preserving the base result selection and output contract.
- **Display value limit**: A human presentation limit in characters, distinct from backend completeness and from the number of returned tasks or variables.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All four input paths—single key, multiple keys, stdin, and search—show the expected effective variables in acceptance checks, with zero duplicate names within a task and zero missing variables after completed pagination.
- **SC-002**: Across bounded search and paging checks, 100% of variable lookups belong to included tasks; excluded tasks and all non-enrichment modes cause zero variable lookups.
- **SC-003**: Every human-format acceptance case matches the established variable conventions, and 100% of shortened-value cases distinguish display shortening from backend incompleteness. Machine-readable values are unchanged by the display limit.
- **SC-004**: All output checks preserve their selected contract: one successful structured envelope, one key per keys-only line, zero-byte empty keys output, the existing count line, and zero prompt contamination of results.
- **SC-005**: Every no-variable case succeeds and every retrieval-failure case reports failure across all three supported versions; unsupported-version cases remain explicit.
- **SC-006**: Each of the four issue examples is supported and documented, enabling an operator to inspect keyed or filtered tasks and interpret value completeness directly from help and output.

## Assumptions

- Issue #309 is authoritative. Its architecture and engineering constraints remain binding for planning: use native effective-variable retrieval, preserve established layer ownership and version-specific adapters, reuse matching models and renderers, and avoid a generic enrichment framework or manual generated-client edits. Those implementation choices belong in the plan.
- This feature depends on the base command from [feature #308](../308-get-user-task/spec.md). Its exclusion of variables bounds that earlier feature; #309 explicitly adds that capability while preserving its other contracts.
- Existing `get pi --with-vars` behavior supplies unspecified formatting, ordering, enrichment shape, and empty-variable conventions. Task output retains task-specific metadata rather than adding unrelated process-instance display metadata.
- Existing authentication, permissions, and tenant policies apply. No new access policy or variable mutation capability is introduced.
- Acceptance collections are stable unless a scenario explicitly exercises concurrent change or failure; no new snapshot guarantee is introduced.
- Planning must carry forward issue #309's command, facade, and service coverage across supported versions, real-terminal paging checks with separate stdout/stderr capture, CLI documentation regeneration with `make docs-content`, and formatting of touched Go files. Validation remains proportional under the constitution; this specification-only change requires lightweight document checks.
