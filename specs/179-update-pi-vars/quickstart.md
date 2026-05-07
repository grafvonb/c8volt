# Quickstart: Process Instance Variable Updates

## Targeted Validation

Run focused command, facade, and versioned service tests while implementing:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1
```

When iterating on waiter behavior, include:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processinstance/waiter ./c8volt/process ./cmd -count=1
```

## Manual CLI Smoke Scenarios

Single-key update with confirmation:

```bash
./c8volt update pi --key <process-instance-key> --vars '{"customerTier":"gold"}'
./c8volt get pi --key <process-instance-key> --with-vars
```

Full command name:

```bash
./c8volt update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}'
```

Multiple keys:

```bash
./c8volt update pi --key <key-a> --key <key-b> --vars '{"customerTier":"gold"}'
```

Stdin keys:

```bash
printf '%s\n' <key-a> <key-b> | ./c8volt update pi - --vars '{"customerTier":"gold"}'
```

Accepted/submitted without confirmation:

```bash
./c8volt update pi --key <process-instance-key> --vars '{"customerTier":"gold"}' --no-wait
```

JSON output:

```bash
./c8volt --json update pi --key <process-instance-key> --vars '{"customerTier":"gold"}'
./c8volt --json update pi --key <process-instance-key> --vars '{"customerTier":"gold"}' --no-wait
```

Validation errors:

```bash
./c8volt update pi --key <process-instance-key>
./c8volt update pi --key <process-instance-key> --vars 'not-json'
./c8volt update pi --key <process-instance-key> --vars '["not","object"]'
./c8volt update pi --vars '{"customerTier":"gold"}'
```

Unsupported version:

```bash
./c8volt update pi --key <process-instance-key> --vars '{"customerTier":"gold"}'
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
