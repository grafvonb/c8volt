---
title: "Purge All Process Definitions"
permalink: /ops/purge-all-process-definitions/
parent: "C8 Ops CLI"
nav_order: 7
has_toc: true
---

# c8volt ops purge all-process-definitions

## Purpose

Deleting process definitions is not just resource cleanup. `c8volt ops purge all-process-definitions` discovers candidate versions, plans process-instance impact, blocks active impact unless forced, then deletes selected definitions with an audit report.

## In Action

<img src="../../assets/screencasts/ops-purge-all-process-definitions.gif" alt="c8volt ops purge all-process-definitions demo" />

## Use When

- process-definition versions should be removed after impact planning
- active process-instance impact must be visible before mutation
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

## Output And Safety

`--dry-run` reports selected definitions and process-instance impact without mutation. Real execution requires confirmation unless automation controls are used. Full process-definition purge is supported from Camunda 8.9 onward.

Selector mode is tenant-filtered discovery: named configuration is shown as `selection scope: tenant-a only`, while an empty tenant or active `--all-tenants` is shown as `selection scope: unfiltered across accessible tenants`. Use `--all-tenants` to clear a configured selector tenant for this run; the unfiltered scope remains limited to resources visible to the authenticated identity. Explicit tenant flag changes are reported before scope; clearing a named configuration with `--tenant ""` warns that selection is unfiltered, and clearing it with `--all-tenants` emits `--all-tenants overrides the configured tenant filter; selection is unfiltered`. The options `--tenant` and `--all-tenants` are mutually exclusive. Explicit `--key` targets use backend authorization instead, so c8volt reports `selection scope: explicit resource keys; tenant filter not applied` and then shows known definition or nested process-instance tenant evidence from the frozen impact plan. Multi-tenant impact emits one warning-level `affected tenants: ...` summary; unknown-metadata warnings appear separately before confirmation. JSON reports include `tenantContext`; quiet and keys-only modes do not add tenant lines to stdout.

## Force-Cleanup Progress

With `--force`, a purge that must clean up process instances reports each stage as it is actually entered: cancelling process-instance root trees, waiting for active process instances to drain, deleting process-instance histories, and deleting process definitions. Cancellation and history deletion count unique root trees. Definition deletion uses its own process-definition count, so root cleanup progress is not carried into the definition stage.

Cancellation may also show `affected scope: N process instance(s)` when the cleanup plan has a trustworthy affected-instance total. That value is planned scope, not completed work. Completed affected totals are shown only when every contributing item reports a trustworthy count; unknown values are omitted instead of estimated.

Default human progress stays completion-driven. Long-running mutation work can emit compact progress milestones on stderr at most once per 10-second interval, and failures remain immediate. Waiting for active instances to drain updates the transient activity but does not fabricate completion counts or timer-only milestones. When verbose or debug output is enabled, c8volt prints one durable item outcome for each root or definition completion instead of aggregate milestones.

Successful progress is suppressed in quiet mode, while immediate failure warnings remain visible. JSON and automation-oriented runs do not add human progress text, so stdout remains reserved for the existing structured result. `--batch-size` controls discovery page size only; it does not batch cancellation, draining, history deletion, or definition deletion.
