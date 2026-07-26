---
title: "Purge Process Instances With Incidents"
permalink: /ops/purge-process-instances-with-incidents/
parent: "C8 Ops CLI"
nav_order: 3
has_toc: true
---

# c8volt ops purge process-instances-with-incidents

## Purpose

Incident cleanup often begins with incident filters, not process-instance keys. `c8volt ops purge process-instances-with-incidents` discovers matching incidents, freezes the affected process-instance keys, then runs deterministic family-scope delete planning.

## In Action

<img src="../../assets/screencasts/ops-purge-process-instances-with-incidents.gif" alt="c8volt ops purge process-instances-with-incidents demo" />

## Use When

- incident filters should define the purge candidate set
- deletion must still use normal process-instance family safety
- an operator or agent needs a stable report of selected incidents and deleted instances

## Basic Usage

```bash
c8volt ops purge process-instances-with-incidents --state active --error-type io_mapping_error --dry-run
```

Generated reference: [ops purge process-instances-with-incidents](/cli/c8volt_ops_purge_process-instances-with-incidents).

## Best Variants

```bash
c8volt ops purge process-instances-with-incidents --inc-key <incident-key> --dry-run
c8volt ops purge process-instances-with-incidents --state active --error-type io_mapping_error --limit 5 --force --report-file incident-purge.md
```

## Built From Lower-Level Commands

```bash
c8volt get incident --state active --error-type io_mapping_error --pi-keys-only
c8volt delete process-instance -
```

Generated references: [get incident](/cli/c8volt_get_incident), [delete process-instance](/cli/c8volt_delete_process-instance).

## Output And Safety

`--dry-run` reports the discovered incidents, frozen process-instance keys, and delete plan without mutation. Real execution requires confirmation unless automation controls are used. Incident matching is discovery only; deletion still follows the same process-instance family rules as `delete process-instance`.
