---
title: "CLI Reference"
nav_order: 7
nav_exclude: false
has_children: true
has_toc: false
---

# CLI reference

Command pages are generated from the executable's command tree. Use them for exact flags, aliases, and supported output modes. Examples target Camunda 8.10; 8.9 and 8.8 remain supported with version-specific API limits.

Standard commands provide inspection, dependency traversal, mutation, and outcome confirmation. `ops` orchestrates multiple base operations into playbooks for special cases. See [why choose c8volt](/camunda-cli/) for the operational benefits.

- [Command tree](/cli/command-tree/)
- [Root command and global flags](/cli/c8volt/)
- [Ops playbooks](/ops/)
- [Runtime elements and listeners](/cli/runtime-elements/)
- [Configuration](/configuration/)

| Task | Commands |
| --- | --- |
| Inspect | `get`, `walk`, `ops analyse` |
| Operate native user tasks | `get user-task`, `update user-task` |
| Inspect and update exact variable scopes | `get element`, `update element`, `update process-instance`, `update user-task` |
| Deploy and start | `deploy`, `embed`, `run` |
| Change state or variables | `update`, `resolve`, `suspend`, `resume` |
| Clean up | `cancel`, `delete`, `ops purge`, `ops execute retention-policy` |
| Verify | `expect`, `config test-connection`, `ops execute smoke-test` |

Use `--dry-run` before repair or deletion where supported. Use `--json` for structured results or `--keys-only` for pipelines where supported. `c8volt capabilities --json` describes each command's available flags, output modes, and automation support.

Bare properties select resources, `--set-*` supplies values to write, and `--with-*` adds query detail. v5 has no aliases for renamed legacy flags.
