---
title: "CLI Reference"
nav_order: 2
nav_exclude: false
has_children: true
has_toc: false
description: "Camunda CLI reference for c8volt: inspect processes, operate user tasks, deploy BPMN, repair incidents, and automate verified operations."
---

# c8volt Camunda CLI reference

Command pages are generated from the executable's command tree. Use them for exact flags, aliases, and supported output modes. Examples target Camunda 8.10; 8.9 and 8.8 remain supported with version-specific API limits.

Standard commands provide inspection, dependency traversal, mutation, and outcome confirmation. `ops` orchestrates multiple base operations into playbooks for special cases. See [why choose c8volt](/camunda-cli/) for the operational benefits.

- [Command tree](/cli/command-tree/)
- [Root command and global flags](/cli/c8volt/)
- [Ops playbooks](/ops/)
- [Runtime elements and listeners](/cli/runtime-elements/)
- [Configuration](/configuration/)

| Task | Commands |
| --- | --- |
| Inspect | [get](./c8volt_get), [walk](./c8volt_walk), [ops analyse](./c8volt_ops_analyse) |
| Operate native user tasks | [get user-task](./c8volt_get_user-task), [update user-task](./c8volt_update_user-task) |
| Inspect and update exact variable scopes | [get element](./c8volt_get_element), [update element](./c8volt_update_element), [update process-instance](./c8volt_update_process-instance), [update user-task](./c8volt_update_user-task) |
| Deploy and start | [deploy](./c8volt_deploy), [embed](./c8volt_embed), [run](./c8volt_run) |
| Change state or variables | [update](./c8volt_update), [resolve](./c8volt_resolve), [suspend](./c8volt_suspend), [resume](./c8volt_resume) |
| Clean up | [cancel](./c8volt_cancel), [delete](./c8volt_delete), [ops purge](./c8volt_ops_purge), [ops execute retention-policy](./c8volt_ops_execute_retention-policy) |
| Verify | [expect](./c8volt_expect), [config test-connection](./c8volt_config_test-connection), [ops execute smoke-test](./c8volt_ops_execute_smoke-test) |

Use `--dry-run` before repair or deletion where supported. Use `--json` for structured results or `--keys-only` for pipelines where supported. `c8volt capabilities --json` describes each command's available flags, output modes, and automation support.

Bare properties select resources, `--set-*` supplies values to write, and `--with-*` adds query detail. v5 has no aliases for renamed legacy flags.
