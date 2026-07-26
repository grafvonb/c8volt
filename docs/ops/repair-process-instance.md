---
title: "Repair Process Instance"
permalink: /ops/repair-process-instance/
parent: "C8 Ops CLI"
nav_order: 5
has_toc: true
---

# c8volt ops repair process-instance

## Purpose

Support work often starts from process-instance keys or filters, not incident keys. `c8volt ops repair process-instance` discovers active incidents for selected process instances, freezes the repairable set, then reuses the incident repair workflow.

## Use When

- process-instance selectors are easier than incident keys
- only active incidents for selected instances should be repaired
- process-instance discovery, incident repair, and reporting should be one workflow

## Basic Usage

```bash
c8volt ops repair process-instance --key <process-instance-key> --dry-run
```

Generated reference: [ops repair process-instance](/cli/c8volt_ops_repair_process-instance/).

## Best Variants

```bash
c8volt ops repair process-instance --direct-incidents-only --state active --limit 5 --dry-run
c8volt ops repair process-instance --key <process-instance-key> --vars '{"hasIncident":false}' --report-file repair-process-instance.md
```

## Built From Lower-Level Commands

```bash
c8volt get process-instance --key <process-instance-key> --with-incidents
c8volt update process-instance --key <process-instance-key> --vars '{"hasIncident":false}'
c8volt resolve incident --key <incident-key>
```

Generated references: [get process-instance](/cli/c8volt_get_process-instance/), [update process-instance](/cli/c8volt_update_process-instance/), [update job](/cli/c8volt_update_job/), [resolve incident](/cli/c8volt_resolve_incident/).

## Output And Safety

`--dry-run` shows selected process instances, discovered active incidents, and planned repair steps without mutation. Real execution skips instances with no repairable incident, deduplicates shared targets, and reports the same repair outcomes as `ops repair incident`.
