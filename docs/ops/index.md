---
title: "C8 Ops CLI"
permalink: /ops/
nav_order: 3
has_children: true
has_toc: true
---

# Ops playbooks

Use `c8volt ops` to inspect performance, repair incidents, and clean up Camunda data. Playbooks discover targets, prepare a plan, execute approved changes, and report outcomes. Examples target Camunda 8.10.

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

- **Preview:** use `--dry-run` on mutation workflows. A preview can read cluster state but makes no changes. Analysis is read-only.
- **Scope:** `--batch-size` controls discovery page size; `--limit` bounds selected candidates. Family expansion may include additional relatives. Complete empty discovery needs no confirmation or mutation.
- **Approval:** inspect the plan before confirming. Use `--auto-confirm` or supported `--automation` for unattended runs. A later invocation discovers a fresh scope.
- **Tenants:** a named tenant narrows discovery; `--all-tenants` clears that filter within authorized access. Explicit keys use backend authorization. Creation requires one destination tenant.
- **Completion:** mutations normally wait for an observed outcome. Where supported, `--no-wait` reports acceptance; required safety checks still run.
- **Reports:** use `--report-file report.md` to save an audit report; use `--report-format json` for a JSON file. `--json` selects machine output. Failed runs retain available partial outcomes and a nonzero exit status.

Use `--verbose` for per-resource outcomes and `--debug` for redacted HTTP diagnostics. See [configuration](/configuration/) for profiles, authentication, and logging, and the [CLI reference](/cli/) for all flags.
