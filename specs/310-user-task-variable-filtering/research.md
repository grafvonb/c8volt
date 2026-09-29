# Research: User-Task Variable Filtering

## 1. Existing grammar is the compatibility boundary

**Decision**: Parameterize the existing `parsePIVariableFilters` orchestration with explicit exists/value/like input slices, retaining the PI wrapper and existing clause helpers. The task wrapper supplies task-owned flags. Keep parsing pure with respect to the other command's globals; do not temporarily assign PI flags. A small `parseVariableFilters` helper can remain beside the existing parser; no broad file/type rename is needed.

**Rationale**: `cmd/get_processinstance_variable_filter.go` already implements every required operator, alias, delimiter rule, error, and wildcard behavior. It groups clauses in exists, var, like order and preserves order within each group. Reimplementing parsing risks divergence. `StringArray` flag registration preserves commas for that parser.

**Alternatives considered**: A second parser duplicates behavior; a new query language violates the issue; relocating every PI type/helper creates unnecessary churn.

**Evidence**: `cmd/get_processinstance_variable_filter.go`, its tests, and variable flag registration in `cmd/get_processinstance.go`.

## 2. Reuse filter records without changing PI ownership

**Decision**: Expose `task.VariableFilterOperator`, `task.VariableFilterClause`, and `task.VariableFilterSet` as aliases of the corresponding public process filter types, with task-prefixed-package operator constants. Add `VariableFilters` to `task.SearchRequest` and `domain.UserTaskSearchQuery`; the domain field uses the existing domain `ProcessInstanceVariableFilterSet`. Convert public clauses mechanically in the task facade.

**Rationale**: `c8volt/task/model.go` already imports process and aliases its variable record. These existing predicate shapes contain no process key or scope constraint: scope comes from their destination request field. Reusing validation and retaining public PI names avoids a cross-repository migration.

**Alternatives considered**: Duplicating domain validators risks drift; introducing a generic public filtering package is disproportionate; importing internal types into the public facade leaks internals.

**Evidence**: `c8volt/task/model.go`, `c8volt/task/convert.go`, `c8volt/process/model.go`, `internal/domain/processinstance.go`, `internal/domain/usertask.go`.

## 3. Native mapping and supported versions

**Decision**: Map clauses to `UserTaskFilter.LocalVariables` in each existing v88/v89/v810 task adapter. Use the matching generated `VariableValueFilterProperty` and `StringFilterProperty` types. Add small adapter-local mapping helpers following existing PI mappers. Retain v87's unsupported native-task behavior. Omit `localVariables` entirely for an empty set.

**Rationale**: All three checked-in generated clients expose the same local-variable value-filter structure and six operators: eq, neq, exists, in, notIn, like. Their types are version-specific even where shapes match. Existing PI mapping provides the exact serialized-value conventions. No new generated client or endpoint is needed.

**Alternatives considered**: Importing PI adapters breaks ownership; sharing mappers across versioned generated types adds abstraction without benefit; extracting common version helpers would expand PI changes. Bounded local mapping with parity request tests is the smaller change for this issue.

**Evidence**: `internal/clients/camunda/{v88,v89,v810}/camunda/client.gen.go`, `internal/services/usertask/{v88,v89,v810}/search.go`, `internal/services/processinstance/{v88,v89,v810}/variable_filter.go`. Independent read-only research confirmed these matching shapes.

## 4. Preserve encoding; do not redefine existence semantics

**Decision**: Equality/inequality retain serialized value text; membership decodes the existing JSON string-array operand into the generated string array; wildcard text is passed unchanged; existence passes its explicit boolean, including false. Validate the domain set and membership decoding before issuing a native request.

**Rationale**: Generated value filters carry strings representing serialized values. A native null comparison is the text `null`, whereas a string containing that word is the text `"null"`. No automatic JSON conversion or additional object parser is introduced. Domain validation checks array shape; adapter decoding must still reject malformed membership arrays from direct facade callers. Existing CLI array validation already decodes string arrays.

**Limits of evidence**: Schemas prove representability, not every server's treatment of missing fields, indexed nulls, or `$exists=false`. Preserve backend semantics without claiming a client-side complement operation. Request tests prove faithful mapping; optional live scope fixtures verify backend results. A backend rejection remains an error, never a client-side fallback. No unresolved implementation choice depends on guessing these semantics.

**Evidence**: Existing PI parser, domain clause validation, PI versioned mappers, generated `AdvancedStringFilter` and `VariableValueFilterProperty` comments.

## 5. Existing traversal and display remain authoritative

**Decision**: Carry predicates through the existing request into every search/count page. Reuse the task walker, count path, visitors, and enrichment gate unchanged. Add the three flags to `hasGetUserTaskSearchFlags` so explicit keys and discovered stdin keys both conflict. Reject parsing errors in argument validation; build the request through an error-returning helper so no error is ignored or stored in mutable parsed state.

**Rationale**: `SearchUserTasksTotal` uses exact metadata or counts traversed rows. The shared walker retains one query while advancing pages. `get_usertask_search.go` already trims before display enrichment, and `shouldEnrichSelectedUserTasks` already handles effective output mode, total, and empty results.

**Alternatives considered**: Filtering already-fetched tasks would invalidate limits/counts; effective-variable reads would change scope and request cost; duplicating paging/output logic adds regression risk.

**Evidence**: `cmd/get_usertask.go`, `cmd/get_usertask_search.go`, `cmd/get_usertask_vars.go`, `internal/services/usertask/search.go`.

## 6. Validation proportional to the change

**Decision**: Plan targeted parser parity, adapter request, facade mapping, traversal propagation, command execution/output, and real-terminal checks. Regenerate docs after implementation metadata changes. This planning pass uses document checks only.

**Rationale**: View-only tests cannot prove native filtering or absence of variable reads; mocked terminal detection cannot prove prompt routing. The change requires no concurrency, dependency, or generated-client update. A full suite is conditional on broader implementation impact or unresolved failures, per constitution v2.0.0.

**Alternatives considered**: A live environment is not required for deterministic contract tests; a broad runtime suite for documentation alone provides no relevant evidence.
