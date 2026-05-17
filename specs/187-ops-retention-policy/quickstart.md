# Quickstart: Ops Execute Retention Policy

## Manual Smoke Scenarios

Build the CLI:

```bash
go build -o /tmp/c8volt-retention .
```

Preview retention cleanup:

```bash
/tmp/c8volt-retention ops execute retention-policy --retention-days 90 --dry-run
```

Preview cleanup with a bounded selection:

```bash
/tmp/c8volt-retention ops execute retention-policy --retention-days 90 --dry-run --bpmn-process-id invoice-process --state completed --limit 25
```

Run confirmed cleanup:

```bash
/tmp/c8volt-retention ops execute retention-policy --retention-days 90 --auto-confirm
```

Run in automation with JSON:

```bash
/tmp/c8volt-retention ops execute retention-policy --retention-days 180 --automation --json
```

Write a Markdown dry-run report:

```bash
/tmp/c8volt-retention ops execute retention-policy --retention-days 90 --dry-run --report-file retention-report.md
```

Write a JSON execution report:

```bash
/tmp/c8volt-retention ops execute retention-policy --retention-days 180 --auto-confirm --report-file retention-report.json --report-format json
```

## Expected Behavior

- `ops execute` alone shows grouping help and performs no cleanup.
- Missing or invalid `--retention-days` fails locally before remote calls.
- `retention-policy --dry-run` discovers retention seed keys, builds the existing delete plan, reports planned scope, and sends no delete or cancel requests.
- No-target cleanup exits successfully and clearly reports that no retention-eligible process instances were found.
- `--automation --json` works without `--auto-confirm` for this supported state-changing ops command and keeps stdout deterministic JSON.
- Existing report files are preserved for dry-run, aborted, unconfirmed, or locally blocked runs unless destructive execution is already confirmed.
- Requested reports include timestamps, retention days, derived boundary, filters, discovered seeds, resolved roots, affected scope, deletion status, errors, and final outcome.

## Validation Commands

Run targeted command tests during implementation:

```bash
go test ./cmd -run 'TestOps|TestCommandContract|TestDeleteProcessInstance|TestGetProcessInstance' -count=1
```

Run facade/service tests when those layers change:

```bash
go test ./c8volt/ops ./c8volt/process ./internal/services/ops ./internal/services/processinstance -count=1
```

Refresh generated CLI docs after command metadata changes:

```bash
make docs-content
```

Run repository validation before committing completed work:

```bash
make test
```
