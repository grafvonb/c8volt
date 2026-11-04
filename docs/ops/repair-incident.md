---
title: "Repair Incident"
permalink: /ops/repair-incident/
parent: "C8 Ops CLI"
nav_order: 4
has_toc: true
---

# Repair incident

Repair selected incidents in one plan: apply requested variables, restore related job retries, resolve incidents, and wait for clearance.

## Examples

```bash
c8volt ops repair incident --key <incident-key> --dry-run
c8volt ops repair incident --key <incident-key> --set-retries 3 --set-vars-file vars.json --report-file repair.md
c8volt ops repair incident --state active --error-type io_mapping_error --limit 10 --dry-run
```

## Execution

Use keys or search filters, not both. Variables are applied once per process-instance scope. `--set-retries 0` skips retry restoration; `--set-job-timeout` optionally changes related job timeouts. Incidents without a related job still proceed to resolution.

See the [shared ops controls](/ops/#shared-controls) for discovery, confirmation, tenant scope, and reports.

[Command reference](/cli/c8volt_ops_repair_incident) · [get incident](/cli/c8volt_get_incident) · [update process-instance](/cli/c8volt_update_process-instance) · [update job](/cli/c8volt_update_job) · [resolve incident](/cli/c8volt_resolve_incident)
