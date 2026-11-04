# Data Model: User-Task Variable Filtering

## Public search inputs

Extend `task.SearchRequest` with `VariableFilters VariableFilterSet` using `json:"variableFilters,omitempty"`. Keep all existing selectors and bounds unchanged. This is an input record, not a change to returned task or envelope schemas; Go struct-valued `omitempty` behavior must not be misrepresented as guaranteed omission when callers marshal this input themselves.

Add public aliases in `c8volt/task/model.go`:

| Task name | Existing target |
| --- | --- |
| `VariableFilterOperator` | `process.ProcessInstanceVariableFilterOperator` |
| `VariableFilterClause` | `process.ProcessInstanceVariableFilterClause` |
| `VariableFilterSet` | `process.ProcessInstanceVariableFilterSet` |

Expose `VariableFilterOperatorEq`, `Neq`, `Exists`, `In`, `NotIn`, and `Like` constants in the task package, referring to the existing process constants. No changes to the process API are required.

## Predicate fields

| Field | Meaning and invariant |
| --- | --- |
| `Name string` | Required nonblank local variable name; existing parser trimming rules apply |
| `Operator VariableFilterOperator` | Canonical existing operator; CLI `$notin` normalizes to `$notIn` |
| `Value string` | Existing serialized value or wildcard text, without new coercion |
| `Exists *bool` | Explicit true or false for existence; nil is invalid for that operator |
| `Source string` | Existing originating flag label for diagnostics |
| `Clauses []VariableFilterClause` | Ordered predicates, all required; empty means no variable predicate |

The task facade maps each field to the existing domain clause shape. Preserve order and boolean presence. Do not mutate caller-owned slices or pointers; allocate mapped clauses and copy optional boolean values.

## Domain query

Extend `domain.UserTaskSearchQuery` with `VariableFilters ProcessInstanceVariableFilterSet`. This name is reused for compatibility; it does not imply process-scope matching. The task adapter's `localVariables` destination determines scope.

Use existing clause/set validation. Before sending a native page request, decode membership operands through the same string-array rules as the PI adapter. Unknown operators, missing required values, missing existence booleans, and invalid membership operands return errors before HTTP, including for direct facade calls.

## Lifecycle and unchanged records

Raw task flags → existing shared parser → task search request → domain query → versioned native local-variable predicates → existing filtered page traversal → optional effective-variable display.

There are no state mutations, migrations, persistence, new result entities, or new task lifecycle transitions. `UserTask`, `UserTasks`, page metadata, visitors, completion dispositions, and effective-variable records remain unchanged. Empty native filter sets are omitted from request bodies to preserve unfiltered searches.
