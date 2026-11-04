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

Use keys or search filters to select instances. The workflow fixes instance and active-incident targets before confirmation and deduplicates shared repairs. `--limit` caps selected instances; instances without active incidents need no change.

Requested variables are applied once per process-instance scope. Repair sets related job retries to 1 by default; `--set-retries` changes that value and `--set-retries 0` skips retry restoration. `--set-job-timeout` optionally changes job timeouts.

By default, wait until selected incidents are no longer active. Failed variable updates block their dependent resolutions. `--no-wait` skips post-action confirmation.

See the [shared ops controls](/ops/#shared-controls) for discovery, confirmation, tenant scope, and reports.

[Command reference](/cli/c8volt_ops_repair_process-instance) · [get process-instance](/cli/c8volt_get_process-instance) · [update process-instance](/cli/c8volt_update_process-instance) · [update job](/cli/c8volt_update_job) · [resolve incident](/cli/c8volt_resolve_incident)
