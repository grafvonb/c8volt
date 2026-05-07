# CLI Contract: Process Instance Variable Updates

## Scope

This contract covers user-visible behavior for:

- `c8volt update`
- `c8volt update process-instance`
- `c8volt update pi`

## Valid Invocations

```bash
c8volt update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}'
c8volt update pi --key <process-instance-key> --vars '{"customerTier":"gold"}'
c8volt update pi --key <key-a> --key <key-b> --vars '{"customerTier":"gold"}'
printf '%s\n' <key-a> <key-b> | c8volt update pi - --vars '{"customerTier":"gold"}'
printf '%s\n' <key-a> | c8volt update pi --key <key-b> - --vars '{"customerTier":"gold"}'
c8volt update pi --key <process-instance-key> --vars '{"customerTier":"gold"}' --no-wait
c8volt --json update pi --key <process-instance-key> --vars '{"customerTier":"gold"}'
```

## Invalid Invocations

```bash
c8volt update pi --key <process-instance-key>
c8volt update pi --key <process-instance-key> --vars 'not-json'
c8volt update pi --key <process-instance-key> --vars '["not","object"]'
c8volt update pi --vars '{"customerTier":"gold"}'
c8volt update pi --key <process-instance-key> --vars '{"customerTier":"gold"}' # with Camunda 8.7 config
```

Expected behavior:

- Missing `--vars` fails with a clear required flag or validation error.
- Malformed JSON fails before mutation.
- JSON whose top-level value is not an object fails before mutation.
- Missing keys from both `--key` and stdin `-` fails through existing target-selector validation.
- Camunda 8.7 fails with an unsupported-version error before mutation.

## Mutation Behavior

The command applies the same parsed variable map to every unique selected process instance.

For Camunda 8.8 and 8.9, each process instance key is used as the `elementInstanceKey` for the generated client call backing:

```text
PUT /element-instances/{elementInstanceKey}/variables
```

The generated method may be named `CreateElementInstanceVariables...`; the external behavior is variable update.

## Default Confirmation Behavior

Unless `--no-wait` is supplied:

- the command waits for every requested variable name to be visible through the same lookup path as `get process-instance --key <key> --with-vars`;
- only requested variable names are checked;
- requested and observed values are compared as normalized JSON values;
- unrelated variables do not affect success;
- timeout or retry exhaustion reports confirmation failure for the affected key.

## No-Wait Behavior

With `--no-wait`:

- the command returns after the mutation request is accepted;
- output reports submitted/accepted status;
- no visibility confirmation is attempted;
- mutation failures are still reported per key.

## Bulk Behavior

When multiple unique keys are selected:

- the same variable map is applied to every key;
- results are reported independently per process instance;
- worker fan-out, `--workers`, `--fail-fast`, and `--no-worker-limit` follow existing command semantics where applicable;
- duplicate keys are updated once.

## Human Output

Human output must be compact and per-key. Exact wording may follow existing result rendering, but it must distinguish:

- submitted/accepted without waiting;
- confirmed success;
- mutation failure;
- confirmation failure;
- unsupported version.

## JSON Output

JSON output must be script-safe and include one result per selected key with enough fields to distinguish:

- key;
- mutation accepted status;
- confirmation status;
- skipped confirmation when `--no-wait` is used;
- error details when a key fails.

## Command Metadata

`update process-instance` / `update pi` must be marked:

- state-changing;
- contract-supported where the repository exposes command contracts;
- automation-compatible following existing metadata patterns.
