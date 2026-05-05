# Quickstart: Process Instance Incident Expectation

## Manual Verification Scenarios

1. Wait for an incident to appear:

   ```bash
   c8volt expect pi --key <process-instance-key> --incident true
   ```

   Expected: the command waits until the selected process instance is present with `Incident: true`, then exits successfully.

2. Wait for incident absence:

   ```bash
   c8volt expect pi --key <process-instance-key> --incident false
   ```

   Expected: the command waits until the selected process instance is present with `Incident: false`; a missing instance does not satisfy this condition.

3. Combine state and incident expectations:

   ```bash
   c8volt expect pi --key <process-instance-key> --state active --incident true
   ```

   Expected: the command succeeds only after the selected process instance is active and incident-bearing.

4. Pipe keys from `get pi`:

   ```bash
   c8volt get pi --key <process-instance-key> --keys-only | c8volt expect pi --incident true -
   ```

   Expected: keys are read from stdin and `--key` is not required.

5. Verify invalid input:

   ```bash
   c8volt expect pi --key <process-instance-key> --incident maybe
   ```

   Expected: the command fails with a clear invalid-input message.

6. Verify missing expectation validation:

   ```bash
   c8volt expect pi --key <process-instance-key>
   ```

   Expected: the command fails clearly because at least one expectation flag is required.

## Automated Validation

Run focused tests while implementing:

```bash
go test ./cmd -run 'TestExpect|TestCommandContract' -count=1
go test ./c8volt/process -run 'TestClient_.*Incident|TestClient_.*Wait' -count=1
go test ./internal/services/processinstance/waiter -count=1
go test ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1
```

Run final validation before commit:

```bash
make docs-content
make test
```
