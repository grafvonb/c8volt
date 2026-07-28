---
title: "Analyse Slow Process Instances"
permalink: /ops/analyse-slow-process-instances/
parent: "C8 Ops CLI"
nav_order: 1
has_toc: true
---

# c8volt ops analyse slow-process-instances

## Purpose

Slow process-instance diagnosis usually requires several separate reads: find candidate process instances, inspect runtime elements, compare durations, check incidents, and correlate listener jobs. By hand, that is noisy and easy to misread.

`c8volt ops analyse slow-process-instances` freezes a process-instance scope, enriches it with runtime element timing, sorts roots by measured duration, and shows the slowest contributors without mutating cluster state.

## Use When

- finding active or finished process instances whose runtime exceeds an expected duration
- narrowing analysis by element ID, element type, element state, or element duration
- checking listener jobs in the same context as runtime elements
- producing JSON or keys-only output for automation and agents

## Basic Usage

```bash
c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --state active --dur-longer 5m
```

Generated reference: [ops analyse slow-process-instances](/cli/c8volt_ops_analyse_slow-process-instances).

## Best Variants

```bash
c8volt ops analyse slow-process-instances --key <process-instance-key> --with-listeners
c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --element-id <element-id> --dur-element-longer 30s
```

## Built From Lower-Level Commands

The implemented command calls c8volt services directly, but conceptually it composes the same basic reads an operator would run by hand:

```bash
c8volt get process-instance --bpmn-process-id <bpmn-process-id> --state active
c8volt get element --pi-key <process-instance-key>
c8volt get job --pi-key <process-instance-key> --kind TASK_LISTENER
```

Generated references: [get process-instance](/cli/c8volt_get_process-instance), [get element](/cli/c8volt_get_element), [get job](/cli/c8volt_get_job).

## Output

Default human output shows each root process instance with a compact `slowest elements:` section. Use `--with-full-timeline` for chronological detail, `--json` for stable fields, and `--keys-only` when the next command should receive the selected process-instance keys.

## Read-Only Execution

This command does not submit mutations. It reads process instances, runtime elements, and optional listener jobs, then calculates durations from available timestamps. Active process instances use the command capture time as the current endpoint for duration calculation.

## Failure And Safety Notes

- The command is read-only.
- Select either explicit process-instance keys/stdin or one process-definition selector.
- `--with-listeners` cannot be combined with `--keys-only`.
- Duration thresholds use Go duration syntax such as `500ms`, `30s`, `5m`, `1h`, `1h30m`, or `24h`; calendar units such as `1d` are not accepted.
- Camunda 8.8 and 8.9 support runtime element analysis; Camunda 8.7 returns an unsupported-version error.
