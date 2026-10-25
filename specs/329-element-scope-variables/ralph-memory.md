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

## Decisions

- Iteration 1 completed foundational T001. Iteration 2 completes T002 only (US1 service/facade enrichment plus discovery tenant correction). T003 CLI exposure remains a separate independently validated unit.
- Version directories expose generated-client interfaces in `contract.go`, not the planned `api.go`. No generated-client interface additions were necessary. Existing consumer doubles use narrower root contracts; the widened variable API is implemented by all three concrete adapters.
- Use explicit-key `WithIgnoreTenant()` with the actual observed expected tenant in later element enrichment/planning/confirmation. Do not derive expected identity from configured discovery tenant.

## Gotchas

- Missing optional tenant evidence is unknown, including an empty observed tenant; compare only nonempty expected and observed values. Missing/blank owner or scope is malformed evidence, never empty success.
- Test HTTP responses must include request metadata: a bare `http.Response{StatusCode: 200}` fails existing HTTP normalization before decoding. Reuse existing adapter response helpers.
- AGENTS.md's active-plan marker is stale (311); the prerequisite script and branch select `specs/329-element-scope-variables`.
- The composed enrichment service accumulates enriched page items and replaces base aggregate Items; base element paging copies structs before the visitor. Forward original page metadata, counts, limits and visitor actions; do not derive them in the facade.
- Complete empty variable responses need both `totalItems` and `hasMoreTotalItems` in raw fake JSON. A missing total-semantic field correctly fails as malformed metadata.
- `domain.ErrForbidden` maps to public `ferrors.ErrLocalPrecondition`; there is no public `ErrForbidden` sentinel. New enrichment APIs return errors without logging or rendering; CLI diagnostics belong to T003.

## Reusable Commands

- `GOFLAGS= go test ./internal/services/variable/... -list 'Test(ScopeVariable|ProcessInstanceVariable|SearchProcessInstanceVariable|UpdateScopeVariables)'`
- `GOFLAGS= go test ./internal/services/variable/... -run 'Test(ScopeVariable|ProcessInstanceVariable|SearchProcessInstanceVariable|UpdateScopeVariables)' -count=1`
- Run `make test` with empty GOFLAGS and sanitized inherited C8VOLT/helper/make settings. The inspected target runs `go test ./... -race -count=1`; integration tests are build-tagged and excluded.
- `gofmt -l` on touched Go files and `git diff --check` validate formatting/whitespace.

- `GOFLAGS= go test ./internal/services/element/... ./c8volt/element ./c8volt -run 'Test(ElementScopeVariables|ElementSearchTenant|Enrich|Client_.*Element|Service_.*Element|New_.*Wires)' -count=1`
- `GOFLAGS= go test ./cmd -run 'Test(GetElementCommand|GetElementHelp|SearchElementsWith|CommandCapabilityForCommand_GetElement)' -count=1`
- `make docs-content` regenerates the element help and README-derived homepage; generated build metadata changes are expected. Run with GOFLAGS empty and inherited make/C8VOLT/helper overrides cleared.

## Do Not Repeat

- Do not fork the traversal loop or tighten root compatibility semantics when extending scoped reads.
- Do not run live Camunda/integration/smoke checks. Isolated tests do not establish real-engine consistency or authorization behavior.

## Current Handoff

- T002 is complete, with focused service/facade/adapter/command regressions, root-client cross-layer fake tests, docs generation, formatting/diff checks and `make test` (full race suite) passing. Next dependency-ready work is T003 (US1 `get element/ei --with-vars` CLI, renderers, mode/empty-result/diagnostic/real-terminal acceptance, metadata/docs); no blockers. Recheck tasks.md before selection. Five feature tasks remain.
- Use the new public enrichment methods: exact keys with IgnoreTenant; variable-only search uses paged enrichment, listener+variable search uses collected enrichment. Suppress variables before dispatch for effective keys-only/total, preserve conflicts and existing envelope shapes, and reuse existing value formatting. No variable CLI flag was introduced by T002; its only command change documents configured-tenant discovery.
