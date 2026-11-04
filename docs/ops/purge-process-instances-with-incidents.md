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

Incident matching selects candidates; deletion still follows process-instance family rules. Unfinished instances require `--force`. The preview includes relatives affected by cancellation and deletion.

See the [shared ops controls](/ops/#shared-controls) for discovery, confirmation, tenant scope, and reports.

[Command reference](/cli/c8volt_ops_purge_process-instances-with-incidents) · [get incident](/cli/c8volt_get_incident) · [delete process-instance](/cli/c8volt_delete_process-instance)
