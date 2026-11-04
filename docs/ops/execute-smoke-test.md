---
title: "Execute Smoke Test"
permalink: /ops/execute-smoke-test/
parent: "C8 Ops CLI"
nav_order: 8
has_toc: true
---

# Execute smoke test

Check that a profile can connect, deploy a bundled fixture, start instances, inspect their families, and clean up.

## Examples

```bash
c8volt ops execute smoke-test --dry-run
c8volt ops execute smoke-test --report-file smoke-test.md
```

## Execution

Dry-run checks connectivity and previews the steps without deploying or starting instances. The workflow requires one destination tenant and rejects `--all-tenants`.

Execution creates and deletes cluster data. Cleanup tracks every deployed fixture definition, including dependencies, and removes it only when no unrelated instances use it. Identical deployments can reuse existing definition versions, which cleanup can also remove. Retained or failed cleanup is reported.

Deployment and instance creation are confirmed before inspecting the families.

See the [shared ops controls](/ops/#shared-controls) for discovery, confirmation, tenant scope, and reports.

[Command reference](/cli/c8volt_ops_execute_smoke-test) · [config test-connection](/cli/c8volt_config_test-connection) · [run process-instance](/cli/c8volt_run_process-instance) · [walk process-instance](/cli/c8volt_walk_process-instance)
