# Data Model: Process Instance Variable Output

## Process Instance Result

Existing process-instance row selected by keyed lookup before enrichment.

**Fields used by this feature**:

- `key`: Process-instance key used for variable filtering.
- `tenantId`: Tenant context displayed in the base row and used by service calls where applicable.
- Existing visible row fields remain unchanged.

**Validation rules**:

- Must be selected through keyed lookup for `--with-vars` in this iteration.
- Missing process instances keep existing not-found behavior.

## Process Instance Variable

Variable directly defined at process-instance scope.

**Fields**:

- `name`: Variable name.
- `value`: Value as received from the API.
- `variableKey`: Unique variable key when returned by the API.
- `processInstanceKey`: Owning process-instance key.
- `scopeKey`: Scope key where the variable is directly defined.
- `tenantId`: Tenant ID.
- `apiTruncated`: Whether Camunda reports the returned value as truncated.

**Validation rules**:

- `processInstanceKey` must equal the selected process-instance key.
- `scopeKey` must equal the selected process-instance key.
- Variables with a different `scopeKey` are element-scoped and excluded.
- Variables are sorted by `name` ascending before output.

## Variable Value Display

Human-only representation of a variable value.

**Fields**:

- `displayValue`: One-line value after JSON compaction and optional CLI shortening.
- `cliTruncated`: Whether c8volt shortened the received value for terminal display.
- `truncationLabels`: Ordered labels rendered as `api-truncated`, `cli-truncated`, or `api-truncated,cli-truncated`.

**Validation rules**:

- JSON-like values are compacted to one line for human output.
- Values are not shortened by default.
- `--var-value-limit <chars>` applies only to human output.
- JSON output keeps the received API value intact.
- The ambiguous label `truncated` is never rendered for variable value truncation.

## Variable-Enriched Process Instance

Stable output association between one process instance and its variables.

**Fields**:

- `item`: The process-instance result.
- `variables`: Sorted process-instance variables belonging to `item.key`.

**Validation rules**:

- Multiple keyed lookups preserve per-key association.
- Lookup failure for any selected key fails the command clearly.
- Empty variable lists are represented as an empty list in JSON and no misleading element-scoped variables in human output.
