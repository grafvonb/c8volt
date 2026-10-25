# Ralph Memory

Feature: 329-element-scope-variables
Started: 2026-10-25T06:34:13Z

## Codebase Patterns

- `variable/search.collect` owns the single complete traversal algorithm. Root `ProcessInstanceVariables` and scoped `ScopeVariables` supply callbacks; raw counts, exact/lower-bound totals, cursor cycles, cursor-to-offset fallback, cancellation, and name/key sorting stay shared.
- `domain.ScopeVariableQuery` separates `ProcessInstanceKey` from `ScopeKey`, with optional `ExpectedTenantId`. `SearchScopeVariablesPage` validates owner/scope before requests; complete reads return an allocated empty slice only after authoritative completion and nil on failure.
- Adapters retain raw JSON decoding because generated models omit value/truncation fields. Scoped reads require name/value and nonblank owner/scope, allow missing optional variable key/tenant, ignore unrelated scopes, and reject selected-scope owner/known-tenant contradictions.
- Root methods remain owner=scope compatibility wrappers. Their existing required raw fields, filtering, tenant options and error wording are preserved. `IgnoreTenant` propagates to every page; expected tenant evidence does not replace the configured discovery filter.

## Decisions

- T001 is the sole iteration-1 work unit (Phase 2 foundational). It adds only internal reading APIs; no command/facade behavior, generated clients or CLI documentation changed.
- Version directories expose generated-client interfaces in `contract.go`, not the planned `api.go`. No generated-client interface additions were necessary. Existing consumer doubles use narrower root contracts; the widened variable API is implemented by all three concrete adapters.
- Use explicit-key `WithIgnoreTenant()` with the actual observed expected tenant in later element enrichment/planning/confirmation. Do not derive expected identity from configured discovery tenant.

## Gotchas

- Missing optional tenant evidence is unknown, including an empty observed tenant; compare only nonempty expected and observed values. Missing/blank owner or scope is malformed evidence, never empty success.
- Test HTTP responses must include request metadata: a bare `http.Response{StatusCode: 200}` fails existing HTTP normalization before decoding. Reuse existing adapter response helpers.
- AGENTS.md's active-plan marker is stale (311); the prerequisite script and branch select `specs/329-element-scope-variables`.
- Future T002 must accumulate enriched visitor items itself: base element paging copies structs before the visitor. Include the configured-tenant element discovery correction in that task.

## Reusable Commands

- `GOFLAGS= go test ./internal/services/variable/... -list 'Test(ScopeVariable|ProcessInstanceVariable|SearchProcessInstanceVariable|UpdateScopeVariables)'`
- `GOFLAGS= go test ./internal/services/variable/... -run 'Test(ScopeVariable|ProcessInstanceVariable|SearchProcessInstanceVariable|UpdateScopeVariables)' -count=1`
- Run `make test` with empty GOFLAGS and sanitized inherited C8VOLT/helper/make settings. The inspected target runs `go test ./... -race -count=1`; integration tests are build-tagged and excluded.
- `gofmt -l` on touched Go files and `git diff --check` validate formatting/whitespace.

## Do Not Repeat

- Do not fork the traversal loop or tighten root compatibility semantics when extending scoped reads.
- Do not run live Camunda/integration/smoke checks. Isolated tests do not establish real-engine consistency or authorization behavior.

## Current Handoff

- T001 is complete and validated by focused adapter/collector/root regression tests and the full race suite. Next dependency-ready work is T002 (US1 service/facade enrichment and element discovery tenant correction); no blockers. Recheck tasks.md dependencies before selection. T003 depends on T002 and is not part of iteration 1.
