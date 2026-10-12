# Service and Facade Contract

## Prepared workflow

Add PI-specific plan and execute operations, analogous to UT, under `internal/services/processinstance` and `c8volt/process`:

- `PlanProcessInstanceVariableUpdates(context, keys, variables, options) → plan, error`
- `ExecuteProcessInstanceVariableUpdates(context, plan, wantedWorkers, options) → existing result collection, error`

These are proposed additive method names. Planning and execution mechanics belong to the internal service; facade signatures use public models and `foptions.FacadeOption` only. Facades map errors through `ferrors.FromDomain` while returning available results alongside errors.

### Planning preconditions and guarantees

1. Accept stable-deduplicated keys and JSON-compatible values; reject malformed keys/values without mutation.
2. Read required selected-instance facts and complete root-variable data before returning a usable plan.
3. Preserve tenant scoping and evidence. The public command continues rejecting missing target input; an empty internal collection can return an empty non-mutating plan.
4. Deep-copy nested requested values, classify deterministically, and mark truncated comparisons as requiring change rather than equality.
5. Any planning failure returns an error and no executable partial plan. No writer is invoked in preparation.

### Execution preconditions and guarantees

1. Validate and freeze the whole supplied plan before submission, including selected keys, root destinations, unique target/name association, requested/classified values, and exact payload correspondence.
2. Do not rediscover or replan destinations/payloads after approval. Backend verification can read values but cannot expand mutation scope.
3. Forward worker, fail-fast, no-worker-limit, tenant, cancellation, no-wait, and suppression options according to the existing service conventions.
4. Submit only nonempty planned payloads via existing `UpdateScopeVariables`; the adapter sets explicit `local=true` and owns retries. Do not submit the original complete input payload.
5. Unless no-wait applies, run a single waiter workflow for each accepted changed root, checking all frozen requested values through complete reads. A successful wait is final; no extra confirmation read follows.
6. Return ordered keyed outcomes, including unchanged/skipped work in mixed collections. Preserve accepted outcomes on later errors; fail-fast does not erase outcomes for in-flight work.
7. No-wait suppresses only result verification. Operator approval remains a command concern.

## Variable page contract

Add a single-page root-variable operation to `internal/services/variable/api.go` with domain request/result types described in [data-model.md](../data-model.md). Version adapters retain generated API details, root and tenant filters, and raw value decoding. A shared variable search helper handles traversal; it must not import the adapter-owning factory package.

Existing `SearchProcessInstanceVariables` keeps its signature and becomes a complete-list read. It must propagate later-page errors instead of returning incomplete success. Request `truncateValues=false` on every page, but still preserve and respect returned truncation flags. Cancellation, cursor progression, offset overflow, malformed metadata, exact totals, and capped lower bounds require tests in the owning layer.

## Composition and compatibility

Wire a composed update service from PI reader, variable reader/writer, configuration, and logger in the existing root client factory. Inject it into a new additive process facade constructor. Preserve existing constructors and legacy update entry points. A facade built without the new dependency returns a precondition error for new prepared operations rather than silently falling back to broad writes.

Add public/domain `unchanged` and `skipped` status values while retaining existing status strings and JSON fields. `submitted` remains the serialized accepted/no-wait state. Preserve existing `confirmationStatus` wording where applicable. The added values are a documented compatibility extension; no new envelope/schema fields are required.

Legacy writer behavior is not redesigned in this feature. The new prepared route bypasses its nested wait; shared complete-list reads and the truncated-safe waiter receive regression coverage for legacy consumers. UT behavior must remain unchanged.
