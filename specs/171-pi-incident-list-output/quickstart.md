# Quickstart: Process Instance Incident List Output

## Targeted Validation

Run focused command and facade tests while implementing:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process -count=1
```

When touching versioned process-instance service code, include:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1
```

## Manual CLI Smoke Scenarios

List/search human output with direct incidents:

```bash
./c8volt get pi --incidents-only --with-incidents
./c8volt get pi --state active --with-incidents
./c8volt get pi --bpmn-process-id <id> --with-incidents
```

Keyed lookup regression:

```bash
./c8volt get pi --key <process-instance-key> --with-incidents
./c8volt get pi --key <process-instance-key> --with-incidents --json
```

JSON list/search enrichment:

```bash
./c8volt get pi --json --with-incidents
./c8volt get pi --state active --json --with-incidents
```

Human truncation:

```bash
./c8volt get pi --with-incidents --incident-message-limit 40
```

Validation errors:

```bash
./c8volt get pi --total --with-incidents
./c8volt get pi --incident-message-limit 40
./c8volt get pi --with-incidents --incident-message-limit -1
```

Walk prefix regression:

```bash
./c8volt walk pi --key <process-instance-key> --with-incidents
```

## Final Validation

Before committing implementation work, run:

```bash
make test
```
