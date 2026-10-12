# Research: Align Process Instance Variable Updates

## Scope and evidence

Research used the local implementation and issue #328 already read in this chat. No runtime tests or Camunda mutations were executed. Two research agents inspected service/facade mechanics and CLI/output compatibility independently. No new technology or dependency is needed.

## 1. Put preparation and execution in a composed PI service

**Decision:** Add a PI-specific `VariableUpdateAPI` and composed service in `internal/services/processinstance/update.go`, following `internal/services/usertask/update.go`. Expose separate plan and execute operations; keep the command responsible for operator approval and rendering.

**Evidence:** `cmd/update_processinstance_variables.go` currently owns discovery and comparison. `cmd/update_processinstance.go` creates a preview but passes the original keys and full variables map to bulk execution. `internal/services/usertask/update.go` and `update_validation.go` already demonstrate frozen plans, deep copies, validation, bounded execution, and partial outcomes.

**Rationale:** The same prepared state must survive approval and supply both writes and verification. One root per PI requires much less machinery than UT inherited/shared scopes.

**Alternatives considered:** Filtering payloads in the command would retain backend planning above its owner layer. Reusing the entire UT workflow would introduce task-specific scope resolution. A general mutation framework is unnecessary and out of scope.

## 2. Complete variable reads in the variable service

**Decision:** Introduce version-neutral root-variable page request/result types and a single-page operation in the variable service. Use a narrow `internal/services/variable/search` traversal package, following the repository's service-helper pattern, so version adapters can delegate without importing their factory-owning parent package. Keep `SearchProcessInstanceVariables` as the compatible complete-list operation.

**Evidence:** `internal/services/variable/{v88,v89,v810}/variables.go` reads only offset 0 with limit 1000. The adapters already constrain process-instance key and scope key, apply tenant configuration, request untruncated values, and decode raw responses. UT's `variables.go` supplies examples of capped totals, sparse pages, overflow checks, and metadata validation.

**Rationale:** Planning, verification, existing enrichment, and variable reads need the same complete root scope. Normalize generated page metadata inside each version adapter, then traverse once below the facade. Prefer advancing cursors where available; otherwise advance by raw backend item count, never locally retained count. Exact totals can establish completion; capped totals are lower bounds. A short page is not sufficient completion evidence while continuation exists. Reject contradictory metadata or non-progressing continuation rather than returning partial success.

**Alternatives considered:** Repeating a loop in every caller risks inconsistent completion. Assuming 1000 variables is enough preserves the bug. Extending command paging is a layering violation. Importing the parent variable package from its version adapters would create an import cycle.

## 3. Preserve version-specific decoding and tenant scope

**Decision:** Retain each adapter's key-filter construction and raw-body value decoding; normalize metadata without losing whether required fields were present. Preserve `isTruncated` and legacy `truncated` interpretation.

**Evidence:** The three `variable/*/convert.go` files decode raw bodies because generated response models omit value/truncation data. v88 uses common key-filter helpers, while v89/v810 use version-specific union helpers.

**Rationale:** Generated defaults must not turn missing metadata into authoritative zero counts. Filtering and tenant behavior are compatibility requirements on every page, not just the first.

**Alternatives considered:** Editing generated clients broadens scope unnecessarily. Sharing generated filter types across versions hides differences. Treating malformed metadata as completion risks incorrect no-ops.

## 4. Submit through the existing exact-scope writer and wait once

**Decision:** Execute each nonempty prepared root payload with `UpdateScopeVariables`, which already sets `local=true`, retries mutations, and returns acceptance without reading for confirmation. The composed PI service then invokes the existing variable waiter once per accepted changed instance, unless no-wait is requested. Confirmation includes every requested value for that instance, including values classified unchanged during preparation.

**Evidence:** Legacy variable adapter `UpdateProcessInstanceVariables` waits internally, then `internal/services/processinstance/variables.go` performs another `ConfirmProcessInstanceVariables` read. `internal/services/variable/waiter/waiter.go` already handles configured backoff and exact root matching, but its equality check lacks an `APITruncated` guard.

**Rationale:** The new route removes double verification and preserves accepted evidence when later verification fails. Harden the waiter against truncated equality and make its reader complete. Entirely unchanged instances are skipped without verification reads. Reuse JSON-normalization behavior from UT for planning, with comparison fixtures proving parity; extract only a narrowly scoped shared comparator if needed, without changing UT semantics.

**Alternatives considered:** Calling the legacy writer and adding another wait preserves duplication. Bypassing all verification violates the default operational contract. Verifying only changed names can miss drift in requested values that were initially unchanged.

## 5. Keep public result fields and make statuses truthful

**Decision:** Preserve existing fields and existing status strings `submitted`, `confirmed`, `mutation_failed`, and `confirmation_failed`. Add explicit `unchanged` and `skipped` status values for prepared workflow results. Accepted no-wait work continues to serialize as `submitted`, not a new `accepted` value. Unchanged is a successful no-op with `mutationAccepted=false`; skipped is unsuccessful unexecuted work. Adjust `OK()`/totals to distinguish successful no-op from accepted mutation, and never use `OK()` alone to choose emitted mutation keys.

**Evidence:** `c8volt/process/model.go` and `internal/domain/processinstance_variable.go` have only four statuses; `OK()` currently requires mutation acceptance. UT has richer outcome handling and failure renderers that retain partial results.

**Rationale:** Reusing `mutation_failed` for work never attempted obscures outcomes. This is an intentional additive status correction within the existing field schema. Consumers exhaustively matching statuses need the compatibility note in help/docs/release-facing documentation. Entirely unchanged CLI runs retain the existing preview payload instead of inventing report entries.

**Alternatives considered:** New JSON fields or a new envelope break the stated scope. Labeling unchanged work submitted is false. Dropping skipped or successful entries loses partial-result evidence.

## 6. Preserve actual CLI grammar and mode distinctions

**Decision:** Missing keys and empty stdin remain validation errors; there is no new selector/discovery mode. An empty internal plan is safe, but the CLI does not manufacture successful empty selection from invalid input. Dry-run keys-only retains selected keys, even when unchanged; executed all-unchanged output remains zero bytes. Retain JSON precedence, JSON confirmation validation, and the JSON/verbose incompatibility.

**Evidence:** `cmd/update_processinstance.go`, `cmd/cmd_views_processinstance_update.go`, `cmd/cmd_views_rendermode.go`, and `TestUpdateProcessInstanceCommand_MissingTargetsFailBeforeMutation`. Existing human summaries use the logger/stderr; JSON and keys use stdout.

**Rationale:** The spec's successful-empty language is conditional on authoritative valid completion. It does not authorize syntax or output changes unrelated to #328. Reserve stdout for results without moving existing human logger output to a different stream.

**Alternatives considered:** Treating missing targets as no-op weakens validation. Removing unchanged selected keys from dry-run output changes an established preview contract.

## 7. Validate execution paths and shared consumers

**Decision:** Extend service, adapter, facade, command, and actual-terminal tests near their owners. Update successful terminal fixtures from three variable reads to two for the one-page immediately-visible case: one planning read and one verification read. Multi-page reads and repeated polling legitimately require more requests.

**Evidence:** `cmd/update_processinstance_terminal_test.go` uses `testx.NewCmdTerminalRunner`; UT bulk/error/output/terminal tests demonstrate partial results and truncated verification.

**Rationale:** Pure view tests cannot establish absence of writes, prompts, or duplicate verification. Shared complete-list/waiter changes also affect legacy callers, so the eventual implementation warrants targeted consumer coverage followed by the race-enabled full suite.

**Alternatives considered:** Mocking terminal eligibility or checking only rendered text misses the regressions. Runtime tests during this documentation-only planning step provide no implementation evidence.

All research questions are resolved. Remaining coding details are implementation choices within these contracts, not open scope decisions.
