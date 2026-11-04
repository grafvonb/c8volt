---
title: "CLI Reference"
nav_order: 7
nav_exclude: false
has_children: true
has_toc: false
---

# CLI reference

Command pages are generated from the executable's command tree. Use them for exact flags, aliases, and supported output modes. Examples target Camunda 8.10; 8.9 and 8.8 remain supported. Features requiring 8.10 are marked **only c8.10+**.

- [Command tree](/cli/command-tree/)
- [Root command and global flags](/cli/c8volt/)
- [Ops playbooks](/ops/)
- [Runtime elements and listeners](/cli/runtime-elements/)
- [Configuration](/configuration/)

| Task | Commands |
| --- | --- |
| Inspect | `get`, `walk`, `ops analyse` |
| Deploy and start | `deploy`, `embed`, `run` |
| Change state or variables | `update`, `resolve`, `suspend`, `resume` |
| Clean up | `cancel`, `delete`, `ops purge`, `ops execute retention-policy` |
| Verify | `expect`, `config test-connection`, `ops execute smoke-test` |

Use `--dry-run` before repair or deletion where supported. Use `--json` for structured results or `--keys-only` for pipelines where supported. `c8volt capabilities --json` describes each command's available flags, output modes, and automation support.

Bare properties select resources, `--set-*` supplies values to write, and `--with-*` adds query detail. v5 has no aliases for renamed legacy flags.
