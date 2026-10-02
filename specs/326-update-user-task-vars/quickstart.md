# Quickstart Validation: Update User-Task Variables

These commands validate the implemented command with fixture-backed tests and optional live checks. The automated suites require no external Camunda credentials; live mutation remains optional.

## Prerequisites

- Repository Go 1.26 / toolchain 1.26.2 and existing build dependencies.
- For fixture tests: no external Camunda credentials required.
- For optional live validation: an authorized disposable Camunda 8.8, 8.9, or 8.10 environment, existing c8volt configuration, and active native user tasks. Use a process with a root variable, an intermediate-scope variable, and a task-local variable; include two tasks inheriting one root variable.
- Use dedicated fixture variable names and record initial values. Parent-scope updates intentionally affect other tasks inheriting those values. Do not use production data for mutation validation.

## 1. Focused automated validation

Begin with the affected packages and test patterns that select the implemented coverage:

```bash
go test ./internal/services/variable/... -run 'TestUpdateScopeVariables' -count=1
go test ./internal/services/usertask/... -run 'Test(PlanUserTaskVariableUpdates|ExecuteUserTaskVariableUpdates)' -count=1
go test ./c8volt/task -run 'Test(VariableUpdateFacade|LegacyTaskConstructorRejectsVariableUpdates)' -count=1
go test ./cmd -run 'Test(UpdateUserTask|CommandCapabilityForCommand_UpdateUserTask)' -count=1
```

Confirm patterns select real tests using `go test <package> -list '<pattern>'`; a run with no selected tests is not acceptance evidence.

Cover the matrices in [CLI contract](contracts/cli.md) and [service contract](contracts/facade-service.md). Assert request bodies, target keys, `local=true`, ordering, deduplication, exact counts, full variable pagination, no hidden reads during rendering, and absence of mutation/prompt calls on dry-run/no-op.

Use `testx.NewCmdTerminalRunner` for confirmation cases; pipe-only stdin or mocked terminal detection is insufficient. Capture stdout and stderr separately. Decode exactly one JSON envelope and then require EOF. Assert zero-byte no-op keys output and exact human wording.

Run existing affected regression suites:

```bash
go test ./cmd -run 'Test.*(UpdateProcessInstance|GetUserTask)' -count=1
go test ./internal/services/variable/... -run 'Test.*UpdateProcessInstanceVariables' -count=1
go test ./c8volt/task -count=1
```

Add targeted race coverage when exercising shared target outcomes, pools, and concurrent HTTP observations. Do not repeat passing suites without relevant changes or failures.

## 2. Build and inspect the command

```bash
go build -o /tmp/c8volt-326 .
/tmp/c8volt-326 update ut --help
/tmp/c8volt-326 update user-task --help
```

Confirm aliases, existing PI controls, version guidance, scope semantics, and absence of new task-search or scope flags. Compare examples and payload errors with `update pi`.

## 3. Preview local, inherited and missing values

Set `TASK_A` and `TASK_B` to existing authorized fixture keys. Configure the target environment using the established c8volt configuration workflow.

```bash
/tmp/c8volt-326 get ut --key "$TASK_A" --with-vars --verbose
/tmp/c8volt-326 update ut --key "$TASK_A" --vars '{"rootReview":"approved","localReview":"approved","newReview":true}' --dry-run
/tmp/c8volt-326 --json update ut --key "$TASK_A" --vars '{"rootReview":"approved","localReview":"approved","newReview":true}' --dry-run
```

Expected: inherited changes retain original scope, local changes retain task scope, newReview targets task scope, preview indicates no mutation, and unrelated values stay untouched. Verify fixture names actually exist at the intended scopes before executing writes.

## 4. Confirm and inspect one task

```bash
/tmp/c8volt-326 update ut --key "$TASK_A" --vars '{"rootReview":"approved","localReview":"approved","newReview":true}' --auto-confirm
/tmp/c8volt-326 get ut --key "$TASK_A" --with-vars --verbose
```

Expected: confirmed status only after matching values at the planned scopes; no task-local shadow for rootReview. Inspect the shared root variable through TASK_B as well.

Repeat the same payload using quiet+JSON and keys-only:

```bash
/tmp/c8volt-326 --quiet --json update ut --key "$TASK_A" --vars '{"rootReview":"approved","localReview":"approved","newReview":true}' --auto-confirm
/tmp/c8volt-326 --keys-only update ut --key "$TASK_A" --vars '{"rootReview":"approved","localReview":"approved","newReview":true}' --auto-confirm
```

Expected: one succeeded no-op envelope with `mutationSubmitted=false`; keys-only emits zero bytes. Request-count assertions belong in automated fixtures, not inferred from these live commands.

## 5. Shared inherited write and no-wait

```bash
printf '%s\n' "$TASK_A" "$TASK_B" "$TASK_A" | /tmp/c8volt-326 update ut - --vars '{"rootReview":"recheck"}' --dry-run
printf '%s\n' "$TASK_A" "$TASK_B" | /tmp/c8volt-326 --automation --json update ut - --vars '{"rootReview":"recheck"}' --no-wait
```

Expected: both tasks refer to the shared parent target, one logical scope/name write, stable unique task ordering, no prompt, and accepted rather than confirmed outcome. Automated tests prove exactly one logical submission and zero confirmation reads.

## 6. Failure and version checks

Use controlled HTTP fixtures for transport/HTTP errors, malformed pages, truncated values, authorization loss, task disappearance, partial acceptance, shared scope failure, fail-fast, and cancellation. Verify unsupported v8.7 produces no mutation. Exercise all three supported adapters; a live test on one version does not prove all version paths.

For normal-wait execution, test mismatched scope despite matching value, incomplete observed values, and confirmation timeout. Each must preserve submission facts and avoid confirmed success.

## 7. Integrated completion

After executable changes and source metadata are complete:

```bash
make docs-content
git diff --check
make test
```

Run targeted `gofmt` on touched Go files before these checks. Review generated CLI pages and README against the final contract. The full race suite is required by issue #326 and justified by shared interfaces, facade wiring, and concurrent execution changes. If the environment prevents a check, record the exact gap; do not claim it passed. No live mutation is required merely to validate planning documents.
