---
title: "Use Cases"
permalink: /use-cases/
nav_order: 2
has_toc: false
---

# Use Cases & Ideas

The c8volt use-case board is where operational workflows are shaped before they become CLI behavior.

[Open the discussion board](https://github.com/grafvonb/c8volt/discussions/categories/1-use-cases-ideas){: .btn .btn-primary }

## Ops use cases

### [Repair commands](https://github.com/grafvonb/c8volt/discussions/191) <span class="status-badge status-accepted">status: accepted</span>

High-level repair workflows for operator-safe remediation. The goal is to turn multi-step recovery work into explicit `c8volt ops repair ...` commands with dry-run previews, confirmation controls, automation support, and a final report that shows what was selected, attempted, skipped, and changed.

### [Orphan cleanup](https://github.com/grafvonb/c8volt/discussions/190) <span class="status-badge status-accepted">status: accepted</span>

Automated cleanup for orphan child process instances. The planned flow selects candidates with `get pi --orphan-children-only --keys-only`, then passes those keys into `delete pi --keys`, reusing existing delete behavior for root traversal, duplicate removal, dry-run reporting, `--auto-confirm`, and `--automation`.

### [Retention policy](https://github.com/grafvonb/c8volt/discussions/189) <span class="status-badge status-accepted">status: accepted</span>

Home-grown retention cleanup for completed process instances older than a configured number of days. The planned flow selects keys with `get pi --end-date-older-days --keys-only`, then deletes them through the existing process-instance delete service so c8volt keeps full control over filtering, concurrency, traversal, reporting, and execution timing.

### [Smoke test](https://github.com/grafvonb/c8volt/discussions/56) <span class="status-badge status-accepted">status: accepted</span>

Operational smoke test for proving a c8volt-to-Camunda environment is usable end to end. The planned flow checks the connection, deploys the embedded `C89_MultipleSubProcessesParentProcess`, starts one or more instances, walks the process tree, deletes the created instances, and removes the deployed process definition only when no independently created instances still exist.

## Status guide

<div class="status-guide">
  <p><span class="status-badge status-idea">status: idea</span> Early concept, open for exploration.</p>
  <p><span class="status-badge status-shaping">status: shaping</span> Being refined before implementation.</p>
  <p><span class="status-badge status-accepted">status: accepted</span> Agreed direction, ready for issue/spec work.</p>
  <p><span class="status-badge status-superseded">status: superseded</span> Replaced by newer discussion or issue.</p>
  <p><span class="status-badge status-implemented">status: implemented</span> Delivered in the codebase.</p>
</div>
