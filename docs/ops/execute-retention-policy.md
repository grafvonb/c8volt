---
title: "Execute Retention Policy"
permalink: /ops/execute-retention-policy/
parent: "C8 Ops CLI"
nav_order: 2
has_toc: true
---

# c8volt ops execute retention-policy

## Purpose

Retention cleanup is simple to describe but risky to perform by hand. `c8volt ops execute retention-policy` discovers finished process instances older than the requested age, freezes the set, builds the normal delete plan, and records the result.

## In Action

<img src="../../assets/screencasts/ops-execute-retention-policy.gif" alt="c8volt ops execute retention-policy demo" />

## Use When

- finished process instances should be removed by age
- cleanup must be previewed before mutation
- the run needs a Markdown or JSON audit report

## Basic Usage

```bash
c8volt ops execute retention-policy --retention-days 90 --dry-run
```

Generated reference: [ops execute retention-policy](/cli/c8volt_ops_execute_retention-policy).

## Best Variants

```bash
c8volt ops execute retention-policy --retention-days 90 --bpmn-process-id <bpmn-process-id> --state completed --limit 25 --dry-run
c8volt ops execute retention-policy --retention-days 90 --bpmn-process-id <bpmn-process-id> --state completed --limit 25 --report-file retention-report.md
```

## Built From Lower-Level Commands

```bash
c8volt get process-instance --end-date-older-days <days> --keys-only
c8volt delete process-instance -
```

Generated references: [get process-instance](/cli/c8volt_get_process-instance), [delete process-instance](/cli/c8volt_delete_process-instance).

## Output And Safety

`--dry-run` reports the frozen retention set and delete plan without mutation. Real execution confirms or runs under automation, deletes through normal process-instance delete planning, waits unless disabled, and can write Markdown or JSON reports. Discovery page size and frozen scope are separate: use `--batch-size` for request size and `--limit` only when the retention scope should intentionally stop early.
