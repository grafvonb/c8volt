---
title: "Orphan Process Instances"
permalink: /ops/orphan-process-instances/
parent: "C8 Ops CLI"
nav_order: 3
has_toc: true
---

# c8volt ops purge orphan-process-instances

## The Problem

Orphan child process instances are hard to delete safely because the operator first has to discover children whose parents are missing, freeze that exact candidate set, validate the process-instance delete plan, and then run deletion without accidentally widening the scope during execution.

## The Promise

`c8volt ops purge orphan-process-instances` discovers orphan child process instances, freezes their keys, runs the shared process-instance delete preview, and either reports the plan with `--dry-run` or deletes only after confirmation. It supports deterministic JSON output and Markdown or JSON audit reports.

Aliases: `orphan-pi`, `opi`.

## Use When

- cleaning up child process instances whose parents are missing
- narrowing orphan cleanup by process-definition, date, state, parent, or incident filters
- previewing the affected process-instance trees before deletion
- writing an audit report for a maintenance cleanup

## Command At A Glance

```bash
c8volt ops purge orphan-process-instances --dry-run
c8volt ops purge orphan-process-instances --dry-run --bpmn-process-id <bpmn-process-id> --limit 25
c8volt ops purge orphan-process-instances --automation --json --dry-run
c8volt ops purge orphan-process-instances --state completed --limit 25 --auto-confirm
c8volt ops purge orphan-process-instances --dry-run --report-file orphan-purge.md
c8volt ops purge orphan-process-instances --state completed --limit 25 --auto-confirm --report-file orphan-purge.json --report-format json
```

## Built From Lower-Level Commands

This is the conceptual flow. The implemented command calls c8volt services directly.

```bash
c8volt get pi --orphan-children-only --keys-only [filters...]
c8volt delete pi -
```

Implemented selection controls include process-definition filters, `--pd-key`, date ranges, `--batch-size`, `--limit`, `--parent-key`, `--state`, `--incidents-only`, and `--no-incidents-only`. Execution controls include `--workers`, `--no-worker-limit`, `--fail-fast`, `--no-wait`, `--force`, `--automation`, `--json`, `--report-file`, and `--report-format`.

## Workflow

```text
discover orphan child process instances
        |
        v
freeze candidate orphan keys
        |
        v
build shared cancel/delete dry-run plan
        |
        v
resolve roots and affected family keys
        |
        +--> --dry-run: report preview, mutate nothing
        |
        v
block non-final affected scope unless --force is set
        |
        v
confirm, auto-confirm, or automation-confirm
        |
        v
delete planned root process-instance trees
        |
        v
write optional audit report
```

## Dry Run

`--dry-run` performs discovery and delete-plan validation only. Human output shows candidate orphan count and the delete preview: orphan candidates, process-instance trees, and affected process instances. Verbose output lists candidate keys.

When no orphan candidates are found, the command reports a skipped preview and exits successfully with outcome `planned`.

## Real Execution

Without `--dry-run`, the command first plans the same frozen scope. If the run is not already implicitly confirmed, it prompts with the candidate and affected counts. If the plan includes affected process instances that are not in a final state, mutation is blocked unless `--force` is supplied.

Deletion submits root process-instance trees through the existing process-instance deletion service. `--no-wait` returns after accepted deletion requests; otherwise the command reports confirmed deletion when all reports are OK.

## Reports

Reports use schema version `ops.orphan-process-instances.v1`. They include command metadata, selection filters, discovered candidate keys, delete plan root and affected keys, deletion item status, confirmation/no-wait flags, errors, and outcome.

Report format is inferred from `--report-file` unless `--report-format markdown|json` is supplied.

## Demo

The VHS source is `demos/vhs/ops-orphan-process-instances.tape`.

```bash
c8volt ops purge orphan-process-instances --dry-run
c8volt ops purge orphan-process-instances --state completed --limit 25 --auto-confirm --report-file /tmp/c8volt-vhs/reports/orphan-purge.md
```

## Failure And Safety Notes

- `--dry-run` never submits deletion requests.
- `--automation` is supported and implicitly confirms supported prompts.
- `--force` is required when the planned affected scope contains non-final process instances.
- Existing report files are preserved unless the run is already confirmed for mutation.
- `--keys-only` is supported as an output mode for the discovered orphan keys.

## Related Commands

- [ops purge orphan-process-instances](/cli/c8volt_ops_purge_orphan-process-instances/)
- [get process-instance](/cli/c8volt_get_process-instance/)
- [delete process-instance](/cli/c8volt_delete_process-instance/)
