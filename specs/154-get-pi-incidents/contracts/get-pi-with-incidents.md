# Contract: `get pi --with-incidents`

## Command Surface

```bash
c8volt get process-instance --key <process-instance-key> --with-incidents
c8volt get pi --key <process-instance-key> --with-incidents
c8volt get pi --key <key-a> --key <key-b> --with-incidents
c8volt get pi --key <process-instance-key> --with-incidents --json
```

## Validation Contract

- `--with-incidents` is valid only when at least one `--key` value is provided.
- `--with-incidents` is invalid with search-mode filters, including `--incidents-only` and `--no-incidents-only`.
- `--with-incidents` is invalid with `--total` because `--total` is search-mode/count output and is already mutually exclusive with `--key`.
- Validation failures use existing invalid flag-combination error handling.

## Human Output Contract

When `--with-incidents` is omitted, process-instance human output is unchanged.

When `--with-incidents` is present:

- The normal process-instance row is still shown.
- Incident error messages are shown as indented `incident:` lines directly below the matching process-instance row.
- Process instances without incidents still render successfully.
- Empty incident messages do not break output.

## JSON Output Contract

When `--with-incidents --json` is present, the command returns a machine-readable enriched payload.

Expected payload shape:

```json
{
  "total": 1,
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
          "state": "ACTIVE",
          "errorType": "JOB_NO_RETRIES",
          "errorMessage": "No retries left"
        }
      ]
    }
  ]
}
```

`incidents` is present as an empty array when enrichment was requested and no incidents were returned for an item.

## Tenant Contract

- Supported versions include configured tenant filtering in the incident search request when a tenant is configured.
- Incident enrichment must not use tenant-unsafe direct incident lookup as its primary path.
- Wrong-tenant or absent incidents are represented by the supported API response and do not change the process-instance lookup result.

## Version Contract

- `v8.8`: Supported through generated process-instance incident search.
- `v8.9`: Supported through generated process-instance incident search.
- `v8.7`: Unsupported for this flag when tenant-safe keyed enrichment cannot be guaranteed.
