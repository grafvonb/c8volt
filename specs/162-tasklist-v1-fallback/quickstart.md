# Quickstart: Tasklist V1 Fallback For Task-Key Process-Instance Lookup

## Goal

Verify that `get pi --has-user-tasks=<task-key>` preserves v2-first behavior and uses Tasklist V1 only when the primary lookup cannot find a task key.

## Manual Scenarios

1. Configure a Camunda 8.9 profile with both Camunda API and Tasklist API credentials.
2. Use a modern Camunda user task key visible to the primary lookup:

   ```bash
   ./c8volt get pi --has-user-tasks=<modern-task-key>
   ```

   Expected: the owning process instance renders successfully and no Tasklist fallback request is required.

3. Use a legacy job-worker-based user task key not visible to the primary lookup but visible through Tasklist V1:

   ```bash
   ./c8volt get pi --has-user-tasks=<legacy-task-key>
   ```

   Expected: the owning process instance renders successfully after fallback resolution.

4. Confirm JSON shape matches direct keyed lookup:

   ```bash
   ./c8volt get pi --has-user-tasks=<legacy-task-key> --json
   ./c8volt get pi --key=<resolved-process-instance-key> --json
   ```

   Expected: both commands use the same process-instance JSON shape.

5. Use a missing task key:

   ```bash
   ./c8volt get pi --has-user-tasks=<missing-task-key>
   ```

   Expected: the command returns the existing not-found style task lookup error after both lookup paths miss.

6. Run against Camunda 8.7:

   ```bash
   ./c8volt get pi --has-user-tasks=<task-key>
   ```

   Expected: the command returns the existing unsupported-version error before fallback lookup.

## Automated Validation Targets

Run focused tests while implementing:

```bash
go test ./internal/services/usertask/v88 ./internal/services/usertask/v89 ./internal/services/usertask/v87 -count=1
go test ./cmd -run 'HasUserTasks|Tasklist|UserTask' -count=1
```

Run broader validation before commit:

```bash
make test
```
