# Quickstart: Review and Refactor CLI Error Code Usage

## Goal

Implement one shared CLI failure model across all existing commands, preserving compatibility where documented while making exit codes and failure semantics more predictable for humans, shell automation, and AI agents.

## Prerequisites

- Go 1.25.3 available locally
- Repository dependencies available through the existing Go module setup
- Working tree on branch `19-cli-error-model`

## Implementation Steps

1. Review `c8volt/ferrors`, `internal/exitcode`, `internal/domain/errors.go`, `internal/services/errors.go`, and `cmd/cmd_errors.go` to define the final shared CLI error classes and numeric exit mapping.
2. Introduce or extend shared classifier helpers in `c8volt/ferrors` so command and root pre-run failures normalize through one consistent path before `HandleAndExit`.
3. Sweep the existing command surface under `cmd/` to replace ad hoc wrapped failure handling with the shared classification model while preserving command names, flags, and output success paths.
4. Cover representative root, get, run, deploy, cancel, delete, expect, walk, embed, and config failure paths with subprocess-based CLI tests that assert exit codes and stderr behavior, including `--no-err-codes`.
5. Update `README.md` and `docs/index.md` so the documented scripting and error-code contract matches the new failure model. Regenerate `docs/cli/` only if help text changes.

## Final Validation Sequence

```bash
go test ./cmd/... ./c8volt/ferrors/... -count=1
make test
```

Run the targeted command and shared-error tests first so exit-code regressions are easier to isolate before the repository-wide suite.

## Validation Notes

- The closeout proof for this feature is the two-step sequence above: targeted CLI/failure-model coverage first, then the repository-wide `make test` run.
- Re-run `make docs` only if Cobra help text changes; the current shared error-model work does not require regenerated CLI reference output.

## Documentation Impact

- `README.md` and `docs/index.md` should be updated because this feature changes documented failure semantics.
- Generated CLI reference docs under `docs/cli/` only need regeneration if command help text or flag descriptions are intentionally changed.

If implementation changes Cobra help text:

```bash
make docs
```

## Completion Checklist

- One shared CLI error classification and exit-code mapping is defined
- `--no-err-codes` still forces exit code `0` without bypassing classification
- All existing CLI commands route failures through the shared model
- Representative failure groups are covered by subprocess-based tests
- Scripting documentation matches shipped behavior
- Targeted tests pass before running `make test`
- `make test` passes
