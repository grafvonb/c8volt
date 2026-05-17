---
title: "Retention Policy"
permalink: /ops/retention-policy/
parent: "C8 Ops CLI"
nav_order: 2
has_toc: true
---

# c8volt ops execute retention-policy

## The Problem

Retention cleanup is simple to describe but risky to perform by hand. The operator has to translate an age policy into a process-instance search, freeze the eligible set, expand it through normal delete planning, handle duplicates and hierarchy behavior, refuse unsafe mutations, and produce an audit trail.

## The Promise

`c8volt ops execute retention-policy` applies a c8volt-owned retention workflow. It discovers finished process instances older than the requested age, freezes the retention seed set, builds the normal delete plan, executes only after validation and confirmation, and records what happened.

## Use When

- deleting old completed or terminated process instances from a supported Camunda cluster
- running scheduled cleanup where the audit trail matters
- narrowing retention by BPMN process ID, process-definition key, version, version tag, state, parent, roots, children, or incident filters
- previewing old-instance cleanup before a destructive run

## Command At A Glance

```bash
c8volt ops execute retention-policy --retention-days 90 --dry-run
c8volt ops execute retention-policy --retention-days 90 --bpmn-process-id invoice-process --state completed --limit 25 --auto-confirm
c8volt ops execute retention-policy --retention-days 90 --automation --json --dry-run
c8volt ops execute retention-policy --retention-days 90 --bpmn-process-id invoice-process --state completed --limit 25 --auto-confirm --force --workers 4
c8volt ops execute retention-policy --retention-days 90 --dry-run --report-file retention-report.md
c8volt ops execute retention-policy --retention-days 90 --bpmn-process-id invoice-process --state completed --limit 25 --auto-confirm --report-file retention-report.json --report-format json
```

## Built From Lower-Level Commands

This is the conceptual flow. The ops command should use c8volt services and facades rather than shelling out to these commands.

```bash
c8volt get pi --end-date-older-days <days> --keys-only [filters...]
c8volt delete pi -
```

The command derives an `endDate <= <boundary>` filter from `--retention-days` and the command start time. It intentionally does not use Camunda native retention policies or Camunda batch deletion APIs. c8volt owns discovery, delete planning, confirmation, waiting, concurrency, and reporting.

## Workflow

```text
validate retention age and selection filters
        |
        v
discover retention seed process instances
        |
        v
freeze discovered seed keys
        |
        v
build c8volt delete plan
        |
        v
resolve roots, descendants, duplicates, and non-final blockers
        |
        +--> --dry-run: report plan, mutate nothing
        |
        v
confirm or run under automation
        |
        v
delete according to existing delete behavior
        |
        v
write outcome and optional audit report
```

## Dry Run

`--dry-run` performs discovery, delete planning, and validation without deleting or canceling process instances. It should show retention days, the derived end-date boundary when available, selection filters, discovered retention seed count, resolved root count, affected process-instance count, duplicate handling, non-final blockers, missing ancestor warnings, and report path or format when supplied.

Verbose output can list the actual seed keys, root keys, affected keys, and blocked keys.

## Real Execution

Real execution operates on the retention seed set discovered at command start. It does not chase newly eligible process instances after discovery.

Deletion reuses existing process-instance delete execution. Before deletion, the command resolves each retention seed to its ancestry root. Seeds whose roots are not final are skipped. Final roots are expanded to descendants; if that final-root scope still contains non-final descendants, the command refuses mutation unless `--force` is supplied.

## Reports

Reports should distinguish retention seeds, resolved roots, and affected process-instance family keys. They should include retention days, derived boundary, filters, discovery status, delete plan, duplicate summary, final-state and non-final counts, missing ancestors, automation flags, per-key or per-batch delete status, errors, timestamps, duration, and final outcome.

Suggested outcomes are `planned`, `deleted`, `partially_failed`, and `failed`.

## Demo

The initial VHS source is `demos/vhs/ops-retention-policy.tape`.

The demo should show the preview-first path:

```bash
c8volt ops execute retention-policy --retention-days 90 --dry-run
c8volt ops execute retention-policy --retention-days 90 --state completed --limit 25 --auto-confirm --report-file /tmp/c8volt-vhs/reports/retention-report.md
```

## Failure And Safety Notes

- `--retention-days` is required and must be a non-negative integer.
- Explicit process-instance keys should not be used as retention selectors.
- Process instances without an end date are excluded by the end-date filter behavior.
- Non-final affected process instances should block deletion unless existing delete controls explicitly allow cancellation before deletion.
- Existing report files should be preserved for dry-run, aborted, unconfirmed, or locally blocked runs.

## Related Commands

- [get process-instance](/cli/c8volt_get_process-instance/)
- [delete process-instance](/cli/c8volt_delete_process-instance/)
