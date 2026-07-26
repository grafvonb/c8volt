---
title: "C8 Ops CLI"
permalink: /ops/
nav_order: 3
has_children: true
has_toc: true
---

# C8 Ops CLI

Low-level commands do work. `c8volt ops` finishes workflows.

The ops command group is the operator-facing layer for predefined Camunda playbooks. Each command composes lower-level c8volt behavior such as discovery, runtime element inspection, tree traversal, delete planning, incident lookup, confirmation, waiting, JSON output, and report writing.

## Playbook Index

| Workflow | Command | Use when |
| --- | --- | --- |
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

## Reports And Demos

Ops reports are stable structured data first, then rendered to Markdown or JSON. Demo recordings live as VHS scripts under `demos/vhs/` and show preview-first usage before deletion, cleanup, or repair execution.
