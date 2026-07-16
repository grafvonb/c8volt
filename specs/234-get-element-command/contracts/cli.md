# CLI Contract: `c8volt get element`

## Command

```text
c8volt get element [flags]
```

Aliases, if added, must not conflict with existing `get` subcommands. The canonical command name is `element`.

## Direct Lookup

```text
c8volt get element --key <element-instance-key>
```

### Rules

- Fetches exactly one runtime element instance when the key exists.
- `--key` is mutually exclusive with all search filters.
- Camunda 8.7 returns a clear unsupported-version error.

## Search

```text
c8volt get element [search filters] [output controls]
```

### Search Filters

| Flag | Meaning |
|------|---------|
| `--pi-key <process-instance-key>` | Match the owning process instance key. |
| `--element-id <bpmn-element-id>` | Match the BPMN element identifier. |
| `--state <state>` | Match runtime element state. |
| `--type <type>` | Match runtime element type. |
| `--pd-key <process-definition-key>` | Match process definition key. |
| `--bpmn-process-id <bpmn-process-id>` | Match BPMN process identifier. |

Search filters combine with AND semantics. Unfiltered search is allowed and follows normal `get` command paging behavior.

### Output Controls

| Flag | Contract |
|------|----------|
| `--batch-size <n>` | Tunes page size only. Must not cap total results by itself. |
| `--limit <n>` | Caps returned element instances across pages. |
| `--total` | Prints only the numeric total. |
| `--json` | Prints one stable JSON payload. |
| `--keys-only` | Prints only element instance keys, one per line. |

`--total` is mutually exclusive with `--json` and `--keys-only`.

## Human Output

Human list output uses compact aligned rows, primary key first, short tags, and a final `found: N` line.

Required row semantics:

```text
<elementInstanceKey> <tenantId> <type> <state> s:<startDate> [e:<endDate>] pi:<processInstanceKey> pd:<processDefinitionKey> element:<elementId> [inc!|inc!:<incidentKey>]
found: <N>
```

Rules:

- `e:` is omitted when no end date exists.
- `inc!` is present when `hasIncident` is true and no incident key is available.
- `inc!:<incidentKey>` is present when `hasIncident` is true and an incident key is available.
- Only one incident marker is rendered per row; never render both `inc!` and `inc!:<incidentKey>` for the same element instance.
- Normal output does not include request, cursor, backend target, or per-page lifecycle diagnostics.
- Timestamps render through existing c8volt timestamp helpers.

## Keys-Only Output

```text
<elementInstanceKey>
<elementInstanceKey>
```

No labels, counts, diagnostics, or blank decoration.

## Total Output

```text
<numeric-total>
```

No labels or decoration.

## JSON Output

Search JSON payload:

```json
{
  "total": 2,
  "items": [
    {
      "elementInstanceKey": "2251799813689002",
      "elementId": "ship-order",
      "elementName": "Ship order",
      "type": "SERVICE_TASK",
      "state": "ACTIVE",
      "startDate": "2026-07-15T10:12:01Z",
      "processInstanceKey": "2251799813688001",
      "rootProcessInstanceKey": "2251799813688001",
      "processDefinitionId": "order-process",
      "processDefinitionKey": "2251799813687001",
      "tenantId": "tenant-a",
      "hasIncident": true,
      "incidentKey": "2251799813687777"
    }
  ]
}
```

Direct lookup JSON may render the same item shape using the existing single-item command JSON convention.

## Error Contract

- Unsupported Camunda 8.7 behavior returns an unsupported-version error and a non-success exit.
- Invalid key-shaped flags fail before contacting Camunda.
- `--key` combined with search filters fails before contacting Camunda.
- Invalid paging controls fail before contacting Camunda.
