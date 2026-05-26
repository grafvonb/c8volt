# Quickstart: Ops Paged Discovery Scope

## Planning And Ralph Context

Every implementation session must start with:

```bash
sed -n '1,240p' specs/ralph-implementation-rules.md
```

Ralph launch instructions must include:

```text
--implementation-context specs/ralph-implementation-rules.md
```

## Targeted Validation

Run targeted tests while implementing each slice:

```bash
go test ./internal/services/ops -count=1
go test ./cmd -run 'TestOps.*(PurgeProcessInstancesWithIncidents|RepairIncident|RepairProcessInstance|PurgeAllProcessDefinitions)' -count=1
go test ./cmd ./internal/services/ops ./c8volt/ops -count=1
```

After command help or docs metadata changes:

```bash
make docs-content
go test ./docsgen ./cmd -count=1
```

Before commit:

```bash
make test
```

## Manual Smoke Commands

Use a configured test environment or fake server fixture capable of returning multiple pages:

```bash
./c8volt ops purge piwi --batch-size 2 --dry-run --json
./c8volt ops repair incident --batch-size 2 --dry-run --json
./c8volt ops repair process-instance --batch-size 2 --dry-run --json
./c8volt ops purge apd --batch-size 2 --dry-run --json
```

Then verify:

- No `--limit` means discovery completes across all pages.
- `--limit N` means frozen candidates stop at N and output is user-limited.
- Confirmation paths reuse the dry-run frozen candidate keys.
- Markdown reports and JSON output agree on discovery status.
