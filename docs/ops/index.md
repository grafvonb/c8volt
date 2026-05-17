---
title: "C8 Ops CLI"
permalink: /ops/
nav_order: 2
has_children: true
has_toc: true
---

# C8 Ops CLI

Low-level commands do work. `c8volt ops` finishes workflows.

The ops command group is the operator-facing layer for predefined Camunda playbooks. Each command composes lower-level c8volt behavior such as discovery, tree traversal, delete planning, incident lookup, confirmation, waiting, JSON output, and report writing. The goal is not to hide the primitives. The goal is to make the whole operational outcome repeatable.

## Playbook Index

| Workflow | Command | Use when |
| --- | --- | --- |
| [Smoke Test](/ops/smoke-test/) | `c8volt ops execute smoke-test` | You need to prove a profile can connect, deploy, run, walk, and clean up. |
| [Retention Policy](/ops/retention-policy/) | `c8volt ops execute retention-policy` | You need an auditable cleanup of old finished process instances. |
| [Orphan Process Instances](/ops/orphan-process-instances/) | `c8volt ops purge orphan-process-instances` | You need to find and delete orphan child process instances. |
| [Incident-Based Purge](/ops/purge-pi-with-incidents/) | `c8volt ops purge process-instances-with-incidents` | You need to delete process-instance families selected from incident filters. |
| [All Process Definitions](/ops/all-process-definitions/) | `c8volt ops purge all-process-definitions` | You need to delete selected process-definition versions after impact planning. |
| [Incident Repair](/ops/repair-incident/) | `c8volt ops repair incident` | You need to repair incidents selected by key, stdin, or incident filters. |
| [Process-Instance Repair](/ops/repair-process-instance/) | `c8volt ops repair process-instance` | You need to repair active incidents discovered from selected process instances. |

## Shared Shape

Every ops playbook page follows the same structure:

- the problem the command solves
- when to use it
- the command at a glance
- the lower-level c8volt commands it composes
- an ASCII workflow diagram
- what `--dry-run` does
- what real execution does
- report behavior
- a VHS demo script
- failure and safety notes

## Safety Model

Ops commands should feel boring in the best way: they discover, freeze, plan, validate, execute, verify, and report.

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

Ops reports should be stable structured data first, then rendered to Markdown or JSON. Markdown is for operator review. JSON is for agents, CI, and audit pipelines.

Demo recordings live as VHS scripts under `demos/vhs/`. The scripts intentionally show preview-first usage before deletion, cleanup, or repair execution.
