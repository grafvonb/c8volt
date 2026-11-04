# Research: Suspended Process-Instance State

## 1. Extend the existing state vocabulary

**Decision:** Add `StateSuspended` to domain and public state constants and the public parser's valid-state list. Preserve existing terminal sets and canceled/terminated equivalence.

**Evidence:** `internal/domain/state.go` owns lifecycle classification and creation-observable states. `c8volt/process/state.go` uses case-insensitive canonical parsing and accepts `cancelled` as an alias. Public state-report conversions use parsing; instance/expectation conversions mostly cast directly. Domain `ABSENT` is terminal while public `ABSENT` is not; retain that intentional existing difference for this issue.

**Rationale:** Extending the canonical vocabulary fixes UNKNOWN fallback without introducing alternate state handling.

**Alternatives considered:** Mapping suspension to ACTIVE or UNKNOWN violates exact filtering and truthful reports; changing all terminal sets is unrelated and risks mutation behavior.

## 2. Guard explicit requests at internal service entry points

**Decision:** Add `ValidateProcessInstanceStates(states domain.States) error` to the existing process-instance service contract and corresponding version contracts. Implement the capability decision in each adapter: v88/v89 reject a case-insensitive suspended member with wrapped `domain.ErrUnsupported`; v810 permits it. This method validates suspension capability, not a new generic grammar for every state.

**Evidence:** `internal/services/processinstance/api.go` has no shared Config method. Concrete adapters own version configuration. `search.go` has a direct-incident-index path that bypasses `SearchForProcessInstancesPage` and can return an empty success before any instance lookup. `wait.go` delegates bulk requests to versioned bulk methods.

**Rationale:** An explicit contract lets shared search validate before incidental discovery or short-circuits while keeping public facades mechanical. It avoids optional type assertions that could silently skip capability enforcement.

**Placement:** Validate at the start of shared `SearchProcessInstancesPages` and `SearchProcessInstancesTotal`, before direct-index discovery; validate adapter page-search entry points for direct callers. Validate adapter single-state, single-expectation, bulk-state, and bulk-expectation entry points before lookup or worker dispatch, including empty keys. Shared bulk wrappers may delegate to these guarded methods without duplicating policy. Update all affected interfaces, implementations, and test doubles together.

**Alternatives considered:** CLI-only checks miss public consumers. Page-search-only guards miss direct incident selection. A new framework or exposing configuration through the API is unnecessary. Optional validator interfaces weaken guarantees.

## 3. Preserve implicit creation visibility

**Decision:** Add suspension to `ObservableProcessInstanceCreationStates`, but keep explicit-request capability checks out of the generic waiter.

**Evidence:** All three `v*/service.go` creation paths call `waiter.WaitForProcessInstanceState` directly using the domain observable-state list. Explicit public waits pass through adapter entry points. `waiter/waiter.go` already compares states case-insensitively and preserves canceled/terminated equivalence.

**Rationale:** A suspended observation proves existence without proving completion. Guarding the generic waiter would reject the expanded implicit creation list on older versions and break their defaults.

**Alternatives considered:** Waiting for ACTIVE after suspension could time out despite successful creation; expanding terminal states would make terminal expectations incorrect.

## 4. Preserve nullable timestamps through current conventions

**Decision:** Add `SuspendedDate string` to domain and public ProcessInstance, with public JSON tag `suspendedDate,omitempty`. Map v810's generated `*time.Time` through existing `formatTimePtr` (UTC RFC3339Nano; nil/zero to empty string); map both public conversion directions.

**Evidence:** `internal/domain/processinstance.go`, `c8volt/process/model.go`, `c8volt/process/convert.go`, and `internal/services/processinstance/v810/convert.go` use this pattern for existing dates. The generated result documents an absent timestamp when currently unsuspended.

**Rationale:** Nullability is represented by absence/empty string in this repository, not a new public pointer convention. Fresh conversion must overwrite absence instead of merging only nonempty fields.

**Alternatives considered:** New pointer semantics would diverge from adjacent public date fields; carrying forward the last nonempty timestamp would invent historical state.

## 5. Keep CLI presentation and mode scope narrow

**Decision:** Extend existing instance read/expectation metadata, validation and value completions. Preserve default human rows. Test refreshed metadata using repeated reads and structured rendering.

**Evidence:** `cmd/get_processinstance.go`, `get_processinstance_validation.go`, `expect_processinstance.go`, and `cmd_views_processinstance.go` own these surfaces. No production state-value `RegisterFlagCompletionFunc` registration currently exists; `completion_test.go` provides bootstrap-free completion tests. Existing watch code is `get_processdefinition_watch.go`, which carries definition snapshots/statistics rather than instance metadata.

**Rationale:** No process-instance watch is needed or in scope. Audit hard-coded selector lists in slow analysis and mutation commands for truthful descriptions, but do not broaden mutation eligibility or purge selection owned by #340/#341.

**Alternatives considered:** Adding a watch mode or definition suspension statistics expands the issue; globally adding state choices to every command without respecting command-specific restrictions changes unrelated workflows.

## 6. Validate against the required server without regenerating clients

**Decision:** Keep client baseline 8.10.3 at the exact commit/hashes in [planning-context.md](planning-context.md). Use server 8.10.4 for live validation and record both versions separately.

**Evidence:** Current provenance names upstream commit `3961b79c3fdaf43c48691ed77672d6db42d0abcc`. The v810 generated client already includes SUSPENDED and SuspendedDate. `integration/cli/batch_test.go` offers generated-client setup and evidence patterns; `isolated_seed_test.go` provides isolated held-instance datasets. Existing release gates check the 8.10 minor line, so this feature's live test must additionally verify observed patch version 8.10.4 before setup mutations.

**Rationale:** Reuse existing integration infrastructure and fixtures for #342. Live suspension/resumption is test setup, not a new public feature.

**Alternatives considered:** Client regeneration is expressly excluded. Minor-only server validation cannot prove the requested patch target. A separate integration runner duplicates existing infrastructure.

All design unknowns are resolved from repository evidence. Live behavior remains to be validated during implementation; no tests or live mutations ran in this planning phase.
