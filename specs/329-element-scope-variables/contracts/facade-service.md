# Facade and Service Contract

These are design signatures, not implemented declarations. The [data model](../data-model.md) defines the corresponding new types and evidence rules.

## Public element facade

Keep all existing `c8volt/element.API` methods and existing `New`/`NewWithListeners` constructor signatures. Add the following operations, using the package's existing facade option conventions:

```go
type Enrichment struct {
    Variables bool
    Listeners bool
}

GetElementWithEnrichment(ctx context.Context, key string, enrichment Enrichment,
    opts ...foptions.FacadeOption) (Element, error)
SearchElementsWithEnrichment(ctx context.Context, request SearchRequest, enrichment Enrichment,
    opts ...foptions.FacadeOption) (SearchResult, error)
SearchElementsPagesWithEnrichment(ctx context.Context, request SearchRequest, enrichment Enrichment,
    visitor SearchPageVisitor, opts ...foptions.FacadeOption) (SearchPagesResult, error)
PlanElementVariableUpdates(ctx context.Context, keys typex.Keys, variables map[string]any,
    opts ...foptions.FacadeOption) (ElementVariableUpdatePlan, error)
ExecuteElementVariableUpdates(ctx context.Context, plan ElementVariableUpdatePlan, wantedWorkers int,
    opts ...foptions.FacadeOption) (ElementVariableUpdateResults, error)
```

An enrichment selection with neither flag preserves ordinary behavior. The CLI uses paged enrichment for variable-only search, collected enrichment for listeners with/without variables, and no enriched call for effective keys-only/total. Combined enrichment must not repeat base selection. The paged facade delegates traversal and aggregation to the composed service; it only maps the visitor/action and domain/public types.

Add a composition constructor for the new services and wire it in `c8volt/client.go` using existing shared clients/factories. Legacy constructors remain usable for existing methods; absent dependencies for new methods return converted precondition errors, not panics. Facades deep-copy mutable inputs/outputs, convert errors through `c8volt/ferrors`, and preserve partial execution results alongside the converted error. No facade implements paging, pools, retries, planning, or polling.

Adding methods to the exported interface requires updating implementers/test doubles; record this source compatibility impact in API release guidance. Existing method signatures and result JSON remain intact. Update in-repository doubles with the interface change in the same compiling work unit.

## Composed internal element services

Add a focused enrichment service and variable-update service under `internal/services/element`. Dependencies are narrow interfaces for existing element selection, listener enrichment, complete scoped reads, and exact-local mutation. Reuse the same configured HTTP/rate-limiter context as other services; do not create a second client or limiter.

Enrichment accepts already selected domain elements, validates identity, and attaches completed local reads. For paged selection, maintain an enriched accumulator while forwarding the original page metadata/visitor action; replace the base aggregate's items before returning. Preserve limit trimming, ordering, early stop, counts, and sparse-page behavior. A variable-page failure propagates as an operation failure, never a successful empty scope.

Planning/execution methods mirror the public operations with domain types and `services.CallOption`. Planning normalizes first-occurrence keys and completes every lookup/local read before returning a plan. Execution validates the full supplied plan, freezes a deep copy, and runs only changed destinations through existing worker helpers. Worker count, fail-fast, no-worker-limit, cancellation, timeout, backoff, and shared pacing follow existing service options.

Complete confirmation reads compare every requested value at the frozen owner/scope. Unrelated or truncated observations cannot confirm. No-wait bypasses this step only. Confirmation never resubmits or redirects mutations. Accepted work whose confirmation was never started remains accepted with confirmation `not_started`, and the original cancellation/fail-fast failure remains; do not turn it into unsubmitted/skipped work. Preserve original joined error causes for classification, but do not log them at normal human levels.

## Scoped variable reader

Extend `internal/services/variable` and v88/v89/v810 implementations consistently:

```go
SearchScopeVariablesPage(ctx context.Context, query domain.ScopeVariableQuery,
    page domain.ProcessInstanceVariablePageRequest, opts ...services.CallOption,
) (domain.ProcessInstanceVariablePage, error)
SearchScopeVariables(ctx context.Context, query domain.ScopeVariableQuery,
    opts ...services.CallOption,
) ([]domain.ProcessInstanceVariable, error)
```

Retain `SearchProcessInstanceVariablesPage` and `SearchProcessInstanceVariables`. Their owner=scope wrappers preserve current tenant behavior, error wording/classification, filtering, and traversal results. Extract one narrow callback collector in `variable/search`; do not fork its progress algorithm. Reuse raw page/item representations despite historical names.

Each adapter must send independent process and scope filters, request untruncated values, and retain raw response metadata. Preserve version-specific cursor and total fields. Count raw items before filtering; validate claimed-scope identity and known tenant evidence; follow sparse continuation; reject cursor cycles and inconsistent totals. Retain deterministic name/key ordering. Missing owner identity is not empty success.

Explicit-key calls carry `IgnoreTenant` throughout all phases. A scoped query may bind known observed tenant evidence, but never uses an unrelated configured discovery tenant as a filter. Validate identity against observations; optional missing evidence alone is not authorization failure. Backend errors always propagate.

## Local mutation and evidence

Keep the existing `UpdateScopeVariables(ctx, scopeKey, values, opts...)` primitive and its `local=true` request. Add internal submission evidence to its response and preserve that response on errors. Use a focused shared evidence accumulator for the three adapters. Do not modify retry policy, endpoint selection, or generated clients.

Within every physical attempt, create a fresh `httpc.BeginRequestRateAttempt` observation. Combine conclusive response/dispatch evidence with observation failures before updating logical-write evidence. Preserve it when the retry helper returns a zero response after backoff cancellation. Never mark a later unsent retry as proof that earlier work was unsent. Apply the conservative documented-response rejection rule in the data model; generic server/proxy failures or response-plus-error do not prove rejection. A successful response ends mutation retry; confirmation errors cannot clear acceptance.

Keep existing PI/UT public mappings compatible, including their serialized schema. Add tests proving this shared internal change does not alter their destination semantics or task completion. Do not retrofit the new element outcome field into existing public schemas.

## Discovery tenant correction

In each element adapter's `SearchElementsPage`, retain `ApplyCallOptions(opts)` and set generated `ElementInstanceFilter.TenantId` from a nonblank configured tenant unless `IgnoreTenant`. All collected/page/total paths inherit this server-side filter. Do not filter elements locally or alter native explicit GET.

This corrects current behavior where configured-tenant element searches omit the filter. It can narrow returned elements and totals; document it explicitly. No generated-client update is required because all three generated filter types already support the field.

## Error ownership and preconditions

Domain/service errors preserve invalid-input, precondition, unauthorized, not-found, cancellation/deadline, and operation failure evidence through normal facade conversion. Plan validation fails before writes. Unsupported configured 8.7 remains the existing bootstrap/precondition exit 1 before remote work, even when output configuration has not yet initialized.

No internal service or facade emits CLI results or prompts. The command owns one envelope or selected-format result and one configured human aggregate diagnostic when applicable. Render failure handling must not replace the original operation failure with success, lose partial outcomes, or trigger a second envelope.
