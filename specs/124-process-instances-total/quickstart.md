# Quickstart: Add Process-Instance Total-Only Output

## Planned Behavior

- `./c8volt get pi --total` returns only the number of matching process instances.
- `./c8volt get pi --state active --total` returns a single numeric result with no instance-detail lines.
- `./c8volt get pi --state completed --total` returns `0` when nothing matches.
- When the backend exposes a capped total, `--total` still returns that numeric lower-bound value unchanged.
- `--total` is search/list-only and should reject `--key`, `--json`, `--keys-only`, and `--with-age`.

## Implementation Notes

- Add the flag and validation in `cmd/get_processinstance.go`.
- Keep default detail rendering in `cmd/cmd_views_get.go` unchanged; implement count-only output as a narrow branch before the existing list renderer.
- Extend shared page models under `internal/domain/processinstance.go` and `c8volt/process/model.go` plus conversions so count-only mode can use backend-reported totals without version-specific command logic.
- Represent backend totals as optional `ReportedTotal{Count, Kind}` page metadata, where `Kind` is `exact` or `lower_bound` and omission means no trustworthy total is available.
- Update `internal/services/processinstance/v87`, `v88`, and `v89` to populate the new reported-total metadata.
- Update `README.md` and regenerate CLI docs with `make docs-content`.

## Verification Focus

1. Confirm `--total` prints only a number for search/list invocations.
2. Confirm zero-match searches print `0`.
3. Confirm capped totals on `v8.8` and `v8.9` remain numeric lower bounds instead of triggering recounts or failures.
4. Confirm `--total` is rejected with `--key`, `--json`, `--keys-only`, and `--with-age`.
5. Confirm default non-`--total` detail output remains unchanged.
6. Confirm README and generated CLI docs both reflect the new flag.

## Suggested Verification Commands

```bash
go test ./c8volt/process -count=1
go test ./internal/services/processinstance/... -count=1
go test ./cmd -count=1
make docs-content
make test
```

Run the focused model/service/command suites first, then regenerate docs, then finish with the full repository gate.

## Manual Smoke Ideas

```bash
./c8volt get pi --state active --total
./c8volt get pi --bpmn-process-id order-process --state active --total
./c8volt get pi --state completed --total
./c8volt get pi --key 2251799813711967 --total
./c8volt get pi --state active --total --json
```

Check that:

- the first three commands print only a number
- the third command prints `0` when no matches exist
- the fourth and fifth commands fail with clear validation errors
- default non-`--total` usage such as `./c8volt get pi --state active` still prints detail lines plus `found:`
