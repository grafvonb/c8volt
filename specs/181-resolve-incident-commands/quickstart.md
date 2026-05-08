# Quickstart: Resolve Incident Commands

## Targeted Validation

Run focused tests while implementing each slice:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./internal/services/incident/... ./c8volt/process ./cmd -run 'TestResolve|TestIncident|TestCommandContract' -count=1
```

Run broader validation before commit:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/incident/... ./docsgen -count=1
make docs-content
make test
```

## Manual Smoke Examples

Resolve one known incident key and wait for confirmation:

```bash
./c8volt resolve incident --key 2251799813685249
```

Preview one known incident key without mutation:

```bash
./c8volt resolve incident --key 2251799813685249 --dry-run
```

Resolve repeated and stdin incident keys:

```bash
printf '%s\n' 2251799813685249 2251799813685251 | ./c8volt resolve inc --key 2251799813685249 -
```

Resolve incidents discovered for one process instance:

```bash
./c8volt resolve pi --key 2251799813685250
```

Preview process-instance incident resolution without mutation:

```bash
./c8volt resolve pi --key 2251799813685250 --dry-run
```

Submit process-instance incident resolution without waiting:

```bash
./c8volt --json resolve process-instance --key 2251799813685250 --no-wait
```

Render a stable JSON dry-run plan:

```bash
./c8volt --json resolve process-instance --key 2251799813685250 --dry-run
```

Verify unsupported-version behavior with a Camunda 8.7 test configuration:

```bash
./c8volt resolve incident --key 2251799813685249
```

Expected result: unsupported-version error before mutation.
