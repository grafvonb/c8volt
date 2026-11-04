---
title: "C8 Ops CLI"
permalink: /ops/
nav_order: 3
has_children: true
has_toc: true
---

# Ops playbooks

`c8volt ops` orchestrates the operations exposed by multiple base commands into one playbook. It is a c8volt highlight for special cases that otherwise need a maintenance script: incident recovery, retention, dependency cleanup, and runtime verification. Examples target Camunda 8.10.

For example, incident repair can inspect incidents and jobs, apply requested variables, restore retries, resolve incidents, and verify clearance. Mutation playbooks coordinate discovery, impact planning, approval, execution, and confirmation, with reports for review and automation. Analysis remains read-only.

| Task | Playbook |
| --- | --- |
| Find slow instances and elements | [Analyse slow process instances](/ops/analyse-slow-process-instances/) |
| Repair incidents by key or filter | [Repair incident](/ops/repair-incident/) |
| Repair an instance's active incidents | [Repair process instance](/ops/repair-process-instance/) |
| Delete old finished instances | [Retention policy](/ops/execute-retention-policy/) |
| Delete incident-affected families | [Purge instances with incidents](/ops/purge-process-instances-with-incidents/) |
| Delete children with missing parents | [Purge orphan instances](/ops/purge-orphan-process-instances/) |
| Delete definitions and history | [Purge process definitions](/ops/purge-all-process-definitions/) |
| Check deployment, execution, and cleanup | [Smoke test](/ops/execute-smoke-test/) |

## Shared controls

Controls vary by playbook; the command reference lists supported flags and output modes.

- **Preview:** use `--dry-run` on mutation workflows. A preview can read cluster state but makes no changes. Analysis is read-only.
- **Scope:** `--batch-size` controls discovery page size; `--limit` bounds selected candidates. Family expansion may include additional relatives. Complete empty discovery needs no confirmation or mutation.
- **Approval:** inspect the plan before confirming. Use `--auto-confirm` or supported `--automation` for unattended runs. A later invocation discovers a fresh scope.
- **Tenants:** a named tenant narrows discovery; `--all-tenants` clears that filter within authorized access. Explicit keys use backend authorization. Creation requires one destination tenant.
- **Completion:** mutations wait for an observed outcome by default. Failed confirmation remains a failure even after an accepted request.
- **Reports:** mutation playbooks support `--report-file report.md`; use `--report-format json` for a JSON file. Use a new path for each preview or execution; existing files are not overwritten. Reports retain available partial outcomes. Analysis returns command results directly. Use `--json` for machine output and the exit status to detect failure.

Use `--verbose` for per-resource outcomes and `--debug` for redacted HTTP diagnostics. See [configuration](/configuration/) for profiles, authentication, and logging, and the [CLI reference](/cli/) for all flags.
