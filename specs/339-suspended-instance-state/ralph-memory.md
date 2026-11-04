# Ralph Memory

Feature: 339-suspended-instance-state
Started: 2026-10-17T11:26:30Z

## Codebase Patterns

- `integration/cli/suspended_fixture_test.go` supplies owned held fixtures, exact pre-mutation 8.10.4 gating, generated suspend/resume setup and independent observations. Cleanup resumes tracked keys, deletes the owned definition, confirms instance absence and writes separate cleanup evidence. The focused `TestGetFamilySuspendedState` belongs to the existing get slice.

- `SuspendedDate` is an optional string in domain/public instances, mapped through v810 `formatTimePtr` and both facade directions. Fresh conversion clears absent metadata; nil/zero dates omit JSON without changing state. Existing nested incident/variable/element payloads inherit the field; human renderers need no change.
- `c8volt/process/state.go` owns canonical parsing: trim whitespace, accept case variants and `cancelled`, and reject UNKNOWN/unrecognized inputs with the UNKNOWN fallback. `validStates` also supplies CLI invalid-value feedback.
- `ValidateProcessInstanceStates` is mandatory on shared/versioned process-instance APIs. Shared page/total traversal validates before direct-incident discovery; adapters validate page search and single/bulk waits before requests/workers. v88/v89 reject trimmed case-insensitive suspension; v810 permits it. Generic waiters stay capability-neutral.
- Lookup and single-state waits parse observed states; instance, combined-expectation and bulk-state report conversions preserve raw states. Extending the parser fixes suspended lookup reports without changing converters or facade orchestration.
- `cmd/cmd_views_processinstance.go` already renders the state in its existing column and JSON payload. No renderer or schema change is needed for recognition. Empty quiet summaries need guards in both incremental search and the collected list view; existing nonempty quiet output is intentionally retained (see `TestGetProcessInstanceSearch_AllTenantsProtectedOutputModesStayClean`).
- `cmd/processinstance_state_completion.go` supplies bootstrap-free candidates from public constants: search/analysis offers ALL, expectations offer ABSENT. Suspension carries an 8.10 description; typed prefix case is preserved. Analysis still rejects ABSENT.

## Decisions

- T001 adds domain/public `StateSuspended` and public parsing only. Preserve terminal sets: domain ABSENT is terminal, public ABSENT is not; suspension is nonterminal in both.
- Creation visibility includes SUSPENDED through the cloned domain observable-state list. All adapters keep using the generic waiter directly; explicit-request capability guards remain outside it, preserving 8.8/8.9 creation defaults.
- Update get/expect state flag descriptions with the 8.10 limitation because the parser expands exposed valid-value feedback. Regenerate CLI docs and the README-derived homepage with `make docs-content`.

## Gotchas

- CLI all-state selection uses the existing lowercase `all` sentinel. Uppercase `ALL` is forwarded to the backend and rejected; preserve this existing behavior. The first T006 run exposed this test mistake; corrected live rerun passed and both runs cleaned up.

- Direct service-factory HTTP tests must include `/v2` in `config.APIs.Camunda.BaseURL`; CLI bootstrap supplies this path separately. Older adapters intentionally leave `SuspendedDate` empty, even if a response contains the unfamiliar field.
- Creation help is asserted by `docsgen/TestGeneratedRunProcessInstanceDocsDocumentPipeline` as well as command help tests; keep its state list aligned.

- `StartConfirmedAt` belongs to internal creation results; public ProcessInstance/JSON does not expose it. Keep confirmation timestamp checks at the adapter layer. CLI no-wait emits `accepted`, omits unobserved state and makes no lookup; confirmed creation emits `succeeded` without implying execution completion.

- Do not normalize every report uniformly: existing unknown/raw and cancellation-alias behavior differs between parsed state reports and direct casts. `TestClient_ProcessInstanceStateReports` captures those contracts.
- Interface updates also affect `c8volt/resource/client_test.go` in addition to process facade and dry-run stubs. Embedded nil API stubs may compile while panicking if a newly called method is not implemented.
- The existing domain unsupported sentinel text is “unsupported media type”; preserve the sentinel and wrapping so public `ferrors` still classifies these as unsupported capability.
- Public `StateReport` has no `Ok` field and preserves UNKNOWN on failed waits. Combined expectation reports expose `Ok`; do not change either schema.
- State help text is asserted in both `cmd/expect_test.go` and `cmd/command_contract_test.go`; update both when the vocabulary changes.
- The generated v810 client/provenance stays at 8.10.3; live acceptance later requires observed server 8.10.4.
- v810 serializes the state equality filter as a plain string (`"SUSPENDED"`), not a `$eq` object. ALL may omit the entire filter. Empty get JSON omits both `total` and `items`; require one envelope followed by EOF. Quote `"8.10"` in raw YAML test fixtures to avoid YAML numeric normalization to 8.1.
- Slow-analysis state help is also asserted in `TestCommandContractOpsAnalyseSlowProcessInstances`; keep it aligned when changing vocabulary.

## Reusable Commands

- Final integrated `make test` passed all packages with race detection in T006. See `validation.md` for commands, evidence, source revision/dirty status and exact client provenance.
- `go test -tags=integration ./integration/cli -list '^TestGetFamilySuspendedState$'` proves discovery. `C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10 go test -tags=integration ./integration/cli -run '^TestGetFamilySuspendedState$' -count=1 -v -timeout=20m` passed against observed 8.10.4. Use a distinct `C8VOLT_IT_WORKDIR` per run. The release variable takes minor `8.10`, not patch `8.10.4`.
- `go test ./internal/domain -run 'State|ObservableProcessInstanceCreation' -count=1`
- `go test ./internal/services/processinstance/... ./c8volt/process -run 'State|Expectation|Search|Suspended|CreateProcessInstance' -count=1`
- `go test ./cmd -run 'GetPI|GetProcessInstance|Expect|Completion|Suspended|RunProcessInstance|OpsSlowProcess' -count=1`
- `make docs-content` regenerates CLI references and `docs/index.md` (including its build provenance line).
- `TestClient_SuspendedDateSnapshots` reuses real versioned adapters/facades for present-to-absent reads/searches; `TestSuspendedDateSnapshots` checks separate streams, one envelope plus EOF, unchanged rows/keys and one request per snapshot.
- `TestSuspendedCommandHelper` accepts JSON arguments through `suspendedCommandEnv`; reuse it for isolated exit/stream tests. `decodeSuspendedEnvelope` requires EOF. The real-terminal matrix covers empty/nonempty six output modes with ordinary, auto-confirm and automation policies.
- Controlled zero-delay creation fixtures cover delayed absence-to-suspension; an already-expired context covers deadline failure without sleeps.

## Do Not Repeat

- Do not add facade/backend orchestration to fix state recognition; the public vocabulary is the missing piece.
- Do not test quiet suppression through a view helper alone. Use actual command execution for T003's quiet/empty/automation contracts, with real terminal stdin for applicable interactive paths.

## Current Handoff

- Feature complete; no handoff required.
