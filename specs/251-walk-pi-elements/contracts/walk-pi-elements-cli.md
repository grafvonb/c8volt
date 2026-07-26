# CLI Contract: Walk PI Elements

## Command Surface

The feature extends these equivalent command paths:

```text
c8volt walk process-instance --key <process-instance-key> --with-elements
c8volt walk pi --key <process-instance-key> --with-elements
```

Supported traversal combinations:

```text
c8volt walk pi --key <process-instance-key> --with-elements
c8volt walk pi --key <process-instance-key> --children --with-elements
c8volt walk pi --key <process-instance-key> --parent --with-elements
c8volt walk pi --key <process-instance-key> --flat --with-elements
c8volt walk pi --key <process-instance-key> --with-vars --with-incidents --with-elements
```

## Flag Contract

| Flag | Type | Required | Compatibility |
|------|------|----------|---------------|
| `--with-elements` | bool | No | Requires keyed walk behavior and a render mode that can carry detail rows. |
| `--key`, `-k` | string | Yes | Identifies the starting process instance for traversal. |
| `--children` | bool | No | Traverses descendants and enriches selected rows. |
| `--parent` | bool | No | Traverses ancestry and enriches selected rows. |
| `--flat` | bool | No | Renders family output as flat path output with detail sections. |
| `--with-vars` | bool | No | May combine with `--with-elements`; `vars:` renders before `elements:`. |
| `--with-incidents` | bool | No | May combine with `--with-elements`; `incidents:` renders before `elements:`. |
| `--keys-only` | bool | No | Must fail when combined with `--with-elements`. |
| `--json` | bool | No | Must preserve traversal metadata and include per-item `elements`. |

## Validation Contract

- `--with-elements` requires `--key`.
- `--keys-only --with-elements` fails before remote element enrichment.
- Existing `--with-vars` and `--with-incidents` validation remains unchanged.
- Camunda 8.7 returns the existing unsupported element-search capability error.

## Human Output Contract

When runtime elements exist, render an `elements:` section under the owning process-instance row.

Element row grammar:

```text
<elementInstanceKey> <type> <elementId> <state> s:<startDate> [e:<endDate>] [dur:<duration>] [inc!|inc!:<incidentKey>]
```

When multiple enrichment sections are present, render in this order:

```text
vars:
incidents:
elements:
```

Family tree shape must keep detail sections and child process instances structurally separate:

```text
<root-pi-row>
├─ elements:
│  ├─ <element-row>
│  └─ <element-row>
└─ <child-pi-row>
   └─ elements:
      └─ <element-row>
```

## JSON Output Contract

`--json walk pi --with-elements` keeps the shared traversal envelope and adds per-item element data:

```json
{
  "mode": "family",
  "outcome": "complete",
  "rootKey": "2251799813688001",
  "keys": ["2251799813688001"],
  "edges": {},
  "items": [
    {
      "item": {
        "key": "2251799813688001"
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

If variables or incidents are also requested, the same item may include `variables`, `incidents`, and `elements`.
