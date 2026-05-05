# CLI Contract: Process Instance Incident Expectation

## Command Surface

```text
c8volt expect process-instance --key <process-instance-key> --incident true
c8volt expect process-instance --key <process-instance-key> --incident false
c8volt expect pi --key <process-instance-key> --incident true
c8volt expect pi --key <process-instance-key> --state active --incident false
c8volt get pi --key <process-instance-key> --keys-only | c8volt expect pi --incident true -
```

## Flags

| Flag | Values | Required | Behavior |
|------|--------|----------|----------|
| `--key`, `-k` | process-instance key strings | Required unless stdin `-` supplies keys | Selects process instances to monitor |
| `--state`, `-s` | existing process-instance state values | Required only when `--incident` is absent | Existing state expectation; current absent and canceled/terminated compatibility remains unchanged |
| `--incident` | exactly `true` or `false` | Required only when `--state` is absent | Waits for the selected present process instances to match the requested incident marker |
| `-` positional target | stdin keys | Optional | Reads process-instance keys from stdin through existing key pipelining |

At least one expectation flag, `--state` or `--incident`, must be present.

## Success Behavior

- `--incident true` succeeds when every selected process instance is present and has `Incident: true`.
- `--incident false` succeeds when every selected process instance is present and has `Incident: false`.
- `--state` alone preserves existing behavior.
- `--state` and `--incident` together succeed only when every selected process instance satisfies both expectations.
- Multiple selected process instances succeed only after all selected instances satisfy the requested expectations.

## Failure And Waiting Behavior

- Missing process instances keep waiting for `--incident true`.
- Missing process instances keep waiting for `--incident false`.
- Missing process instances preserve existing `--state absent` behavior when state absence is requested.
- Invalid incident values fail through the standard invalid-input path.
- Invocations without `--state` and without `--incident` fail through the standard local precondition path.
- Existing timeout, context cancellation, backend error, shared envelope, and rendering behavior remain authoritative.

## Documentation Contract

- Help for `expect process-instance` and `expect pi` documents `--incident true|false`.
- Examples include a direct `--incident` wait and a stdin key pipelining example.
- README/docs examples are updated where process-instance expectation examples are listed.
- Generated CLI documentation is refreshed from command metadata.
