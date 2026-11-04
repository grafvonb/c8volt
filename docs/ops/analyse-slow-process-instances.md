---
title: "Analyse Slow Process Instances"
permalink: /ops/analyse-slow-process-instances/
parent: "C8 Ops CLI"
nav_order: 1
has_toc: true
---

# Analyse slow process instances

Find slow process instances and identify the runtime elements contributing to their duration. This command is read-only.

## Examples

```bash
c8volt ops analyse slow-process-instances --bpmn-process-id order --state active --dur-longer 5m
c8volt ops analyse slow-process-instances --key <process-instance-key> --with-listeners
c8volt ops analyse slow-process-instances --bpmn-process-id order --element-id payment --dur-element-longer 30s
```

## Execution

Use `--limit` to bound the selected instances. Durations use values such as `30s`, `5m`, or `24h`; `1d` is not supported. Listener duration includes worker waiting time. Select by exact `--bid` when useful (**only c8.10+**).

See the [shared ops controls](/ops/#shared-controls) for discovery, confirmation, tenant scope, and reports.

[Command reference](/cli/c8volt_ops_analyse_slow-process-instances) · [get process-instance](/cli/c8volt_get_process-instance) · [get element](/cli/c8volt_get_element) · [get job](/cli/c8volt_get_job)
