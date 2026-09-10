# Validation Guide: Terminal Cancellation Confirmation

## Prerequisites

Use the repository root, Go 1.26 with toolchain go1.26.2 available, and the existing module dependencies. Regression tests use controlled clients and require no live Camunda credentials. This guide describes validation after implementation; the planning phase has not added or run the proposed regressions.

## Targeted checks

```sh
go test ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 ./internal/services/processinstance/v810 -run 'TestService_(CancelProcessInstance|DeleteProcessInstance)' -count=1
go test ./internal/services/processinstance/waiter -run 'TestWaitForProcessInstance' -count=1
go test ./internal/services/processinstance -run 'Test.*Cancel' -count=1
go test ./internal/services/processdefinition -run 'Test(CleanupProcessDefinitionDeletePlanForceScope|DeleteProcessDefinitions)' -count=1
go test ./cmd -run 'Test(Cancel|Expect)' -count=1
```

Place new cases under these existing test-name prefixes, or update these commands when generating tasks. A passing command with no matching tests is not evidence. Use `go test <package> -list '<pattern>'` to confirm coverage if naming changes.

## Scenarios and expected outcomes

Use the [acceptance matrix](contracts/cancellation-confirmation.md) and [state table](data-model.md). In every version, verify a completed descendant, active-to-completed transition, disappearance after family keys are known, each terminal-root no-op, and active/unknown timeout controls. Assert cancellation request counts and `Ok` values for no-ops; preserve existing status text and HTTP code.

Exercise forced-delete recovery so a second narrow cancellation wait cannot reintroduce the timeout. Preserve existing opt-out guards, including the forced-delete recovery wait. A terminal cancellation outcome must not bypass final history absence verification.

For the focused process-definition regression, configure a real 8.8 process-instance service with controlled backend responses: active root, completed descendant, successful root cancellation, drained active statistics, successful history deletion, and the existing definition deletion/verification result. Enter through existing process-definition deletion orchestration so the test proves the entire path. Confirm that replacing the four-state cancellation set with the old two-state set makes the regression fail. Never substitute a canned successful cancellation callback for this proof.

Check explicit cancelled-state expectations against all four terminal states; completed/absent must fail to match within the configured bounded wait. Confirm normal read not-found errors and unrelated errors remain errors. Run existing command regressions for JSON/human output, prompts, dry-run, flags, and activity.

## Documentation and final validation

Clarify README and cancellation command long help, then run:

```sh
make docs-content
git diff --check
make test
```

Run `gofmt -w` on each touched Go file before these checks. Inspect generated documentation changes for only the intended clarification and any known generator effects. Do not hand-edit generated CLI references. The full test target runs `go test ./... -race -count=1` and must pass before implementation commit/merge. Record any unavailable validation explicitly.
