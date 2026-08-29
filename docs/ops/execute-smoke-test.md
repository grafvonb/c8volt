---
title: "Execute Smoke Test"
permalink: /ops/execute-smoke-test/
parent: "C8 Ops CLI"
nav_order: 8
has_toc: true
---

# c8volt ops execute smoke-test

## Purpose

A profile can look valid and still fail at the first real operational step. `c8volt ops execute smoke-test` proves the configured environment by connecting, deploying a c8volt-owned fixture, starting process instances, walking them, and cleaning up.

## In Action

<img src="../../assets/screencasts/ops-execute-smoke-test.gif" alt="c8volt ops execute smoke-test demo" />

## Use When

- a profile should be verified end to end
- deployment, runtime, traversal, and cleanup should be tested together
- CI or operators need a report that the profile passed

## Basic Usage

```bash
c8volt ops execute smoke-test --dry-run
```

Generated reference: [ops execute smoke-test](/cli/c8volt_ops_execute_smoke-test).

## Best Variants

```bash
c8volt ops execute smoke-test --report-file smoke-test.md
c8volt ops execute smoke-test --no-cleanup --report-file smoke-test.md
```

## Built From Lower-Level Commands

```bash
c8volt config test-connection
c8volt run process-instance --bpmn-process-id <bpmn-process-id>
c8volt walk process-instance --key <created-process-instance-key>
```

Generated references: [config test-connection](/cli/c8volt_config_test-connection), [run process-instance](/cli/c8volt_run_process-instance), [walk process-instance](/cli/c8volt_walk_process-instance), [delete process-instance](/cli/c8volt_delete_process-instance).

## Output And Safety

`--dry-run` reports the planned smoke-test steps without mutation. Real execution creates c8volt-owned runtime data and cleans it up unless `--no-cleanup` is supplied. On Camunda 8.8, prefer `--no-cleanup` because full process-definition deletion is supported by c8volt from Camunda 8.9 onward.

Smoke-test creation reports the target tenant before creating runtime data: `Create in tenant: tenant-a` for a named tenant or `Create in tenant: <default>` when no tenant is configured. Cleanup evidence in the report uses the same tenant-context contract and may include known resource tenants plus non-blocking cross-tenant or unknown-metadata warnings. JSON reports expose `tenantContext`; quiet and automation-oriented runs avoid extra human chatter on stdout.
