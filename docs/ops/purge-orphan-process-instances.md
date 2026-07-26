---
title: "Purge Orphan Process Instances"
permalink: /ops/purge-orphan-process-instances/
parent: "C8 Ops CLI"
nav_order: 6
has_toc: true
---

# c8volt ops purge orphan-process-instances

## Purpose

Orphan child process instances are hard to delete safely by hand. `c8volt ops purge orphan-process-instances` discovers children whose parents are missing, freezes their keys, then runs the shared process-instance delete preview or execution.

## In Action

<img src="../../assets/screencasts/ops-purge-orphan-process-instances.gif" alt="c8volt ops purge orphan-process-instances demo" />

## Use When

- orphan child process instances should be found and removed
- deletion must stay limited to the frozen orphan set
- a cleanup report is needed for operators or agents

## Basic Usage

```bash
c8volt ops purge orphan-process-instances --dry-run
```

Generated reference: [ops purge orphan-process-instances](/cli/c8volt_ops_purge_orphan-process-instances/).

## Best Variants

```bash
c8volt ops purge orphan-process-instances --state completed --limit 25 --dry-run
c8volt ops purge orphan-process-instances --state completed --limit 25 --report-file orphan-purge.md
```

## Built From Lower-Level Commands

```bash
c8volt get process-instance --orphan-children-only --keys-only
c8volt delete process-instance -
```

Generated references: [get process-instance](/cli/c8volt_get_process-instance/), [delete process-instance](/cli/c8volt_delete_process-instance/).

## Output And Safety

`--dry-run` reports orphan candidates and the delete plan without mutation. Real execution deletes only after confirmation unless automation controls are used. Process-instance delete safety, waiting, report format, worker controls, and fail-fast behavior follow the underlying delete workflow.
