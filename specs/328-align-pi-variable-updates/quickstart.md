# Validation Guide: Align PI Variable Updates

This guide is for the later implementation phase. These runtime commands were not run during planning. Start from the repository root on `328-align-pi-variable-updates`, with the repository Go toolchain installed and dependencies available. Service and command tests use local HTTP fixtures; no live Camunda cluster is required. Actual-terminal tests require a platform supported by the existing `testx.NewCmdTerminalRunner` helper.

## 1. Targeted checks

After implementing the relevant unit, format touched Go files and run the corresponding checks:

```sh
go test ./internal/services/variable/... -count=1
go test ./internal/services/processinstance/... -run 'Variable' -count=1
go test ./c8volt/process -run 'Variable' -count=1
go test ./cmd -run 'Test(UpdatePI|UpdateProcessInstance|FormatProcessInstanceVariable)' -count=1
```

New service tests should retain `Variable` in their test names so these selections include them. Check test discovery if naming differs; a zero-test success is not evidence. Use the new variable traversal package tests as part of `./internal/services/variable/...`.

## 2. Fixture scenarios and expected observations

| Scenario | Fixture setup | Required observation |
| --- | --- | --- |
| Complete planning | Root variables across several pages, requested name on a later page | Classification matches complete root data, not first page |
| Sparse/capped pages | Short or locally empty intermediate page with continuation; capped total lower bound | Continue to later values; no premature no-op |
| Root isolation | Same name at root and nested scope | Only root values affect plan/verification; write has `local=true` at selected key |
| Mixed collection | Two instances already match; third has one addition, one change, one unchanged requested value | Exactly one mutation to third instance, containing only two planned values |
| Planning failure | Last instance or later variable page fails | Zero mutation requests for the entire collection |
| Frozen plan | Change caller's input/nested map after planning; supply a malformed plan separately | Caller mutation does not change prepared payload; invalid plan fails before all writes |
| Truncation | Matching visible prefix with `apiTruncated=true` | Never classified unchanged or confirmed from that observation |
| Default wait | One planning page; first post-write observation matches | Two variable searches total, one mutation; no extra confirmation pass |
| No-wait | Same plan with no-wait | Planning reads and selective mutation only, zero verification reads; status submitted |
| Partial failure | One writer accepts, another rejects; separately fail confirmation after acceptance | Accepted evidence retained, correct failure class, one result envelope |
| Fail-fast/cancel | Stop after failure with pending and in-flight roots | Actual completed results retained; unstarted roots skipped, no fabricated rejection |
| All unchanged | Valid explicit keys and complete matching values | No writes, no approval prompt, no verification, no accepted/pending claims |
| Missing input | No keys or empty stdin | Existing validation error, zero writes; not a successful empty selection |

Run adapter cases for 8.8, 8.9, and 8.10, asserting tenant/root filters and untruncated-value request settings on every page. Include raw-body truncation fields, exact/capped totals, cursor and offset progress, overflow, malformed metadata, later-page errors, and cancellation.

## 3. Output and terminal matrix

Extend existing `cmd/update_processinstance_test.go` and `cmd/update_processinstance_terminal_test.go` fixtures. Exercise normal execution and dry-run across human, JSON, keys-only, quiet, quiet+JSON, and quiet+keys, plus supported auto-confirm, automation, and no-wait combinations. Preserve invalid-combination validation.

Capture stdout and stderr separately. Decode exactly one JSON envelope and require EOF after it, assert exact established human text on its existing stream, and require zero-byte keys output for executed all-unchanged plans. Dry-run keys still list selected keys, including unchanged selections. Verify nonempty executed keys are pure accepted-key lines and partial errors do not produce a second envelope.

Use real terminal stdin while holding stdout/stderr in separate captures to prove no-op completion without input when approval would otherwise be eligible. Cover explicit and inherited stderr destinations. Retain nonempty approval, decline, default-answer/EOF, redirected stdout, repeated/stdin keys, explicit-key errors, and activity-output regressions. Assert request counts and payloads, not merely output text.

The contracts are defined in [contracts/cli.md](contracts/cli.md) and [contracts/service.md](contracts/service.md); outcome invariants are in [data-model.md](data-model.md).

## 4. Shared-consumer regression and docs

```sh
go test ./internal/services/usertask/... -count=1
go test ./cmd -run 'TestUpdateUserTask' -count=1
go test ./c8volt/... -count=1
make docs-content
git diff --check
```

UT behavior must remain unchanged. Existing get/enrichment and legacy writer/waiter consumers must remain compatible with complete reads. Update command help and README before generating docs, then inspect generated changes for this command and accidental unrelated churn.

After targeted checks pass, run the full race-enabled suite for the implementation's shared reader/waiter and cross-layer concurrency impact:

```sh
make test
```

This broad validation is justified by shared runtime changes, not by creating or committing these planning artifacts. Record actual results and any platform limitations; skipped terminal coverage must not be presented as passed. No live mutation is necessary to complete this validation matrix.
