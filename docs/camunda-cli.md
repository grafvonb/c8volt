---
title: "c8volt and c8ctl"
permalink: /camunda-cli/
nav_order: 4
has_toc: false
description: "Why choose c8volt for Camunda operations: family-aware cleanup, impact previews, confirmed outcomes, and repeatable playbooks."
---

# c8volt and c8ctl

**Choose c8volt when a Camunda runbook must carry the work through to an observable outcome.** Its advantage is the operational coordination built into the commands: dependency traversal, impact planning, prerequisite mutations, retries, confirmation, and useful failure evidence.

`c8ctl` is Camunda's official CLI, designed for developer access during development and testing. c8volt is an independent operations specialist for DevOps teams, support engineers, and CI pipelines. It is the better fit when recovery or cleanup needs several coordinated steps and you want less custom runbook code to maintain.

## What c8volt handles for you

| Operator need | What c8volt handles | Practical benefit |
| --- | --- | --- |
| Understand the impact | Walk ancestry and descendants; preview the affected family before mutation. | Review the full scope even when selection starts with one child. |
| Finish cancellation and deletion | Cancel relevant roots, confirm terminal state, delete children before parents, and observe deletion. | Keep dependency ordering and polling out of maintenance scripts. |
| Repair the intended variables | Preview root, element-local, or task-owned destinations and confirm requested values. | Make variable ownership visible before changing runtime state. |
| Operate user tasks and jobs | Search and inspect tasks; assign, unassign, update, or complete explicit task collections; manage job retries and worker outcomes. | Recover user tasks and service jobs from one CLI. |
| Coordinate special cases | `ops` orchestrates multiple base operations for incident repair, retention, purge, analysis, and runtime verification. | Run a reviewed playbook with outcome reporting. |
| Automate with evidence | Use structured results, supported key streams, partial outcomes, and failure exit codes. | Let CI distinguish completed work from failed or unconfirmed work. |

## Done is done

An accepted request can precede its observable result. By default, cancellation waits for affected family members to become terminal or absent; deletion waits for disappearance; variable updates verify the requested values at their intended scopes. Failed confirmation remains a failure even after an accepted mutation.

Confirmation follows each operation's contract. Starting an instance confirms creation; `expect process-instance --state completed` waits for process completion. Where supported, `--no-wait` skips final confirmation while retaining required prerequisites. Definition disappearance alone cannot confirm deferred history deletion without a returned batch.

## From a command to a playbook

The standard commands provide precise inspection, mutation, and verification. `ops` combines those operations for special cases. Incident repair, for example, can discover incidents and jobs, apply requested variables, restore retries, resolve incidents, verify clearance, and save a report in one workflow.

Start with the [ops playbooks](/ops/), or use the [CLI reference](/cli/) to build your own operational sequence. v5 adds native user tasks, explicit variable scopes, native business IDs, suspension/resumption, and definition lifecycle visibility. Examples target Camunda 8.10; 8.9 and 8.8 remain supported with version-specific API limits.

For an existing c8ctl or zbctl script, map the workflow explicitly to c8volt's command grammar. It is not a drop-in replacement. Use a Camunda SDK or REST API when the work belongs in a long-running application integration.

Use `c8volt capabilities --json` to inspect supported flags, output modes, and automation support.

[Quick start](/#fast-start) · [Configuration](/configuration/) · [Command tree](/cli/command-tree/)
