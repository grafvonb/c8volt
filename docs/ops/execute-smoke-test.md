---
title: "Execute Smoke Test"
permalink: /ops/execute-smoke-test/
parent: "C8 Ops CLI"
nav_order: 1
has_toc: true
---

# c8volt ops execute smoke-test

## The Problem

A profile can look valid and still fail at the first real operational step. Operators need to know whether c8volt can reach the cluster, deploy BPMN, start process instances, traverse the created process tree, and clean up after itself.

## The Promise

`c8volt ops execute smoke-test` proves the configured environment with a real c8volt-owned workflow. It runs the whole loop and reports each step: connection, fixture selection, deployment, process-instance creation, traversal, cleanup, and final outcome.

## In Action

The recording previews the smoke-test workflow without touching the cluster, then runs it with confirmation already handled, writes an audit report, and opens the first report section. It shows the normal operator rhythm: plan first, execute once the plan is understood, then inspect the evidence.

<img src="../../assets/screencasts/ops-execute-smoke-test.gif" alt="c8volt ops execute smoke-test demo" />

Generic command shape:

```bash
# read-only: preview the smoke-test workflow
c8volt ops execute smoke-test --dry-run

# harmless: deploys, runs, walks, and cleans up c8volt-owned smoke-test data
c8volt ops execute smoke-test --report-file smoke-test.md
```

## Use When

- validating a new local, CI, or production-support profile
- proving a Camunda upgrade is usable before incident work starts
- checking whether credentials, tenant settings, deployment, runtime, and cleanup all work together
- producing an audit trail that a profile passed an end-to-end c8volt check

## Command At A Glance

```bash
# read-only: preview the full smoke-test workflow
c8volt ops execute smoke-test --dry-run

# harmless: creates disposable smoke-test data and removes run-owned instances during cleanup
c8volt ops execute smoke-test --report-file smoke-test.md
```

On dirty clusters, smoke-test still proves deploy, run, walk, and run-owned process-instance cleanup. If unrelated instances still use the deployed fixture definition, c8volt skips process-definition cleanup, reports the retained definition, and exits successfully with `passed_cleanup_skipped`. On Camunda 8.8, prefer `--no-cleanup` because full process-definition deletion is supported by c8volt from Camunda 8.9 onward.

## Built From Lower-Level Commands

This is the conceptual flow. The ops command should use c8volt services and facades rather than shelling out to these commands.

```bash
c8volt config test-connection
c8volt embed deploy -f processdefinitions/<embedded-process>.bpmn
c8volt run pi -b <bpmn-process-id>
c8volt walk pi --key <created-process-instance-key>
c8volt delete pi --key <created-process-instance-key>
c8volt delete pd -b <bpmn-process-id>
```

## Workflow

```text
validate profile and connection
        |
        v
select embedded fixture for Camunda version
        |
        v
deploy smoke-test BPMN
        |
        v
start requested process instances
        |
        v
walk each created process tree
        |
        +--> --no-cleanup: keep resources, report keys
        |
        v
delete created process instances
        |
        v
delete process definition when no unrelated instances still use it
        |
        v
write outcome and optional audit report
```

## Dry Run

`--dry-run` validates local configuration, checks the planned smoke-test shape, verifies that the matching embedded fixture exists, and reports whether cleanup would run. It must not deploy BPMN, create process instances, walk newly created instances, delete process instances, or delete process definitions.

Dry-run output should show the selected fixture, requested count, cleanup mode, report path and format when supplied, and the ordered steps that would run.

## Real Execution

Real execution retrieves cluster topology when that service is available, deploys the embedded fixture, starts the requested number of process instances, walks each created family, and cleans up unless `--no-cleanup` is supplied. Cleanup deletes the process-instance families created by this smoke-test run, including normal family expansion performed by existing c8volt delete planning. Process-definition cleanup runs only when no unrelated process instances still use the deployed fixture definition; otherwise the command skips that final cleanup and reports retained resources.

When cleanup is enabled, the command prompts before deployment/run/cleanup unless `--auto-confirm` or `--automation` has already confirmed supported prompts. `--no-wait` applies to cleanup deletion confirmation.

## Reports

Markdown reports should be easy for an operator to read. JSON reports should keep the full structured step model for automation.

Important fields include selected fixture, BPMN process ID, deployed process-definition key, requested count, created process-instance keys, walk status, cleanup status, skipped cleanup reasons, errors, timestamps, duration, and final outcome.

## Failure And Safety Notes

- Missing embedded fixture fails before mutation.
- `--no-cleanup` leaves created resources in place and must report their keys.
- Process-definition cleanup is skipped, not failed, when unrelated instances exist.
- Automation JSON output should keep stdout deterministic.
- Reports should not expose unrelated variables or sensitive profile details.

## Related Commands

- [config test-connection](/cli/c8volt_config_test-connection/)
- [embed deploy](/cli/c8volt_embed_deploy/)
- [run process-instance](/cli/c8volt_run_process-instance/)
- [walk process-instance](/cli/c8volt_walk_process-instance/)
- [delete process-instance](/cli/c8volt_delete_process-instance/)
- [delete process-definition](/cli/c8volt_delete_process-definition/)
