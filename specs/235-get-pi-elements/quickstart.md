# Quickstart: Process Instance Element Enrichment

## Prerequisites

- c8volt configured for a Camunda 8.8 or 8.9 environment.
- The standalone runtime element command from issue #241 is available.
- A process instance with runtime element instances.
- For unsupported-version validation, a c8volt config targeting Camunda 8.7.

## Build

```bash
go build -o /tmp/c8volt-get-pi-elements .
```

## Manual Validation Scenarios

### 1. Fetch one process instance with elements

```bash
/tmp/c8volt-get-pi-elements get pi --key <process-instance-key> --with-elements
```

Expected:

- One process-instance row is printed.
- An `elements:` section appears below the process instance when matching runtime elements exist.
- Element rows use `<elementInstanceKey> <type> <elementId> <state> s:<startDate> [e:<endDate>] [inc!]`.

### 2. Search process instances with elements

```bash
/tmp/c8volt-get-pi-elements get pi --state active --limit 5 --with-elements
```

Expected:

- At most five process instances are returned.
- Each returned process instance is enriched independently.
- `found: N` counts process instances, not element rows.

### 3. Search by BPMN process identifier with elements

```bash
/tmp/c8volt-get-pi-elements get pi -b <bpmn-process-id> --limit 5 --with-elements
```

Expected:

- Existing BPMN process selector validation still applies.
- Selected process instances preserve existing process-instance search behavior.
- Attached elements do not change process-instance paging or limits.

### 4. Combine elements with variables and incidents

```bash
/tmp/c8volt-get-pi-elements get pi --key <process-instance-key> --with-vars --with-incidents --with-elements
```

Expected:

- Sections render in this order when data exists: `vars:`, `incidents:`, `elements:`.
- No duplicate sections appear.
- Missing sections are omitted according to existing behavior.

### 5. Reject total with element enrichment

```bash
/tmp/c8volt-get-pi-elements get pi --state active --with-elements --total
```

Expected:

- Command fails before remote lookup where validation can detect the combination.
- Error explains that `--total` cannot be combined with `--with-elements`.

### 6. Reject keys-only with element enrichment

```bash
/tmp/c8volt-get-pi-elements get pi --state active --with-elements --keys-only
```

Expected:

- Command fails before remote lookup where validation can detect the combination.
- Error explains that `--keys-only` cannot be combined with `--with-elements`.

### 7. Verify JSON payload

```bash
/tmp/c8volt-get-pi-elements get pi --key <process-instance-key> --with-elements --json
```

Expected:

- One valid shared JSON result envelope is printed.
- The payload item includes an `elements` array.
- Each element preserves fields listed in [data-model.md](data-model.md).

### 8. Unsupported Camunda 8.7

```bash
/tmp/c8volt-get-pi-elements --config <camunda-87-config> get pi --key <process-instance-key> --with-elements
```

Expected:

- Command fails with a clear unsupported-version error.
- No success output is printed.

## Targeted Automated Validation

```bash
go test ./internal/services/processinstance -run 'TestEnrichProcessInstancesWithElements' -count=1
go test ./c8volt/process -run 'TestClient_EnrichProcessInstancesWithElements' -count=1
go test ./cmd -run 'TestGetProcessInstance.*Elements|TestProcessInstanceActivity.*Elements|TestCommandCapabilityForCommand_GetProcessInstance' -count=1
go test ./docsgen -count=1
```

## Full Validation Before Merge

```bash
make docs-content
make test
```

Expected:

- Generated CLI docs include `--with-elements` and examples.
- README-facing examples describe element enrichment.
- Full repository tests pass with the race-enabled project test target.
