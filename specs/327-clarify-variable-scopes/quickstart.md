# Quickstart: Validate Variable Destination Previews

Run from the repository root after implementation. This is a validation guide, not a record of tests executed during planning. See [preview contract](contracts/preview-output.md) for exact output rules and [data model](data-model.md) for metadata invariants.

## Prerequisites

- Repository Go toolchain from `go.mod` (Go 1.26, toolchain 1.26.2), dependencies available, and loopback networking for HTTP fixtures.
- A platform supported by `testx.NewCmdTerminalRunner` for real terminal tests; report any platform skips rather than treating them as proof.
- No live Camunda instance is required for automated tests. Optional manual scenarios require a configured disposable environment and selected native task/process keys.

## Focused automated validation

After changing task metadata and conversions:

```sh
go test ./internal/services/usertask/... -run 'TestPlanUserTaskVariableUpdates|TestExecuteUserTaskVariableUpdates' -count=1
go test ./c8volt/task -run 'TestVariableUpdateFacade|TestLegacyTaskConstructor' -count=1
```

Extend the existing service plan test to assert owner identity alongside sparse-page completeness, root/local/intermediate destinations, target deduplication, and unchanged request counts. Extend facade tests to verify both conversion directions and identical marshaled JSON with the new metadata present or absent. Keep missing-owner plans executable under existing rules.

After changing views and command fixtures:

```sh
go test ./cmd -run 'Test(UpdatePI|UpdateProcessInstance|FormatProcessInstanceVariableUpdatePlan|UserTaskVariableUpdate|UpdateUserTask)' -count=1
```

Use the existing test-name families for new regressions so these filters select them. Verify the run actually includes the new cases. Cover the following through command execution as well as focused view assertions:

| Scenario | Required evidence |
| --- | --- |
| UT single preview and dry-run | Root/intermediate/local changes and local addition all have correct annotations |
| PI single preview and dry-run | Every addition/change names the selected process root |
| UT/PI bulk and verbose | Compact summary unchanged; verbose details complete; UT inherited scope order and raw-key deduplication unchanged |
| Metadata and requests | Task read, variable-page read, mutation, and total request counts unchanged; fail on unexpected endpoints |
| Machine/quiet modes | Exact existing payload shape/content and key results; one JSON envelope followed by EOF; separate stdout/stderr; quiet preserves explicit machine output |
| No-op, empty payload, dry-run | No mutation or confirmation; existing mode-specific no-work payload/wording preserved, with zero-byte keys when no keys are emitted |
| Accept, decline, default answer, EOF | Real terminal stdin; existing prompt wording/destination and final status; no writes on abort |
| Automation, auto-confirm, no-wait | Existing behavior and accepted/confirmed distinction; no extra preview output introduced |
| Errors, partial results, supported versions | Existing regression coverage still passes without new scope lookups or altered errors |

`TestUpdateUserTaskOutputModes` and `TestUpdateUserTaskTerminal` provide reusable subprocess and terminal setup. Add analogous PI confirmation assertions using the same runner. Normal confirmation must be exercised without auto-confirm or automation because those skip the preview. Assert configured and inherited stderr routing with stdout redirected.

If shared concurrent fixture capture changes, run the affected command tests with `-race`:

```sh
go test -race ./cmd -run 'Test(UpdatePI|UpdateProcessInstance|UserTaskVariableUpdate|UpdateUserTask)' -count=1
```

## Optional manual preview walkthrough

Set `C8VOLT_SCOPE_CONFIG`, `C8VOLT_SCOPE_TASK_A`, `C8VOLT_SCOPE_TASK_B`, and `C8VOLT_SCOPE_PI` to your test configuration path and real keys. Prepare task A with an inherited process-root `customerTier="silver"`, an inherited intermediate-element `approvalLimit=100`, local `reviewed=false`, and no effective `incident` variable. Use task B sharing an inherited destination to inspect bulk deduplication. Populate the PI fixture with `customerTier="silver"` and no root `incident`.

```sh
go build -o /tmp/c8volt-scope-preview .
/tmp/c8volt-scope-preview --config "$C8VOLT_SCOPE_CONFIG" update ut --key "$C8VOLT_SCOPE_TASK_A" --vars '{"customerTier":"gold","approvalLimit":200,"reviewed":true,"incident":"buba"}' --dry-run
/tmp/c8volt-scope-preview --config "$C8VOLT_SCOPE_CONFIG" update ut --key "$C8VOLT_SCOPE_TASK_A" --key "$C8VOLT_SCOPE_TASK_B" --vars '{"customerTier":"gold","incident":"buba"}' --dry-run
/tmp/c8volt-scope-preview --config "$C8VOLT_SCOPE_CONFIG" --verbose update ut --key "$C8VOLT_SCOPE_TASK_A" --key "$C8VOLT_SCOPE_TASK_B" --vars '{"customerTier":"gold","incident":"buba"}' --dry-run
/tmp/c8volt-scope-preview --config "$C8VOLT_SCOPE_CONFIG" update pi --key "$C8VOLT_SCOPE_PI" --vars '{"customerTier":"gold","incident":"buba"}' --dry-run
/tmp/c8volt-scope-preview --config "$C8VOLT_SCOPE_CONFIG" --quiet --json update ut --key "$C8VOLT_SCOPE_TASK_A" --vars '{"customerTier":"gold"}' --dry-run
```

Expect the destination identities from your fixture in the notation defined by the contract, unchanged aggregate wording, no prompt, and no mutation. JSON must contain no new process-identity metadata field in task plans. For normal confirmation, repeat the corresponding single-resource invocation without `--dry-run` in a real terminal and answer `no`; compare scope annotations with dry-run and verify no mutation. Use automated terminal tests for accepted execution and exact stream assertions.

## Documentation and final checks

With each delivered command slice, update its help/examples and README explanation, then regenerate and review documentation:

```sh
make docs-content
git diff --check
git diff -- README.md docs/index.md docs/cli/c8volt_update_process-instance.md docs/cli/c8volt_update_user-task.md
```

Run `gofmt` on touched Go files before testing. Do not hand-edit generated CLI markdown. Inspect unrelated generated changes before retaining them. Broaden to `make test` only when the actual implementation affects shared runtime behavior or unresolved failures justify it; otherwise retain the focused evidence. Record actual results and any skips or limitations in the implementation progress report.
