---
title: "Purge Process Instances With Incidents"
permalink: /ops/purge-process-instances-with-incidents/
parent: "C8 Ops CLI"
nav_order: 3
has_toc: true
---

# Purge process instances with incidents

Use incident keys or filters to select process-instance families for deletion.

## Examples

```bash
c8volt ops purge process-instances-with-incidents --error-type io_mapping_error --dry-run
c8volt ops purge process-instances-with-incidents --inc-key <incident-key> --force --report-file purge.md
```

## Execution

Use incident keys or search filters. Search defaults to active incidents; `--limit` caps selected incidents, not the number of affected process instances. The preview includes relatives affected by cancellation and deletion.

Unfinished instances require `--force`. Deletion processes children before parents and waits for disappearance by default. Required cancellation checks run before deletion.

See the [shared ops controls](/ops/#shared-controls) for discovery, confirmation, tenant scope, and reports.

[Command reference](/cli/c8volt_ops_purge_process-instances-with-incidents) · [get incident](/cli/c8volt_get_incident) · [delete process-instance](/cli/c8volt_delete_process-instance)
