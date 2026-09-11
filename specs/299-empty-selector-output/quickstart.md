# Quickstart Validation: Empty Selector Result Output

## Prerequisites

- Work from the repository root on `codex/299-empty-selector-output`.
- Use the toolchain specified by `go.mod` (Go 1.26, toolchain go1.26.2), with module dependencies available.
- Have a supported local terminal/PTY environment for `testx.NewCmdTerminalRunner` and the platform tooling needed by Go's race detector.
- Implement the planned regression cases before expecting the new behavior to pass. This guide records validation to perform during implementation, not tests already run during planning.
- No live Camunda instance or credentials are required for the local HTTP-fixture command tests.

## Targeted command validation

```sh
go test ./cmd -run 'Test(Delete|Cancel)ProcessInstance' -count=1
```

Extend the existing selector test files with empty-output cases whose names fit this pattern. Run both delete and cancel, normal and dry-run, against zero-match fixtures. Expected output and payloads are defined in [contracts/cli-output.md](contracts/cli-output.md); field semantics are in [data-model.md](data-model.md).

For each of the four command/execution combinations verify JSON, keys-only, human, and quiet output. Add quiet with JSON/keys-only, auto-confirm, automation with explicit JSON/keys-only, and supported no-wait variants. Require exactly one JSON document and EOF, or exactly zero keys-only bytes. Capture stderr separately and verify it has no empty-result summary in quiet mode.

Use existing `TestDeleteProcessInstanceBpmnSelectorVisiblePreservesSearchNoOp` and its cancel counterpart as two-request fixture foundations: definition validation followed by process-instance search. Use state/date-filter empty-search fixtures to prove the simpler one-request path. Reject unexpected request paths and assert zero mutation calls. Preserve tests that continue past an empty selected page when discovery has more work.

## Terminal, renderer, and help validation

```sh
go test ./cmd -run 'TestProcessInstance|TestConfirm|TestConfirmation' -count=1
```

Use matching names for new focused view and terminal regressions. Reuse `testx.NewCmdTerminalRunner` as in `cmd/cmd_confirmation_terminal_test.go`: real terminal stdin, no input exchanges, separate stdout/stderr, and the existing bounded timeout. Each zero-match command must finish successfully without asking for input. Add confirmation/mutation spies where the facade-stub tests already support them; terminal validation complements those guards.

Also preserve existing invalid-selector, discovery-failure, explicit-key, nonempty, paging-abort, dry-run summary, and no-wait tests. Reset package-level command globals after each case; do not parallelize tests that mutate them.

## Documentation and complete validation

Update README and the delete/cancel command `Long`/`Example` metadata with the empty-result behavior. Regenerate the reference rather than editing generated pages:

```sh
make docs-content
make test
git diff --check
```

`make test` runs `go test ./... -race -count=1`. Run `gofmt` on the Go files changed during implementation before these final checks. Inspect regenerated documentation for correct no-op examples and review unrelated output changes before committing. A failing or unavailable check must be reported and resolved before implementation completion; do not treat this planning guide as evidence of passing tests.

## Iteration 1 validation (2026-09-11)

- Confirmed branch, active feature selection, Go 1.26/toolchain 1.26.2, constitution, AGENTS guidance, and Ralph implementation rules with no conflict.
- Demonstrated the pre-fix machine-output defect with the new delete/cancel empty-selector tests.
- Passed `go test ./cmd -run 'Test(Delete|Cancel)ProcessInstanceEmptySelectorOutput|TestProcessInstance.*Empty|Test(Delete|Cancel)ProcessInstanceBpmnSelectorVisiblePreservesSearchNoOp' -count=1`.
- Passed `make test` (`go test ./... -race -count=1`).
- Passed `git diff --check` before coordinated persistence.

## Iteration 2 validation (2026-09-11)

- Demonstrated the pre-fix quiet defect in delete/cancel normal and dry-run human cases; the new matrix failed only because `found: 0` remained on stdout.
- Passed `go test ./cmd -run 'Test(Delete|Cancel)ProcessInstanceEmptySelectorOutput|TestProcessInstance.*Empty' -count=1` after adding human-only quiet suppression.
- Verified ordinary human output is exactly `found: 0\n`, quiet human output is absent from both streams, quiet JSON remains one successful envelope, and quiet keys-only remains byte-empty for both commands and execution types.
- Passed `make test` (`go test ./... -race -count=1`).
- Passed `git diff --check` before coordinated persistence.
