---
title: "Repair Process Instance"
permalink: /ops/repair-process-instance/
parent: "C8 Ops CLI"
nav_order: 5
has_toc: true
---

# Repair process instance

Discover active incidents for selected process instances, then apply the incident repair workflow.

## Examples

```bash
c8volt ops repair process-instance --key <process-instance-key> --dry-run
c8volt ops repair process-instance --bpmn-process-id order --set-retries 3 --report-file repair.md
c8volt ops repair process-instance --key <process-instance-key> --set-vars-file vars.json --dry-run
```

## Execution

The workflow fixes instance and incident targets before confirmation and deduplicates shared repairs. Instances without repairable incidents need no change. `--set-retries 0` skips retry restoration; `--set-job-timeout` optionally changes related job timeouts.

See the [shared ops controls](/ops/#shared-controls) for discovery, confirmation, tenant scope, and reports.

[Command reference](/cli/c8volt_ops_repair_process-instance) · [get process-instance](/cli/c8volt_get_process-instance) · [update process-instance](/cli/c8volt_update_process-instance) · [update job](/cli/c8volt_update_job) · [resolve incident](/cli/c8volt_resolve_incident)
