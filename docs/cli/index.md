---
title: "CLI Reference"
nav_order: 2
has_toc: true
---

## [c8volt](../) CLI Reference

**Camunda 8 Operations CLI for workflows that must actually finish.**

> **done is done**
>
> If an action needs retries, waiting, tree traversal, state checks, or cleanup before it is truly finished, `c8volt` should do that work for you instead of making you script the last mile yourself.

This section is the generated command reference for `c8volt`. It is best read as an operations guide first and a raw command catalog second.

## At A Glance

`c8volt` is strongest when you need more than a bare request/response CLI:

- deploy BPMN and use it immediately
- start process instances and confirm activation
- inspect process families before changing them
- cancel with root escalation when Camunda requires it
- delete thoroughly instead of partially
- wait for a precise target state in automation
- validate effective `c8volt` configuration before acting
- work in tenant-aware environments with config or `--tenant`

## Start Here

If you are new to `c8volt`, start with the workflows that define the tool:

- [root command](./c8volt)
- [deploy bundled BPMN fixtures](./c8volt_embed_deploy)
- [deploy BPMN files](./c8volt_deploy_process-definition)
- [run process instances](./c8volt_run_process-instance)
- [walk process trees](./c8volt_walk_process-instance)
- [cancel process instances](./c8volt_cancel_process-instance)
- [delete process instances](./c8volt_delete_process-instance)

## Signature Workflows

These are the workflows where `c8volt` stands apart from a basic CRUD-oriented CLI:

- `embed deploy --all`
  Bootstrap a local environment with bundled BPMN fixtures.
- `run process-instance`
  Start process instances and confirm they are actually active.
- `walk process-instance`
  Inspect parent/child structure before changing a live tree.
- `cancel process-instance --force`
  Escalate from a child process to the root instance that must actually be cancelled.
- `delete process-instance --force`
  Handle cancel-before-delete cleanup flows thoroughly.
- `expect process-instance`
  Wait for a target state as part of verification or automation.

## Precision Tools

The following implemented commands are also especially useful:

- [show config](./c8volt_config_show)
  Render sanitized effective `c8volt` config, validate it, or print a copy-paste-ready template.
- [get cluster topology](./c8volt_get_cluster_topology)
  Verify the connected Camunda cluster shape quickly as a human-readable tree.
- [get cluster version](./c8volt_get_cluster_version)
  Show the gateway version, optionally including sorted broker versions.
- [get cluster license](./c8volt_get_cluster_license)
  Inspect license details from the connected cluster; `licence` works as an alias.
- [get process definition](./c8volt_get_process-definition)
  List definitions, fetch latest versions, or retrieve raw XML for one key.
- [get resource](./c8volt_get_resource)
  Fetch a single resource by id.
- [embed export](./c8volt_embed_export)
  Export bundled BPMN fixtures to local files for editing or custom deployment.

Tenant-aware operations are supported through the global `--tenant` flag and the `app.tenant` config setting, and the `config` command family is specifically about configuring `c8volt` itself.

## Command Map

- [embed commands](./c8volt_embed)
  Work with bundled BPMN fixtures through list, deploy, and export flows.
- [deploy commands](./c8volt_deploy)
  Deploy BPMN process definitions from files or stdin.
- [run commands](./c8volt_run)
  Start process instances and confirm activation by default.
- [walk commands](./c8volt_walk)
  Inspect ancestors, descendants, and full family trees.
- [cancel commands](./c8volt_cancel)
  Cancel resources and wait for a confirmed state change.
- [delete commands](./c8volt_delete)
  Delete process instances or process definitions with safety checks.
- [expect commands](./c8volt_expect)
  Wait until process instances reach a target state.
- [get commands](./c8volt_get)
  Read cluster, definition, instance, and resource data.
- [config commands](./c8volt_config)
  Inspect, validate, and template `c8volt` configuration.
- [version command](./c8volt_version)
  Print build, compatibility, and copyright information.

## Common Flags

These flags appear across many operational commands:

- `--no-wait` for explicitly asynchronous behavior
- `--json` for structured output
- `--keys-only` for pipelines
- `--auto-confirm` for non-interactive runs
- `--workers` for controlled concurrency

For the higher-level product story and quick-start examples, see the [project overview](../).
