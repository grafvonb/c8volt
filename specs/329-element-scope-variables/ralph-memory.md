# Ralph Memory

Feature: 329-element-scope-variables
Started: 2026-10-25T06:34:13Z

## Codebase Patterns

- `variable/search.collect` owns the single complete traversal algorithm. Root `ProcessInstanceVariables` and scoped `ScopeVariables` supply callbacks; raw counts, exact/lower-bound totals, cursor cycles, cursor-to-offset fallback, cancellation, and name/key sorting stay shared.
- `domain.ScopeVariableQuery` separates `ProcessInstanceKey` from `ScopeKey`, with optional `ExpectedTenantId`. `SearchScopeVariablesPage` validates owner/scope before requests; complete reads return an allocated empty slice only after authoritative completion and nil on failure.
- Adapters retain raw JSON decoding because generated models omit value/truncation fields. Scoped reads require name/value and nonblank owner/scope, allow missing optional variable key/tenant, ignore unrelated scopes, and reject selected-scope owner/known-tenant contradictions.
- Root methods remain owner=scope compatibility wrappers. Their existing required raw fields, filtering, tenant options and error wording are preserved. `IgnoreTenant` propagates to every page; expected tenant evidence does not replace the configured discovery filter.

- `element.NewEnrichment` composes base element selection, existing listener helpers, and complete `SearchScopeVariables` reads. `domain.ElementEnrichment`/public `element.Enrichment` select the two independent enrichments; the three public `WithEnrichment` operations map/delegate through `c8volt/element/enrichment.go`.
- Root client wiring reuses the existing element, job, and variable services. `element.NewWithEnrichment` accepts the composed service; legacy constructors retain ordinary/listener behavior and return converted local-precondition errors for unwired variable enrichment.
- Enriched exact GET automatically appends `WithIgnoreTenant()` to all dependent reads and binds known observed tenant evidence. With neither enrichment flag, it delegates ordinary behavior. Searches preserve discovery options and server-side tenant selection.
- Element adapters now set generated `ElementInstanceFilter.TenantId` for a nonblank configured tenant unless IgnoreTenant; collected/page/total paths inherit it. Native GET and traversal arithmetic are unchanged.

- `getElementVariableEnrichmentEnabled` resolves effective mode before facade dispatch. Exact keys pass IgnoreTenant; variable-only searches use paged enrichment, combined listener/variable searches retain collected selection. Unrequested enrichment uses the old methods. Existing conflicts remain validated first.
- `cmd/cmd_views_element.go` owns keyed, collected and page rendering plus the mode-aware final summary. Human rows/summary respect quiet; explicit JSON/keys/totals remain. Variable rows reuse `variableValueHumanLine` with limit 0 and sibling connectors beside listeners. Empty selection JSON remains `{"total":0,"items":[]}`.
- Element adapter overflow normalization must not treat an empty page as complete while `totalItems > from` or `hasMoreTotalItems` advertises continuation. Existing offset advancement by request size handles those sparse pages. Final empty fake pages must supply authoritative no-more metadata.
- Element read failures pass a bounded human message to `handleCommandError`; the configured logger emits one ERROR and JSON retains the original error class/detail exactly once. Inspection stops at the first failed scope: already streamed complete rows survive, but a failed scope never becomes successful empty enrichment. DEBUG exchange logging/redaction stays in the existing HTTP path.

- `element.NewVariableUpdates` owns collection-wide planning using a narrow exact element getter and the existing complete scoped reader. It appends IgnoreTenant itself, binds owner/scope plus known observed tenant, deduplicates keys in input order and discards partial plans on any lookup/read/identity failure. An empty object still reads each target; empty key selections and nil objects fail before remote work.
- `validateAndCopyElementVariableUpdatePlan` validates the whole frozen collection: key/order/scope/owner, all counts, disjoint complete classifications, mutation/verification equality, unsent state and canonical TenantContext derived from per-element evidence. Execution calls it before any workers. All classifications are allocated empty slices when appropriate; mutation maps contain only additions/changes, verification maps contain every requested value.
- `element.NewWithVariableUpdates` adds the planning dependency without changing earlier constructor signatures. Root composition wires it to the same element/variable services; the facade only snapshots inputs, delegates, maps all plan fields and converts errors. Custom marshal-only JSON values fall back to independent decoded snapshots when the concrete type cannot round-trip faithfully; ordinary maps/slices, typed collections, RawMessage and number semantics remain covered.

- `common.RecordScopeVariableSubmission` accumulates per-attempt `SubmissionStatus` and observed HTTP status outside retry return values. A fresh `httpc.BeginRequestRateAttempt` in each adapter callback distinguishes unsent local admission from dispatched uncertainty; its local error must be returned to the existing retry helper when Client.Timeout substitutes the error type. Pre-dispatch canceled contexts do not invoke the generated client.
- Exact-local acceptance evidence requires a documented empty 204 response with valid HTTP metadata and no call error. Conclusive rejection requires a 400 `application/problem+json` body with all five non-null, correctly typed ProblemDetail fields and matching status. Other statuses, malformed/proxy responses and response-plus-error remain unknown; unknown survives later rejection/unsent attempts, and accepted supersedes uncertainty. No retry-policy changes were needed.

- `element/update_execution.go` validates/copies the complete plan, initializes unchanged/skipped outcomes, and uses separate mutation/confirmation `pool.ExecuteSlice` phases with attempted markers. Fail-fast mutation failure prevents fresh confirmation work; accepted in-flight writes remain submitted/not_started with failure evidence. Other successful writes can still confirm after an independent non-fail-fast failure.
- `element/update_wait.go` owns complete local polling, validates observations with the planner's identity/duplicate checks, and compares every VerificationValue (including unchanged requested names). It reuses configured backoff timeout/delay/retry semantics and IgnoreTenant; it never writes. An interrupted started read is confirmation_failed; unscheduled read-back stays submitted/not_started.
- Public `ExecuteElementVariableUpdates` only maps/copies/delegates/converts and returns partial results with the error. `ElementVariableUpdateResults.Items` uses JSON `items`; each result retains frozen owner/scope and mutation values. Public `VariableSubmissionStatus` constants mirror the internal writer evidence. `OK()` accepts unchanged, confirmed, or deliberate no-wait skipped confirmation only, never not_started or any result with Error.

## Decisions

- Iterations 1–6 implement T001–T006: shared scoped reading, service/facade enrichment and tenant correction, read-only CLI MVP, frozen planning, exact-local writer evidence, and frozen execution/confirmation with public results. The update CLI remains T007.
- Version directories expose generated-client interfaces in `contract.go`, not the planned `api.go`. No generated-client interface additions were necessary. Existing consumer doubles use narrower root contracts; the widened variable API is implemented by all three concrete adapters.
- Use explicit-key `WithIgnoreTenant()` with the actual observed expected tenant in later element enrichment/planning/confirmation. Do not derive expected identity from configured discovery tenant.

## Gotchas

- Missing optional tenant evidence is unknown, including an empty observed tenant; compare only nonempty expected and observed values. Missing/blank owner or scope is malformed evidence, never empty success.
- Test HTTP responses must include request metadata: a bare `http.Response{StatusCode: 200}` fails existing HTTP normalization before decoding. Reuse existing adapter response helpers.
- AGENTS.md's active-plan marker is stale (311); the prerequisite script and branch select `specs/329-element-scope-variables`.
- The composed enrichment service accumulates enriched page items and replaces base aggregate Items; base element paging copies structs before the visitor. Forward original page metadata, counts, limits and visitor actions; do not derive them in the facade.
- Complete empty variable responses need both `totalItems` and `hasMoreTotalItems` in raw fake JSON. A missing total-semantic field correctly fails as malformed metadata.
- `domain.ErrForbidden` maps to public `ferrors.ErrLocalPrecondition`; there is no public `ErrForbidden` sentinel. New enrichment APIs return errors without logging or rendering; CLI diagnostics belong to T003.

- `runElementScopeTerminal` in `cmd/get_element_terminal_test.go` uses actual terminal stdin and a redirected result file, comparing command-specific/inherited stderr with a capture file. The plain page prompt ends `More matching elements remain.\nContinue? [y/N]: `; it does not append a second confirmation question.
- `newElementScopeFixture` records generated-adapter requests and bodies against a test-owned server. `truncateValues=false` is a query parameter, not a body field. Cursor-only empty variable pages need an advancing `endCursor`; every raw page still requires both total-semantic fields.

- Duplicate variable observations can learn previously missing optional key/tenant evidence; retain that evidence when merging so a third row cannot hide a conflict behind the first row's missing metadata. Missing optional evidence stays unknown.
- JSON representability is checked before recursive facade copying, so cyclic/unsupported payloads fail instead of recursing. Existing variable-update comparison preserves number spelling through json.Number; backend `1.0` and a requested marshaled `1` remain different under the established PI/UT convention.
- Facade cancellation remains `context.Canceled` through `FromDomain`; command normalization later assigns its local-precondition class. Unauthorized/forbidden already convert to `ferrors.ErrLocalPrecondition`.

- The scope writer's legacy `Accepted` field and generic successful-HTTP mapping remain compatible for PI/UT callers. Element execution derives `MutationAccepted` and its success decision from `SubmissionStatus == ScopeVariableSubmissionAccepted`, not that legacy boolean; an unexpected 200 can retain legacy Accepted=true while strict evidence stays unknown. Execution preserves an operation failure for nonaccepted strict evidence even if the legacy writer returns nil error.
- Retry cancellation can return a zero generated response while an earlier response was observed. Keep the accumulated status/code on every error return. The generated decoder can discard the entire response on decode failure; do not invent a status code from the error.
- `TestUpdateScopeVariablesPacingEvidence` uses an in-memory base RoundTripper below real shared pacing and the real generated client. Delaying local rejection until `req.Cancel` closes forces actual Client.Timeout type substitution; assertions verify both the substituted client error and restored local classification, with no network dispatch after admission failure.

- Snapshot `ctx.Err()` once when joining mutation cancellation and deciding whether confirmation may begin; separate reads permit a cancellation race with a nil aggregate error. Pool cancellation can leave zero-value output slots even for queued jobs; attempted markers, not slot indexes, distinguish started work.
- Existing `NewVariableUpdates` discovers the optional scope writer from the supplied variable reader, preserving its constructor signature and root wiring. Read-only custom compositions still plan but execution returns a precondition error. The root's generated adapters and shared HTTP limiter are reused without new clients.
- Execution Error/Message strings are machine evidence only. T007 must derive bounded human reasons from known error classes/statuses, never log these fields or the raw joined error. Render accepted keys from MutationAccepted even for failed/not_started confirmation; select failure envelopes from the returned error before considering NoWait.

## Reusable Commands

- `GOFLAGS= go test ./internal/services/variable/... -list 'Test(ScopeVariable|ProcessInstanceVariable|SearchProcessInstanceVariable|UpdateScopeVariables)'`
- `GOFLAGS= go test ./internal/services/variable/... -run 'Test(ScopeVariable|ProcessInstanceVariable|SearchProcessInstanceVariable|UpdateScopeVariables)' -count=1`
- Run `make test` with empty GOFLAGS and sanitized inherited C8VOLT/helper/make settings. The inspected target runs `go test ./... -race -count=1`; integration tests are build-tagged and excluded.
- `gofmt -l` on touched Go files and `git diff --check` validate formatting/whitespace.

- `GOFLAGS= go test ./internal/services/element/... ./c8volt/element ./c8volt -run 'Test(ElementScopeVariables|ElementSearchTenant|Enrich|Client_.*Element|Service_.*Element|New_.*Wires)' -count=1`
- `GOFLAGS= go test ./cmd -run 'Test(GetElementCommand|GetElementHelp|SearchElementsWith|CommandCapabilityForCommand_GetElement)' -count=1`
- `make docs-content` regenerates the element help and README-derived homepage; generated build metadata changes are expected. Run with GOFLAGS empty and inherited make/C8VOLT/helper overrides cleared.

- `GOFLAGS= go test ./cmd -run 'Test(ElementScopeVariablesGet|GetElement|SearchElements|ElementFlat|ElementListener|RuntimeElement|CommandCapabilityForCommand_GetElement)' -count=1 -timeout=90s`
- `GOFLAGS= go test ./internal/services/element/... -run 'Test(ElementScopeVariablesSparseSelection|ElementSearchTenant|Service_SearchElements)' -count=1 -timeout=60s`
- `GOFLAGS= go test ./docsgen -run 'TestGeneratedGetElementDocsDocumentLookupSearchAndOutput' -count=1`

- `GOFLAGS= go test ./internal/services/element ./c8volt/element ./c8volt -run '^TestElementVariableUpdatesPlan' -count=1`
- `GOFLAGS= go test ./internal/services/element/... ./internal/services/processinstance ./internal/services/usertask -run 'Test(ElementVariableUpdatesPlan|PlanProcessInstanceVariableUpdates|PlanUserTaskVariableUpdates)' -count=1`

- `GOFLAGS= go test ./internal/services/common ./internal/services/variable/... -run 'Test(ScopeVariableSubmission|UpdateScopeVariables)' -count=1`
- `GOFLAGS= go test ./internal/services/common ./internal/services/processinstance ./internal/services/usertask ./c8volt/task -run 'Test(.*ProcessInstanceVariable|.*UserTaskVariable|CompleteUserTask)' -count=1`

- `GOFLAGS= go test ./internal/services/element ./c8volt/element ./c8volt -run 'TestElementVariableUpdates' -count=1 -timeout=60s`
- `GOFLAGS= go test ./internal/services/processinstance ./internal/services/usertask/... ./c8volt/task -run 'Test(.*ProcessInstanceVariable|.*UserTaskVariable|CompleteUserTask)' -count=1`
- Root facade `TestElementVariableUpdatesExecuteWiring` covers exact local requests and sparse confirmation in v88/v89/v810; `TestElementVariableUpdatesPacing` uses the real burst-one limiter over an in-memory transport for planning, submission and read-back admission failures.

## Do Not Repeat

- Do not fork the traversal loop or tighten root compatibility semantics when extending scoped reads.
- Do not run live Camunda/integration/smoke checks. Isolated tests do not establish real-engine consistency or authorization behavior.

## Current Handoff

- T006 is complete. Next dependency-ready work is T007 (US4 update CLI, output/diagnostic/terminal acceptance and documentation), with T003/T004/T006 satisfied and no blockers. Recheck selection against tasks.md; only T007 remains unchecked.
- Use the implemented planning/execution facade APIs and frozen results without introducing command-side loops, polling or replanning. Results use mutation values, preserve per-key identity and strict acceptance, and return partial outcomes alongside the original converted error. T007 must use MutationAccepted for executed keys, Error/Message only for machine evidence, and a bounded configured logger diagnostic for human failures. Interrupted not_started confirmation is failed execution, even though its write was accepted.
- T006 targeted element service/facade/root tests, PI/UT/completion regressions, formatting/diff review, make docs-content and make test (full race suite) passed. Tests use mocks, in-memory transports with the actual limiter, test-owned servers and existing isolated subprocess/terminal fixtures. No live checks or integration build tags ran; real-engine behavior remains unverified. The update CLI is still unexposed and belongs entirely to T007.
