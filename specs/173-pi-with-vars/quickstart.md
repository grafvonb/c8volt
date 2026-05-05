# Quickstart: Process Instance Variable Output

## Manual Verification Scenarios

1. Inspect variables for one process instance:

   ```bash
   ./c8volt get pi --key 2251799813711967 --with-vars
   ```

   Expected: the normal process-instance row appears first, followed by sorted indented variable lines.

2. Inspect variables for multiple process instances:

   ```bash
   ./c8volt get pi --key 2251799813711967 --key 2251799813711977 --with-vars
   ```

   Expected: each process-instance row shows only variables whose `processInstanceKey` and `scopeKey` match that row's key.

3. Apply an explicit human value limit:

   ```bash
   ./c8volt get pi --key 2251799813711967 --with-vars --var-value-limit 80
   ```

   Expected: values longer than 80 characters are shortened only in human output and marked `cli-truncated`.

4. Request JSON:

   ```bash
   ./c8volt get pi --key 2251799813711967 --with-vars --json
   ```

   Expected: JSON includes each process instance and sorted variables with received values and metadata intact.

5. Verify invalid combinations:

   ```bash
   ./c8volt get pi --with-vars
   ./c8volt get pi --state active --with-vars
   ./c8volt get pi --key 2251799813711967 --with-vars --total
   ./c8volt get pi --key 2251799813711967 --var-value-limit 80
   ./c8volt get pi --key 2251799813711967 --with-vars --var-value-limit -1
   ```

   Expected: each invalid invocation fails with a clear validation error.

## Automated Verification Targets

```bash
GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetProcessInstance.*Var|TestVariableEnriched' -count=1
GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process -run 'TestClient_.*Var' -count=1
GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'Test.*Variable' -count=1
make docs-content
make test
```
