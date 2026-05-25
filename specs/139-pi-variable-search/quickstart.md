# Quickstart: Native Process Instance Variable Search

## Setup

Build a local binary or run tests through the repository's existing Go test commands. Use a config profile for Camunda 8.8 or 8.9 when exercising live behavior.

## Manual Verification Scenarios

### Existence Search

```bash
./c8volt get pi --var-exists customerId --limit 5
```

Expected:

- Output includes only process instances where `customerId` exists.
- JSON and keys-only modes remain clean when requested.

### Equality Search

```bash
./c8volt get pi --var 'status="approved"' --limit 5
```

Expected:

- Output includes only process instances where `status` equals the serialized approved value.

### Combined Equality Search

```bash
./c8volt get pi --var 'status="canceled",payload="payload"' --limit 5
```

Expected:

- Both variable clauses are applied together.
- The comma between clauses splits correctly.

### Array Operator Search

```bash
./c8volt get pi --var 'status.$in=["approved","pending"]' --limit 5
```

Expected:

- The array remains one clause.
- Matching process instances have `status` in the supplied set.

### Like Search

```bash
./c8volt get pi --var-like 'email=*@example.com' --limit 5
```

Expected:

- Native wildcard behavior is used.
- The command does not add extra wildcards.

### Unsupported Version

```bash
./c8volt --camunda-version 8.7 get pi --var-exists customerId
```

Expected:

- The command fails with an explicit unsupported-version message.
- Existing `get pi` searches without variable flags continue to behave as before.

## Automated Validation Targets

Run targeted tests first:

```bash
go test ./cmd -run 'TestGetProcessInstance|TestCommandContract' -count=1
go test ./c8volt/process -run 'TestClient_SearchProcessInstances' -count=1
go test ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'Test.*Variable' -count=1
go test ./internal/services/processinstance/v87 -run 'Test.*Unsupported' -count=1
```

Refresh docs after command metadata or help changes:

```bash
make docs-content
```

Before commit readiness:

```bash
make test
```

## Ralph Launch Requirement

Any Ralph launch for this feature must include:

```text
--implementation-context specs/ralph-implementation-rules.md
```

Do not start Ralph for this feature without that implementation context.
