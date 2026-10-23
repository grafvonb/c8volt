---
title: "Purge All Process Definitions"
permalink: /ops/purge-all-process-definitions/
parent: "C8 Ops CLI"
nav_order: 7
has_toc: true
---

# c8volt ops purge all-process-definitions

## Purpose

Deleting process definitions is not just resource cleanup. `c8volt ops purge all-process-definitions` discovers candidate versions, plans process-instance impact, blocks unfinished impact unless forced, then deletes selected definitions with an audit report.

## In Action

<img src="../../assets/screencasts/ops-purge-all-process-definitions.gif" alt="c8volt ops purge all-process-definitions demo" />

## Use When

- process-definition versions should be removed after impact planning
- active and suspended process-instance impact must be visible before mutation
- Camunda 8.9+ deletion behavior should be driven by one audited workflow

## Basic Usage

```bash
c8volt ops purge all-process-definitions --bpmn-process-id <bpmn-process-id> --latest --dry-run
```

Generated reference: [ops purge all-process-definitions](/cli/c8volt_ops_purge_all-process-definitions).

## Best Variants

```bash
c8volt ops purge all-process-definitions --key <process-definition-key> --force --report-file process-definition-purge.md
c8volt ops purge all-process-definitions --bpmn-process-id <bpmn-process-id> --latest --force --report-file process-definition-purge.md
```

## Built From Lower-Level Commands

```bash
c8volt get process-definition --bpmn-process-id <bpmn-process-id>
c8volt delete process-definition --key <process-definition-key>
```

Generated references: [get process-definition](/cli/c8volt_get_process-definition), [delete process-definition](/cli/c8volt_delete_process-definition).

## Safety

`--dry-run` reports selected definitions and process-instance impact without mutation. Real execution requires confirmation unless automation controls are used. Full process-definition purge is supported from Camunda 8.9 onward.

A named tenant limits discovery. An empty tenant or `--all-tenants` searches across tenants visible to the authenticated identity. `--all-tenants` conflicts with an explicit `--tenant` value. Explicit keys use backend authorization without tenant filtering.

## Forced Cleanup

With `--force`, cleanup cancels root process instances, waits for unfinished instances to drain, deletes process-instance histories, and then deletes process definitions.
On Camunda 8.10, unfinished work includes ACTIVE and SUSPENDED instances. Earlier supported versions retain ACTIVE-only cleanup. Counts for active instances keep their original meaning; previews and audit reports add suspended counts with availability and an authoritative unfinished count. Expanded affected families can include instances outside the selected definitions and are deduplicated by key.

`--no-wait` skips final deletion confirmation, while required unfinished-work verification still runs before history and definition deletion. Failed cancellation or verification prevents dependent deletion. Additional unfinished work does not expand the confirmed cancellation scope. Purge has no state-check bypass flag. These observations do not prevent new starts or eliminate exporter lag.

Failures retain partial results in a single JSON envelope or the audit report; normal and verbose diagnostics use a compact aggregate. HTTP diagnostics remain available only at DEBUG with existing redaction.
