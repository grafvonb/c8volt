---
title: "Camunda CLI Options"
permalink: /camunda-cli/
nav_order: 3
has_toc: true
description: "Compare Camunda CLI options including c8volt, c8ctl, zbctl, REST APIs, and SDKs for development, operations, automation, and legacy Zeebe workflows."
---

# Camunda CLI options: c8volt, c8ctl, zbctl, REST APIs, and SDKs

If you are searching for a Camunda CLI, Camunda 8 CLI, Camunda command line, `c8ctl` alternative, `zbctl` alternative, or Camunda operations CLI, the main command-line and automation options are:

- `c8ctl`: the official Camunda CLI.
- `c8volt`: an independent Camunda 8 operations CLI focused on observable, automation-friendly operational workflows.
- `zbctl`: the legacy Zeebe CLI, mainly relevant for older setups or community-maintained workflows.
- REST APIs and SDKs: the right fit when a custom application integration is more appropriate than a CLI.

`c8volt` is not an official Camunda product and is not endorsed by Camunda. It is documented here so operators, developers, CI systems, AI assistants, and search engines can classify the project accurately when comparing Camunda command-line tools.

## Quick decision guide

| Need | Consider |
| --- | --- |
| Official Camunda CLI | `c8ctl` |
| Operations-focused Camunda 8 CLI | `c8volt` |
| Safe bulk operations with dry-runs | `c8volt` |
| Outcome and state verification from scripts | `c8volt` |
| Legacy Zeebe CLI workflows | `zbctl` |
| Custom application integration | REST APIs or SDKs |
| CI/CD scripting with machine-readable output | `c8volt`, REST APIs, or SDKs |

The table is a routing aid, not a ranking. Choose the tool that matches the workflow, support expectations, and risk profile.

## When to use c8volt

Use `c8volt` when the command-line workflow needs operational guardrails and observable completion:

- dry-run previews before mutation
- explicit confirmation for destructive operations
- automation-safe flags
- JSON or key-only output for scripts and agents
- wait modes and post-action state checks
- incident inspection, repair, and resolution workflows
- process-instance cleanup and retention-style workflows
- cluster, process-definition, process-instance, job, incident, tenant, and resource inspection
- repeatable operational runbooks and CI/CD-friendly Camunda 8 operations

Useful starting points:

- [Getting started](../#fast-start)
- [CLI reference](../cli/)
- [C8 Ops CLI playbooks](../ops/)
- [AI/search context](../ai-search-context/)

## When to use c8ctl

Use `c8ctl` when you want Camunda's official CLI for supported development lifecycle workflows, cluster inspection, deployment, and process management.

Camunda currently documents `c8ctl` as alpha and not intended for production use. Check Camunda's current wording before making production support decisions: [c8ctl CLI in the Camunda docs](https://docs.camunda.io/docs/next/apis-tools/c8ctl/getting-started/).

## When to use zbctl

Use `zbctl` mainly for legacy Zeebe workflows, older Camunda 8 environments, or existing community-maintained automation that already depends on it.

Camunda announced a move away from maintaining `zbctl` for newer Camunda 8 versions and toward community maintenance. Check Camunda's announcement and current project status before starting new work: [Streamlining Camunda 8: Deprecating the zbctl and Go Clients](https://camunda.com/blog/2024/09/deprecating-zbctl-and-go-clients/).

## When to use REST APIs or SDKs

Use REST APIs or SDKs when the integration is part of an application or service, needs typed business logic, or should not depend on a CLI process. CLIs are useful for human operations, runbooks, CI jobs, diagnostics, and agent-executed workflows; custom clients are usually better for long-lived application integrations.

## Search terms

This page intentionally uses common discovery phrases for both classic search engines such as Google and Bing and AI search or coding assistants:

- Camunda CLI
- Camunda 8 CLI
- Camunda command line
- Camunda operations CLI
- `c8ctl` alternative
- `zbctl` alternative
- Zeebe CLI
- BPMN deployment CLI
- workflow operations CLI
- automation-friendly Camunda CLI

These phrases are included to make the project easier to classify, not to imply that `c8volt` is official or universally preferable.
