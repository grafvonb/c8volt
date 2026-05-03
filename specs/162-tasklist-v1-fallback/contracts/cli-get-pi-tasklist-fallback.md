# CLI Contract: `get pi --has-user-tasks` Tasklist Fallback

## Supported Commands

```bash
./c8volt get pi --has-user-tasks=2251799815391233
./c8volt get process-instance --has-user-tasks=2251799815391233
./c8volt get pi --has-user-tasks=2251799815391233 --has-user-tasks=2251799815391244
./c8volt get pi --has-user-tasks=2251799815391233 --json
```

## Success Behavior

- On Camunda 8.8 and 8.9, each task key is resolved through the primary Camunda v2 user-task lookup first.
- If the primary lookup resolves the task, Tasklist V1 fallback is not called.
- If the primary lookup returns not found or an empty result for that task key, Tasklist V1 fallback is attempted.
- If fallback resolves the task, the owning process-instance key is passed through existing keyed process-instance lookup.
- Human output matches `get pi --key=<resolved-process-instance-key>` as closely as possible.
- JSON output matches the existing direct keyed lookup shape.
- Existing valid keyed lookup render behavior, including default age output and `--keys-only`, remains valid with `--has-user-tasks`.

## Fallback Miss Behavior

```bash
./c8volt get pi --has-user-tasks=2251799815391233
```

If both primary lookup and Tasklist V1 fallback cannot find the task, the command returns the existing not-found style error for task-key lookup.

## Unsupported Version Behavior

```bash
./c8volt get pi --has-user-tasks=2251799815391233
```

On Camunda 8.7, the command fails with an explicit unsupported-version error for `--has-user-tasks`. It must not call either the primary lookup or Tasklist V1 fallback.

## Terminal Error Behavior

- Primary lookup auth, config, malformed response, network, or server errors fail immediately and do not trigger fallback.
- Fallback lookup auth, config, malformed response, network, or server errors are surfaced to the user and are not reported as not found.
- A primary or fallback task result without a usable process-instance key fails with a clear resolution error.
- Process instance not found after resolution uses existing direct keyed lookup error behavior.
- Process-instance API errors after resolution use existing direct keyed lookup error behavior.

## Invalid Combinations

Each invocation below must fail before task or process-instance API resolution:

```bash
./c8volt get pi --has-user-tasks=2251799815391233 --key=2251799813711967
printf '2251799813711967\n' | ./c8volt get pi --has-user-tasks=2251799815391233 -
./c8volt get pi --has-user-tasks=2251799815391233 --state=active
./c8volt get pi --has-user-tasks=2251799815391233 --bpmn-process-id C88_SimpleUserTask_Process
./c8volt get pi --has-user-tasks=2251799815391233 --total
./c8volt get pi --has-user-tasks=2251799815391233 --limit=1
```

## Documentation Contract

- Command help includes `--has-user-tasks` and describes primary lookup followed by Tasklist V1 fallback for Camunda 8.8 and 8.9.
- README and generated CLI docs mention Camunda 8.8/8.9 support, Camunda 8.7 unsupported behavior, and Tasklist V1 fallback for legacy user-task compatibility.
- Documentation must mention that Tasklist V1 fallback is transitional compatibility for deprecated upstream behavior.
