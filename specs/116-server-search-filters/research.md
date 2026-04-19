# Research: Push Supported Get Filters Into Search Requests

## Decision 1: Extend the shared process-instance filter model instead of teaching command code version-specific request fields

- **Decision**: Add explicit request-capable fields to the shared `ProcessInstanceFilter` model for parent presence and incident presence, then map those fields through the facade into versioned services.
- **Rationale**: `cmd/get_processinstance.go` currently translates CLI selectors into a shared filter and then relies on `applyPISearchResultFilters(...)` for extra local narrowing. Keeping the new semantics in the shared model preserves the existing command-to-facade-to-service flow and avoids version branching in command code.
- **Alternatives considered**:
  - Build ad hoc request-only structs inside `cmd/get_processinstance.go`: rejected because it would duplicate version-specific search logic in the command layer.
  - Continue inferring roots/children/incidents only after fetch: rejected because it preserves the overfetching and inaccurate paging behavior reported in issue `#116`.

## Decision 2: Represent roots/children semantics as parent-presence, not overloaded parent-key equality

- **Decision**: Model `--roots-only` and `--children-only` as a tri-state parent-presence filter separate from the existing explicit `ParentKey=<key>` equality filter.
- **Rationale**: The current shared filter only has `ParentKey`, which means "exact parent key match." Roots/children semantics instead depend on whether the parent key exists at all, so reusing `ParentKey` would blur two distinct meanings and make request generation ambiguous.
- **Alternatives considered**:
  - Treat roots as `ParentKey=""` and children as `ParentKey!= ""`: rejected because the shared filter type cannot currently express presence/absence rules distinctly from equality.
  - Keep roots/children purely CLI-local and leave the shared model unchanged: rejected because `v88` and `v89` generated search filters support existence-based predicates that should be used when available.

## Decision 3: Represent incidents semantics as explicit boolean presence on the shared filter

- **Decision**: Add an explicit incident-presence field to the shared filter so `--incidents-only` and `--no-incidents-only` can flow directly into the request builders on supported versions.
- **Rationale**: The public and domain process-instance models already expose `Incident bool` on returned items, and the `v88`/`v89` generated Camunda search filters expose `hasIncident`. A shared filter field keeps the CLI semantics aligned with the generated request surface.
- **Alternatives considered**:
  - Leave incident filtering local everywhere: rejected because the generated `v88` and `v89` search APIs already support the same boolean filter server-side.
  - Reuse state filtering to approximate incidents: rejected because incident presence is not equivalent to process instance state.

## Decision 4: Keep `v8.7` on client-side fallback for these four semantics

- **Decision**: Preserve local post-fetch filtering for roots/children/incidents/no-incidents on `v8.7`.
- **Rationale**: `internal/services/processinstance/v87/searchProcessInstancesRequest(...)` currently maps only explicit equality-style fields such as process ID, version, state, and `ParentKey`. The Operate request shape in current repo code does not provide a reliable parent-key existence predicate or incident-presence predicate that matches the CLI flags.
- **Alternatives considered**:
  - Approximate roots/children by mixing broad search plus partial request fields on `v8.7`: rejected because it would still overfetch and would not faithfully represent the CLI flag meaning.
  - Declare the flags unsupported on `v8.7`: rejected because the issue and clarified spec prefer preserving current semantics through client-side fallback where request-side support is absent.

## Decision 5: Push down all four semantics on `v8.8` and `v8.9`

- **Decision**: Implement request-side filtering for roots-only, children-only, incidents-only, and no-incidents-only in both `v88` and `v89`.
- **Rationale**: The generated Camunda client surfaces for `v88` and `v89` expose `hasIncident` and `parentProcessInstanceKey` filter properties, including existence-style advanced filters. Those versions already build JSON request bodies directly in the versioned services, so adding the missing fields is repository-native.
- **Alternatives considered**:
  - Push down only incidents and leave roots/children local: rejected because the generated parent-key filter properties support the same presence semantics needed for roots/children.
  - Push down only roots/children and leave incidents local: rejected because the generated `hasIncident` filter is already available on both versions.

## Decision 6: Keep `--orphan-children-only` client-side across all versions

- **Decision**: Do not add request-side support for `--orphan-children-only`.
- **Rationale**: The current behavior depends on fetching children and then checking each parent through `FilterProcessInstanceWithOrphanParent(...)`. That semantic requires verifying missing parents, not just filtering on one process-instance field in the initial search request.
- **Alternatives considered**:
  - Approximate orphan detection with parent presence alone: rejected because an orphan child still has a parent key; the missing parent is only discoverable through follow-up lookup logic.
  - Add a new multi-query server-side orphan detection flow in this feature: rejected because it would broaden scope beyond the issue’s request-side pushdown goal and would not produce a single authoritative search request.

## Decision 7: The audit of other `get` commands should be explicit, but the planning scan found no other qualifying late-filter seams yet

- **Decision**: Keep an explicit audit task for other `get` commands, with a bounded no-addition rationale unless implementation finds another server-capable post-fetch filter seam.
- **Rationale**: The issue asks for an audit broader than one example, but the current planning scan found the concrete late local filters in `cmd/get_processinstance.go` and did not identify the same pattern in other `get` command families. Recording the audit boundary keeps the task honest without inventing speculative scope.
- **Alternatives considered**:
  - Treat the audit as automatically satisfied by process-instance work alone: rejected because the issue explicitly broadens the scope.
  - Expand scope preemptively to unrelated `get` commands without evidence of the same pattern: rejected because it risks needless churn and parallel structures.

## Decision 8: Use command paging regressions as the main user-visible proof

- **Decision**: Prove the behavioral improvement primarily through command-level paging regressions and service-level request-capture tests.
- **Rationale**: The issue’s observed pain is not just request construction; it is that users see misleading "Fetched 1000..." prompts after local filtering. The command paging seam is where that visible correctness improvement shows up.
- **Alternatives considered**:
  - Prove only service request bodies: rejected because that would miss the user-visible paging accuracy goal.
  - Prove only command output: rejected because that would not guarantee the new request-side predicates are actually being sent on supported versions.

## Confirmed Current Seams

- [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go) translates shared search filters in `populatePISearchFilterOpts()` and then applies `roots-only`, `children-only`, `orphan-children-only`, `incidents-only`, and `no-incidents-only` afterward in `applyPISearchResultFilters(...)`.
- [`c8volt/process/model.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go) and [`internal/domain/processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go) currently expose only `ParentKey` and no request-side boolean fields for parent presence or incident presence.
- [`internal/services/processinstance/v88/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go) and [`internal/services/processinstance/v89/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go) already build explicit Camunda search request bodies and can be extended to encode additional predicates.
- [`internal/services/processinstance/v87/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go) still builds an Operate search request with only equality-style fields and therefore remains the fallback version for these semantics.

## Generated-Client Capability Notes

- `internal/clients/camunda/v88/camunda/client.gen.go` exposes `HasIncident` and `ParentProcessInstanceKey` on the process-instance search filter shape, and the filter-property types support `"$exists"` semantics.
- `internal/clients/camunda/v89/camunda/client.gen.go` exposes the same `HasIncident` and `ParentProcessInstanceKey` search filter fields, and `internal/services/processinstance/v89/service_test.go` already verifies request-body serialization for end-date existence with `"$exists":true`.
- The current `v88` and `v89` service tests already capture request JSON bodies, making them the best seams for new predicate assertions.

## Regression Anchors

- [`cmd/get_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go) and [`cmd/cmd_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go) are the natural command-level anchors for paging and request-capture behavior.
- [`c8volt/process/client_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go) should cover the new shared filter field mapping into the domain layer.
- [`internal/services/processinstance/v88/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go) and [`internal/services/processinstance/v89/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go) should prove request-side predicate encoding for supported versions.
- [`internal/services/processinstance/v87/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go) should prove that unsupported pushdown is intentionally absent while final results still honor the CLI flags through fallback behavior.

## Setup Inventory: Late-Filtering And Version Capability Seams

- `cmd/get_processinstance.go` keeps the request-side search filter narrow to equality and date fields in `populatePISearchFilterOpts()` and only applies `--roots-only`, `--children-only`, `--orphan-children-only`, `--incidents-only`, and `--no-incidents-only` afterward in `applyPISearchResultFilters(...)`.
- `cmd/get_processinstance.go` runs paging through `searchProcessInstancesWithPaging(...)`, which means any filter left client-side can distort the page-level counts and continuation prompt before `applyPISearchResultFilters(...)` trims the visible items.
- `internal/services/processinstance/v87/service.go` builds the Operate request in `searchProcessInstancesRequest(...)` with tenant, explicit selectors, state, and `ParentKey` equality only; it has no request-side seam for parent presence/absence or incident presence/absence.
- `internal/services/processinstance/v88/service.go` builds a typed `camundav88.ProcessInstanceFilter` in `SearchForProcessInstancesPage(...)`, so `HasIncident` and advanced `ParentProcessInstanceKey` encoding can be added there without changing the command surface.
- `internal/services/processinstance/v89/service.go` builds the JSON search body in `SearchForProcessInstancesPage(...)` through the local `processInstanceFilter` wrapper, which already supports advanced filter serialization such as `"$exists"` and is the right place for the same pushdown semantics as `v8.8`.

## Setup Inventory: Shared Filter-Model Seams

- `c8volt/process/model.go` and `internal/domain/processinstance.go` both define `ProcessInstanceFilter` with only `ParentKey` for parent semantics; there is no shared request-capable field for parent presence/absence or incident presence/absence yet.
- `c8volt/process/convert.go` maps the public filter to the domain filter in `toDomainProcessInstanceFilter(...)`, so any new shared filter field must be added symmetrically on both structs and carried through this conversion seam.
- `c8volt/process/client_test.go` already asserts canonical filter mapping in `TestClient_SearchProcessInstances_MapsDateBoundsToDomainFilter`, making it the correct regression anchor for new shared `HasParent` and `HasIncident` mapping coverage.

## Setup Inventory: Existing Request-Capture And Paging Regression Anchors

- `internal/services/processinstance/v88/service_test.go` already captures the serialized request body in `TestService_SearchForProcessInstances` and page metadata in `TestService_SearchForProcessInstancesPage_UsesNativePageMetadata`, so parent/incident pushdown assertions can extend those existing seams.
- `internal/services/processinstance/v89/service_test.go` already captures raw JSON bodies in `TestService_SearchAndLookup` and asserts advanced filter serialization such as `"$exists":true`, making it the strongest existing anchor for parent-presence pushdown on `v8.9`.
- `internal/services/processinstance/v87/service_test.go` already covers `SearchForProcessInstances` request construction plus fallback overflow behavior in `TestService_SearchForProcessInstancesPage_FallbackOverflowDetection`, so unsupported-predicate omission belongs there rather than in command-local tests.
- `cmd/cmd_processinstance_test.go` provides reusable request-decoding helpers such as `decodeCapturedPISearchFilter(...)` and `decodeCapturedTopLevelPISearchPages(...)`, which are the shared test seams for command paging regressions.
- `cmd/get_processinstance_test.go` already exercises tenant-scoped search requests, date-filter serialization, and `populatePISearchFilterOpts()`, so supported-version paging and fallback regressions should extend that file instead of creating parallel command fixtures.
