# Quickstart: Slow Process Timeline Readability

## Prerequisites

- c8volt configured for a Camunda 8.8 or 8.9 environment.
- At least one process instance with runtime element timeline detail.
- For full-timeline comparison, a process instance with multiple completed, instant, transition, active, or incident-bearing rows is preferred.

## Build

```bash
go build -o /tmp/c8volt-slow-timeline .
```

## Manual Validation Scenarios

### 1. Default human output shows hotspot summary

```bash
/tmp/c8volt-slow-timeline ops analyse slow-process-instances --key <process-instance-key>
```

Expected:

- The process-instance root row is printed.
- A nested `slowest elements:` section appears when detail rows exist.
- Completed contributors at or above 1% process-duration share are visible.
- Active and incident-bearing rows are visible even if below 1%.
- Completed sub-1% rows and instant timeline noise are omitted unless active or incident-bearing.
- Output includes `hidden: ... use --with-full-timeline` when rows are omitted.
- Output ends with `process instances: <count>`.

### 2. Full timeline restores chronological detail

```bash
/tmp/c8volt-slow-timeline ops analyse slow-process-instances \
  --key <process-instance-key> \
  --with-full-timeline
```

Expected:

- The human output renders complete chronological detail using the existing timeline row style.
- Zero-duration gateways, transitions, and sub-1% rows are visible when included by existing filters.
- Element instance keys are visible in full-timeline detail rows.
- No hidden-row summary appears.

### 3. Detail filters keep current meaning

```bash
/tmp/c8volt-slow-timeline ops analyse slow-process-instances \
  --key <process-instance-key> \
  --element-id <element-id>
```

Expected:

- The process-instance root remains visible.
- Default human summary is selected from the existing filtered detail set.
- Filtering does not create synthetic transitions.

```bash
/tmp/c8volt-slow-timeline ops analyse slow-process-instances \
  --key <process-instance-key> \
  --element-id <element-id> \
  --with-full-timeline
```

Expected:

- The same existing filter meaning is preserved.
- Full-timeline mode restores complete chronological detail within the filtered detail set.

### 4. JSON output remains stable

```bash
/tmp/c8volt-slow-timeline ops analyse slow-process-instances \
  --key <process-instance-key> \
  --json

/tmp/c8volt-slow-timeline ops analyse slow-process-instances \
  --key <process-instance-key> \
  --json \
  --with-full-timeline
```

Expected:

- Both commands emit valid JSON using the existing payload shape.
- `--with-full-timeline` does not add summary or hidden-row fields.
- Any value differences are limited to already variable live-analysis values such as captured analysis time.

### 5. Keys-only output remains stable

```bash
/tmp/c8volt-slow-timeline ops analyse slow-process-instances \
  --bpmn-process-id <bpmn-process-id> \
  --keys-only

/tmp/c8volt-slow-timeline ops analyse slow-process-instances \
  --bpmn-process-id <bpmn-process-id> \
  --keys-only \
  --with-full-timeline
```

Expected:

- Only process-instance keys are printed.
- One key appears per line.
- No summary, hidden-row, or full-timeline text appears.

### 6. American spelling exposes the same behavior

```bash
/tmp/c8volt-slow-timeline ops analyze slow-process-instances \
  --key <process-instance-key> \
  --with-full-timeline
```

Expected:

- Behavior matches `ops analyse slow-process-instances --with-full-timeline`.

## Targeted Automated Validation

```bash
go test ./cmd -run 'TestRenderOpsSlowProcessAnalysisResultHuman|TestOpsAnalyseSlowProcessInstances|TestCommandContractOpsAnalyseSlowProcessInstances|TestOpsContract' -count=1
go test ./docsgen -count=1
```

Expected:

- Renderer tests cover default summary, full timeline, hidden-row summary, active rows, incident rows, JSON stability, and keys-only stability.
- Command tests cover `--with-full-timeline` flag wiring and output-mode isolation.
- Command contract tests cover help/examples/metadata for the new flag.

## Full Validation Before Merge

```bash
make docs-content
make test
```

Expected:

- README and generated CLI docs describe the compact default view and `--with-full-timeline`.
- Full repository tests pass with the race-enabled project test target.
