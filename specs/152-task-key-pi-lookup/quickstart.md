# Quickstart: Resolve Process Instance From User Task Key

## Manual Verification Scenarios

1. Configure c8volt for Camunda 8.9 with a user task whose owning process instance is known.
2. Run:

   ```bash
   ./c8volt get pi --has-user-tasks=2251799815391233
   ./c8volt get pi --has-user-tasks=2251799815391233 --has-user-tasks=2251799815391244
   ./c8volt get pi --key=<resolved-process-instance-key>
   ```

3. Confirm the has-user-tasks output matches direct keyed process-instance output for the resolved key or keys.
4. Run:

   ```bash
   ./c8volt get pi --has-user-tasks=2251799815391233 --json
   ```

5. Confirm the JSON shape matches direct keyed lookup for the resolved key.
6. Repeat the successful lookup on Camunda 8.8.
7. Configure Camunda 8.7 and confirm:

   ```bash
   ./c8volt get pi --has-user-tasks=2251799815391233
   ```

   fails explicitly as unsupported without Tasklist or Operate fallback usage.

## Validation Commands

Run targeted checks first:

```bash
go test ./cmd ./c8volt/process ./c8volt/task ./internal/services/usertask/... ./internal/services/processinstance/... -count=1
```

Regenerate user-facing docs after command metadata changes:

```bash
make docs-content
```

Run the required broader validation before commit:

```bash
make test
```
