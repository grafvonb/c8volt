---
title: "Repair Incident"
permalink: /ops/repair-incident/
parent: "C8 Ops CLI"
nav_order: 4
has_toc: true
---

# c8volt ops repair incident

## Purpose

Incident repair is rarely one API call. `c8volt ops repair incident` freezes incident targets, optionally updates variables or related jobs, resolves incidents, confirms the result where possible, and records every step.

## Use When

- incident keys or incident filters define the repair target
- variable repair, job retry/timeout repair, and incident resolution should stay in one plan
- the run needs dry-run preview and audit output

## Basic Usage

```bash
c8volt ops repair incident --key <incident-key> --dry-run
```

Generated reference: [ops repair incident](/cli/c8volt_ops_repair_incident/).

## Best Variants

```bash
c8volt ops repair incident --state active --error-type io_mapping_error --limit 5 --dry-run
c8volt ops repair incident --key <incident-key> --vars '{"hasIncident":false}' --report-file repair-incident.md
```

## Built From Lower-Level Commands

```bash
c8volt get incident --key <incident-key>
c8volt update process-instance --key <process-instance-key> --vars '{"hasIncident":false}'
c8volt resolve incident --key <incident-key>
```

Generated references: [get incident](/cli/c8volt_get_incident/), [update process-instance](/cli/c8volt_update_process-instance/), [update job](/cli/c8volt_update_job/), [resolve incident](/cli/c8volt_resolve_incident/).

## Output And Safety

`--dry-run` shows selected incidents and planned variable, job, and resolution steps without mutation. Real execution submits only the requested repair actions and reports planned, skipped, submitted, confirmed, and failed work. Keyed mode and search mode are mutually exclusive.
