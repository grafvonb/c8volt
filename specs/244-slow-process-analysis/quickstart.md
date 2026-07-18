# Quickstart: Slow Process Instance Analysis

## Prerequisites

- c8volt configured for a Camunda 8.8 or 8.9 environment.
- Runtime process instances with associated runtime element instances.
- For process-definition search scenarios, a known BPMN process ID or process-definition key.
- For unsupported-version validation, a c8volt config targeting Camunda 8.7.

## Build

```bash
go build -o /tmp/c8volt-slow-pi-analysis .
```

## Manual Validation Scenarios

### 1. Analyze one process instance by key

```bash
/tmp/c8volt-slow-pi-analysis ops analyse slow-process-instances --key <process-instance-key>
```

Expected:

- One process-instance root row is printed.
- The root row includes `dur:`.
- Runtime details appear under a nested `└─ elements:` section when runtime elements exist.
- Element rows appear in chronological order using `├─` and `└─` child connectors.
- Transition timing rows use `A -> B: duration`.
- Relative-duration bars include ten visual cells plus a rounded percentile when enough comparable measurements exist.
- Output ends with the final process-instance count.

### 2. Analyze repeated keys and stdin keys

```bash
printf '<process-instance-key>\n<other-process-instance-key>\n' |
  /tmp/c8volt-slow-pi-analysis ops analyse slow-process-instances --key <process-instance-key> -
```

Expected:

- Duplicate keys are analyzed once.
- Results are ordered by available duration longest to shortest, followed by unavailable durations.
- Invalid, missing, or unauthorized keys fail instead of being silently discarded.

### 3. Pipe from process-instance keys-only output

```bash
/tmp/c8volt-slow-pi-analysis get pi --state active --keys-only |
  /tmp/c8volt-slow-pi-analysis ops analyse slow-process-instances -
```

Expected:

- The analysis accepts newline-separated keys.
- Keys-only pipeline output from `get pi` remains compatible with the analysis command.

### 4. Discover by BPMN process ID

```bash
/tmp/c8volt-slow-pi-analysis ops analyse slow-process-instances \
  --bpmn-process-id <bpmn-process-id> \
  --state all \
  --limit 10
```

Expected:

- At most ten process instances are selected for analysis.
- `--limit` affects process-instance discovery only.
- Element and transition details are not truncated by the process-instance limit.

### 5. Discover by process-definition key with date filters

```bash
/tmp/c8volt-slow-pi-analysis ops analyse slow-process-instances \
  --pd-key <process-definition-key> \
  --start-date-after 2026-07-18 \
  --end-date-before 2026-07-19
```

Expected:

- Date filters accept `YYYY-MM-DD`.
- The selected process instances are frozen before element inspection.
- Active durations use one captured analysis time.

### 6. Empty process-definition search

```bash
/tmp/c8volt-slow-pi-analysis ops analyse slow-process-instances \
  --bpmn-process-id <bpmn-process-id-with-no-matches> \
  --state completed
```

Expected:

- Command succeeds.
- Human output shows a zero process-instance count.
- No root or detail rows are rendered.

### 7. Detail filtering

```bash
/tmp/c8volt-slow-pi-analysis ops analyse slow-process-instances \
  --key <process-instance-key> \
  --element-id <element-id> \
  --duration-after 2s
```

Expected:

- Process-instance root remains visible.
- Element rows match all element predicates and the duration threshold.
- Transition rows remain visible only when at least one original endpoint matches the element predicates and the transition duration passes the threshold.
- No synthetic transition across hidden elements appears.
- Matching detail rows remain nested under the original process-instance root.

### 8. JSON output

```bash
/tmp/c8volt-slow-pi-analysis ops analyse slow-process-instances \
  --key <process-instance-key> \
  --json
```

Expected:

- Valid JSON output is printed.
- Payload includes captured analysis time, ordered process-instance items, timeline entries, durations, duration milliseconds, comparison sample counts, relative percentiles, and process-duration shares.
- Human-only compact tokens are represented as explicit fields.

### 9. Keys-only output

```bash
/tmp/c8volt-slow-pi-analysis ops analyse slow-process-instances \
  --bpmn-process-id <bpmn-process-id> \
  --keys-only
```

Expected:

- Only unique process-instance keys are printed.
- One key appears per line.
- Detail filters do not change which root keys are emitted.

### 10. American spelling alias

```bash
/tmp/c8volt-slow-pi-analysis ops analyze slow-process-instances --key <process-instance-key>
```

Expected:

- Output and validation match `ops analyse slow-process-instances`.

### 11. Invalid selection combinations

```bash
/tmp/c8volt-slow-pi-analysis ops analyse slow-process-instances \
  --key <process-instance-key> \
  --bpmn-process-id <bpmn-process-id>
```

Expected:

- Command fails before remote lookup where validation can detect the combination.
- Error explains that explicit-key mode and process-definition search mode are mutually exclusive.

### 12. Unsupported Camunda 8.7

```bash
/tmp/c8volt-slow-pi-analysis --config <camunda-87-config> \
  ops analyse slow-process-instances --key <process-instance-key>
```

Expected:

- Command fails with the established unsupported-version error.
- No partial success output is printed.

## Targeted Automated Validation

```bash
go test ./internal/services/ops -run 'TestSlowProcessAnalysis' -count=1
go test ./c8volt/ops -run 'TestClientAnalyseSlowProcessInstances' -count=1
go test ./cmd -run 'Test.*SlowProcessAnalysis|TestOps.*SlowProcess|TestCommandContract|TestOpsContract' -count=1
go test ./docsgen -count=1
```

## Full Validation Before Merge

```bash
make docs-content
make test
```

Expected:

- README-facing examples describe the slow process-instance analysis workflow.
- Generated CLI docs include both command spellings, flags, output modes, and examples.
- Full repository tests pass with the race-enabled project test target.
