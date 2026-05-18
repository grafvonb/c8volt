---
title: "Purge All Process Definitions"
permalink: /ops/purge-all-process-definitions/
parent: "C8 Ops CLI"
nav_order: 5
has_toc: true
---

# c8volt ops purge all-process-definitions

## The Problem

Deleting process definitions is not just a resource cleanup. A selected process-definition version can still have active process instances, so the operator needs discovery, process-instance impact planning, force gating, deletion execution, and an audit report in one workflow.

## The Promise

`c8volt ops purge all-process-definitions` discovers candidate process-definition versions with `get pd`-style selectors, freezes their keys, previews delete impact, blocks active process-instance scope unless `--force` is supplied, and deletes the selected definitions only after confirmation.

## In Action

The recording previews process-definition purge impact before deleting anything, then runs the purge with `--force`, confirms the prompt, writes an audit report, and opens the first report section. It shows the key safety point for this workflow: process definitions are selected first, but c8volt still plans and reports process-instance impact before mutation.

<img src="../../assets/screencasts/ops-purge-all-process-definitions.gif" alt="c8volt ops purge all-process-definitions demo" />

Core commands shown:

```bash
c8volt ops purge all-process-definitions --dry-run
c8volt ops purge all-process-definitions --force --report-file /tmp/c8volt-vhs/reports/process-definition-purge.md
```

## Use When

- cleaning up old process-definition versions by BPMN process ID, key, version, version tag, or latest scope
- previewing process-instance impact before deleting process definitions
- deleting process-definition versions and their affected instances in a controlled maintenance window
- producing a Markdown or JSON audit report for process-definition cleanup

## Command At A Glance

```bash
c8volt ops purge all-process-definitions --dry-run
c8volt ops purge all-process-definitions --dry-run --report-file process-definition-purge.md
c8volt ops purge all-pds --bpmn-process-id <bpmn-process-id> --latest --dry-run
c8volt ops purge all-process-definitions --bpmn-process-id <bpmn-process-id> --pd-version 3 --dry-run --report-file process-definition-purge.json --report-format json
c8volt ops purge all-process-definitions --automation --json --dry-run
c8volt ops purge all-process-definitions --key <process-definition-key> --auto-confirm --force --report-file process-definition-purge.md
```

## Built From Lower-Level Commands

This is the conceptual flow. The implemented command calls c8volt services directly.

```bash
c8volt get pd [process-definition filters...]
c8volt delete pd --key <candidate-process-definition-key>
```

The command supports `--key`, `--bpmn-process-id`, `--pd-version`, `--pd-version-tag`, and `--latest` for candidate discovery. Execution controls include `--workers`, `--no-worker-limit`, `--fail-fast`, `--no-wait`, `--force`, `--automation`, `--json`, `--report-file`, and `--report-format`.

## Workflow

```text
discover candidate process definitions
        |
        v
freeze unique process-definition keys
        |
        v
preview delete-pd impact
        |
        v
count affected and active process instances
        |
        +--> --dry-run: report preview, mutate nothing
        |
        v
block active process-instance impact unless --force is set
        |
        v
confirm, auto-confirm, or automation-confirm
        |
        v
delete selected process definitions
        |
        v
write optional audit report
```

## Dry Run

`--dry-run` discovers matching process definitions and runs the delete preview. Human output shows candidate process-definition count, grouped BPMN/version impact when available, and a delete preview with affected process-instance count. If active instances are in scope, output reports that `--force` is required before deletion.

Verbose output lists candidate process-definition details and planned keys.

## Real Execution

Without `--dry-run`, the command plans the same frozen candidate set before mutation. If the run is interactive, it prompts with candidate and affected counts. If active process instances are affected and `--force` is not set, the command fails before submitting deletion.

Deletion uses the existing process-definition deletion service. With `--force`, affected active process instances may be canceled according to existing delete process-definition behavior. `--no-wait` returns after accepted deletion requests.

## Reports

Reports use schema version `ops.all-process-definitions.v1`. They include selection filters, latest-only scope, candidate process-definition keys and details, duplicate candidates, delete-plan items, affected and active process-instance counts, deletion items, no-wait/force/fail-fast flags, notices, errors, and final outcome.

Report format is inferred from `--report-file` unless `--report-format markdown|json` is supplied.

## Failure And Safety Notes

- `--pd-version` must be positive when supplied.
- `--workers` must be positive when supplied.
- Active process-instance impact blocks mutation unless `--force` is set.
- `--latest` narrows candidate discovery to latest matching process definitions.
- Existing report files are preserved unless the run is already confirmed for mutation.

## Related Commands

- [ops purge all-process-definitions](/cli/c8volt_ops_purge_all-process-definitions/)
- [get process-definition](/cli/c8volt_get_process-definition/)
- [delete process-definition](/cli/c8volt_delete_process-definition/)
