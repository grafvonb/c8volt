# CLI Contract: `get pi --with-vars`

## Command Surface

```text
c8volt get pi --key <process-instance-key> --with-vars
c8volt get pi --key <process-instance-key> --key <another-key> --with-vars
c8volt get pi --key <process-instance-key> --with-vars --var-value-limit <chars>
c8volt get pi --key <process-instance-key> --with-vars --json
```

## Valid Inputs

- `--with-vars` is valid with one or more `--key` values.
- `--var-value-limit <chars>` is valid only when `--with-vars` is set.
- `--var-value-limit 0` or leaving the flag unset means c8volt does not shorten received values for human output.
- Positive `--var-value-limit` values shorten only human display values.
- `--json --with-vars` is valid with keyed lookup and keeps received values intact.

## Invalid Inputs

- `--with-vars` without `--key` is out of scope for this iteration and must fail clearly.
- `--with-vars` with search/list-only filters is out of scope and must fail clearly.
- `--with-vars` with `--total` must fail clearly.
- `--var-value-limit` without `--with-vars` must fail clearly.
- Negative `--var-value-limit` values must fail clearly.

## Human Output

The base process-instance row remains the first line for each selected process instance. Variables render as indented lines below their owning process instance:

```text
2251799813711967 tenant-a order-process v3 ACTIVE s:2026-05-05T10:00:00.000Z p:<root>
  customerId = "C-123"
  order      = {"id":"O-9","amount":42}
  retryCount = 3
found: 1
```

Rules:

- Variable lines do not repeat the word `var`.
- Variables are sorted by name ascending.
- JSON-like values are compacted to one line.
- No CLI shortening happens unless `--var-value-limit <chars>` is positive.
- API-shortened values are marked `api-truncated`.
- CLI-shortened values are marked `cli-truncated`.
- Values shortened by both sources are marked `api-truncated,cli-truncated`.
- The label `truncated` is not used by itself.

## JSON Output

JSON output returns a stable enriched shape:

```json
{
  "total": 1,
  "items": [
    {
      "item": {
        "key": "2251799813711967"
      },
      "variables": [
        {
          "name": "customerId",
          "value": "\"C-123\"",
          "variableKey": "2251799813711999",
          "processInstanceKey": "2251799813711967",
          "scopeKey": "2251799813711967",
          "tenantId": "tenant-a",
          "apiTruncated": false
        }
      ]
    }
  ]
}
```

Rules:

- Values are exactly the values received from the API.
- Human display limits do not alter JSON values.
- Variable ordering is stable by name ascending.
- Metadata includes name, value, variable key, process instance key, scope key, tenant ID, and API truncation state when available.
