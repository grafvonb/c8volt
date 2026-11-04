---
title: "Purge Orphan Process Instances"
permalink: /ops/purge-orphan-process-instances/
parent: "C8 Ops CLI"
nav_order: 6
has_toc: true
---

# Purge orphan process instances

Find child process instances whose parent is missing, then plan deletion of their affected families.

## Examples

```bash
c8volt ops purge orphan-process-instances --dry-run
c8volt ops purge orphan-process-instances --state completed --report-file orphans.md
```

## Execution

`--limit` bounds child candidates inspected, including candidates that still have a parent. Normal family-deletion rules apply; unfinished instances require `--force`.

See the [shared ops controls](/ops/#shared-controls) for discovery, confirmation, tenant scope, and reports.

[Command reference](/cli/c8volt_ops_purge_orphan-process-instances) · [get process-instance](/cli/c8volt_get_process-instance) · [delete process-instance](/cli/c8volt_delete_process-instance)
