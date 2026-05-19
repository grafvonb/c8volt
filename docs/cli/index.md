---
title: "CLI Reference"
nav_order: 2
nav_exclude: false
has_toc: true
---

# c8volt CLI Reference

This section is the command reference for `c8volt`. The individual command
pages are generated from the Cobra command tree; this page explains how to read
that reference and links the command groups that matter for day-to-day
operation.

For the project overview, installation notes, and broader examples, use the
[home page](../). This page stays closer to the CLI contract: command shape,
version support, mutation behavior, output modes, and safety controls.

## Scope

`c8volt` operates Camunda 8 clusters from the command line. The current command
surface covers:

- connection and configuration checks
- cluster topology, version, and license reads
- process-definition discovery, XML retrieval, and deployment
- process-instance start, search, wait, walk, cancel, delete, and variable update
- incident search, incident resolution, and process-instance incident repair
- job lookup, retry update, and timeout update
- tenant and resource reads
- high-level `ops` workflows for smoke tests, retention, purge, and repair
- machine-readable discovery through `capabilities --json`

The reference pages document the available flags and examples for each command.
They do not replace a dry-run plan for destructive operations.

## Version Support

`c8volt` supports Camunda `8.7`, `8.8`, and `8.9`, but not every command is
available on every upstream version.

| Area | 8.9 | 8.8 | 8.7 |
| --- | --- | --- | --- |
| Cluster, config, tenant, process-definition, process-instance reads | supported | supported | supported |
| Deploy and run process instances | supported | supported | supported |
| Cancel and delete process instances | supported | supported | limited |
| Process-instance variable update | supported | supported | unsupported |
| `get job` and `update job` | supported | supported | unsupported |
| Incident resolution and repair workflows | supported | supported | unsupported |
| `delete process-definition` | supported | unsupported | unsupported |
| `ops purge all-process-definitions` | supported | unsupported | unsupported |

Process-definition deletion requires Camunda `8.9` or newer because c8volt
depends on the endpoint shape that supports full process-definition history
deletion. On `8.8`, use `delete process-instance --bpmn-process-id
BPMN_PROCESS_ID` when the intended operation is to remove process instances for
a definition.

## Execution Model

Most state-changing commands follow the same pattern:

1. select targets by key, stdin, or filters
2. preview the target set when `--dry-run` is available
3. require confirmation unless `--auto-confirm` or `--automation` is used
4. submit the Camunda mutation
5. wait for the observable result unless `--no-wait` is used

For process-instance family operations, c8volt expands the selected instance to
the relevant root or descendant scope before mutating. For process-definition
delete and all-process-definition purge on Camunda `8.9`, c8volt also waits
until the deleted process definition is no longer visible unless `--no-wait` is
set.

## Output Modes

Common output controls:

- `--json` writes structured command output where the command supports it
- `--keys-only` emits keys for pipelines where the command supports it
- `--quiet` suppresses normal output except errors
- `--verbose` includes additional diagnostic detail
- `--automation` enables non-interactive behavior only for commands that
  explicitly support automation
- `capabilities --json` exposes a machine-readable command and flag contract

Human logs are written separately from JSON payloads where a command provides a
machine-output mode.

## Safety Controls

Use these flags deliberately:

- `--dry-run` previews supported destructive and repair workflows without
  mutation
- `--auto-confirm` answers command prompts for unattended runs
- `--no-wait` returns after Camunda accepts work instead of waiting for
  confirmation
- `--force` broadens the mutation scope when the command needs root cancellation
  or cancel-before-delete behavior
- `--workers` limits concurrency for batch operations
- `--fail-fast` stops scheduling additional work after the first failure

Prefer `--dry-run` before broad selectors such as BPMN process id, incident
filters, retention windows, or all-process-definition purge.

## Command Map

| Command | Purpose | Reference |
| --- | --- | --- |
| `c8volt` | Root command, global flags, and top-level examples | [c8volt](./c8volt) |
| `capabilities` | Machine-readable command contract | [capabilities](./c8volt_capabilities) |
| `config` | Show, validate, template, and test configuration | [config](./c8volt_config) |
| `get cluster` | Cluster topology, version, and license | [get cluster](./c8volt_get_cluster) |
| `get process-definition` | List definitions, fetch latest versions, and retrieve XML | [get process-definition](./c8volt_get_process-definition) |
| `get process-instance` | Search or fetch process instances, variables, incidents, and task context | [get process-instance](./c8volt_get_process-instance) |
| `get incident` | Search incidents, fetch incident keys, and emit process-instance keys | [get incident](./c8volt_get_incident) |
| `get job` | Fetch a job by key | [get job](./c8volt_get_job) |
| `get tenant` | List visible tenants | [get tenant](./c8volt_get_tenant) |
| `get resource` | Fetch a resource by id | [get resource](./c8volt_get_resource) |
| `deploy process-definition` | Deploy BPMN process definitions from files or stdin | [deploy process-definition](./c8volt_deploy_process-definition) |
| `embed` | List, deploy, or export bundled BPMN fixtures | [embed](./c8volt_embed) |
| `run process-instance` | Start process instances and confirm activation by default | [run process-instance](./c8volt_run_process-instance) |
| `update process-instance` | Update process-instance-scope variables | [update process-instance](./c8volt_update_process-instance) |
| `update job` | Update job retries or timeout | [update job](./c8volt_update_job) |
| `expect process-instance` | Wait for process-instance state or incident conditions | [expect process-instance](./c8volt_expect_process-instance) |
| `walk process-instance` | Inspect ancestry, descendants, full family trees, variables, and incidents | [walk process-instance](./c8volt_walk_process-instance) |
| `cancel process-instance` | Cancel process instances by key or filters | [cancel process-instance](./c8volt_cancel_process-instance) |
| `delete process-instance` | Delete process-instance history, with optional cancel-before-delete handling | [delete process-instance](./c8volt_delete_process-instance) |
| `delete process-definition` | Delete process definitions on Camunda `8.9+` | [delete process-definition](./c8volt_delete_process-definition) |
| `resolve incident` | Resolve incident keys | [resolve incident](./c8volt_resolve_incident) |
| `resolve process-instance` | Resolve active incidents for selected process instances | [resolve process-instance](./c8volt_resolve_process-instance) |
| `ops execute smoke-test` | Deploy, run, walk, and clean up a smoke-test fixture | [ops execute smoke-test](./c8volt_ops_execute_smoke-test) |
| `ops execute retention-policy` | Delete finished process instances selected by age and state | [ops execute retention-policy](./c8volt_ops_execute_retention-policy) |
| `ops purge orphan-process-instances` | Find and purge orphan child process instances | [ops purge orphan-process-instances](./c8volt_ops_purge_orphan-process-instances) |
| `ops purge process-instances-with-incidents` | Purge process instances selected through incident filters | [ops purge process-instances-with-incidents](./c8volt_ops_purge_process-instances-with-incidents) |
| `ops purge all-process-definitions` | Purge selected process definitions on Camunda `8.9+` | [ops purge all-process-definitions](./c8volt_ops_purge_all-process-definitions) |
| `ops repair incident` | Repair variables/jobs and resolve selected incidents | [ops repair incident](./c8volt_ops_repair_incident) |
| `ops repair process-instance` | Discover and repair incidents for selected process instances | [ops repair process-instance](./c8volt_ops_repair_process-instance) |
| `version` | Print build and compatibility information | [version](./c8volt_version) |

## Selector Conventions

Common selectors are shared across command groups:

- `--key` selects explicit resource, process-instance, incident, or
  process-definition keys
- `-` reads keys from stdin for commands that document stdin support
- `--bpmn-process-id` selects process instances or definitions by BPMN id
- `--pd-version`, `--pd-version-tag`, and `--latest` narrow
  process-definition selection
- `--tenant` overrides tenant selection from config for tenant-aware flows
- date filters such as `--start-date-*` and `--end-date-*` bound
  process-instance searches

When a command accepts both explicit keys and filters, the command page describes
which combinations are valid.

## Reports And Playbooks

The `ops` command family composes lower-level commands into audited workflows.
These pages are written as operational playbooks rather than raw Cobra output:

- [Ops playbooks](../ops/)
- [Execute smoke test](../ops/execute-smoke-test/)
- [Execute retention policy](../ops/execute-retention-policy/)
- [Purge orphan process instances](../ops/purge-orphan-process-instances/)
- [Purge process instances with incidents](../ops/purge-process-instances-with-incidents/)
- [Purge all process definitions](../ops/purge-all-process-definitions/)
- [Repair incident](../ops/repair-incident/)
- [Repair process instance](../ops/repair-process-instance/)

Use the playbooks for workflow behavior and audit-report fields; use the
generated command pages for exact flags and aliases.
