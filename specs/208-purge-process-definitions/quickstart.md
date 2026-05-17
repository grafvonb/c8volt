# Quickstart: Ops Purge All Process Definitions

## Manual Smoke Scenarios

Build the CLI:

```bash
go build -o /tmp/c8volt-all-pds-purge .
```

Preview all process-definition purge:

```bash
/tmp/c8volt-all-pds-purge ops purge all-process-definitions --dry-run
```

Preview cleanup with process-definition filters:

```bash
/tmp/c8volt-all-pds-purge ops purge all-process-definitions --bpmn-process-id invoice-process --dry-run
```

Preview latest definitions only:

```bash
/tmp/c8volt-all-pds-purge ops purge all-process-definitions --latest --dry-run
```

Preview a specific version or version tag:

```bash
/tmp/c8volt-all-pds-purge ops purge all-process-definitions --bpmn-process-id invoice-process --pd-version 3 --dry-run
/tmp/c8volt-all-pds-purge ops purge all-process-definitions --pd-version-tag cleanup-target --dry-run
```

Run through the alias:

```bash
/tmp/c8volt-all-pds-purge ops purge all-pds --dry-run
```

Run confirmed cleanup:

```bash
/tmp/c8volt-all-pds-purge ops purge all-process-definitions --auto-confirm
```

Run confirmed cleanup with force:

```bash
/tmp/c8volt-all-pds-purge ops purge all-process-definitions --auto-confirm --force
```

Run in automation with JSON:

```bash
/tmp/c8volt-all-pds-purge ops purge all-process-definitions --automation --json --dry-run
```

Write a Markdown dry-run report:

```bash
/tmp/c8volt-all-pds-purge ops purge all-process-definitions --dry-run --report-file all-pds-purge.md
```

Write a JSON execution report:

```bash
/tmp/c8volt-all-pds-purge ops purge all-process-definitions --auto-confirm --report-file all-pds-purge.json --report-format json
```

## Expected Behavior

- `ops purge` alone shows grouping help and performs no cleanup.
- `all-process-definitions --dry-run` discovers candidate process-definition versions, freezes candidate process-definition keys, builds the existing delete preflight, reports planned scope, and sends no delete, cancel, or history cleanup requests.
- No-target cleanup exits successfully and clearly reports zero candidate process definitions.
- `--latest` narrows scope explicitly and is visible in output/report data.
- Unsafe active process-instance impact blocks destructive runs unless `--force` is supplied.
- `--automation --json` works without `--auto-confirm` for this supported state-changing ops command and keeps stdout deterministic JSON.
- Existing report files are preserved for dry-run, aborted, unconfirmed, or locally blocked runs unless overwrite is already allowed.
- Requested reports include timestamps, filters, candidate process definitions, delete preflight, deletion status, errors, notices, and final outcome.

## Validation Commands

Run targeted command tests during implementation:

```bash
go test ./cmd -run 'TestOpsPurge|TestCommandContract|TestDeleteProcessDefinition|TestGetProcessDefinition' -count=1
```

Run facade/service tests when those layers change:

```bash
go test ./c8volt/ops ./c8volt/processdefinition ./c8volt/resource ./internal/services/ops ./internal/services/processdefinition ./internal/services/resource -count=1
```

Refresh generated CLI docs after command metadata changes:

```bash
make docs-content
```

Run repository validation before committing completed work:

```bash
make test
```
