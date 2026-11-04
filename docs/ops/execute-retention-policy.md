---
title: "Execute Retention Policy"
permalink: /ops/execute-retention-policy/
parent: "C8 Ops CLI"
nav_order: 2
has_toc: true
---

# Execute retention policy

Delete finished process instances older than the retention age. The workflow fixes the candidate set, plans family deletion, and records the outcome.

## Examples

```bash
c8volt ops execute retention-policy --retention-days 90 --bpmn-process-id order --dry-run
c8volt ops execute retention-policy --retention-days 90 --bpmn-process-id order --auto-confirm --report-file retention.md
```

## Execution

Instances whose root is still unfinished are skipped, even with `--force`. Unfinished descendants of a finished root require `--force`. Review the preview before scheduling recurring cleanup.

See the [shared ops controls](/ops/#shared-controls) for discovery, confirmation, tenant scope, and reports.

[Command reference](/cli/c8volt_ops_execute_retention-policy) · [get process-instance](/cli/c8volt_get_process-instance) · [delete process-instance](/cli/c8volt_delete_process-instance)
