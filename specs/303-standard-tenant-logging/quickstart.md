# Validation Guide: Standard Tenant Logging

## Prerequisites

- Repository checkout on `codex/303-standard-tenant-logging`.
- Go toolchain declared by `go.mod` (go1.26.2), make, and downloaded module dependencies.
- Host support for the repository's real-terminal subprocess fixtures and race detector.
- Apply the planned implementation and regression tests first. These commands also run current tests before implementation, but a baseline pass alone does not prove the new behavior.
- No live Camunda environment is required for the stub-backed command tests below.

## Verified Implementation Environment

- Verified on 2026-09-11 from repository branch `codex/303-standard-tenant-logging`.
- `go.mod` declares Go 1.26 with toolchain `go1.26.2`; the active toolchain is `go version go1.26.2 darwin/arm64`.
- `make` is available at `/usr/bin/make`.
- The feature specification, plan, tasks, research, data model, contract, validation guide, `AGENTS.md`, and `specs/ralph-implementation-rules.md` were reviewed with no conflicts found.
- No implementation-environment blockers were identified.

## Pre-Implementation Baseline

- On 2026-09-11, `go test ./cmd -run 'Test(ProcessInstanceMutation|CancelProcessInstance|DeleteProcessInstance|Confirmation)' -count=1` passed (`ok github.com/grafvonb/c8volt/cmd`, 1.905s).
- The passing baseline confirms the existing command and fixture behavior only. It does not prove the planned attached-logger severity, formatting, or threshold-filtering contract.
- Existing support includes `logging.New` attached with `logging.ToContext`, separate stdout/stderr buffers, `resetProcessInstanceCommandGlobals` cleanup, and `testx.NewCmdTerminalRunner` for real-terminal prompt coverage.

Run from the repository root:

```sh
go test ./cmd -run 'TestProcessInstanceMutation(Tenant|Progress)' -count=1
go test ./cmd -run 'Test(Cancel|Delete)ProcessInstance' -count=1
go test ./cmd -run 'TestConfirmation' -count=1
```

Name new focused tests with the `TestProcessInstanceMutationTenant` prefix so the first command includes them. Require actual executed subtests for both emitters, plain/JSON formats, and INFO/WARN/ERROR levels; a successful command with no matching tests is insufficient.

## Expected Evidence

Follow [contracts/tenant-logging.md](contracts/tenant-logging.md) and [data-model.md](data-model.md):

1. Attach an existing standard logger to each test command, directing diagnostics to a buffer separate from stdout. At INFO threshold verify informational and warning records with exact original text/order. At WARN verify only warnings; at ERROR verify zero tenant records and no raw fallback. Decode JSON log records instead of checking only string fragments.
2. Repeat reporting to verify no duplicates. Check missing context and the existing no-logger fallback. Preserve flags with existing cleanup helpers.
3. Exercise both command execution paths across ordinary, verbose, quiet, automation, auto-confirm, dry-run, JSON-result, keys-only, and supported combinations. Verify exact machine results, JSON envelope plus EOF, zero-byte empty keys output, and no extra discovery/mutation calls. Keep dry-run preview output unchanged.
4. Run real-terminal confirmation fixtures with separate result/diagnostic capture. Retain configured/inherited stderr, acceptance/abort behavior, and prompt-free empty selections. Prompts remain plain even when diagnostics use JSON logging.
5. Retain sparse-page continuation, direct-key, failure, and no-wait regressions through the existing command suite.

## Documentation and Full Validation

After updating README and command help metadata:

```sh
make docs-content
git diff --check
make test
```

Run gofmt on all touched Go files before the final checks. Inspect regenerated documentation and confirm it only reflects the planned correction. `make test` must pass with the race detector before implementation commit or merge. Record any unavailable terminal checks or other validation failures explicitly; do not report an unrun check as passing.
