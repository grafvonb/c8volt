# Feature Specification: Process Instance Variable Output

**Feature Branch**: `173-pi-with-vars`  
**Created**: 2026-05-05  
**Status**: Draft  
**Input**: User description: "GitHub issue #173: feat(pi): add --with-vars process-instance inspection output"

## GitHub Issue Traceability

- **Issue Number**: 173
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/173
- **Issue Title**: feat(pi): add --with-vars process-instance inspection output

## Clarifications

### Session 2026-05-05

- Q: What human variable value limit behavior should `--with-vars` use? -> A: No default CLI limit; only shorten when a user passes `--var-value-limit`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect Process Variables (Priority: P1)

As an operator, I want `c8volt get pi --key <key> --with-vars` to show variables defined directly on the selected process instance so that I can inspect runtime data without switching tools.

**Why this priority**: Direct process-instance variable inspection is the core requested behavior and is the smallest independently useful slice.

**Independent Test**: Run `c8volt get pi --key <process-instance-key> --with-vars` against a process instance with process-scope variables and verify the normal process-instance row is followed by sorted, indented variable lines.

**Acceptance Scenarios**:

1. **Given** a process instance has process-scope variables, **When** the user runs `c8volt get pi --key <key> --with-vars`, **Then** the process-instance row is shown with its variables below it.
2. **Given** returned variables have names in mixed order, **When** human output renders them, **Then** variables appear sorted by name ascending.
3. **Given** a returned variable belongs to a task, subprocess, gateway, event, or other element scope, **When** `--with-vars` output is rendered, **Then** that variable is excluded from the process-instance variable list.
4. **Given** the user provides multiple `--key` values, **When** `--with-vars` is enabled, **Then** each process instance shows only its own process-scope variables.

---

### User Story 2 - Keep Large Values Usable (Priority: P2)

As an operator, I want long variable values to be compact and clearly marked when shortened so that process-instance output stays readable while still warning me when displayed data is incomplete.

**Why this priority**: Variable values can be large JSON payloads, so readability and explicit truncation markers are required before the feature is operationally safe for human output.

**Independent Test**: Run `c8volt get pi --key <key> --with-vars` with compact JSON values, large values, API-truncated values, and an explicit value limit, then verify human output remains one-line, applies CLI shortening only when requested, and marks truncation source precisely.

**Acceptance Scenarios**:

1. **Given** a variable value is JSON-like but contains whitespace or line breaks, **When** human output renders it, **Then** the displayed value is compacted to one line.
2. **Given** a received variable value exceeds an explicit `--var-value-limit <chars>`, **When** human output renders it, **Then** the displayed value is shortened and marked `cli-truncated`.
3. **Given** Camunda marks a returned variable value as truncated, **When** human output renders it, **Then** the variable line is marked `api-truncated`.
4. **Given** both Camunda and c8volt truncate a value, **When** human output renders it, **Then** the variable line is marked `api-truncated,cli-truncated`.
5. **Given** the user does not pass `--var-value-limit`, **When** human output renders variables, **Then** c8volt does not shorten received values for terminal display while still preserving API truncation markers.

---

### User Story 3 - Provide Enriched JSON Output (Priority: P3)

As an automation user, I want `--with-vars --json` to include variables in a stable enriched structure so that scripts can consume process instances and variables together.

**Why this priority**: Machine-readable output is part of the requested command contract, but it depends on the variable lookup and association behavior from the first story.

**Independent Test**: Run `c8volt get pi --key <key> --with-vars --json` and verify each returned process instance includes sorted process-scope variables with value and metadata preserved from the API response.

**Acceptance Scenarios**:

1. **Given** a keyed process-instance lookup includes variables, **When** the user runs `c8volt get pi --key <key> --with-vars --json`, **Then** JSON output contains the process instance and its sorted variables.
2. **Given** a variable value exceeds any human display limit, **When** JSON output is requested, **Then** the JSON keeps the received API value intact.
3. **Given** variable metadata is available, **When** JSON output is requested, **Then** each variable includes at least name, value, variable key, process instance key, scope key, tenant ID, and API truncation state when available.

### Edge Cases

- A keyed process instance has no process-scope variables.
- Multiple keyed process instances have disjoint variable sets.
- The variable search response includes element-scoped variables with the same process instance key but a different scope key.
- Variable names differ only by case or include punctuation.
- Variable values are strings, numbers, booleans, null-like values, JSON objects, JSON arrays, or opaque non-JSON text.
- JSON-like values contain whitespace, newlines, or already compact one-line payloads.
- A variable value is empty.
- Camunda reports a variable value as API-truncated.
- c8volt shortens a received value only for human output.
- Both API and CLI truncation apply to the same variable line.
- The user leaves `--var-value-limit` unset and receives full returned values in human output.
- JSON output is requested with or without a human value display limit.
- Variable lookup fails after the process instance is fetched.
- Existing `get pi` behavior without `--with-vars` remains unchanged.
- `--with-vars` is not part of the first iteration for process-instance search by variable value, element-scoped variables, a standalone variable command, or multi-line pretty JSON.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add `--with-vars` to keyed process-instance inspection.
- **FR-002**: The system MUST support `--with-vars` with one or multiple `--key` values.
- **FR-003**: The system MUST fetch each selected process instance normally before variable enrichment.
- **FR-004**: The system MUST search variables through the Camunda Orchestration Cluster API variable search capability.
- **FR-005**: Variable lookup MUST filter by `processInstanceKey` equal to the selected process-instance key.
- **FR-006**: Variable lookup MUST filter by `scopeKey` equal to the selected process-instance key.
- **FR-007**: Output MUST include only variables defined directly at the process-instance scope.
- **FR-008**: Output MUST exclude local task, subprocess, gateway, event, and other element-scoped variables.
- **FR-009**: Variables MUST be sorted by name ascending before rendering.
- **FR-010**: Human output MUST preserve the normal process-instance list row before rendering variables.
- **FR-011**: Human output MUST render variables as indented lines below their matching process instance.
- **FR-012**: Human variable lines MUST NOT repeat the word `var` on each line.
- **FR-013**: Human output MUST render JSON-like variable values compactly on one line by default.
- **FR-014**: Human output MUST NOT shorten received variable values by default.
- **FR-015**: Human output MUST provide `--var-value-limit <chars>` to shorten received variable values for terminal display when requested.
- **FR-016**: Human output MUST mark Camunda-shortened values as `api-truncated`.
- **FR-017**: Human output MUST mark c8volt display-shortened values as `cli-truncated`.
- **FR-018**: Human output MUST mark values affected by both sources as `api-truncated,cli-truncated`.
- **FR-019**: Human output MUST NOT use the ambiguous truncation label `truncated` for variable values.
- **FR-020**: JSON output with `--with-vars` MUST keep the received API value intact.
- **FR-021**: JSON output with `--with-vars` MUST include each process instance and its sorted variables in a stable enriched structure.
- **FR-022**: JSON variable metadata MUST include at least name, value, variable key, process instance key, scope key, tenant ID, and API truncation state when available.
- **FR-023**: Variable enrichment MUST preserve per-process-instance association for multiple keyed lookups.
- **FR-024**: Variable lookup failures MUST fail the command clearly rather than rendering partially enriched output.
- **FR-025**: Existing output and behavior without `--with-vars` MUST remain unchanged.
- **FR-026**: Automated tests MUST cover human output with sorted process-scope variables.
- **FR-027**: Automated tests MUST cover exclusion of element-scoped variables.
- **FR-028**: Automated tests MUST cover human value compaction and CLI truncation markers.
- **FR-029**: Automated tests MUST cover API truncation markers and combined truncation markers.
- **FR-030**: Automated tests MUST cover JSON enriched output preserving received API values and metadata.

### Key Entities *(include if feature involves data)*

- **Process Instance Result**: A keyed process-instance row selected by the user before optional variable enrichment.
- **Process Instance Variable**: A variable whose process instance key and scope key both match the selected process-instance key.
- **Element-Scoped Variable**: A variable associated with the same process instance but scoped to a local element; excluded from this feature's output.
- **Variable Value Display**: The human-rendered value text after optional JSON compaction and CLI display shortening.
- **Variable Truncation State**: The explicit source markers that describe whether the API, CLI display layer, or both shortened a variable value.
- **Variable-Enriched Process Instance**: A stable output shape that associates one process instance with its sorted variables.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show `c8volt get pi --key <key> --with-vars` renders process-scope variables below the selected process instance.
- **SC-002**: Automated tests show variables are sorted by name ascending.
- **SC-003**: Automated tests show variables where `scopeKey` differs from the process-instance key are excluded.
- **SC-004**: Automated tests show multiple keyed lookups preserve per-process-instance variable association.
- **SC-005**: Automated tests show human output compacts JSON-like values to one line.
- **SC-006**: Automated tests show human output does not shorten received values by default.
- **SC-007**: Automated tests show API-shortened values are marked `api-truncated`.
- **SC-008**: Automated tests show values shortened by both sources are marked `api-truncated,cli-truncated`.
- **SC-009**: Automated tests show `--var-value-limit <chars>` shortens received values and marks them as `cli-truncated`.
- **SC-010**: Automated tests show JSON output keeps received API values intact and includes required metadata.
- **SC-011**: Existing process-instance get command tests pass without `--with-vars` behavior changes.
- **SC-012**: Relevant command, facade, service, documentation generation, and repository validation checks pass.

## Assumptions

- The affected command surface is `c8volt get process-instance` and its alias `c8volt get pi`.
- `--with-vars` is limited to keyed process-instance lookup in the first iteration.
- Variable lookup should reuse existing process-instance command selection, rendering, facade, options, error, and JSON patterns where practical.
- The explicit human display limit flag should be named `--var-value-limit` unless planning finds a direct conflict with existing repository conventions.
- The generated Camunda client may need a small repository-consistent fix if the current variable search model lacks value or truncation metadata.
- User-facing help, README examples, and generated CLI documentation need updates because the command surface changes.
