# Contract: Listener Timestamp Output

## Affected commands

- `get element --with-listeners` (keyed and existing search forms).
- `get process-instance --with-elements --with-listeners`.
- `walk process-instance --with-elements --with-listeners` (existing supported tree and flat modes).
- `ops analyse slow-process-instances --with-listeners`.

Existing aliases, required context, output-mode validation, exit codes, and unsupported-version errors remain unchanged.

## Human rows

The existing identity/status/retry/worker columns are followed by three optional timestamp columns in this order, then existing error columns:

| Tag | Source | Visibility |
| --- | --- | --- |
| `s:` | Job creation time | Whenever present, for any state. |
| `e:` | Job end time | Whenever present, for any state. |
| `d:` | Activation deadline | Only when present and state is exactly `ACTIVATED`. |

Creation time is not worker execution start. End time is not a deadline. Missing values omit tags; internal blank alignment columns may remain to align mixed rows, but no placeholder timestamp is printed. Execution and task listeners use the same rules. Completed, canceled, failed, created, blank, and unfamiliar states never show `d:`.

Reuse fixed-millisecond full-date/time formatting and `app.show_timezone_offset`. For example, a compact completed row with no worker or error fields can read:

```text
job-1 TASK_LISTENER lsnr:CREATING COMPLETED tp:updateTaskData r:0 s:2026-09-16T13:07:16.359 e:2026-09-16T13:07:16.842
```

With numeric offset display enabled, the corresponding supplied offset is appended to each timestamp. Tree prefixes and row alignment remain command-specific. Do not shorten timestamps to time-only as a consequence of the issue's illustrative example.

## Public and JSON contracts

`job.Job` and the listener objects in element, process, and ops expose `CreationTime` and `EndTime` as optional times. JSON encodes these as `creationTime` and `endTime`. Standard time JSON formatting preserves the instant; it need not duplicate the millisecond-only human rendering. Nil values are omitted, not emitted as null, empty strings, or zero-time defaults.

Example fragment within an existing listener collection:

```json
{
  "jobKey": "job-1",
  "kind": "TASK_LISTENER",
  "listenerEventType": "CREATING",
  "state": "COMPLETED",
  "retries": 0,
  "creationTime": "2026-09-16T13:07:16.359Z",
  "endTime": "2026-09-16T13:07:16.842Z",
  "deadline": "2026-09-16T13:08:00Z"
}
```

This is a fragment, not a new envelope. Preserve each command's existing envelope and nesting, existing fields, requested-empty listener arrays, and omission of unrequested listeners. Public standalone job JSON gains the same optional timestamp fields while its existing key naming remains unchanged. Human standalone job output is outside this change.

## Invariants

Rendering adds no requests. Adding timestamps changes neither listener association nor result order. Process/element durations and slow-analysis results are unchanged for identical inputs. Existing output without enrichment is unchanged apart from the specified additive fields where public job objects are already serialized. Results stay on stdout and existing diagnostics/control text stay on stderr.
