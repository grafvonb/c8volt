# Data Model: Process Definition Watch Mode

## Watch Session

Represents one invocation of watch mode.

Fields:

- `command`: Command path being watched, initially `get process-definition`.
- `startedAt`: Time the watch session starts.
- `interval`: Watch interval between snapshot attempts after the immediate first snapshot.
- `retryBudget`: Number of consecutive transient failures allowed before stopping.
- `timeout`: Overall command timeout when configured.
- `outputMode`: Selected human output mode.
- `terminationReason`: Why the session ended.

Validation rules:

- `interval` must be positive and defaults to `1s`.
- `retryBudget` defaults to the existing command retry default when no retry flag is provided.
- `outputMode` must be compatible with human watch; JSON, keys-only, XML, quiet, and automation are rejected.
- A watch session must emit its first snapshot before sleeping for `interval`.

State transitions:

```text
configured -> running -> snapshot_attempted* -> stopped
```

## Watch Tick

Represents one attempt to collect and emit a snapshot.

Fields:

- `index`: Monotonic tick number starting at 1.
- `startedAt`: Tick start time.
- `completedAt`: Tick completion time when snapshot collection succeeds or fails.
- `failure`: Transient or terminal failure, if any.
- `consecutiveFailures`: Failure count after this tick.

Validation rules:

- Tick 1 starts immediately when watch mode begins.
- Successful ticks reset `consecutiveFailures` to zero.
- Failed ticks increment `consecutiveFailures` only for transient lookup failures covered by the retry policy.

## Process Definition Watch Request

Represents the selector and output options used for every snapshot.

Fields:

- `filter`: Process-definition selector fields such as BPMN process ID, key, version, version tag, tenant, or broad search.
- `latest`: Whether only latest matching process-definition versions are selected.
- `stat`: Whether statistics are included.
- `page`: Page request for broad search snapshots.
- `watchAllWhenUnselected`: True when watch mode has no selector and must watch all process definitions.

Validation rules:

- `watchAllWhenUnselected` applies only in watch mode.
- Non-watch selector validation remains unchanged.
- JSON, keys-only, XML, quiet, and automation output are incompatible with watch mode.
- Key lookup remains valid in watch mode.

## Process Definition Watch Snapshot

Represents one complete watch result.

Fields:

- `tick`: Tick number that produced the snapshot.
- `capturedAt`: Time the snapshot was captured.
- `items`: Process definitions visible for the selected lookup.
- `total`: Number of items emitted in the snapshot.
- `pages`: Number of backend pages traversed when paging applies.
- `reportedTotal`: Backend-reported total and certainty when available.
- `empty`: Whether the snapshot contains no process definitions.

Validation rules:

- A snapshot must contain the complete selected result set for that tick, including all pages.
- Snapshot item order must remain compatible with existing non-watch ordering.
- Empty snapshots are successful results for explicit selectors and broad searches with no visible definitions.
- Machine-oriented watch rendering is not supported; incompatible output modes must fail before lookup work.

## Snapshot Selector

Represents the query identity for a watch snapshot.

Fields:

- `bpmnProcessId`: BPMN process ID filter.
- `processDefinitionKey`: Direct process-definition key.
- `processVersion`: Process-definition version filter.
- `processVersionTag`: Process-definition version tag filter.
- `latest`: Latest-version selector.
- `tenantId`: Tenant selector inherited from command context.
- `broad`: True when no selector is provided in watch mode.

Validation rules:

- `broad` is watch-only and means all process definitions visible to the command context.
- Explicit key selection remains backend-authorized admin input according to existing command behavior.
- Tenant scoping follows the existing process-definition command contract.

## Retry Budget

Represents the consecutive failure threshold for watch mode.

Fields:

- `maxRetries`: Maximum consecutive transient failures allowed.
- `consecutiveFailures`: Current consecutive transient failure count.
- `resetOnSuccess`: Always true.

Validation rules:

- Defaults come from the existing command retry default.
- Successful snapshots reset `consecutiveFailures`.
- Exceeding `maxRetries` stops the watch session with a non-success exit status.

## Watch Termination Reason

Represents why the watch session stopped.

Values:

- `interrupted`: Operator canceled the command.
- `timeout`: Overall timeout ended the watch session.
- `retry_exhausted`: Consecutive transient failures exceeded the retry budget.
- `fatal_error`: Non-retryable validation or lookup failure occurred.

Validation rules:

- `interrupted` and `timeout` after successful snapshots must not be reported as lookup failures.
- `retry_exhausted` and `fatal_error` must end with clear errors and non-success exit status.

## Output Mode Contract

Represents the selected rendering contract for each snapshot.

Values:

- `human`: Compact snapshot-oriented output; status/progress may use activity/stderr.
- `verbose`: Human output plus durable watch context when useful.
- `debug`: Human output plus debug logging according to existing rules.
- `json_rejected`: JSON output combined with watch is rejected before lookup work.
- `keys_only_rejected`: Keys-only output combined with watch is rejected before lookup work.
- `xml_rejected`: XML output combined with watch is rejected before lookup work.
- `quiet_rejected`: Quiet output combined with watch is rejected before lookup work.
- `automation_rejected`: Automation mode combined with watch is rejected before lookup work.

Validation rules:

- Machine-oriented result streams must not contain human watch headers, progress, retry diagnostics, or timestamps because watch mode rejects those combinations.
- Default human output must avoid low-level endpoint, request, cursor, and page lifecycle detail.
