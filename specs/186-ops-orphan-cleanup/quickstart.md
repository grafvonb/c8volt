# Quickstart: Ops Purge Orphan Process Instances

## Manual Smoke Scenarios

Build the CLI:

```bash
go build -o /tmp/c8volt-ops-cleanup .
```

Preview cleanup:

```bash
/tmp/c8volt-ops-cleanup ops purge orphan-process-instances --dry-run
```

Preview cleanup with a bounded selection:

```bash
/tmp/c8volt-ops-cleanup ops purge orphan-process-instances --dry-run --bpmn-process-id C89_SimpleUserTask_Process --limit 25
```

Run confirmed cleanup:

```bash
/tmp/c8volt-ops-cleanup ops purge orphan-process-instances --auto-confirm
```

Run in automation with JSON:

```bash
/tmp/c8volt-ops-cleanup ops purge orphan-process-instances --automation --auto-confirm --json
```

Write a Markdown report:

```bash
/tmp/c8volt-ops-cleanup ops purge orphan-process-instances --dry-run --report-file orphan-purge.md
```

Write a JSON report:

```bash
/tmp/c8volt-ops-cleanup ops purge orphan-process-instances --auto-confirm --report-file orphan-purge.json --report-format json
```

## Expected Behavior

- `ops purge` alone shows grouping help and performs no cleanup.
- `orphan-process-instances --dry-run` discovers and validates purge targets but sends no delete requests.
- No-target cleanup exits successfully and clearly reports that no orphan child process instances were found.
- `--automation` without `--auto-confirm` fails before mutation when targets exist.
- JSON output uses the shared command result envelope and keeps stdout deterministic.
- Requested reports include timestamps, filters, discovered keys, delete status, errors, and final outcome.

## Validation Commands

Run targeted command tests during implementation:

```bash
go test ./cmd -run 'TestOps|TestCommandContract|TestDeleteProcessInstance' -count=1
```

Run facade/service tests when those layers change:

```bash
go test ./c8volt/ops ./c8volt/process ./internal/services/processinstance -count=1
```

Refresh generated CLI docs after command metadata changes:

```bash
make docs-content
```

Run repository validation before committing completed work:

```bash
make test
```
