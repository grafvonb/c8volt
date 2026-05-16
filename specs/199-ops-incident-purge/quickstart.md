# Quickstart: Ops Purge Process Instances With Incidents

## Manual Smoke Scenarios

Build the CLI:

```bash
go build -o /tmp/c8volt-incident-purge .
```

Preview incident-based purge:

```bash
/tmp/c8volt-incident-purge ops purge process-instances-with-incidents --dry-run
```

Preview cleanup with incident filters:

```bash
/tmp/c8volt-incident-purge ops purge process-instances-with-incidents --state active --error-type io_mapping_error --dry-run
```

Preview a bounded cleanup:

```bash
/tmp/c8volt-incident-purge ops purge process-instances-with-incidents --state active --limit 5 --dry-run
```

Run through the alias:

```bash
/tmp/c8volt-incident-purge ops purge pi-with-incidents --state active --dry-run
```

Run confirmed cleanup:

```bash
/tmp/c8volt-incident-purge ops purge process-instances-with-incidents --auto-confirm
```

Run confirmed cleanup with force:

```bash
/tmp/c8volt-incident-purge ops purge process-instances-with-incidents --auto-confirm --force
```

Run in automation with JSON:

```bash
/tmp/c8volt-incident-purge ops purge process-instances-with-incidents --automation --json --dry-run
```

Write a Markdown dry-run report:

```bash
/tmp/c8volt-incident-purge ops purge process-instances-with-incidents --dry-run --report-file incident-purge.md
```

Write a JSON execution report:

```bash
/tmp/c8volt-incident-purge ops purge process-instances-with-incidents --auto-confirm --report-file incident-purge.json --report-format json
```

## Expected Behavior

- `ops purge` alone shows grouping help and performs no cleanup.
- `process-instances-with-incidents --dry-run` discovers candidate incidents, freezes candidate process-instance keys, builds the existing delete plan, reports planned scope, and sends no delete or cancel requests.
- No-target cleanup exits successfully and clearly reports zero candidate incidents and zero candidate process instances.
- `--key` filters incident key, not process-instance key.
- `--limit` limits matching incidents before candidate process-instance dedupe.
- Non-final affected process instances block destructive runs unless `--force` is supplied.
- `--automation --json` works without `--auto-confirm` for this supported state-changing ops command and keeps stdout deterministic JSON.
- Existing report files are preserved for dry-run, aborted, unconfirmed, or locally blocked runs unless overwrite is already allowed.
- Requested reports include timestamps, filters, candidate incidents, candidate process instances, delete plan, deletion status, errors, notices, and final outcome.

## Validation Commands

Run targeted command tests during implementation:

```bash
go test ./cmd -run 'TestOpsPurge|TestCommandContract|TestDeleteProcessInstance|TestGetIncident' -count=1
```

Run facade/service tests when those layers change:

```bash
go test ./c8volt/ops ./c8volt/incident ./c8volt/process ./internal/services/ops ./internal/services/incident ./internal/services/processinstance -count=1
```

Refresh generated CLI docs after command metadata changes:

```bash
make docs-content
```

Run repository validation before committing completed work:

```bash
make test
```
