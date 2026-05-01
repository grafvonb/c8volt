# CLI Contract: `get pi --task-key`

## Supported Commands

```bash
./c8volt get pi --task-key=2251799815391233
./c8volt get process-instance --task-key=2251799815391233
./c8volt get pi --task-key=2251799815391233 --json
```

## Success Behavior

- On Camunda 8.8 and 8.9, `--task-key` resolves the user task through the native Camunda v2 user-task lookup.
- The owning process-instance key from the user-task result is passed through existing single process-instance lookup.
- Human output matches `get pi --key=<resolved-process-instance-key>` as closely as possible.
- JSON output matches the existing direct keyed lookup shape.
- Existing valid single lookup render flags, including `--with-age` and `--keys-only` where supported, remain valid with `--task-key`.

## Unsupported Version Behavior

```bash
./c8volt get pi --task-key=2251799815391233
```

On Camunda 8.7, the command fails with an explicit unsupported-version error for `--task-key`. It must not call Tasklist or Operate.

## Invalid Combinations

Each invocation below must fail before API resolution:

```bash
./c8volt get pi --task-key=2251799815391233 --key=2251799813711967
printf '2251799813711967\n' | ./c8volt get pi --task-key=2251799815391233 -
./c8volt get pi --task-key=2251799815391233 --state=active
./c8volt get pi --task-key=2251799815391233 --bpmn-process-id C88_SimpleUserTask_Process
./c8volt get pi --task-key=2251799815391233 --total
./c8volt get pi --task-key=2251799815391233 --limit=1
```

## Error Behavior

- Missing user task: clear not-found style error.
- User task without a usable process-instance key: clear resolution error.
- Process instance not found after resolution: existing direct keyed lookup error behavior.
- Process-instance API error after resolution: existing direct keyed lookup error behavior.

## Documentation Contract

- Command help includes `--task-key`.
- Examples include human and JSON task-key lookup.
- README and generated CLI docs mention Camunda 8.8/8.9 support, 8.7 unsupported behavior, and the no Tasklist/Operate fallback rule.
