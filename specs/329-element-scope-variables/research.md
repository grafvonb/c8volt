# Research: Element Scope Variables

**Date**: 2026-10-25

**Branch**: `329-element-scope-variables`

**Reviewed implementation**: `6711d829`

**Sources**: [specification](spec.md), repository source, `AGENTS.md`, and constitution 2.0.0. Research was read-only; no live services or runtime tests were used.

## 1. Exact-local reads without changing process-root reads

**Decision**: Add a version-neutral `ScopeVariableQuery` with separate owning process and scope keys, and optional expected tenant evidence. Add page and complete scoped-read methods. Reuse the existing variable page/item types and extract the current traversal into a narrowly scoped callback-based collector; retain the existing process-root methods as compatibility wrappers with owner equal to scope.

**Rationale**: `internal/services/variable/search/search.go` already handles cursor/offset progress, raw page counts, sparse pages, exact/lower-bound totals, cancellation, and stable sorting. `internal/services/variable/v88/variables.go` and the v89/v810 equivalents currently filter both process and scope to the process key. Merely passing an element key would create the wrong process filter. Raw decoders in each `convert.go` preserve value/truncation facts omitted by generated models.

**Alternatives considered**: Copying a second pagination loop risks divergent completion rules. Replacing root semantics would break existing callers. Renaming every historically process-named page/item type adds unrelated churn. A generic workflow framework is unnecessary.

**Constraints**: Preserve raw counts before filtering, `truncateValues=false`, version-specific cursor fields and total semantics. Ignore unrelated scopes, but reject an item claiming the selected scope with inconsistent owner or known tenant. Missing required identity is failure. Apply this stricter identity validation to the new scoped path without silently changing the root wrapper's established filtering/error behavior. A failed complete read cannot return a successful empty collection.

## 2. Selection and enrichment composition

**Decision**: Compose existing element, listener, and variable dependencies in `internal/services/element`. Enrich the selected objects, never rediscover them. Provide additive facade enrichment entry points, retaining current constructors and methods. Add `Variables` as a pointer to a slice on domain/public elements.

**Rationale**: Existing listener enrichment already resides in `internal/services/element/enrichment.go`. Variable-only search must preserve `cmd/get_element_search.go` page prompts and early stops; listener search currently uses collected search, which remains the combined-mode policy. Nil enrichment means omitted; a pointer to an allocated empty slice means a successful empty local read.

**Alternatives considered**: Fetching each element again wastes requests and can attach data to a different discovery snapshot. Putting variable loops in the command or facade violates layering. Always collecting searches would change ordinary paging behavior.

**Paging trap**: Versioned `SearchElementsPages` implementations append element structs to their result before invoking the visitor. Updating only the visitor's slice does not update the final aggregate. The composed service must accumulate enriched page items itself, forward existing step metadata and visitor actions, and replace the returned aggregate items. A scope is published only after its complete variable read; JSON cannot report an incompletely enriched selection as success.

## 3. Tenant behavior and an explicit compatibility correction

**Decision**: Explicit-key workflows use `WithIgnoreTenant` through lookup, all variable pages, and confirmation. Scoped reads are bound to resolved owner/scope identity and validate known tenant evidence; they must never substitute the configured discovery tenant. Search discovery applies its configured tenant server-side unless `IgnoreTenant` is set.

**Rationale**: All three element `SearchElementsPage` adapters currently discard applied call options and omit `TenantId`. All three generated `ElementInstanceFilter` types support `TenantId`. Satisfying FR-017 therefore requires a small correction to these adapters, rather than claiming tenant filtering already works. Setting that filter in the existing page request also covers collected search and totals without local filtering or changed limit arithmetic.

**Compatibility note**: A configured-tenant element search that previously leaked other tenants' elements will now return only the configured tenant. Document this correction and test filtered/unfiltered requests and totals in v88/v89/v810. Native explicit GET remains unchanged; backend authorization still applies. An absent optional tenant observation is unknown, not proof of mismatch or authorization. No extra lookup is allowed solely to display tenant context.

**Alternatives considered**: Client-side tenant filtering breaks totals and page/limit semantics. Retaining the defect conflicts with the specification. Applying the configured tenant to explicit scoped reads can create false additions.

## 4. Frozen collection-wide planning

**Decision**: Add an element variable-update service modeled on `internal/services/processinstance/update.go` and `update_validation.go`. Normalize unique keys in input order, read all selected identities and local variables before approval, classify each local name, and freeze separate mutation and verification maps. Validate and deep-copy the supplied plan before starting workers.

**Rationale**: PI planning provides the relevant all-before-any safety boundary. UT planning resolves effective owners and therefore cannot implement an explicit local override. Missing local names are additions regardless of parent values; only complete local equality can establish unchanged status.

**Alternatives considered**: Replanning after approval changes the approved operation. Planning and writing one key at a time permits writes before a later planning failure. Reusing UT target grouping risks redirecting a local write to its parent.

## 5. Mutation and confirmation

**Decision**: Reuse `UpdateScopeVariables`, which already sends `local=true` across all supported adapters. One changed unique element is one logical write containing additions/changes, subject to existing retry policy. Confirm all requested values through complete exact-local reads with existing timeout/backoff conventions. No-wait skips only confirmation.

**Rationale**: `UpdateProcessInstanceVariables` has root-oriented semantics and must not be used for this operation. Existing `toolx.DetermineNoOfWorkers`, `toolx/pool.ExecuteSlice`, and service retry helpers supply bounded concurrency, ordering, and cancellation. A focused element wait helper can reuse value equality and backoff while preserving root/UT behavior.

**Alternatives considered**: Writing requested unchanged names expands the mutation unnecessarily. Confirming only changed names misses drift in the approved requested state. Resubmitting after a read-back failure obscures accepted evidence. Recursive scope walks and rollback are outside scope.

## 6. Submission evidence survives retry and pacing

**Decision**: Extend the internal `ScopeVariableUpdateResponse` additively with submission evidence: `not_sent`, `accepted`, `rejected`, or `unknown`. Accumulate evidence across physical attempts outside the retry callback. Expose it on the new element outcomes; retain existing PI/UT public schemas and their acceptance boolean mappings.

**Rationale**: Current adapters return early on retry error and lose response evidence. `RetryCamundaMutation` can return a zero response when backoff is canceled. A final pacing failure proves only that attempt unsent, not that an earlier attempt was unsent. Use `httpc.BeginRequestRateAttempt` and its observation, including when `http.Client.Timeout` replaces the typed transport error. The UT completion evidence helper demonstrates sticky uncertainty but cannot be copied unchanged because it does not account for this pacing evidence.

**Aggregation**: No attempted dispatch gives `not_sent`; conclusive rejection without ambiguity gives `rejected`; a dispatched attempt without conclusive response gives `unknown`. Unsent retries cannot erase earlier rejection or uncertainty. A later documented successful response establishes `accepted`; subsequent confirmation errors preserve it. No further write retries follow successful acceptance. Preserve the original error chain/class independently of the evidence. Use the conservative rejection rule in [data-model.md](data-model.md): documented well-formed 400 InvalidData without accompanying error is rejection; server/proxy errors, unexpected/malformed responses and transport ambiguity do not prove rejection. Generated CreateElementInstanceVariablesResponse contracts expose 400/500/503/504 in all three versions; notably 504 may occur while listener-related processing is still unresolved.

**Alternatives considered**: A boolean conflates rejection and uncertainty. Classifying only the final error loses earlier evidence. Changing global retry behavior or PI/UT public JSON is unnecessary.

## 7. CLI output and failure ownership

**Decision**: Keep construction/validation in `cmd/update_element.go`, the batch lifecycle in `cmd/update_element_variables.go`, and rendering in `cmd/cmd_views_element*.go`. Reuse shared envelopes, payload parsing, configured logging, and explicit-writer prompts. Apply the safer task-completion aggregate-failure pattern, with bounded allowlisted reasons and exactly one human ERROR.

**Rationale**: Existing PI/UT variable renderers send raw outcome errors through informational human lines, and `exitAfterRenderedResult` does not itself log. Copying those paths would violate AGENTS.md. `cmd/cmd_views_job_logging.go`, `cmd/cmd_views_usertask_complete.go`, and `boundedCommandFailureReason` provide safer local patterns. Generic no-wait result selection can incorrectly label a no-op accepted; preview/no-op must call an explicit succeeded renderer.

**Alternatives considered**: Direct stderr printing bypasses configured logging. Logging `err.Error()` still leaks transport detail. Rendering an envelope and returning into another error renderer duplicates JSON. Fixing unrelated PI/UT human formatting is excluded.

## 8. Payload, model, and dependency choices

**Decision**: Preserve established variable-update JSON decoding semantics, support exactly one non-null object source, retain raw received value text and API truncation metadata for inspection, and reuse shared display formatting. No new dependency, persistence layer, display flag, or generated-client change is needed.

**Rationale**: Go 1.26 with the existing Cobra/Viper, domain/facade conversion, JSON and test helpers suffices. Numeric parsing changes belong in a separate compatibility change; task-completion parsing is not a reason to change variable-update payload semantics here.

**Alternatives considered**: A new generic variable framework or global number-format rewrite broadens the feature. Display shortening cannot be reused as evidence of backend completeness.

## 9. Validation and delivery boundaries

**Decision**: Plan isolated adapter, service, facade, command/subprocess, and real-terminal tests using repository helpers. Test request counts and deliberate failure signatures, not only output substrings. Implementation will start with targeted package checks and run the race-enabled full suite once shared traversal/interface/concurrency changes are integrated. This planning-only change uses document/link/diff checks only.

**Rationale**: `testx` supplies test-owned HTTP servers, temporary configuration, separate streams, and terminal runners. AGENTS.md excludes live services from implementation gates. The shared collector and writer-evidence changes have cross-package impact that justifies broader regression validation after targeted checks.

**Alternatives considered**: Live Camunda smoke/integration gates violate the implementation boundary. Mocked terminal eligibility cannot prove actual prompt routing. A passing substring test cannot prove severity, destination, redaction, or single emission.

**Resolution**: All design questions above are resolved. Real-engine behavior remains outside isolated validation coverage and must not be reported as verified.
