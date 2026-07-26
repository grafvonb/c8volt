---
title: "Runtime Elements"
permalink: /cli/runtime-elements/
parent: "CLI Reference"
nav_order: 2
nav_exclude: false
has_toc: true
---

# Runtime Elements

Runtime element inspection shows what is happening inside a process instance: active BPMN elements, completed elements, timing, incidents, and optional listener jobs. Use it when the process-instance row alone is too coarse.

## Use When

- a process instance is active but the slow or blocked BPMN element is not obvious
- incident context needs the owning element, job, or listener detail beside it
- operators or developers need the lower-level data behind slow-process analysis

## Process Instance Context

Use `get process-instance --with-elements` when the process instance is the primary target and runtime element rows are supporting context.

```bash
c8volt get process-instance --key <process-instance-key> --with-elements
c8volt get process-instance --key <process-instance-key> --with-elements --with-listeners
c8volt get process-instance --bpmn-process-id <bpmn-process-id> --state active --with-elements --limit 5
```

Element rows appear under the owning process-instance row. `--with-listeners` requires `--with-elements` and nests execution/task listener jobs under matching element rows.

Generated reference: [get process-instance](./c8volt_get_process-instance).

## Element-Centric Search

Use `get element` when the element instance is the primary target or when element-specific filters should drive the result set.

```bash
c8volt get element --key <element-instance-key>
c8volt get element --pi-key <process-instance-key> --element-id <element-id> --state active
c8volt --json get element --pi-key <process-instance-key> --with-listeners
```

Search filters combine with AND semantics. Use the generated reference for totals, keys-only output, page size, and all supported filters.

Generated reference: [get element](./c8volt_get_element).

## Walk With Runtime Context

Use `walk process-instance --with-elements` when ancestry, descendants, or the full process-instance family tree must stay visible beside runtime element state.

```bash
c8volt walk process-instance --key <process-instance-key> --with-elements
c8volt walk process-instance --key <process-instance-key> --with-elements --with-listeners
```

Generated reference: [walk process-instance](./c8volt_walk_process-instance).

## Slow Analysis

`ops analyse slow-process-instances` is the higher-level workflow built from the same runtime data. Use it when the goal is to rank slow roots and explain their slowest contributors.

```bash
c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --state active --dur-longer 5m
c8volt ops analyse slow-process-instances --key <process-instance-key> --with-listeners
```

Playbook: [Analyse Slow Process Instances](../ops/analyse-slow-process-instances/). Generated reference: [ops analyse slow-process-instances](./c8volt_ops_analyse_slow-process-instances).
