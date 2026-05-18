---
title: "Purge Process Instances With Incidents"
permalink: /ops/purge-process-instances-with-incidents/
parent: "C8 Ops CLI"
nav_order: 4
has_toc: true
---

# c8volt ops purge process-instances-with-incidents

## The Problem

Incident cleanup often begins with incident filters, not process-instance keys. A manual pipeline can find matching incidents and pass their process-instance keys into deletion, but that leaves the operator responsible for dedupe, family-scope planning, non-final blockers, confirmation, and audit output.

## The Promise

`c8volt ops purge process-instances-with-incidents` discovers candidate incidents, extracts candidate process-instance keys, freezes that set, then runs the same deterministic family-scope delete planning used by `delete pi`.

Aliases: `pi-with-incidents`, `piwi`.

## Use When

- deleting process-instance families selected by incident type or message
- cleaning failed test data by BPMN process ID and incident state
- previewing the delete impact of incident-based selection before mutation
- producing an audit report that separates incident candidates from delete scope

## Command At A Glance

```bash
c8volt ops purge process-instances-with-incidents --dry-run
c8volt ops purge process-instances-with-incidents --state active --error-type io_mapping_error --dry-run
c8volt ops purge pi-with-incidents --state active --limit 5 --dry-run
c8volt ops purge process-instances-with-incidents --automation --json --dry-run
c8volt ops purge process-instances-with-incidents --dry-run --report-file incident-purge.md
c8volt ops purge process-instances-with-incidents --state active --error-type job_no_retries --limit 5 --auto-confirm --force
c8volt ops purge process-instances-with-incidents --state active --error-type job_no_retries --limit 5 --auto-confirm --force --workers 4 --report-file incident-purge.json --report-format json
```

## Built From Lower-Level Commands

This is the conceptual flow. The ops command should use c8volt services and facades rather than shelling out to these commands.

```bash
c8volt get incident <incident-filters> --pi-keys-only
c8volt delete pi -
```

The command defaults to `--state active`; `--key` selects incident keys, not process-instance keys. The full command and aliases are equivalent:

```bash
c8volt ops purge process-instances-with-incidents
c8volt ops purge pi-with-incidents
c8volt ops purge piwi
```

## Workflow

```text
search or fetch candidate incidents
        |
        v
extract candidate process-instance keys
        |
        v
dedupe and freeze candidate process instances
        |
        v
build normal c8volt delete plan
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

`--dry-run` searches incidents, extracts candidate process-instance keys, runs delete preflight, and reports the planned purge without mutation.

Normal output should emphasize counts:

```text
dry run: purge process-instances with incidents
candidate incidents: N
candidate process instances: M
delete plan: planned (candidate process instances: M, roots: R, affected process instances: A)
outcome: planned; no changes applied; use --verbose to list process-instance keys
```

Verbose output can list incident keys, candidate process-instance keys, resolved root keys, affected keys, and blocked keys.

## Real Execution

Real execution deletes the process-instance family scope produced by existing delete planning. Matching an incident is only the discovery step. The delete phase must still honor root resolution, descendant traversal, duplicate removal, non-final checks, `--force` cancel-before-delete behavior, `--no-wait`, workers, fail-fast, and confirmation.

If no incidents match, no delete request should be submitted.

## Reports

Reports should preserve the full chain: incident selection filters, incident discovery result, candidate process-instance set, delete plan, deletion result, notices, errors, automation flags, timestamps, duration, and final outcome.

Human-facing Markdown should use candidate terminology: candidate incidents, candidate process instances, resolved roots, and affected process instances.

## Demo

The initial VHS source is `demos/vhs/ops-purge-process-instances-with-incidents.tape`.

The demo should show the preview-first path:

```bash
c8volt ops purge process-instances-with-incidents --state active --error-type io_mapping_error --dry-run
c8volt ops purge process-instances-with-incidents --state active --error-type io_mapping_error --limit 5 --auto-confirm --report-file /tmp/c8volt-vhs/reports/incident-purge.md
```

## Failure And Safety Notes

- `--key` means incident key, not process-instance key.
- Duplicate incidents for the same process instance must not create duplicate delete submissions.
- Non-final affected process instances should block deletion unless `--force` is supplied.
- Help examples should teach safe automation first and avoid bare destructive automation examples.
- JSON and reports should retain complete notice data even when human output stays compact.

## Related Commands

- [get incident](/cli/c8volt_get_incident/)
- [delete process-instance](/cli/c8volt_delete_process-instance/)
