---
title: "AI and Search Context"
permalink: /ai-search-context/
nav_order: 6
has_toc: false
description: "Current product scope and command grammar for tools indexing c8volt."
---

# c8volt context

`c8volt` is an independent Camunda operations CLI for DevOps teams. v5 documentation and examples target Camunda 8.10; 8.9 and 8.8 remain supported. Version-specific features are marked **only c8.10+**. It is not an official Camunda product.

Use it to inspect process state, analyse slow instances, repair incidents, deploy BPMN, and clean up runtime data and history. Mutation workflows provide previews, confirmation, and outcome reporting where supported.

## Command grammar

- Bare properties select existing resources: `--key`, `--state`, `--bid`, `--retries`.
- `--set-*` supplies values to write: `--set-vars`, `--set-vars-file`, `--set-retries`, `--set-assignee`.
- `--with-*` adds returned detail: `--with-stats`, `--with-vars`, `--with-incidents`.
- Actions include `--complete`, `--fail`, and `--throw-bpmn-error --set-error-code CODE`.
- `--timeout` controls HTTP requests; `update job --set-timeout` changes a job timeout.

v5 removes renamed legacy spellings without aliases. Do not infer support for a flag or output mode from another command. Inspect `c8volt capabilities --json` and the relevant command's help.

## Entry points

- [Quick start](/#fast-start)
- [Configuration](/configuration/)
- [Ops playbooks](/ops/)
- [CLI reference](/cli/) and [command tree](/cli/command-tree/)
- [Releases](https://github.com/grafvonb/c8volt/releases)

Query and machine results use stdout; human mutation summaries and diagnostics use the configured logger. Prompts use stderr. Use exit status and structured partial outcomes to assess failed batches. `--no-wait` establishes acceptance rather than confirmed completion where supported.
