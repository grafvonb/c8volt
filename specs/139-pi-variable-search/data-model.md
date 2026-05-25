# Data Model: Native Process Instance Variable Search

## Variable Filter Clause

Represents one variable-search condition supplied by the user.

Fields:

- `Name`: Variable name. Required and non-empty.
- `Operator`: One of `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, or `$like`.
- `Value`: Serialized native value text for operators that compare values.
- `Exists`: Boolean value for `$exists` clauses.
- `Source`: User-facing source flag, used for diagnostics and tests.

Validation:

- `Name` must not be blank.
- `Operator` must be one of the supported native operators.
- `$notin` is accepted only as input alias and normalized to `$notIn`.
- `$exists` requires a boolean value when supplied through advanced syntax.
- `$in` and `$notIn` require an array-shaped serialized value.
- `$eq`, `$neq`, and `$like` require a non-empty serialized value.

## Variable Filter Set

Represents all variable-search clauses for a process-instance search.

Fields:

- `Clauses`: Ordered list of variable filter clauses.

Validation:

- Empty sets preserve existing process-instance search behavior.
- Non-empty sets combine all clauses with all-clauses-must-match semantics.
- Commas inside quoted values or JSON arrays remain part of the clause value and do not split the set.

## Process Instance Search Filter

Extends the existing process-instance filter with optional variable search criteria.

Fields added or affected:

- `VariableFilters`: Optional variable filter set.

Relationships:

- Existing process definition, state, parent, incident, date, tenant, limit, and pagination filters remain compatible with variable filters unless an existing command-mode validation rule forbids that combination.
- Tenant filtering remains owned by the existing process-instance search flow.

## Version Capability

Represents whether the configured Camunda runtime may execute variable search.

Rules:

- Camunda 8.8: supports native variable-search behavior for the new flags.
- Camunda 8.9: supports native variable-search behavior for the new flags.
- Camunda 8.7: rejects the new flags with an explicit unsupported-version error.

## Variable Scope

Represents where a variable is directly defined.

Rules:

- For a process-level variable, `scopeKey` equals the process instance key.
- For a local variable, `scopeKey` is the element-instance key where the variable was set.
- `scopeKey` must not be described as inherited visibility through parent scopes.
