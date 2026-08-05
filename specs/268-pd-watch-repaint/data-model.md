# Data Model: Process Definition Watch Repaint

## Watch Refresh

Represents one complete refresh cycle during process-definition watch mode.

Fields:

- `index`: Monotonic refresh number starting at 1.
- `startedAt`: Time the refresh starts.
- `completedAt`: Time collection and rendering complete.
- `duration`: Time between `startedAt` and `completedAt`.
- `interval`: Configured watch interval used to evaluate slow refreshes.
- `outcome`: Successful refresh, retryable failure, fatal failure, cancellation, or timeout.

Validation rules:

- Refreshes must be serial; refresh `N+1` cannot start before refresh `N` completes.
- Slow refresh detection compares `duration` with `interval`.
- Deliberate sleeping between refreshes is not part of `duration`.

## Repainted Watch View

Represents the terminal view shown for a successful refresh.

Fields:

- `repaintApplied`: Whether the command attempted to clear/reposition the terminal before writing the view.
- `resultBody`: Human process-definition rows and summary for the current refresh.
- `renderMode`: Human output mode.

Validation rules:

- `resultBody` must match the equivalent non-watch human process-definition output for the same selectors and display options.
- `resultBody` must not contain watch-only labels, counters, or lifecycle rows such as `snapshot 1:`.
- Empty results are rendered using the same normal human empty-list shape as non-watch output.
- Repaint/status control must not change JSON, keys-only, XML, quiet, automation, or non-watch output contracts.

## Process Definition Watch Snapshot

Represents the data collected for one refresh.

Fields:

- `items`: Process definitions visible for the selected lookup.
- `total`: Number of items emitted in the snapshot.
- `pages`: Number of backend pages traversed when paging applies.
- `reportedTotal`: Backend-reported total and certainty when available.
- `empty`: Whether the snapshot contains no process definitions.

Validation rules:

- A snapshot must contain the complete selected result set for that refresh.
- Snapshot item order must remain compatible with existing non-watch ordering.
- Key, latest, broad, filtered, tenant, and stat selections follow existing process-definition command rules.

## Slow-Refresh Warning Streak

Represents default human warning state for refreshes that exceed the configured interval.

Fields:

- `active`: Whether the current refresh streak is slow.
- `firstSlowRefresh`: The first refresh index in the current slow streak.
- `lastWarnedRefresh`: The refresh index that emitted the current streak warning.

Validation rules:

- When a refresh duration exceeds the interval and `active` is false, emit one default warning and set `active` true.
- When a refresh duration exceeds the interval and `active` is true, suppress additional default warnings.
- When a refresh duration is within the interval, reset `active` so a later slow streak can warn again.
- Verbose mode may emit per-refresh timing/status even while default warnings are suppressed.

## Output Mode Compatibility

Represents whether the selected output mode may be used with watch.

Values:

- `human`: Compatible with watch.
- `verbose_human`: Compatible with watch and may include additional timing/status outside the result body.
- `json_rejected`: JSON output combined with watch is rejected before lookup work.
- `keys_only_rejected`: Keys-only output combined with watch is rejected before lookup work.
- `xml_rejected`: XML output combined with watch is rejected before lookup work.
- `quiet_rejected`: Quiet output combined with watch is rejected before lookup work.
- `automation_rejected`: Automation mode combined with watch is rejected before lookup work.

Validation rules:

- Incompatible output modes fail local validation before process-definition lookup work begins.
- Non-watch output modes remain unchanged.
