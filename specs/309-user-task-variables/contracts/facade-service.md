# Facade and Service Contract

## Public facade

Add to `c8volt/task.API`:

```go
EnrichUserTasksWithVariables(ctx context.Context, tasks UserTasks, opts ...foptions.FacadeOption) (VariableEnrichedUserTasks, error)
```

The method maps selected task records into domain inputs, maps facade options using existing helpers, delegates to the internal enrichment function, maps the successful result using [data-model.md](../data-model.md), and converts errors with `ferrors.FromDomain`. It adds no pagination, sorting, filtering, per-task requests, or worker policy to the facade. `Total` is derived from selected items rather than an upstream matching count. Nil or empty input produces initialized empty results without requests.

The existing constructor and all lookup/search/count/resolver methods retain their signatures and behavior. The enclosing `c8volt.API` already embeds task API. Update affected stubs and compile-time assertions. No separate public variable-search endpoint is required for this command.

## Internal adapter contract

Add to the shared usertask API and matching version APIs:

```go
SearchUserTaskEffectiveVariablesPage(ctx context.Context, key string, page domain.UserTaskVariablePageRequest, opts ...services.CallOption) (domain.UserTaskVariablePage, error)
```

v88, v89, and v810 extend their generated client interfaces with `SearchUserTaskEffectiveVariablesWithResponse`. v87 implements a no-request unsupported-operation result. All implementations retain configured HTTP policy, context, shared payload/status validation, and domain error mapping. Discovery tenant filters are not added to this keyed effective-variable operation; backend authorization applies. Do not call the legacy task resolver or Tasklist.

### Native request

- Method/path: `POST /v2/user-tasks/{userTaskKey}/effective-variables/search`.
- Query: `truncateValues=false` for every page, independent of human display settings.
- Generated body: `UserTaskEffectiveVariableSearchQueryRequest`, aliased by `SearchUserTaskEffectiveVariablesJSONRequestBody`.
- Body page: offset `from` and `limit`; default internal size 1000. No `after`/`before`.
- Sort: name ascending. No variable predicates, tenant discovery predicates, or scope filters.
- Response: require successful payload, then decode raw body to retain `value`, `isTruncated`, and `truncated`; generated fields alone are insufficient.
- `isTruncated` takes precedence when present, otherwise use `truncated`, otherwise false, matching existing variable conversion. Preserve actual returned identifiers and scope.
- Do not issue full-value recovery GETs. Any remaining truncation is represented explicitly.

## Internal service workflows

Add `SearchUserTaskEffectiveVariables` for complete per-task pagination and `EnrichUserTasksWithVariables` for selected-task attachment in `internal/services/usertask/variables.go` (or a focused enrichment sibling if needed).

Enrichment is sequential, matching existing process-variable enrichment. It preserves order, checks cancellation, makes no requests for empty inputs, and fails the overall result on any retrieval failure. Existing task-key bulk concurrency is unaffected. No generic enrichment framework or new worker flags are introduced.

### Pagination algorithm

1. Start at offset zero with fixed positive size. Validate each adapter page, including request echo, raw item count, and exact/lower-bound total metadata.
2. Accumulate raw observed counts independently of duplicate normalization. Preserve all returned records until effective-name validation.
3. For exact totals, reject a total below observed raw count; finish when equal. A short or empty page below the total continues.
4. For capped totals, retain the highest reported lower bound across pages and do not stop at that bound or on a short nonempty page. Finish only on an empty page after satisfying that bound with no further continuation evidence. Empty pages below the bound or carrying a nonempty end cursor continue by offset. Preserve this evidence in `HasContinuationEvidence`; never turn the cursor into request input. A later exact total contradicting the retained lower bound is malformed.
5. Advance offset by raw item count, or by requested page size for an empty page that requires continuation. Use checked arithmetic and reject int32 overflow. Context cancellation stops further requests.
6. Normalize identical duplicate records by name and sort ascending only after collection. Conflicting records for one name fail as malformed responses; never pick a scope winner locally.
7. Missing body/payload/required total metadata, invalid field types, inconsistent totals, and failures on later pages return errors, not empty collections.

Tests must exercise exact and capped populations, retained lower bounds, sparse pages including empty pages with continuation evidence beyond a cap, empty results, duplicate normalization, cancellation and overflow. The existing task cursor walker remains unchanged; its validation patterns can be reused without broad refactoring.

## CLI call sequence

- Keyed: existing strict `GetUserTasks` → eligible enrichment → task view.
- Incremental search: existing service trims `step.Page.Items` → eligible enrichment of that selected page → view → existing paging decision. The callback does no backend loop.
- Collected search (including JSON): existing service completes selected collection → enrich once → one final view.
- Already streamed results do not enter a second enrichment pass; final summary is task count only.
- Non-enriched, effective keys-only, count, and empty results do not trigger variable requests.

Callbacks propagate enrichment and writer errors. Existing completion dispositions, prompt eligibility, quiet/automation behavior, and task search metadata remain intact.
