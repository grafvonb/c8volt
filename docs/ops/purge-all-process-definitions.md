---
title: "Purge All Process Definitions"
permalink: /ops/purge-all-process-definitions/
parent: "C8 Ops CLI"
nav_order: 7
has_toc: true
---

# Purge all process definitions

Delete selected process-definition versions and associated history after reviewing instance impact (c8.9+).

## Examples

```bash
c8volt ops purge all-process-definitions --bpmn-process-id order --dry-run
c8volt ops purge all-process-definitions --bpmn-process-id order --force --report-file definitions.md
```

## Execution

Select definition versions by key or filters. `--limit` caps selected definitions; expanded families can include instances outside those definitions.

Unfinished instances block deletion. `--force` cancels affected roots, waits for unfinished work to drain, deletes instance history, then deletes definitions. Suspended instances count as unfinished (c8.10+).

By default, confirm definition removal and any returned history-deletion batch. `--no-wait` skips final confirmation; required drain checks still run. Deferred history deletion without a returned batch cannot be independently confirmed.

See the [shared ops controls](/ops/#shared-controls) for discovery, confirmation, tenant scope, and reports.

[Command reference](/cli/c8volt_ops_purge_all-process-definitions) · [get process-definition](/cli/c8volt_get_process-definition) · [delete process-definition](/cli/c8volt_delete_process-definition)
