# CLI Contract: Process Instance Incident List Output

## Scope

This contract covers user-visible behavior for:

- `c8volt get process-instance`
- `c8volt get pi`
- `c8volt walk process-instance`
- `c8volt walk pi`

## Valid Invocations

```bash
c8volt get pi --with-incidents
c8volt get pi --incidents-only --with-incidents
c8volt get pi --state active --with-incidents
c8volt get pi --bpmn-process-id <id> --with-incidents
c8volt get pi --key <process-instance-key> --with-incidents
c8volt get pi --json --with-incidents
c8volt get pi --with-incidents --incident-message-limit 80
c8volt walk pi --key <process-instance-key> --with-incidents
```

## Invalid Invocations

```bash
c8volt get pi --total --with-incidents
c8volt get pi --incident-message-limit 80
c8volt get pi --with-incidents --incident-message-limit -1
```

Expected behavior:

- `--with-incidents` with `--total` fails with a clear mutually exclusive flag error.
- `--incident-message-limit` without `--with-incidents` fails with a clear dependency error.
- Negative `--incident-message-limit` fails with a clear invalid value error.

## Human Output

Normal process-instance rows remain unchanged.

Direct incident details render immediately below the owning row:

```text
2251799813711967 tenant-a demo v3 active s:2026-03-23T18:00:00.000Z inc!
  inc 4503599627370497: No retries left
found: 1
```

Rows with an incident marker but no direct incident details render a short row-local note:

```text
2251799813711967 tenant-a demo v3 active s:2026-03-23T18:00:00.000Z inc!
  no direct incidents found for this process instance
```

If any listed row has that indirect marker condition, one warning is printed after the list:

```text
warning: one or more incident markers may refer to incidents in the process-instance tree; inspect with walk pi --key <key> --with-incidents
```

The exact short note and warning text may follow existing repository wording, but the row-local note must stay short and the tree-inspection guidance must be de-duplicated per list output.

## Human Message Truncation

When `--incident-message-limit <chars>` is set:

- The limit applies to the incident error message only.
- Truncation is character-safe.
- `...` is appended only when truncation occurs.
- A limit of `0` means unlimited.
- The prefix `inc <incident-key>:` is not counted as part of the message limit.

## JSON Output

`c8volt get pi --json --with-incidents` in list/search mode returns the existing incident-enriched payload shape used by keyed lookup.

JSON output:

- includes full incident messages;
- preserves per-process-instance association;
- is not affected by `--incident-message-limit`;
- preserves default process-instance metadata already present in keyed incident JSON.

## Paging And Limits

List/search selectors, paging, and `--limit` determine the process instances to render. Incident details are attached to those rendered process instances and must not cause extra rows to be selected or displayed.

## Prefix Compatibility

Both `get pi --with-incidents` and `walk pi --with-incidents` human incident lines use:

```text
inc <incident-key>: <error-message>
```

The old `incident <incident-key>:` human prefix is no longer used.
