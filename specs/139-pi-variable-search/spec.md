# Feature Specification: Native Process Instance Variable Search

**Feature Branch**: `139-pi-variable-search`

**Created**: 2026-05-25

**Status**: Draft

**GitHub Issue**: [#139](https://github.com/grafvonb/c8volt/issues/139) - feat(get pi): add native variable search for process instances

**Input**: GitHub issue #139 requires native variable-based search for `get process-instance` / `get pi`, including variable existence, equality, pattern matching, and advanced value filters without using Operate-backed behavior.

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.

  Assign priorities (P1, P2, P3, etc.) to each story, where P1 is the most critical.
  Think of each story as a standalone slice of functionality that can be:
  - Developed independently
  - Tested independently
  - Deployed independently
  - Demonstrated to users independently
-->

### User Story 1 - Find Instances By Variable Existence (Priority: P1)

Operators can find process instances by requiring business variables to exist directly from the existing process-instance search command.

**Why this priority**: Existence checks are the smallest useful native variable-search slice and establish the shared variable-filter plumbing for later filter forms.

**Independent Test**: Can be tested by running `get pi --var-exists` and verifying only process instances with the named variables are returned while existing non-variable filters still combine correctly.

**Acceptance Scenarios**:

1. **Given** process instances with and without a `customerId` variable, **When** a user runs `get pi --var-exists customerId`, **Then** only process instances where `customerId` is directly searchable as an existing variable are returned.
2. **Given** process instances with different variable sets, **When** a user runs `get pi --var-exists payload,email`, **Then** only process instances with both variables are returned.
3. **Given** a user supplies repeated `--var-exists` flags, **When** the command searches, **Then** all supplied existence clauses are applied together.

---

### User Story 2 - Find Instances By Variable Equality (Priority: P2)

Operators can find process instances by matching one or more variable names to exact serialized values.

**Why this priority**: Equality checks are the most common value-based operational search case and build directly on the existence-search plumbing.

**Independent Test**: Can be tested by running `get pi --var` with one or more equality clauses and verifying only process instances matching every clause are returned.

**Acceptance Scenarios**:

1. **Given** process instances with different serialized `status` values, **When** a user runs `get pi --var 'status="approved"'`, **Then** only process instances matching that equality filter are returned.
2. **Given** a user runs `get pi --var 'status="canceled",payload="payload"'`, **When** the command parses the filter, **Then** both equality clauses are applied together.
3. **Given** a quoted value contains a comma, **When** the command parses the filter, **Then** the comma remains part of the value rather than splitting the clause.

---

### User Story 3 - Search With Like Patterns (Priority: P3)

Operators can find process instances using native wildcard pattern matching for variable values.

**Why this priority**: Like-pattern matching is a common business-variable search need and has user-visible wildcard semantics that must be documented and tested separately.

**Independent Test**: Can be tested by running `get pi --var-like` and verifying the native wildcard expression is preserved without implicit wildcards.

**Acceptance Scenarios**:

1. **Given** process instances with email-like variables, **When** a user runs `get pi --var-like 'email=*@example.com'`, **Then** the search follows the documented native wildcard semantics.
2. **Given** a user supplies `customerId=CUST-????`, **When** the command searches, **Then** `?` is preserved as a single-character wildcard.
3. **Given** a user needs a literal wildcard character, **When** the command parses an escaped wildcard, **Then** the escaped character remains literal for the native search contract.

---

### User Story 4 - Use Advanced Native Operators (Priority: P4)

Operators can use advanced native variable value operators when equality and like shortcuts are not expressive enough.

**Why this priority**: Advanced operators expose the native search power requested by the issue after the safer shortcut grammar has been established.

**Independent Test**: Can be tested by invoking `get pi --var` with each advanced operator and verifying the outgoing search criteria and returned rows match the requested variables.

**Acceptance Scenarios**:

1. **Given** process instances with several possible `status` values, **When** a user runs `get pi --var 'status.$in=["approved","pending"]'`, **Then** matching instances for any listed value are returned.
2. **Given** a user runs `get pi --var 'status.$notIn=["failed","canceled"]'`, **When** the command searches, **Then** process instances with excluded values are omitted.
3. **Given** a user writes `$notin`, **When** the command parses the variable filter, **Then** it is accepted as an alias for `$notIn`.
4. **Given** a user supplies `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, or `$like`, **When** the command parses the filter, **Then** the operator is accepted and represented in the native search criteria.

---

### User Story 5 - Preserve Version And Tenant Contracts (Priority: P5)

Operators receive predictable behavior across supported Camunda versions, including a clear failure when the selected runtime cannot support variable search.

**Why this priority**: Version clarity prevents false confidence in search results and keeps tenant-aware process-instance behavior consistent with existing workflows.

**Independent Test**: Can be tested by running the same variable-search command under supported 8.8 and 8.9 configurations and under an 8.7 configuration, then verifying supported versions search natively and 8.7 fails before implying unsupported behavior.

**Acceptance Scenarios**:

1. **Given** the configured runtime is 8.8 or 8.9, **When** a user supplies variable-search flags, **Then** the command searches process instances using the native variable search contract for that runtime.
2. **Given** the configured runtime is 8.7, **When** a user supplies any new variable-search flag, **Then** the command fails with an explicit unsupported-version error.
3. **Given** an existing tenant-aware process-instance search, **When** variable filters are added, **Then** tenant handling remains consistent with the same command without variable filters.

---

### User Story 6 - Understand The User-Facing Contract (Priority: P6)

Operators and automation authors can discover the new variable filters, quoting rules, wildcard behavior, scope semantics, and examples through command help and generated documentation.

**Why this priority**: The feature has compact but syntax-sensitive flags, so discoverable examples are necessary for reliable day-to-day use.

**Independent Test**: Can be tested by inspecting command help, command metadata, and generated documentation for concrete examples and the expected scope and escaping semantics.

**Acceptance Scenarios**:

1. **Given** a user opens help for `get pi`, **When** they inspect variable-search flags, **Then** the help explains existence, equality, like, and advanced operator forms with concrete quoting examples.
2. **Given** generated CLI documentation is rebuilt, **When** users read the process-instance search page, **Then** it includes the same variable-search examples and native wildcard behavior.
3. **Given** scope-related filtering is exposed for variables, **When** users read the docs, **Then** `scopeKey` is described as the scope where the variable is directly defined, not inherited visibility through parent scopes.

### Edge Cases

- Commas inside JSON arrays or quoted values must not split a variable clause.
- Repeated `--var` and `--var-like` flags must combine with comma-separated clauses using the same all-clauses-must-match semantics.
- Empty variable names, missing values, unknown operators, malformed arrays, malformed booleans, and invalid shorthand syntax must fail before remote search.
- `$like` must follow documented native wildcard behavior: `*` matches zero or more characters, `?` matches one character, and escaped wildcards remain literal.
- The command must not silently fall back to Operate-backed process-instance search for variable filters.
- Existing non-variable filters, output modes, pagination, and tenant behavior must not regress when no variable filters are supplied.
- 8.7 must remain valid for existing process-instance search behavior when the new variable-search flags are absent.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add variable-search support to `get process-instance` and `get pi` without creating a separate parallel command family.
- **FR-002**: The system MUST support `--var-exists` with one or more comma-separated variable names.
- **FR-003**: The system MUST treat `--var-exists payload,email` as requiring all listed variables to exist.
- **FR-004**: The system MUST support `--var name=value` as equality shorthand for `name.$eq=value`.
- **FR-005**: The system MUST support comma-separated `--var` clauses and repeated `--var` flags with all clauses applied together.
- **FR-006**: The system MUST support `--var-like name=pattern` as shorthand for `name.$like=pattern`.
- **FR-007**: The system MUST support repeated and comma-separated `--var-like` clauses with all clauses applied together.
- **FR-008**: The system MUST support advanced variable operators `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, and `$like`.
- **FR-009**: The system MUST accept `$notin` as an alias when practical and normalize it to `$notIn`.
- **FR-010**: The system MUST preserve commas inside JSON arrays and quoted strings when parsing variable filter clauses.
- **FR-011**: The system MUST fail with clear input errors for malformed variable filter expressions.
- **FR-012**: The system MUST support variable-search behavior for configured Camunda 8.8 and 8.9 runtimes using native Camunda variable search semantics.
- **FR-013**: The system MUST return an explicit unsupported-version error for Camunda 8.7 when any new variable-search flag is used.
- **FR-014**: The system MUST avoid Operate-backed implementation paths for this feature.
- **FR-015**: The system MUST keep tenant-aware behavior consistent with existing process-instance search flows.
- **FR-016**: The system MUST document `$like` wildcard and escaping behavior with examples.
- **FR-017**: The system MUST document `scopeKey`, if exposed, as the key of the scope where the variable is directly defined.
- **FR-018**: The system MUST update command help, command metadata, tests, README-facing documentation, and generated CLI documentation where user-facing command behavior changes.
- **FR-019**: The Ralph implementation launch instructions for this feature MUST include `--implementation-context specs/ralph-implementation-rules.md`.

### Key Entities *(include if feature involves data)*

- **Variable Filter Clause**: A single user-supplied condition that names a variable, chooses an operator, and supplies a serialized value or existence assertion.
- **Variable Filter Set**: The complete collection of variable filter clauses supplied by `--var-exists`, `--var`, and `--var-like`; all clauses are applied together.
- **Serialized Variable Value**: The string, boolean, or array value representation supplied by the user for native variable matching.
- **Variable Scope**: The scope where a variable is directly defined; for a process-level variable this is the process instance, and for a local variable this is the element instance where the variable was set.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests prove variable existence, equality shorthand, like shorthand, and each advanced operator produce the expected process-instance search behavior.
- **SC-002**: Automated tests prove comma parsing preserves arrays and quoted values without splitting clauses incorrectly.
- **SC-003**: Automated tests prove 8.8 and 8.9 support the new variable-search flags and 8.7 returns an explicit unsupported-version error for those flags.
- **SC-004**: Help text and generated documentation include concrete examples for `--var-exists`, `--var`, `--var-like`, `$in`, `$notIn`, `$like`, and wildcard escaping.
- **SC-005**: Existing process-instance search tests continue to pass for searches that do not use variable-search flags.
- **SC-006**: Repository validation for the affected command, facade, service, docs, and contract areas passes before tasks are marked complete.

## Assumptions

- Variable filters are intended for operators and automation users who already use `get process-instance` / `get pi`.
- Existing process-instance filtering, sorting, pagination, output modes, and tenant behavior remain in scope for compatibility unless explicitly changed by variable filters.
- Variable values follow the native serialized value contract expected by supported Camunda runtimes.
- The feature does not add write behavior; it only filters process-instance search results.
- Camunda 8.7 remains supported for existing process-instance search behavior but not for the new variable-search flags.
- Planning, task generation, and Ralph iterations must apply `specs/ralph-implementation-rules.md`.
