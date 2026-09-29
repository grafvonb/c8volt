# Feature Specification: Filter User Tasks by Local Variables

**Feature Branch**: `codex/310-user-task-variable-filtering`

**Created**: 2026-09-19

**Status**: Draft

**Input**: [GitHub issue #310 — feat(cmd): add variable filtering to get user-task](https://github.com/grafvonb/c8volt/issues/310), incorporating the existing `get pi` grammar and independent effective-variable display.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find Tasks Using Familiar Variable Filters (Priority: P1)

As an operator, I want to find user tasks by their local variables using the variable-filter expressions I already use with process instances, so that I can investigate work without learning a different query language.

**Why this priority**: Selecting the right tasks is the feature's primary value. Familiar syntax reduces mistakes during operations.

**Independent Test**: Search a known set of tasks using the existing equality, existence, wildcard, and advanced-clause examples. Compare selected task identities with the expected local-variable matches and check that equivalent expressions retain the existing process-instance parsing behavior.

**Acceptance Scenarios**:

1. **Given** tasks with different local `status` values, **When** `c8volt get ut --var 'status="approved"'` runs, **Then** only tasks whose local value matches are selected, without requiring `--with-vars`.
2. **Given** tasks with different local variable names, **When** `--var-exists payload,email` runs, **Then** both names are required; repeated flags retain the same all-clauses-must-match behavior.
3. **Given** local values containing ordinary text and literal wildcard characters, **When** `--var-like` or `name.$like=value` is used, **Then** `*`, `?`, and escaped wildcards follow the existing `get pi` behavior without added wildcards.
4. **Given** valid existing `get pi` clauses, **When** the same clauses are supplied to `get ut`, **Then** equality shorthand, `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, `$like`, and the `$notin` alias retain the same grammar, quoting, serialized-value encoding, and validation behavior for supported operations.
5. **Given** matching tasks assigned to different users, **When** `--assignee alice --var 'status="approved"'` runs, **Then** both the ordinary task filter and variable filter must match, within the existing tenant selection.
6. **Given** a matching variable defined only in a parent or process-instance scope, **When** the task is searched by that variable, **Then** the inherited value does not satisfy a local-variable predicate. If a task-local value shadows a parent value, the local value determines the match.
7. **Given** malformed syntax or an unsupported operator, **When** the command is invoked, **Then** it fails before issuing requests, using established flag-error conventions and without falling back to another interpretation.

---

### User Story 2 - Bound and Count Filtered Work Reliably (Priority: P1)

As an operator or automation author, I want filtered searches to preserve task limits, paging, counting, and output contracts, so that adding a variable filter does not change how I control or consume a search.

**Why this priority**: A correct filter must remain reliable for large workloads and scripts, including empty searches and incomplete pages.

**Independent Test**: Search a known filtered collection across multiple and sparse pages, with a result limit, exact counting, interactive paging, and each existing output mode. Verify selected identities, counts, continuation, and separate result and prompt streams.

**Acceptance Scenarios**:

1. **Given** more matching tasks than the requested limit, **When** `--assignee alice --var 'status="approved"' --limit 20` runs, **Then** at most 20 matching tasks are returned, with the same filters retained throughout discovery.
2. **Given** matching tasks spread across pages, including an empty intermediate page with continuation evidence, **When** a filtered search continues, **Then** later matches remain discoverable and the empty page is not treated as authoritative completion.
3. **Given** a known collection with an exact or capped reported total, **When** `--var 'status="approved"' --total` runs, **Then** the command returns the exact filtered count through its existing counting behavior and retains existing incompatible-flag validation.
4. **Given** interactive paging is eligible, **When** the user continues, declines, or reaches end of input, **Then** paging eligibility, default answers, stop behavior, and prompts on configured or inherited stderr remain unchanged, including when stdout is redirected.
5. **Given** authoritative discovery finds no matching tasks, **When** human, JSON, keys-only, quiet, or total output is requested, **Then** the established empty result is preserved: human `found: 0`, one successful JSON envelope with the existing empty payload, zero-byte keys-only output, suppressed quiet human output, or numeric `0` for total. Quiet does not suppress explicitly requested machine results.
6. **Given** a search with results, **When** JSON, keys-only, quiet, auto-confirm, or automation modes are selected, **Then** the existing output precedence, envelope structure, one-key-per-line contract, and unattended behavior are preserved.
7. **Given** explicit task keys or stdin key input, **When** any variable search flag is also supplied, **Then** the existing key-versus-search conflict is reported before requests.
8. **Given** discovery fails, including on a later page, **When** the command reports the outcome, **Then** it preserves established errors and exit behavior instead of reporting an empty or complete successful result.

---

### User Story 3 - Inspect Effective Variables After Local Filtering (Priority: P2)

As an operator, I want to add the existing `--with-vars` option to a filtered search, so that I can inspect selected tasks without changing either the filtering scope or the display semantics.

**Why this priority**: Combining selection and inspection is useful, but variable filtering must also work independently without additional variable reads.

**Independent Test**: Run the same local-variable search with and without `--with-vars`, including a limit and excluded display modes. Verify identical task selection and existing effective-variable output only for selected tasks in eligible modes.

**Acceptance Scenarios**:

1. **Given** tasks selected by a local `status` value and effective variables from multiple scopes, **When** `--var 'status="approved"' --with-vars` runs, **Then** the selection remains local while displayed variables retain the existing backend-selected effective values.
2. **Given** a filtered search without `--with-vars`, **When** it executes, **Then** no individual task-variable retrieval is performed to evaluate the filter, and no discovery is added solely to validate variable names.
3. **Given** filtered results subject to a limit or a paging stop, **When** `--with-vars` is requested, **Then** only selected tasks eligible for display have their effective variables retrieved; filtering does not introduce duplicate retrieval.
4. **Given** `--with-vars` with effective keys-only output, `--total`, or an empty result, **When** the command executes, **Then** no effective-variable retrieval occurs. JSON-over-keys precedence and quiet combined with JSON retain their existing behavior.
5. **Given** effective variables with long, structured, Unicode, or backend-truncated values, **When** filtered results are displayed, **Then** existing `--var-value-limit`, truncation labels, JSON preservation, and display-failure behavior remain unchanged.

### Edge Cases

- Repeated flags and comma-separated clauses combine identically to `get pi`; commas inside supported quoted values and arrays remain part of their values.
- Serialized strings, numbers, booleans, nulls, and array operands retain existing parsing and encoding rules; this feature does not broaden accepted value syntax or add automatic conversion.
- An absent local variable and a present value of JSON `null` remain distinct under existing native matching semantics; inherited values do not establish local existence.
- Conflicting predicates may produce a successful empty search; they must not be silently dropped or changed to OR semantics.
- Empty names, malformed arrays, unterminated quoting, missing assignments, and unknown operators follow existing validation conventions.
- Sparse pages and capped totals retain established completion rules; permission failures and retrieval errors are not empty matches.
- An unsupported backend or operation produces a clear error, without unfiltered fallback or per-task filtering.
- Interactive results keep prompts out of stdout. Empty completed searches do not add paging or mutation confirmation.
- Filtering and optional display can observe changing backend data; this feature adds no snapshot-consistency guarantee.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `get user-task` and its existing aliases MUST accept `--var`, `--var-exists`, and `--var-like` for list/search execution independently of `--with-vars`.
- **FR-002**: Variable-filter grammar MUST match existing `get pi` behavior, including equality shorthand, `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, `$like`, the `$notin` alias, repeated flags, comma-separated clauses, quoting, escaping, serialized-value encoding, and validation. No new grammar, operators, conversion, or matching semantics may be introduced.
- **FR-003**: Every variable predicate MUST apply only to variables defined locally on the task. Parent-scope and process-instance variables MUST NOT satisfy local predicates.
- **FR-004**: Variable predicates MUST combine with one another and ordinary task filters using existing AND semantics while preserving tenant selection and authorization behavior.
- **FR-005**: The backend MUST select matching tasks. Filtering MUST NOT retrieve individual task variables, perform client-side task matching, or add variable-name validation discovery.
- **FR-006**: Malformed clauses and unsupported operators MUST fail before search requests using established error conventions. Explicit keys and stdin key input combined with variable filters MUST fail before requests.
- **FR-007**: All discovery pages and counting operations MUST preserve the requested predicates. Existing batch sizing, limits, sparse-page traversal, exact counting, capped-total handling, and failure behavior MUST remain unchanged.
- **FR-008**: Existing human, JSON, keys-only, quiet, total, auto-confirm, automation, and combined-mode behavior MUST be preserved for both nonempty and authoritative empty results. No shared result schema or output precedence may change.
- **FR-009**: Interactive paging MUST preserve eligibility, wording, default answers, end-of-input behavior, and stopping behavior; prompts MUST use configured or inherited stderr while stdout remains command results only.
- **FR-010**: Explicit `--with-vars` MUST preserve effective-variable display for selected tasks, with unchanged value limits, truncation metadata, output formatting, and errors. It MUST NOT affect which tasks match local predicates.
- **FR-011**: Display-related retrieval MUST remain absent without `--with-vars`, for effective keys-only output, for `--total`, and for empty results. Eligible display MUST retrieve only selected tasks under existing bounds and paging decisions.
- **FR-012**: The feature MUST cover the base command's supported Camunda 8.8, 8.9, and 8.10 versions where native operations are supported, retain Camunda 8.7's clear unsupported behavior, and reject unsupported operations without fallback or silently removing predicates.
- **FR-013**: Existing commands without variable filters and existing process-instance filtering MUST retain their current behavior.
- **FR-014**: Command help, metadata, examples, README, and generated CLI documentation MUST describe familiar grammar and clearly distinguish local filtering from effective display, including examples for equality, existence, wildcard, ordinary-filter combination, total, and display combination.
- **FR-015**: Scope MUST remain read-only user-task variable filtering. Changes to process-instance filtering, effective-variable filtering across scopes, variable display semantics, variable mutation, and task mutations are excluded.

### Key Entities *(include if feature involves data)*

- **User task**: The existing selectable work item, with its identity, state, assignment, process, and tenant context unchanged.
- **Local task variable**: A named value defined directly on a task, used to determine whether that task matches a predicate.
- **Variable predicate**: An existing c8volt expression comprising a variable name, supported operator, and value or existence condition.
- **Filtered task selection**: Tasks satisfying all requested local-variable and ordinary task predicates within existing selection bounds.
- **Effective variable display**: The existing optional view of variables visible to selected tasks, independent of local filtering.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can complete all six issue workflows—equality, existence, wildcard, bounded combined search, exact total, and filtered effective-variable display—with the documented commands and expected task identities.
- **SC-002**: All compatibility cases for existing variable-filter syntax retain the same interpretation or validation outcome when moved from process-instance searches to supported user-task searches; no new expression syntax must be learned.
- **SC-003**: Known local-scope test collections produce exactly the expected matching identities and totals, including shadowed and parent-only values, sparse pages, and bounded results.
- **SC-004**: All covered output and interaction modes preserve their existing contracts for matching and empty searches, with zero prompt text on stdout and zero bytes for empty keys-only output.
- **SC-005**: Filter-only workflows perform zero individual task-variable reads and zero variable-name discovery operations; optional display retrieves variables only for eligible selected tasks.
- **SC-006**: Every malformed-clause and key-conflict acceptance case fails before requests; unsupported operations and retrieval failures never become successful unfiltered or empty searches.
- **SC-007**: Filtered and unfiltered inspection retain the same effective-variable display semantics, and existing process-instance variable-filter behavior remains unchanged.

## Assumptions

- The base user-task command and effective-variable display already available on `develop` are the compatibility baseline; this feature does not reimplement them.
- Existing `get pi` grammar and validation are authoritative. Apparent opportunities to expand accepted values or redesign operators are outside this issue.
- Native local-variable matching determines scope, existence, and value-comparison semantics. Unsupported behavior is rejected rather than approximated locally.
- Existing authentication, tenant scope, output selection, paging, count validation, and error conventions are reused.
- Specification work is limited to requirements and quality validation. Implementation mechanics and version-specific mappings belong in the subsequent plan, following repository architecture and issue constraints.
