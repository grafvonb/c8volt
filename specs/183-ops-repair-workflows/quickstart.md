# Quickstart: Ops Repair Workflows

This quickstart describes expected operator behavior for validation after implementation.

## Build

```bash
go build -o /tmp/c8volt-ops-repair .
```

## Inspect Command Help

```bash
/tmp/c8volt-ops-repair ops repair --help
/tmp/c8volt-ops-repair ops repair incident --help
/tmp/c8volt-ops-repair ops repair process-instance --help
```

Expected:

- `ops repair` remains a grouping command and exposes no top-level `--key`.
- Incident repair documents explicit keys, stdin keys, incident search filters, dry-run, variable flags, job flags, bulk controls, automation, and reports.
- Process-instance repair documents explicit keys, stdin keys, process-instance search filters, required incident-bearing selectors for search mode, dry-run, variable flags, job flags, bulk controls, automation, and reports.

## Preview Incident Repair

```bash
/tmp/c8volt-ops-repair ops repair incident --state active --error-type io_mapping_error --limit 25 --dry-run
```

Expected:

- Discovery and validation run.
- No variable, job, or incident mutation is submitted.
- Output shows frozen incident keys, process-instance keys, job keys where present, job applicability, retry/timeout requests, and resolution targets.

## Repair Explicit Incidents

```bash
/tmp/c8volt-ops-repair ops repair incident --key "$INCIDENT_A" --key "$INCIDENT_B" --auto-confirm
printf '%s\n' "$INCIDENT_A" "$INCIDENT_B" | /tmp/c8volt-ops-repair ops repair incident - --auto-confirm
```

Expected:

- The command freezes the explicit incident keys before mutation.
- Job-backed incidents receive applicable retry and timeout work.
- Non-job incidents show job steps as `not_applicable`.
- Incident resolution is confirmed before success is reported.

## Preview Process-Instance Repair

```bash
/tmp/c8volt-ops-repair ops repair process-instance --state active --incidents-only --bpmn-process-id order-process --limit 25 --dry-run
```

Expected:

- Process instances are selected only with an incident-bearing selector.
- Active incidents for the selected process instances are frozen before mutation.
- Duplicate incident references are repaired once.

## Repair With Variables And Report

```bash
/tmp/c8volt-ops-repair ops repair incident \
  --state active \
  --error-type io_mapping_error \
  --vars '{"customerStatus":"verified"}' \
  --job-timeout 30s \
  --report-file repair-report.json \
  --report-format json \
  --auto-confirm
```

Expected:

- Variables are updated once per unique process-instance scope.
- Only requested variable names are confirmed.
- Incidents dependent on failed variable scopes are blocked from resolution.
- Timeout is applied only to incidents with related job keys.
- The JSON audit report includes discovery, frozen targets, step statuses, notices, errors, and final outcome.

## Validation Commands

Run focused tests first, then broader validation once the work unit is complete:

```bash
go test ./cmd -run 'TestOpsRepair|TestCommandContract' -count=1
go test ./c8volt/ops ./internal/services/ops -count=1
go test ./internal/services/incident ./internal/services/processinstance ./internal/services/job -count=1
make docs-content
make test
```
