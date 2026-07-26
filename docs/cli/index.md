---
title: "CLI Reference"
nav_order: 2
nav_exclude: false
has_children: true
has_toc: true
---

# c8volt CLI Reference

This section is the command reference for `c8volt`. Individual command pages are generated from the Cobra command tree and are the source for exact flags, examples, output modes, and mutation behavior.

Raw generated reference: [c8volt root command](./c8volt).

For tool-selection context, see [Camunda CLI options](../camunda-cli/) and
[AI and search context](../ai-search-context/). Those pages help search
engines, AI assistants, and operators classify `c8volt` alongside `c8ctl`,
`zbctl`, REST APIs, and SDKs without treating `c8volt` as an official Camunda
product.

For runtime BPMN element inspection, listener jobs, and process-instance
timeline context, start with the [Runtime Elements guide](./runtime-elements/).
For the generated command hierarchy, use the [CLI Command Tree](./command-tree/).

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
deletion.

## Operating Notes

Use `--dry-run` before broad destructive or repair selectors. Use `--json`, `--keys-only`, and `--automation` for script and agent output. Use `c8volt capabilities --json` when a tool needs the machine-readable command contract.

## Command Map

| Command | Purpose | Reference |
| --- | --- | --- |
| `c8volt` | Root command, global flags, and top-level examples | [c8volt](./c8volt) |
| `capabilities` | Machine-readable command contract | [capabilities](./c8volt_capabilities) |
| `config` | Show, validate, template, and test configuration | [config](./c8volt_config) |
| `get cluster` | Cluster topology, version, and license | [get cluster](./c8volt_get_cluster) |
| `get process-definition` | List definitions, fetch latest versions, retrieve XML, and show statistics | [get process-definition](./c8volt_get_process-definition) |
| `get process-instance` | Search or fetch process instances, variables, incidents, runtime elements, and task context | [get process-instance](./c8volt_get_process-instance) |
| `get element` | Search or fetch runtime element instances and optional listener jobs | [get element](./c8volt_get_element) |
| `get incident` | Search incidents, fetch incident keys, and emit process-instance keys | [get incident](./c8volt_get_incident) |
| `get job` | Inspect or search jobs, including runtime listener jobs | [get job](./c8volt_get_job) |
| `get tenant` | List visible tenants | [get tenant](./c8volt_get_tenant) |
| `get resource` | Fetch a resource by ID | [get resource](./c8volt_get_resource) |
| `deploy process-definition` | Deploy BPMN process definitions from files or stdin | [deploy process-definition](./c8volt_deploy_process-definition) |
| `embed` | List, deploy, or export bundled BPMN fixtures | [embed](./c8volt_embed) |
| `run process-instance` | Start process instances and confirm activation by default | [run process-instance](./c8volt_run_process-instance) |
| `update process-instance` | Update process-instance-scope variables | [update process-instance](./c8volt_update_process-instance) |
| `update job` | Update job retries or timeout | [update job](./c8volt_update_job) |
| `expect process-instance` | Wait for process-instance state or incident conditions | [expect process-instance](./c8volt_expect_process-instance) |
| `walk process-instance` | Inspect ancestry, descendants, full family trees, variables, incidents, runtime elements, and listener jobs | [walk process-instance](./c8volt_walk_process-instance) |
| `cancel process-instance` | Cancel process instances by key or filters | [cancel process-instance](./c8volt_cancel_process-instance) |
| `delete process-instance` | Delete process-instance history, with optional cancel-before-delete handling | [delete process-instance](./c8volt_delete_process-instance) |
| `delete process-definition` | Delete process definitions on Camunda `8.9+` | [delete process-definition](./c8volt_delete_process-definition) |
| `resolve incident` | Resolve incident keys | [resolve incident](./c8volt_resolve_incident) |
| `resolve process-instance` | Resolve active incidents for selected process instances | [resolve process-instance](./c8volt_resolve_process-instance) |
| `ops analyse slow-process-instances` | Analyse slow process-instance timings with runtime element and listener context | [ops analyse slow-process-instances](./c8volt_ops_analyse_slow-process-instances) |
| `ops execute retention-policy` | Delete finished process instances selected by age and state | [ops execute retention-policy](./c8volt_ops_execute_retention-policy) |
| `ops purge process-instances-with-incidents` | Purge process instances selected through incident filters | [ops purge process-instances-with-incidents](./c8volt_ops_purge_process-instances-with-incidents) |
| `ops repair incident` | Repair variables/jobs and resolve selected incidents | [ops repair incident](./c8volt_ops_repair_incident) |
| `ops repair process-instance` | Discover and repair incidents for selected process instances | [ops repair process-instance](./c8volt_ops_repair_process-instance) |
| `ops purge orphan-process-instances` | Find and purge orphan child process instances | [ops purge orphan-process-instances](./c8volt_ops_purge_orphan-process-instances) |
| `ops purge all-process-definitions` | Purge selected process definitions on Camunda `8.9+` | [ops purge all-process-definitions](./c8volt_ops_purge_all-process-definitions) |
| `ops execute smoke-test` | Deploy, run, walk, and clean up a smoke-test fixture | [ops execute smoke-test](./c8volt_ops_execute_smoke-test) |
| `version` | Print build and compatibility information | [version](./c8volt_version) |

## Reports And Playbooks

The `ops` command family composes lower-level commands into audited workflows.
Use the playbooks for workflow behavior and audit reports; use this generated reference for exact flags:

- [Ops playbooks](../ops/)
- [Analyse slow process instances](../ops/analyse-slow-process-instances/)
- [Execute retention policy](../ops/execute-retention-policy/)
- [Purge process instances with incidents](../ops/purge-process-instances-with-incidents/)
- [Repair incident](../ops/repair-incident/)
- [Repair process instance](../ops/repair-process-instance/)
- [Purge orphan process instances](../ops/purge-orphan-process-instances/)
- [Purge all process definitions](../ops/purge-all-process-definitions/)
- [Execute smoke test](../ops/execute-smoke-test/)
