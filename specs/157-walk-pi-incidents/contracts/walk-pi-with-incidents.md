# Contract: `walk pi --with-incidents`

## Command Scope

`--with-incidents` is available on:

```bash
c8volt walk process-instance --key <process-instance-key> --with-incidents
c8volt walk pi --key <process-instance-key> --with-incidents
```

The flag applies to all supported walk modes:

- default children mode
- `--parent`
- `--children`
- `--family`
- `--family --tree`

## Validation Contract

- `--with-incidents` requires keyed walk input through `--key`.
- Validation must fail before process-instance or incident lookup side effects when the required keyed input is absent.
- `--keys-only --with-incidents` must be rejected with a clear validation error because key-only output cannot carry incident details.
- If incident lookup fails for any walked process instance, the command must fail instead of rendering partial incident output.

## Human Output Contract

When `--with-incidents` is provided:

- Normal process-instance rows remain the primary output.
- Incident messages render directly below the process-instance row that owns them.
- Incident message lines use the issue #154 convention: `incident: <message>`.
- Tree output keeps branch prefixes and indents incident lines beneath the matching node.
- Partial traversal warnings and missing ancestor warnings remain visible.

When `--with-incidents` is omitted:

- Existing human, key-only, tree, and JSON output remain unchanged.

## JSON Output Contract

JSON output with `--with-incidents` preserves traversal metadata and enriches `items`.

Example shape:

```json
{
  "mode": "children",
  "outcome": "complete",
  "rootKey": "2251799813711967",
  "keys": ["2251799813711967", "2251799813711977"],
  "edges": {
    "2251799813711967": ["2251799813711977"]
  },
  "items": [
    {
      "item": {
        "key": "2251799813711967",
        "tenantId": "tenant-a",
        "incident": true
      },
      "incidents": [
        {
          "incidentKey": "4503599627370497",
          "processInstanceKey": "2251799813711967",
          "tenantId": "tenant-a",
          "errorMessage": "failed to complete job"
        }
      ]
    },
    {
      "item": {
        "key": "2251799813711977",
        "tenantId": "tenant-a"
      },
      "incidents": []
    }
  ]
}
```

The shared top-level JSON envelope behavior remains whatever the command renderer already provides for JSON mode.

## Incident Lookup Contract

- Incident lookup is performed only for process-instance keys returned by traversal.
- Incident details attach to the matching process-instance key.
- Tenant filtering is included when a tenant is configured and the generated request body supports it.
- v8.8 and v8.9 use the generated `SearchProcessInstanceIncidentsWithResponse` operation.
- v8.7 returns the existing unsupported-capability style for requested incident enrichment.
- Any incident lookup failure propagates as command failure.
- No tenant-unsafe direct incident lookup may be used as the primary path.
