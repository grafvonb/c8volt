# Quickstart: Runtime Element Instance Command

## Prerequisites

- c8volt configured for a Camunda 8.8 or 8.9 environment with runtime element instances available.
- A known `processInstanceKey` and at least one known `elementInstanceKey`.
- For unsupported-version validation, a c8volt config targeting Camunda 8.7.

## Build

```bash
go build -o /tmp/c8volt-get-element .
```

## Manual Validation Scenarios

### 1. Fetch one element instance

```bash
/tmp/c8volt-get-element get element --key <element-instance-key>
```

Expected:

- One compact row for the requested element instance.
- The element instance key is the first column.
- No `found:` line is required for single-item output.

### 2. Search by process instance

```bash
/tmp/c8volt-get-element get element --pi-key <process-instance-key> --limit 10
```

Expected:

- Rows include only element instances for the selected process instance.
- Output ends with `found: N`.
- Looped or multi-instance elements appear as distinct rows when the runtime contains them.

### 3. Combine search filters

```bash
/tmp/c8volt-get-element get element --pi-key <process-instance-key> --element-id <bpmn-element-id> --state active --limit 10
```

Expected:

- Every returned row matches all supplied filters.
- The command does not treat filters as OR conditions.

### 4. Reject direct lookup mixed with search filters

```bash
/tmp/c8volt-get-element get element --key <element-instance-key> --pi-key <process-instance-key>
```

Expected:

- Command fails before lookup.
- Error explains that `--key` cannot be combined with search filters.

### 5. Keys-only output

```bash
/tmp/c8volt-get-element get element --pi-key <process-instance-key> --limit 5 --keys-only
```

Expected:

- Only element instance keys are printed, one per line.
- No `found:` line or diagnostics.

### 6. Total-only output

```bash
/tmp/c8volt-get-element get element --pi-key <process-instance-key> --total
```

Expected:

- Only a numeric total is printed.

### 7. JSON output

```bash
/tmp/c8volt-get-element get element --pi-key <process-instance-key> --limit 5 --json
```

Expected:

- One valid shared JSON result envelope.
- The envelope `payload` contains `total` and `items`.
- Each item includes stable element fields listed in [data-model.md](data-model.md).

### 8. Incident markers

```bash
/tmp/c8volt-get-element get element --pi-key <process-instance-key> --limit 10
```

Expected when matching elements have incidents:

- Rows with incidents include `inc!` when no incident key is available.
- Rows include `inc!:<incidentKey>` when an incident key is available.
- Rows never include both incident markers for the same element instance.

### 9. Unsupported Camunda 8.7

```bash
/tmp/c8volt-get-element --config <camunda-87-config> get element --key <element-instance-key>
```

Expected:

- Command fails with a clear unsupported-version error.
- No success output is printed.

## Targeted Automated Validation

```bash
go test ./internal/services/element/... -count=1
go test ./c8volt/element -count=1
go test ./cmd -run 'TestGetElement|TestElement|TestCommandContract' -count=1
go test ./docsgen -count=1
```

## Full Validation Before Merge

```bash
make docs-content
make test
```

Expected:

- Generated CLI docs include `get element` command metadata and examples.
- Full repository tests pass with the race-enabled project test target.
