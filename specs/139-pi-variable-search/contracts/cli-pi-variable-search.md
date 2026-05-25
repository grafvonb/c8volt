# CLI Contract: Process Instance Variable Search

## Traceability

- GitHub Issue: #139
- GitHub URL: https://github.com/grafvonb/c8volt/issues/139
- Feature: `139-pi-variable-search`

## Affected Commands

The feature changes only the existing process-instance search commands:

- `c8volt get process-instance`
- `c8volt get pi`

It must not introduce a separate parallel command family for variable search.

## New Flags

### `--var-exists <name[,name...]>`

Adds one `$exists=true` clause per variable name.

Example:

```bash
./c8volt get pi --var-exists payload,email
```

Contract:

- All listed variables must exist.
- Repeated flags add more required clauses.
- Blank names are invalid.

### `--var <clause[,clause...]>`

Adds equality shorthand or advanced variable value clauses.

Examples:

```bash
./c8volt get pi --var 'status="canceled"'
./c8volt get pi --var 'status="canceled",payload="payload"'
./c8volt get pi --var 'status.$in=["approved","pending"]'
```

Contract:

- `name=value` is shorthand for `name.$eq=value`.
- Advanced form is `name.$operator=value`.
- Supported operators are `$eq`, `$neq`, `$exists`, `$in`, `$notIn`, and `$like`.
- `$notin` may be accepted as an input alias and normalized to `$notIn`.
- Commas inside arrays or quoted values must not split clauses.
- Repeated flags and comma-separated clauses are all applied together.

### `--var-like <name=pattern[,name=pattern...]>`

Adds `$like` clauses.

Example:

```bash
./c8volt get pi --var-like 'email=*@example.com,customerId=CUST-????'
```

Contract:

- `name=pattern` is shorthand for `name.$like=pattern`.
- The CLI must not add implicit wildcard characters.
- Native wildcard semantics apply: `*` matches zero or more characters, `?` matches one character, and escaped wildcards remain literal.

## Version Behavior

- Camunda 8.8 and 8.9 must execute variable filters through native Camunda variable-search semantics.
- Camunda 8.7 must return an explicit unsupported-version error when any new variable-search flag is supplied.
- Searches without the new variable-search flags must preserve existing 8.7 behavior.
- No variable-search path may silently fall back to Operate-backed filtering.

## Output And Metadata

- Existing human, JSON, keys-only, total, paging, automation, and enrichment behavior must remain stable unless variable filters deliberately narrow the search result.
- Command help and examples must describe the three new flags, advanced operators, quoting, arrays, and `$like` wildcard behavior.
- Command contract metadata must expose the new flags and preserve existing mutation/read-only/automation support semantics.
- Generated CLI documentation must be refreshed from command metadata.

## ScopeKey Documentation

If `scopeKey` is exposed in variable-search help or docs, it must be described as the scope where the variable is directly defined:

- Process-level variable: `scopeKey == processInstanceKey`
- Local variable: `scopeKey` is the element-instance key where the variable was set

It must not imply inherited visibility through parent scopes.

## Invalid Inputs

The command must fail locally where practical for:

- Blank variable names
- Missing values for value operators
- Unknown operators
- Malformed boolean values for `$exists`
- Malformed array values for `$in` and `$notIn`
- Clauses that cannot be split or parsed unambiguously
