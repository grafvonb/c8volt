---
title: "C8 Ops Discussions"
permalink: /use-cases/
nav_order: 3
has_toc: false
---

# C8 Ops Use-Case Discussions

The c8volt use-case board is the GitHub discussion space where C8 Ops CLI workflows are shaped before, during, and after implementation.

[Open the discussion board](https://github.com/grafvonb/c8volt/discussions/categories/1-use-cases-ideas){: .btn .btn-primary }

Implemented operational workflows now live in the [C8 Ops CLI playbooks](/ops/). This page remains the idea board for accepted, upcoming, and superseded workflow concepts.

## Discussion Links

### [Repair commands](https://github.com/grafvonb/c8volt/discussions/189) <span class="status-badge status-implemented">status: implemented</span>

High-level repair workflows for operator-safe remediation. The implemented `c8volt ops repair incident` and `c8volt ops repair process-instance` flows freeze repair targets, optionally update process-instance variables once per scope, apply related-job retry/timeout repair where applicable, resolve incidents, confirm clearance unless `--no-wait` is set, and write a final report that shows what was selected, attempted, skipped, and changed.

### [Orphan cleanup](https://github.com/grafvonb/c8volt/discussions/190) <span class="status-badge status-implemented">status: implemented</span>

Automated cleanup for orphan child process instances. The implemented `c8volt ops purge orphan-process-instances` flow selects candidates with orphan-child discovery, freezes those keys, then reuses existing process-instance delete planning for root traversal, affected-scope validation, dry-run reporting, `--auto-confirm`, and `--automation`.

### [Retention policy](https://github.com/grafvonb/c8volt/discussions/191) <span class="status-badge status-implemented">status: implemented</span>

Home-grown retention cleanup for completed process instances older than a configured number of days. The implemented `c8volt ops execute retention-policy` flow derives the end-date boundary, discovers candidate process instances, skips candidates whose roots are not final, and deletes final-root scopes through the existing process-instance delete service.

### [Smoke test](https://github.com/grafvonb/c8volt/discussions/56) <span class="status-badge status-implemented">status: implemented</span>

Operational smoke test for proving a c8volt-to-Camunda environment is usable end to end. The implemented `c8volt ops execute smoke-test` flow checks topology where available, deploys the version-matched embedded multiple-subprocess fixture, starts one or more instances, walks each created family, and cleans up unless `--no-cleanup` is set.

### [Purge all selected process definitions](https://github.com/grafvonb/c8volt/discussions/213) <span class="status-badge status-implemented">status: implemented</span>

Process-definition cleanup for selected versions. The implemented `c8volt ops purge all-process-definitions` flow discovers candidate process definitions with `get pd`-style filters, previews process-instance impact, blocks active-instance impact unless `--force` is supplied, and deletes selected definitions through the existing process-definition deletion service.

### [Purge process instances selected by incidents](https://github.com/grafvonb/c8volt/discussions/212) <span class="status-badge status-implemented">status: implemented</span>

Incident-driven cleanup for process-instance families. The implemented `c8volt ops purge process-instances-with-incidents` flow discovers candidate incidents, freezes candidate process-instance keys, deduplicates them, builds the shared process-instance delete plan, and deletes resolved roots after confirmation.

## Status guide

<div class="status-guide">
  <p><span class="status-badge status-idea">status: idea</span> Early concept, open for exploration.</p>
  <p><span class="status-badge status-shaping">status: shaping</span> Being refined before implementation.</p>
  <p><span class="status-badge status-accepted">status: accepted</span> Agreed direction, ready for issue/spec work.</p>
  <p><span class="status-badge status-superseded">status: superseded</span> Replaced by newer discussion or issue.</p>
  <p><span class="status-badge status-implemented">status: implemented</span> Delivered in the codebase.</p>
</div>
