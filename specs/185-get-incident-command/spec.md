# Feature Specification: Get Incident Command

**Feature Branch**: `185-get-incident-command`
**Created**: 2026-05-09
**Status**: Draft
**Input**: GitHub issue [#185](https://github.com/grafvonb/c8volt/issues/185) - `feat(get): add get incident command for keyed lookup and searchable incident listing`

## Source Issue

- **Issue Number**: 185
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/185
- **Issue Title**: feat(get): add get incident command for keyed lookup and searchable incident listing

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Fetch Known Incidents (Priority: P1)

As a Camunda operator with one or more incident keys, I want to fetch incidents directly so I can inspect their current state and failure details without first searching process instances.

**Why this priority**: Direct keyed lookup is the smallest useful command path and establishes the command, aliases, target selection, service facade, and output contracts.

**Independent Test**: Can be tested by fetching one or more known incident keys through repeated flags or stdin and verifying human, JSON, and keys-only output without requiring search filters.

**Acceptance Scenarios**:

1. **Given** a known incident key, **When** the user runs `c8volt get incident --key <incident-key>`, **Then** c8volt fetches and renders that incident.
2. **Given** repeated incident keys and newline-separated stdin keys, **When** the user runs `c8volt get incident --key <key-a> --key <key-b> -`, **Then** c8volt merges, validates, and deduplicates all keys before lookup.
3. **Given** a missing incident key, **When** the user requests keyed lookup, **Then** c8volt returns a clear not-found style error consistent with existing keyed commands.

---

### User Story 2 - Search Incidents By Core Fields (Priority: P2)

As an operator investigating failures, I want to list incidents by state, process context, flow node, and error type so I can narrow related failures quickly.

**Why this priority**: Search/list mode is the main operator workflow and provides the basis for totals, local filtering, and creation-time filtering.

**Independent Test**: Can be tested by running `get incident` without keys and applying each supported server-safe filter independently and in combination.

**Acceptance Scenarios**:

1. **Given** incidents in multiple states, **When** the user runs `c8volt get incident --state active`, `--state resolved`, or `--state all`, **Then** c8volt returns incidents matching the requested state semantics.
2. **Given** a valid generated Camunda incident error type in any letter case, **When** the user runs `c8volt get incident --error-type <value>`, **Then** c8volt validates and normalizes it through the existing incident filter helper.
3. **Given** process and flow-node context filters, **When** the user supplies those filters, **Then** c8volt applies them through version-appropriate incident search behavior.

---

### User Story 3 - Search Incident Messages Safely (Priority: P3)

As an operator searching by error text, I want case-insensitive substring matching across all relevant incidents so I do not miss matches because of backend paging or backend text semantics.

**Why this priority**: Error-message filtering has compatibility and correctness risk, so it should be isolated from the simpler server-safe filters.

**Independent Test**: Can be tested by searching for mixed-case message substrings across more than one page of candidate incidents and verifying matches are complete.

**Acceptance Scenarios**:

1. **Given** incidents whose messages contain a substring in different cases, **When** the user runs `c8volt get incident --error-message <substring>`, **Then** c8volt matches messages case-insensitively from the user's point of view.
2. **Given** backend text filtering is not guaranteed to be compatible with case-insensitive substring matching, **When** message filtering is needed, **Then** c8volt pages all relevant candidates up to the explicit command limit and applies local filtering.
3. **Given** v8.8 runtimes reject known broken scoped request shapes, **When** message filtering or scoped related incident lookup is needed, **Then** c8volt uses a tenant-safe compatibility path instead of the failing shape.

---

### User Story 4 - Filter By Creation Time (Priority: P4)

As an operator investigating incidents in a time window, I want to filter by incident creation time so I can isolate incidents created before or after a specific point.

**Why this priority**: Time filters are independent operator refinements once basic search works.

**Independent Test**: Can be tested by searching incidents with `--creation-time-after`, `--creation-time-before`, and both flags together using valid and invalid date values.

**Acceptance Scenarios**:

1. **Given** incidents created before and after a timestamp, **When** the user runs `c8volt get incident --creation-time-after <date-or-timestamp>`, **Then** only incidents created after that point are returned.
2. **Given** a time window, **When** the user combines `--creation-time-after` and `--creation-time-before`, **Then** c8volt returns only incidents within the requested window.
3. **Given** an invalid date value, **When** the user supplies it to a creation-time flag, **Then** c8volt fails locally with a clear flag validation error before making a remote request.

---

### User Story 5 - Render Incident Lists And Counts (Priority: P5)

As an operator reading incident output or building scripts, I want human rows, JSON, keys-only, and totals to preserve the right fields so I can inspect or automate incident workflows.

**Why this priority**: Output modifiers depend on fetched or searched incident results and must remain consistent with existing c8volt rendering conventions.

**Independent Test**: Can be tested by running the same filtered result set with human output, `--json`, `--keys-only`, `--total`, and `--error-message-limit`.

**Acceptance Scenarios**:

1. **Given** human incident output, **When** incidents are rendered, **Then** each row includes incident key, tenant, state, error type, creation time, process instance key, flow node ID, flow node instance key, job key as `n/a` when absent, message, and age from `creationTime`.
2. **Given** JSON output, **When** `--json` is used, **Then** c8volt preserves full incident fields, full `errorMessage`, and `creationTime` without truncation.
3. **Given** count output, **When** `--total` is used, **Then** c8volt prints only the exact numeric count after all filters are applied.

---

### User Story 6 - Preserve Command Contracts (Priority: P6)

As a c8volt user, I want `get incident` to follow existing command validation, help, documentation, and version behavior so the new command does not blur process-instance incident concepts or regress current workflows.

**Why this priority**: Contract preservation is cross-cutting and should finish the feature after the core lookup, search, and output slices are available.

**Independent Test**: Can be tested through help text, invalid flag combinations, unsupported-version behavior, documentation checks, and regression tests for existing `get pi` incident flows.

**Acceptance Scenarios**:

1. **Given** invalid flag combinations, **When** the user combines keyed lookup with search filters or combines `--total` with other machine output modes, **Then** c8volt rejects the command before remote calls with existing wording style.
2. **Given** a v8.7 target or another unsupported version, **When** the user runs unsupported incident operations, **Then** c8volt fails clearly before issuing unsupported requests.
3. **Given** existing process-instance incident commands, **When** this feature is added, **Then** `get pi --with-incidents`, `--incidents-only`, `--direct-incidents-only`, and related filters keep their existing semantics.

### Edge Cases

- `get incident` supports aliases `incident`, `incidents`, and `inc`.
- Keyed lookup cannot be combined with search/list filters.
- Repeated `--key` values and stdin keys are merged and deduplicated before lookup.
- Default search state is `active`; `--state all` means no state filter.
- Incident detail fields are plain `get incident` filters such as `--error-type`, `--error-message`, and `--state`; process-instance view extender flags are not added to `get incident`.
- Invalid state values report valid states: `active`, `pending`, `resolved`, `migrated`, `unknown`, and `all`.
- Invalid error types report generated enum-backed valid values.
- Local post-filtering never inspects only the first page; it continues until search is exhausted or the explicit command limit is reached.
- Backend totals are used only when they remain exact after all filters; otherwise totals are counted after local filtering.
- Long error messages are truncated in human output only when `--error-message-limit` is explicitly supplied.
- `--error-message-limit` requires human incident output.
- `--total` cannot be combined with `--json` or `--keys-only`.
- Missing or unparsable `creationTime` does not crash rendering or age calculation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST add a first-class `get incident` command with aliases `incidents` and `inc`.
- **FR-002**: `get incident` MUST support direct incident lookup by repeated `--key`, stdin keys via positional `-`, or both.
- **FR-003**: Keyed lookup MUST merge, validate, and deduplicate incident keys consistently with existing `get pi` key handling.
- **FR-004**: Keyed lookup MUST return clear not-found errors for missing keys.
- **FR-005**: Search/list mode MUST work when no incident keys are supplied.
- **FR-006**: Search/list mode MUST default `--state` to `active`.
- **FR-007**: `--state` MUST accept `active`, `pending`, `resolved`, `migrated`, `unknown`, and `all`; `all` MUST mean no state filter.
- **FR-008**: Search/list mode MUST support `--error-type <incident-error-type>`.
- **FR-009**: `--error-type` MUST validate against generated Camunda incident error enum values, accept any letter case from the user, and normalize internally to the generated enum value.
- **FR-010**: Error type validation and matching MUST reuse `internal/services/incidentfilter` instead of duplicating enum lists.
- **FR-011**: Search/list mode MUST support `--error-message <substring>` using case-insensitive substring semantics from the user's point of view.
- **FR-012**: Error-message filtering MUST NOT silently miss matches after the first page.
- **FR-013**: Backend message filtering MUST be used only when backend behavior is confirmed compatible with required case-insensitive substring semantics; otherwise c8volt MUST page candidates and filter locally.
- **FR-014**: Search/list mode MUST support `--process-instance-key`, `--root-process-instance-key`, `--process-definition-key`, `--process-definition-id`, `--flow-node-id`, and `--flow-node-instance-key`.
- **FR-015**: Search/list mode MUST support `--creation-time-after` and `--creation-time-before` using the incident `creationTime` field.
- **FR-016**: Invalid date or timestamp values MUST fail locally with clear flag validation errors before remote calls.
- **FR-017**: v8.7 MUST fail clearly before unsupported tenant-safe incident operations.
- **FR-018**: v8.8 compatibility paths MUST avoid request shapes known to fail with `Request property [filter] cannot be parsed`.
- **FR-019**: v8.9 SHOULD use server-side filters where semantics are safe, including tenant, state, error type, process context, flow node, and creation time bounds.
- **FR-020**: Any local post-filtering MUST page through all relevant results until search is exhausted or the explicit command limit is reached.
- **FR-021**: Search/list mode MUST follow existing `get pi` pagination, limit, interactive paging, auto-confirm, and non-interactive conventions where applicable.
- **FR-022**: Human output MUST include compact incident rows with incident key, tenant, state, error type, creation time, process instance key, flow node ID, flow node instance key, job key, message, and age.
- **FR-023**: Human output MUST render absent job keys as `n/a`.
- **FR-024**: Human output MUST truncate long error messages only when `--error-message-limit <chars>` is explicitly supplied.
- **FR-025**: JSON output MUST preserve full incident fields, full `errorMessage`, and `creationTime` without truncation.
- **FR-026**: `--keys-only` MUST print incident keys only.
- **FR-027**: `--total` MUST print only the numeric count of matching incidents after all filters are applied.
- **FR-028**: `--key` lookup MUST be rejected when combined with search filters.
- **FR-029**: `--total` MUST be rejected with `--json` and with `--keys-only`.
- **FR-030**: `--error-message-limit` MUST be rejected with non-human incident output using dependency wording consistent with c8volt commands.
- **FR-031**: The implementation MUST reuse `internal/domain.ProcessInstanceIncidentDetail`, `c8volt/process.ProcessInstanceIncidentDetail`, existing `internal/services/incident` service boundaries, and existing facade option/call option patterns where relevant.
- **FR-032**: Incident search logic MUST fit the incident service API rather than living primarily in command code.
- **FR-033**: The implementation MUST NOT add `--with-incidents`, `--incidents-only`, or `--direct-incidents-only` to `get incident`.
- **FR-034**: Existing `get pi --with-incidents`, `get pi --incidents-only`, `get pi --direct-incidents-only`, and `get pi --no-incidents-only` behavior MUST remain unchanged.
- **FR-035**: User-facing documentation, generated CLI docs, and help text MUST be updated for `get incident` and enum-backed error type values.

### Key Entities *(include if feature involves data)*

- **Incident Query**: A keyed lookup or search/list request built from user-supplied incident keys, search filters, output modifiers, pagination controls, and command limits.
- **Incident Detail**: The reusable incident domain record exposed to command rendering and JSON output, including state, error type, error message, creation time, tenant, process context, flow-node context, and optional job key.
- **Incident Filter**: The validated user intent for state, error type, error message, process context, flow node, and creation-time bounds.
- **Incident Search Result**: A paginated result set after server-side and local filters are applied, including exact counts when requested.
- **Incident Output View**: Human, JSON, keys-only, or total representation of the final incident result set.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can fetch one or more known incidents with `get incident --key` and receive accurate human, JSON, or keys-only output.
- **SC-002**: A user can list active incidents with `get incident` or `get incident --state active` without adding process-instance-specific incident flags.
- **SC-003**: State, error type, process context, flow node, and creation-time filters return only matching incidents or fail locally with clear validation errors.
- **SC-004**: Case-insensitive error-message substring filtering finds matches across all relevant pages up to the explicit command limit.
- **SC-005**: `--total` returns an exact numeric count after all server-side and local filters are applied.
- **SC-006**: Human incident rows include `creationTime` and an age derived from `creationTime`.
- **SC-007**: JSON output preserves full incident details and does not truncate messages.
- **SC-008**: v8.7 and incompatible v8.8 request paths fail or fall back before issuing known unsupported or broken requests.
- **SC-009**: Existing process-instance incident behavior remains unchanged under regression tests.

## Assumptions

- Operators already have c8volt configuration and Camunda permissions needed to read incident data for the selected tenant context.
- Existing c8volt date parsing conventions are reused for creation-time filters where possible.
- The existing incident domain model can represent all fields required by `get incident` output.
- Backend case-insensitive `$like` behavior for incident messages is not assumed unless verified by the implementation.
- This feature is read-only and does not change `resolve incident`, incident mutation behavior, process-instance filtering semantics, or job retry behavior.
