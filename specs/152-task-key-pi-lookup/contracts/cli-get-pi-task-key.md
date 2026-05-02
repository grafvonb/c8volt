# CLI Contract: `get pi --has-user-tasks`

## Supported Commands

```bash
./c8volt get pi --has-user-tasks=2251799815391233
./c8volt get process-instance --has-user-tasks=2251799815391233
./c8volt get pi --has-user-tasks=2251799815391233 --has-user-tasks=2251799815391244
./c8volt get pi --has-user-tasks=2251799815391233 --json
```

## Success Behavior

- On Camunda 8.8 and 8.9, `--has-user-tasks` resolves user tasks through tenant-aware native Camunda v2 user-task search.
- The owning process-instance keys from user-task results are passed through existing keyed process-instance lookup.
- Human output matches `get pi --key=<resolved-process-instance-key>` as closely as possible.
- JSON output matches the existing direct keyed lookup shape.
- Existing valid keyed lookup render behavior, including default age output and `--keys-only`, remains valid with `--has-user-tasks`.

## Unsupported Version Behavior

```bash
./c8volt get pi --has-user-tasks=2251799815391233
```

On Camunda 8.7, the command fails with an explicit unsupported-version error for `--has-user-tasks`. It must not call Tasklist or Operate.

## Invalid Combinations

Each invocation below must fail before API resolution:

```bash
./c8volt get pi --has-user-tasks=2251799815391233 --key=2251799813711967
printf '2251799813711967\n' | ./c8volt get pi --has-user-tasks=2251799815391233 -
./c8volt get pi --has-user-tasks=2251799815391233 --state=active
./c8volt get pi --has-user-tasks=2251799815391233 --bpmn-process-id C88_SimpleUserTask_Process
./c8volt get pi --has-user-tasks=2251799815391233 --total
./c8volt get pi --has-user-tasks=2251799815391233 --limit=1
```

## Error Behavior

- Missing user task: clear not-found style error.
- User task without a usable process-instance key: clear resolution error.
- Process instance not found after resolution: existing direct keyed lookup error behavior.
- Process-instance API error after resolution: existing direct keyed lookup error behavior.

## Documentation Contract

- Command help includes `--has-user-tasks`.
- Examples include human and JSON has-user-tasks lookup.
- README and generated CLI docs mention Camunda 8.8/8.9 support, 8.7 unsupported behavior, and the no Tasklist/Operate fallback rule.
