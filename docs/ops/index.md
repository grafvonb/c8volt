---
title: "C8 Ops CLI"
permalink: /ops/
nav_order: 3
has_children: true
has_toc: true
---

# C8 Ops CLI

Low-level commands do work. `c8volt ops` finishes workflows.

The ops command group is the operator-facing layer for predefined Camunda playbooks. Each command composes lower-level c8volt behavior such as discovery, API latency measurement, runtime element inspection, tree traversal, delete planning, incident lookup, confirmation, waiting, JSON output, and report writing.

## Playbook Index

| Workflow | Command | Use when |
| --- | --- | --- |
| [Analyse API Latency](./analyse-api-latency/) | `c8volt ops analyse api-latency` | You need bounded read-path latency evidence without changing cluster state. |
| [Execute API Latency Test](./execute-api-latency-test/) | `c8volt ops execute api-latency-test` | You need confirmed active write/read/visibility latency evidence with exact-key cleanup. |
| [Analyse Slow Process Instances](./analyse-slow-process-instances/) | `c8volt ops analyse slow-process-instances` | You need to find slow runtime work and explain it with element timing and listener context. |
| [Execute Retention Policy](./execute-retention-policy/) | `c8volt ops execute retention-policy` | You need an auditable cleanup of old finished process instances. |
| [Purge Process Instances With Incidents](./purge-process-instances-with-incidents/) | `c8volt ops purge process-instances-with-incidents` | You need to delete process-instance families selected from incident filters. |
| [Repair Incident](./repair-incident/) | `c8volt ops repair incident` | You need to repair incidents selected by key, stdin, or incident filters. |
| [Repair Process Instance](./repair-process-instance/) | `c8volt ops repair process-instance` | You need to repair active incidents discovered from selected process instances. |
| [Purge Orphan Process Instances](./purge-orphan-process-instances/) | `c8volt ops purge orphan-process-instances` | You need to find and delete orphan child process instances. |
| [Purge All Process Definitions](./purge-all-process-definitions/) | `c8volt ops purge all-process-definitions` | You need to delete selected process-definition versions after impact planning. |
| [Execute Smoke Test](./execute-smoke-test/) | `c8volt ops execute smoke-test` | You need to prove a profile can connect, deploy, run, walk, and clean up. |

## Shared Shape

Every ops playbook page keeps the same compact structure: purpose, use when, basic usage, best variants, lower-level commands, output/report behavior, and safety notes. Generated reference pages remain the exact flag contract.

## API Latency Diagnostics

Use the read-only analysis first when production safety is the priority:

```bash
c8volt ops analyse api-latency
c8volt ops analyse api-latency --count 6 --workers 2 --report-file api-latency.md
```

Use the active test only when write and search-visibility evidence is needed. Preview the plan with `--dry-run`; real execution deploys a c8volt-owned fixture, creates a bounded number of process instances, and cleans up exact run-owned resources unless `--no-cleanup` is explicit.

```bash
c8volt ops execute api-latency-test --dry-run
c8volt ops execute api-latency-test --auto-confirm --report-file api-latency-test.json
```

Both commands support compact human output, one-document JSON output, and shared Markdown or JSON reports. `ops analyse api-latency` performs no mutation; `ops execute api-latency-test` requires one concrete tenant and rejects `--all-tenants`.

## Safety Model

Ops commands discover, freeze, plan, validate, execute, verify, and report.

```text
discover candidates
        |
        v
freeze target set
        |
        v
build c8volt plan
        |
        v
validate safety
        |
        +--> --dry-run: report plan, mutate nothing
        |
        v
confirm or run under automation
        |
        v
execute lower-level action
        |
        v
wait and verify
        |
        v
write audit report
```

Tenant-aware discovery playbooks accept `--all-tenants` when the configured tenant filter should be cleared for one run. The scope is unfiltered only across tenants visible to the authenticated identity; it does not enumerate tenants or bypass backend authorization. If the option clears a named configured tenant, human output emits `--all-tenants overrides the configured tenant filter; selection is unfiltered` before the unfiltered selection scope. It is mutually exclusive with any explicit `--tenant` value. Workflows that create resources in one concrete tenant, such as `ops execute smoke-test` and `ops execute api-latency-test`, reject `--all-tenants`.

## Reports And Demos

Ops reports are stable structured data first, then rendered to Markdown or JSON. Demo recordings live as VHS scripts under `demos/vhs/` and show preview-first usage before deletion, cleanup, or repair execution.
