---
title: "Execute API Latency Test"
permalink: /ops/execute-api-latency-test/
parent: "C8 Ops CLI"
nav_order: 2
has_toc: true
---

# c8volt ops execute api-latency-test

## Purpose

Read-only latency checks cannot show write response or search-visibility delay. `c8volt ops execute api-latency-test` runs a bounded active diagnostic by deploying the existing c8volt-owned `SimpleUserTask` fixture, creating a limited number of process instances, measuring create/read/visibility behavior, and cleaning up exact run-owned resources.

Use this command only when mutation is acceptable in the selected tenant. Start with `--dry-run` to see the exact plan and version capability result before any resource is created.

## Use When

- read-only evidence is not enough to distinguish write response from search visibility
- a disposable or approved tenant can receive a small c8volt-owned fixture run
- exact created-resource ownership and cleanup evidence are required
- a Markdown or JSON report should preserve the active run, cleanup, and recovery state

## Basic Usage

```bash
c8volt ops execute api-latency-test --dry-run
```

Generated reference: [ops execute api-latency-test](/cli/c8volt_ops_execute_api-latency-test).

## Best Variants

```bash
c8volt ops execute api-latency-test --count 20 --workers 4 --auto-confirm
c8volt ops execute api-latency-test --auto-confirm --report-file api-latency-test.md
c8volt ops execute api-latency-test --automation --json --report-file api-latency-test.json --report-format json
c8volt ops execute api-latency-test --no-cleanup --auto-confirm --report-file retained-api-latency.md
```

`--count` is the total primary process-instance create sample budget. `--workers` is the maximum closed-loop worker count and final active stage width. JSON execution requires `--dry-run`, `--auto-confirm`, or `--automation` so standard output remains one JSON document.

## Preview And Confirmation

The preview includes the run ID, profile, concrete tenant, configured and observed Camunda version, selected embedded fixture, stage plan, primary sample allocation, derived-request ceiling, visibility attempt ceiling, worker limit, cleanup behavior, and known version or evidence limits.

Real execution prompts unless the normal `--auto-confirm` or `--automation` path applies. `--no-cleanup` still requires confirmation because it intentionally retains resources. The command rejects `--all-tenants`, including for dry runs, because deployment, creation, ownership, and cleanup require one concrete destination tenant.

## Version Behavior

| Camunda | Active behavior |
| --- | --- |
| 8.7 | Blocked before mutation because exact created keys cannot be guaranteed. |
| 8.8 | Cleanup-enabled runs are blocked before mutation; explicit `--no-cleanup` may proceed. |
| 8.9 | Cleanup-enabled and retained runs are supported. |
| 8.10 | Cleanup-enabled and retained runs are supported. |

A blocked dry run reports the same capability decision without mutating resources. It is evidence that the selected request was checked, not proof that a later real run will complete.

## What It Measures

After confirmation, the command deploys the version-matched `SimpleUserTask` fixture and records the returned process-definition key immediately. It creates no more than the requested primary sample count, records each returned process-instance key immediately, measures create response, runs one lightweight read per primary cycle, records whether that read overlapped active writes, and polls exact-key search visibility within the previewed attempt limit.

Fixture deployment, preflight, and cleanup are reported separately from primary samples. The local run ID is for correlation only; cleanup authority comes only from exact returned keys.

## Cleanup And Recovery

Cleanup is enabled by default. The service attempts cleanup after success, stage failure, request timeout, visibility exhaustion, and operator cancellation. Cleanup uses an independent bounded context, deletes exact process-instance keys before the exact process-definition key, and never deletes by BPMN process ID, tenant-wide search, or later rediscovery.

Every owned resource ends as deleted, retained, failed, or unknown. Failed and unknown resources include exact-key recovery guidance when the configured Camunda version supports it, such as:

```bash
c8volt delete process-instance --key <key> --force --auto-confirm
c8volt delete process-definition --key <key> --auto-confirm
```

An explicit `--no-cleanup` run can finish as `completed_retained`. Requested cleanup failure returns a non-success outcome and preserves remaining keys in output and reports.

## Output And Reports

Default output shows the active plan, stage metrics, findings, ownership summary, visibility summary, cleanup summary, notices, limitations, and outcome without per-key lifecycle chatter. Retained, failed, and unknown resources are listed because they require operator attention.

Use `--report-file` to write a Markdown or JSON report. Confirmed mutation uses the established overwrite policy only after a deployment is submitted or exact ownership keys are recorded. Reports and JSON output exclude endpoints, credentials, authorization headers, client secrets, variables, payloads, raw upstream bodies, and unbounded error strings.
