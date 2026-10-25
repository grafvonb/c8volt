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

## Decisions

- Iterations 1–3 completed T001–T003: shared scoped reading, service/facade enrichment and tenant correction, then the read-only CLI MVP. Update stories remain unexposed.
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

## Do Not Repeat

- Do not fork the traversal loop or tighten root compatibility semantics when extending scoped reads.
- Do not run live Camunda/integration/smoke checks. Isolated tests do not establish real-engine consistency or authorization behavior.

## Current Handoff

- T003 is complete and US1's read-only MVP is delivered. Next dependency-ready work is T004 (US2 frozen element planning/validation API and facade), with T001/T002 prerequisites complete and no blockers. Recheck tasks.md before selection; T004–T007 remain unchecked (four feature tasks).
- Targeted command/adapter/docs checks, actual-terminal and diagnostic matrices, generated documentation, formatting/diff review and `make test` (full race suite) passed. T003 also corrected empty element-page continuation in all three adapters after an execution-path test reproduced a false empty result; no later-story implementation started.
- Reuse the complete scoped reader and independent owner/scope identities for T004. Preserve existing PI/UT planning conventions, explicit-key IgnoreTenant, known observed tenant evidence, immutable collection-wide plans and fake-only validation. T007 can reuse the terminal/output testing patterns without rebuilding inspection behavior.
