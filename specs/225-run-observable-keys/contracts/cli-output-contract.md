# CLI Output Contract: Run Process Instance Composition

## `c8volt run pi`

### Normal Output

When process instance details are rendered, the output must include the created process instance key and actual observed state.

Example shape:

```text
<process-instance-key> <tenant> <bpmn-process-id> v<version> <state> s:<start-date> ...
found: <count>
```

The `<state>` token is the observed state that satisfied creation confirmation.
When `--no-wait` is used, no lifecycle state has been observed, so the state token is omitted rather than replaced with a placeholder.

### JSON Output

`c8volt --json run pi ...` remains a full-contract command and must render the shared result envelope. The payload must contain the created process instance items and each item must include `state` when the state was observed.

Example shape:

```json
{
  "outcome": "succeeded",
  "command": "run process-instance",
  "payload": {
    "total": 1,
    "items": [
      {
        "key": "2251799813685249",
        "bpmnProcessId": "C89_NoOpCompletion_Process",
        "state": "COMPLETED"
      }
    ]
  }
}
```

### Keys-Only Output

`c8volt run pi --keys-only ...` must print only created process instance keys, one per line.

```text
2251799813685249
```

No `found:` line, labels, warnings, or other stdout content may be emitted in keys-only mode.

## Pipeline Contract

The output of `run pi --keys-only` must be valid stdin for `expect pi`:

```sh
c8volt run pi -b C89_NoOpCompletion_Process --keys-only \
  | c8volt expect pi --state completed -
```

`expect pi` remains strict: it succeeds only when the explicit `--state` expectation matches its observed process instance state.

## Creation Confirmation Contract

Run-style creation confirmation succeeds when the created process instance is observable as one of:

- `ACTIVE`
- `COMPLETED`
- `CANCELED`
- `TERMINATED`

Creation confirmation fails or continues waiting for:

- absent/not-found observations
- `UNKNOWN`
- any other non-observable state
