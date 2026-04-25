# Quickstart: Process-Instance Limit and Batch Size Flags

## Targeted Command Validation

Use focused command tests first:

```bash
go test ./cmd -run 'Test.*ProcessInstance.*Limit|Test.*ProcessInstance.*BatchSize|Test.*ProcessInstance.*Count' -count=1
```

If test names differ after implementation, run the closest affected package tests:

```bash
go test ./cmd -count=1
```

## Manual CLI Scenarios

The following examples describe expected behavior and should be covered by automated tests where possible:

```bash
./c8volt get pi --state active --limit 25
./c8volt get pi --state active -l 25
./c8volt get pi --state active --batch-size 250
./c8volt get pi --state active -n 250
./c8volt cancel pi --state active --limit 10 --auto-confirm
./c8volt delete pi --state completed --batch-size 250 --limit 25 --auto-confirm
```

Expected outcomes:

- `--limit` caps total matched process instances across pages.
- `--batch-size` controls per-page search size only.
- Reaching the limit stops without a continuation prompt.
- Direct `--key` plus `--limit` fails.
- `--total` plus `--limit` fails as mutually exclusive.
- Removed `--count` fails on affected command paths.

## Documentation Validation

After command metadata changes, refresh generated docs:

```bash
make docs-content
```

Then inspect:

```bash
rg -n -- '--count|--batch-size|--limit' README.md docs/cli/c8volt_get_process-instance.md docs/cli/c8volt_cancel_process-instance.md docs/cli/c8volt_delete_process-instance.md
```

Affected process-instance get/cancel/delete docs should use `--batch-size` and `--limit`, not `--count`.

## Final Validation

Run repository validation before commit:

```bash
make test
```
