# CLI Contract: `c8volt get process-instance --with-elements`

## Command

```text
c8volt get process-instance [flags]
c8volt get pi [flags]
c8volt get pis [flags]
```

The feature adds `--with-elements` to the existing process-instance command and aliases.

## Keyed Lookup

```text
c8volt get pi --key <process-instance-key> --with-elements
```

### Rules

- Fetches the selected process instance using existing keyed lookup behavior.
- Attaches runtime element instances whose `processInstanceKey` matches the selected process instance key.
- Keyed mode rejects process-instance search filters when `--with-elements` is present.
- Camunda 8.7 returns a clear unsupported-version error.

## List/Search Lookup

```text
c8volt get pi --state active --limit 5 --with-elements
c8volt get pi -b <bpmn-process-id> --limit 5 --with-elements
```

### Rules

- Existing process-instance filters decide which process instances are selected.
- `--limit` caps process instances only.
- `--batch-size` controls process-instance page size only.
- Interactive prompts are based on process-instance pages only.
- Element-specific filters are not part of this command.

## Invalid Combinations

| Combination | Contract |
|-------------|----------|
| `--with-elements --total` | Reject before remote lookup with a clear validation error. |
| `--keys-only --with-elements` | Reject before remote lookup with a clear validation error. |
| `--key <key> --with-elements` plus search filters | Reject before remote lookup with a clear validation error. |

## Human Output

When `--with-elements` is present, each enriched process instance may include an `elements:` section.

Required section order when multiple enrichments are requested:

```text
vars:
incidents:
elements:
```

Element rows under `elements:` use this compact shape:

```text
<elementInstanceKey> <type> <elementId> <state> s:<startDate> [e:<endDate>] [inc!|inc!:<incidentKey>]
```

Rules:

- Element rows are indented beneath the owning process instance.
- Element row columns are aligned across sibling rows.
- `elementId` is a positional column between `type` and `state`.
- Active elements without an end date omit `e:`.
- Incident markers match standalone `get element` marker behavior.
- The old `element:<elementId>` suffix is not rendered.
- Normal output does not include request, cursor, backend target, or per-element lifecycle diagnostics.

## JSON Output

JSON output uses the existing shared result envelope. Each enriched process-instance item includes attached elements.

Payload shape:

```json
{
  "total": 1,
  "items": [
    {
      "item": {
        "key": "2251799813688001",
        "tenantId": "tenant-a",
        "bpmnProcessId": "order-process",
        "processVersion": 3,
        "state": "ACTIVE",
        "startDate": "2026-07-15T10:12:00Z"
      },
      "elements": [
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
  ]
}
```

Combined enrichment payloads may include `variables`, `incidents`, and `elements` fields in the same process-instance item.

## Error Contract

- Validation errors fail before remote lookup where possible.
- Unsupported Camunda 8.7 behavior returns a non-success unsupported-version result.
- Element enrichment lookup failures fail the command instead of rendering partially enriched process instances as success.
- `--json` errors use the existing shared command error envelope for full-contract commands.
