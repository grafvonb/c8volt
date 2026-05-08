# Quickstart: Job Lookup And Updates

## Targeted Validation

Run focused command, facade, and versioned service tests while implementing:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/job ./internal/services/job ./internal/services/job/v87 ./internal/services/job/v88 ./internal/services/job/v89 -count=1
```

When iterating on retry confirmation behavior, include:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./internal/services/job/waiter ./c8volt/job ./cmd -count=1
```

When checking regressions around related existing behavior, include:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./cmd ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'Test(GetProcessInstance|UpdateProcessInstance)' -count=1
```

## Manual CLI Smoke Scenarios

Lookup a job discovered from incident output:

```bash
./c8volt get pi --with-incidents --key <process-instance-key>
./c8volt get job --key <job-key>
```

Lookup in JSON mode:

```bash
./c8volt --json get job --key <job-key>
```

Update retries with confirmation:

```bash
./c8volt update job --key <job-key> --retries 3 --dry-run
./c8volt update job --key <job-key> --retries 3
./c8volt get job --key <job-key>
```

Update timeout without deadline confirmation:

```bash
./c8volt update job --key <job-key> --timeout 5m --dry-run
./c8volt update job --key <job-key> --timeout 5m
./c8volt get job --key <job-key>
```

Update retries and timeout together, confirming retries only:

```bash
./c8volt update job --key <job-key> --retries 3 --timeout 5m --dry-run
./c8volt update job --key <job-key> --retries 3 --timeout 5m
./c8volt get job --key <job-key>
```

Accepted/submitted without confirmation:

```bash
./c8volt update job --key <job-key> --retries 3 --no-wait
```

JSON update output:

```bash
./c8volt --json update job --key <job-key> --retries 3 --dry-run
./c8volt --json update job --key <job-key> --timeout 5m --dry-run
./c8volt --json update job --key <job-key> --retries 3 --auto-confirm
```

Validation errors:

```bash
./c8volt get job
./c8volt update job
./c8volt update job --key <job-key>
./c8volt update job --key <job-key> --retries invalid
./c8volt update job --key <job-key> --timeout invalid
./c8volt --json update job --key <job-key> --retries 3
./c8volt --json --verbose update job --key <job-key> --retries 3 --dry-run
```

Unsupported version:

```bash
./c8volt update job --key <job-key> --retries 3
```

Run the unsupported-version scenario with a Camunda 8.7 configuration and verify it fails before mutation.

## Documentation Validation

Regenerate generated CLI docs after command metadata and examples change:

```bash
make docs-content
```

## Final Validation

Before committing implementation work, run:

```bash
make test
```
