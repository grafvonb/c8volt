# Research: Harden Tenant Handling Across Tenant-Aware Commands

## Decision 1: Keep tenant correctness in the versioned process-instance services, not in command-level post-filtering

- **Decision**: Implement tenant enforcement inside the existing `internal/services/processinstance/v87` and `v88` services, with walker/waiter flows inheriting the same contract through the shared API surface.
- **Rationale**: The spec explicitly forbids client-side post-filtering of already-fetched cross-tenant data. The command layer is already thin and should stay that way; the versioned services are the right place to select tenant-safe upstream calls.
- **Alternatives considered**:
  - Filter returned process instances in `cmd/` or `c8volt/process`: rejected because it would still fetch cross-tenant data and only hide it after the fact.
  - Add a parallel tenant safety wrapper around the facade: rejected because it would duplicate the current repository-native service structure instead of strengthening it.

## Decision 2: Treat `v88` search/query paths as the preferred tenant-safe baseline

- **Decision**: Use `v88` generated search/query paths as the primary tenant-safe mechanism, especially where direct retrieval endpoints cannot enforce tenant scope.
- **Rationale**: `internal/services/processinstance/v88/service.go` already injects `TenantId` into `SearchProcessInstancesWithResponse`, while direct methods like `GetProcessInstanceWithResponse` and `GetProcessInstanceStateByKey` still look like unscoped direct gets. That makes search the safest current upstream contract for tenant enforcement.
- **Alternatives considered**:
  - Continue relying on direct `GetProcessInstanceWithResponse` for all `v88` lookups: rejected because it does not show any tenant constraint in the request shape and is the exact kind of mixed-flow leak the issue reports.
  - Keep mixed behavior and document caveats: rejected because the feature is about correctness, not disclosure of known unsafe paths.

## Decision 3: Narrow `v87` support to only the operations that can be made tenant-safe through existing upstream calls

- **Decision**: Audit `v87` operation by operation and only preserve supported tenant-aware behavior where the generated client and upstream semantics can enforce it without local filtering; otherwise return an explicit unsupported outcome for that exact segment.
- **Rationale**: `v87` currently mixes Operate direct-get (`GetProcessInstanceByKeyWithResponse`) with search (`SearchProcessInstancesWithResponse`), and the direct-get path shows no tenant scoping. The clarified spec requires unsupported behavior to be scoped narrowly instead of pretending the version is fully safe.
- **Alternatives considered**:
  - Mark all `v87` tenant-aware commands unsupported: rejected by clarification because unsupported behavior should be scoped to the unsafe segment, not the whole command family.
  - Fake tenant correctness by performing a direct get and then comparing `tenantId`: rejected because it still leaks cross-tenant existence and violates the no-post-filtering rule.

## Decision 4: Treat supported tenant mismatch as the same `not found` outcome as an absent resource

- **Decision**: Normalize supported tenant mismatches to the same user-facing `not found` contract used for genuinely absent resources.
- **Rationale**: This is the safest contract for tenant isolation because it does not reveal whether the resource exists in another tenant. The clarified spec explicitly chose this outcome.
- **Alternatives considered**:
  - Return an explicit tenant mismatch error: rejected because it leaks that the resource exists elsewhere.
  - Return generic access denied: rejected because it changes the command contract without improving tenant privacy over `not found`.

## Decision 5: Scope unsupported `v87` behavior to exact operations and flow segments

- **Decision**: If a `v87` direct get, state check, wait, delete preflight, or ancestry step cannot be made tenant-safe, fail that specific segment explicitly rather than blocking all related commands.
- **Rationale**: Walker and waiter compose service methods transitively. Narrow unsupported outcomes let safe search-based flows remain usable while still preventing unsafe direct-get-based traversal or mutation steps.
- **Alternatives considered**:
  - Block the whole command family when one sub-step is unsafe: rejected by clarification because it would unnecessarily reduce usable CLI behavior.
  - Leave unsupported handling to per-command discretion: rejected because it would fragment the tenant contract again.

## Decision 6: Treat `8.9` as a planning audit note, not an immediate implementation target in this repository

- **Decision**: Record `8.9` as audited scope at the planning level, but do not promise `v89` code changes in this feature unless the repository first gains a `processinstance` `v89` implementation.
- **Rationale**: `toolx.NormalizeCamundaVersion(...)` accepts `8.9`, but `toolx.SupportedCamundaVersions()` and `internal/services/processinstance/factory.go` still only support `8.7` and `8.8`. The plan must stay aligned with the repository’s real support surface.
- **Alternatives considered**:
  - Ignore `8.9` entirely: rejected because the clarified spec asked for `8.9` to be audited for parity if relevant.
  - Assume a `v89` implementation can be added as part of this feature: rejected because it would expand scope into a new versioned service and generated-client surface that is not currently present.

## Decision 7: Use walker/waiter composition as the main mixed-flow audit seam

- **Decision**: Audit tenant correctness through `GetProcessInstance`, `GetProcessInstanceStateByKey`, `GetDirectChildrenOfProcessInstance`, `walker.Ancestry`, `walker.Descendants`, `waiter.WaitForProcessInstanceState`, and the cancel/delete flows that compose them.
- **Rationale**: The reported bug is not just a single command problem; it arises when direct lookups and follow-up searches mix. The shared walker/waiter helpers are the repository-native place where those mixed flows converge.
- **Alternatives considered**:
  - Audit only `walk pi`: rejected by the issue and clarified spec as too narrow.
  - Audit only the low-level search methods: rejected because wait/cancel/delete can still leak via direct state checks or ancestry lookups.

## Decision 8: Add regression coverage for every tenant-aware command family and every tenant source

- **Decision**: Add or update tests for all tenant-aware command families and cover explicit `--tenant`, environment-derived tenant, profile-derived tenant, and base-config-derived tenant separately.
- **Rationale**: The clarified spec requires full command-family audit coverage and treats env/profile/base-config as distinct propagation paths. Existing command tests under `cmd/` plus versioned service tests under `internal/services/processinstance/` provide the right seams.
- **Alternatives considered**:
  - Cover only changed code paths: rejected by clarification because every tenant-aware command family needs explicit regression proof.
  - Collapse non-flag tenant sources into one “derived tenant” case: rejected because issue `#107` already demonstrated source-specific propagation bugs.

## Decision 9: Freeze the unsupported-version boundary at the operation level before code changes

- **Decision**: Treat the current repository boundary as an operation matrix, not as a blanket version toggle: `v8.8` should converge on tenant-safe search-backed lookup semantics for direct-get-adjacent flows, `v8.7` should keep only the segments that can stay tenant-safe through existing upstream calls, and `v8.9` remains audit-only until a repository-native service exists.
- **Rationale**: The shared API refactor needs one authoritative contract before interface changes start. Without an operation-level boundary, later code could over-block all of `v8.7`, leave unsafe `v8.8` direct-key seams in place, or accidentally imply that `8.9` is supported at runtime when the factory still stops at `v88`.
- **Alternatives considered**:
  - Defer the boundary definition until implementation: rejected because the shared API refactor would otherwise encode assumptions ad hoc.
  - Treat `v8.8` as already correct because tenant filtering exists on search: rejected because direct get and state-check seams still look unscoped.
  - Treat `v8.9` as implicitly supported because `toolx` normalizes it: rejected because the factory and tests still restrict runtime support to `v87` and `v88`.

## Audit Inventory: Current Tenant-Sensitive Seams

### Shared bootstrap and config seam

- [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go) binds and logs the effective tenant via `cfg.App.ViewTenant()`, so commands may appear tenant-aware even when the downstream service path is not.
- `config/app.go` still normalizes tenant defaults differently for `v87` and `v88`, which means source-specific tenant coverage must include version-specific expectations.

### Tenant-aware command family inventory

- [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go) has two materially different paths: keyed lookup goes through `cli.GetProcessInstances(...)`, while filter mode goes through `searchProcessInstancesWithPaging(...)`. The tenant audit must treat the keyed branch as the direct-get seam and the paged branch as the tenant-filtered search seam.
- [`cmd/walk_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go) routes `--parent`, `--children`, and default family traversal through `cli.Ancestry(...)`, `cli.Descendants(...)`, and `cli.Family(...)`. Those flows mix direct lookup and child expansion through the shared walker and are the primary traversal-leak surface.
- [`cmd/cancel_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go) supports both direct keys and search-selected batches. The direct path validates impact through `cli.DryRunCancelOrDeleteGetPIKeys(...)` before `cli.CancelProcessInstances(...)`; the search path pages through search results before running the same preflight and mutation logic.
- [`cmd/delete_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go) mirrors cancel: search-first selection is possible, but the actual delete path still depends on `cli.DryRunCancelOrDeleteGetPIKeys(...)`, descendant/root discovery, optional cancel-first behavior, and waiter-backed absence confirmation.
- [`cmd/run_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_processinstance.go) seeds `TenantId` directly into `process.ProcessInstanceData`, so creation already receives the selected tenant. The tenant-sensitive follow-up seam is confirmation and wait behavior inside the versioned service after creation, not the Cobra flag parsing itself.

### Facade and versioned service seam

- [`c8volt/process/client.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go) simply forwards process-instance calls into the shared service API, so tenant correctness must live in the underlying service implementations.
- [`internal/services/processinstance/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go) currently supports only `v87` and `v88`.
- `v87` uses Operate `GetProcessInstanceByKeyWithResponse` for direct get and Operate `SearchProcessInstancesWithResponse` for search.
- `v88` uses Camunda `GetProcessInstanceWithResponse` for direct get and Camunda `SearchProcessInstancesWithResponse` for search, with explicit `TenantId` filter injection only on the search path.

### Confirmed version and factory boundary

- [`toolx/version.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/toolx/version.go) normalizes `8.7`, `8.8`, and `8.9`, but `SupportedCamundaVersions()` still returns only `8.7` and `8.8`.
- [`internal/services/processinstance/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go) matches that support surface exactly by constructing only `v87.New(...)` and `v88.New(...)`; all other normalized values fail through `services.ErrUnknownAPIVersion`.
- [`internal/services/processinstance/factory_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory_test.go) already locks that behavior in with concrete-type assertions for `v87` and `v88` and an unsupported-version assertion that expects the rendered version to normalize to `"unknown"`.

### Authoritative unsupported-version boundary for this feature

- `v8.8` is the preferred tenant-safe baseline, but direct get and state-by-key must not remain authoritative if they cannot carry tenant scope; those paths need search-backed or equivalent tenant-safe behavior before they count as supported.
- `v8.7` may keep search-backed tenant-safe flows such as filtered search and child lookup built on that search path, but any direct-get-dependent segment that cannot be made tenant-safe must surface an explicit unsupported outcome at that exact seam.
- Walker, waiter, cancel, and delete inherit the contract of the lookup and state-check methods they compose, so unsupported behavior must be assigned per composed segment rather than per command family.
- `v8.9` is audited only for parity and documentation in this feature; no runtime-support claim is valid until `internal/services/processinstance/factory.go` and companion tests admit a real `v89` implementation.

### Implemented verification anchors

- `internal/services/processinstance/factory_test.go` now locks in the honest runtime matrix: `v87` and `v88` construct real services, while normalized `8.9` still fails as an unsupported process-instance runtime version.
- `config/app_test.go` now proves config normalization accepts `8.9` as a known version value without silently treating it like `v87` default-tenant behavior; audit scope and runtime support remain separate concerns.
- `internal/services/processinstance/v87/service_test.go` now proves the unsupported surface stays narrow: direct key lookup and state-by-key remain unsupported, but search-backed child lookup still works with tenant scoping.
- `cmd/get_processinstance_test.go` now proves the command family follows the same split in real CLI flows: `get process-instance --key` stays unsupported on `v87`, while tenant-scoped search mode still succeeds.

### Mixed-flow helpers

- [`internal/services/processinstance/walker/walker.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go) composes `GetProcessInstance` and `GetDirectChildrenOfProcessInstance`, making it a primary cross-tenant leakage seam.
- [`internal/services/processinstance/waiter/waiter.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter.go) polls `GetProcessInstanceStateByKey`, so wait-based commands inherit whatever tenant safety that method provides.
- `CancelProcessInstance` and `DeleteProcessInstance` in both versioned services call state checks, ancestry, descendants, and family traversal before mutating operations.

## Audit Inventory: Existing Regression Seams

- `internal/services/processinstance/v88/service_test.go` already contains tenant assertions for search request bodies, including omission when config tenant is empty. This is the cleanest seam for direct-get vs search tenant-safe behavior in `v88`.
- `internal/services/processinstance/v87/service_test.go` is the corresponding seam for `v87` search/direct-get behavior and unsupported-operation handling.
- `internal/services/processinstance/walker/walker_test.go` and `waiter/waiter_test.go` are the right places to verify helper behavior once service-level contracts change.
- Command-family tests under `cmd/` already exist for `get`, `walk`, `cancel`, `delete`, `run`, and related flows, and can be extended with temp config plus env/profile/base-config tenant cases.
- `internal/services/processinstance/factory_test.go` currently proves only `v87` and `v88` support, which should be preserved as the honest repository support boundary until a `v89` service exists.

### Confirmed regression anchors for this feature

- [`cmd/get_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go) already covers paged search behavior and explicit `--tenant` request capture; it is the natural place to add direct-lookup tenant mismatch and derived-tenant-source cases.
- [`cmd/walk_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go) already exercises walker-backed JSON/tree flows with tenant-bearing fixtures, which makes it the anchor for ancestry/descendants/family tenant propagation checks.
- [`cmd/cancel_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go) and [`cmd/delete_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go) already cover paged search selection, dependency expansion, and v87/v88 behavior splits; these tests can be extended to prove that preflight, wait, and mutation steps preserve the same tenant contract.
- [`cmd/run_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go) already proves profile-derived tenant selection feeds the run payload, so it is the command-side seam for start-plus-confirmation tenant propagation.
- [`config/config_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config_test.go) already distinguishes base-config, profile, env, and flag precedence, and [`cmd/bootstrap_errors_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/bootstrap_errors_test.go) already shows the resolved tenant in bootstrap output. Those are the best starting points for T007.
