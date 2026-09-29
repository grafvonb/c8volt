# Quickstart Validation: User-Task Variable Filtering

## Prerequisites

Work from the repository root on `codex/310-user-task-variable-filtering` after implementation. Use the pinned Go toolchain. Deterministic tests use existing HTTP fixtures and terminal helpers; no live mutation or credentials are required.

For optional live checks, use an authorized Camunda 8.8, 8.9, or 8.10 configuration with existing user tasks and known local/parent variables. Never create or modify live tasks solely for these checks without separate authorization.

## Focused automated checks

```sh
go test ./cmd -run 'Test.*(VariableFilter|PIVariable|UserTask.*Filter|GetUserTask.*Variable)' -count=1
go test ./c8volt/task -run 'Test.*(Search|VariableFilter)' -count=1
go test ./internal/services/usertask/... -run 'Test.*(Search|VariableFilter|Native)' -count=1
go test ./internal/services/processinstance/... -run 'Test.*Variable' -count=1
go test ./cmd -run 'TestGetUserTaskPagingTerminal' -count=1
go test ./cmd -run 'TestCommandCapabilityForCommand_UserTaskReadContract' -count=1
```

Confirm filters match actual implemented test names; a no-tests-to-run result is not evidence. Tests should prove the [CLI contract](contracts/cli.md) and [facade/service contract](contracts/facade-service.md), especially:

- PI/task parser parity and global isolation; no new value conversion.
- Exact native mappings across versions, including false existence, null text, quoted string-null, and omitted empty filters.
- Unchanged predicates across pages/counts; no per-task reads attributable to filtering.
- Explicit and stdin key conflicts, malformed input, and direct facade validation before requests.
- Separate stdout/stderr; one JSON envelope followed by EOF; exact empty human output and zero-byte keys-only output.
- Real terminal stdin with inherited/configured stderr, redirected stdout, continue/decline/EOF, sparse pages, empty completion, and supported unattended modes.
- Existing effective display with limits/paging and no retrieval for excluded modes.

Run focused race checks for the changed command and service/facade paths after ordinary checks pass. Do not run the full suite merely for this planning phase; implementation uses the constitution's actual-impact rule.

## Build and exercise the six issue workflows

```sh
go build -o /tmp/c8volt-310 .
/tmp/c8volt-310 get ut --var 'status="approved"'
/tmp/c8volt-310 get ut --var-exists payload
/tmp/c8volt-310 get ut --var-like 'email=*@example.com'
/tmp/c8volt-310 get ut --assignee alice --var 'status="approved"' --limit 20
/tmp/c8volt-310 get ut --var 'status="approved"' --total
/tmp/c8volt-310 get ut --var 'status="approved"' --with-vars
```

Use the normal global `--config` option if needed. Expected: known local matches only, all ordinary predicates also satisfied, at most the requested limit, exact filtered total, and unchanged effective-variable display for the same selected tasks.

A parent-only positive match must not select a task; a locally shadowed value determines selection. Record observed backend absent/null/negative-existence behavior rather than inferring it from request fixtures.

## Output, validation, and interaction

```sh
/tmp/c8volt-310 --json get ut --var 'status="approved"'
/tmp/c8volt-310 --keys-only get ut --var 'status="approved"' --with-vars
/tmp/c8volt-310 --quiet --json get ut --var 'status="approved"' --with-vars
/tmp/c8volt-310 --automation get ut --var-exists payload --batch-size 1
/tmp/c8volt-310 get ut --var 'status.$contains=approved'
/tmp/c8volt-310 get ut --key 2251799815391233 --var-exists payload
printf '%s\n' '2251799815391233' | /tmp/c8volt-310 get ut --var-exists payload
```

Expected: existing JSON result; pure keys without display reads; quiet JSON with requested display; no automation paging prompt; and pre-request errors for the final three invocations. Validate request counts with HTTP fixtures, not live output alone.

Repeat output cases with a known nonmatching predicate and assert existing empty contracts. Compare ordinary searches and representative existing `get pi` variable filters before/after implementation.

For real interactive paging, use terminal stdin with a matching predicate and `--batch-size 1`; verify prompts stay on stderr when stdout is redirected. Automated terminal tests are the authoritative repeatable stream check.

## Documentation and final review

Format touched Go files. Update command help/examples/metadata and README, then run:

```sh
make docs-content
git diff --check
```

Review generated task help and grammar compatibility, confirm no generated client edits, no schema changes to task results, and no new filtering/discovery loops. Record actual checks and any live validation gaps. Planning-only changes need document/link/whitespace checks, not runtime tests or CLI regeneration.

## Implementation validation log

### Iteration 1 — setup baseline (2026-09-19)

- Confirmed the implementation branch is `codex/310-user-task-variable-filtering`.
- Reviewed the feature plan, specification, data model, research, contracts, quickstart, task list, repository guidance, constitution, and Ralph implementation rules. No conflicts were found.
- The prerequisite check selected `specs/310-user-task-variable-filtering` and reported the expected design and task artifacts.
- Scope remained documentation-only for T001; no runtime tests or CLI documentation generation were warranted under constitution principle III.
- `git diff --check` passed for the coordinated setup changes.

### Iteration 2 — foundational facade plumbing (2026-09-19)

- Added public task filter aliases/constants, the additive search/domain fields, and mechanical ordered conversion with copied existence pointers.
- `go test ./c8volt/task -run 'Test.*(Search|VariableFilter)' -count=1` passed.
- `go test ./c8volt/task -count=1` passed.
- `go test ./internal/domain -count=1` passed.
- `go test -race ./c8volt/task -run 'Test.*(Search|VariableFilter)' -count=1` passed.
- `git diff --check` passed before coordinated persistence; no live backend validation was attempted because this work unit only establishes facade/domain propagation.

### Iteration 3 — parser parity and isolation (2026-09-19)

- Extracted explicit-input orchestration while retaining the existing PI wrapper and all lower-level grammar helpers.
- Added task-wrapper parity coverage for all operators, aliases, repeated/grouped inputs, quoted commas, arrays, assignment characters, wildcard escapes, malformed inputs, exact diagnostics, and independent globals.
- `go test ./cmd -run 'Test(UserTaskVariableFilterParser|ParsePIVariableFilters)' -count=1` passed.
- `go test -race ./cmd -run 'Test(UserTaskVariableFilterParser|ParsePIVariableFilters)' -count=1` passed.
- `git diff --check` passed; no live backend validation was attempted because this work unit does not yet construct native task requests.

### Iteration 4 — v8.8 native local-variable mapping (2026-09-19)

- Added v8.8 request coverage for all six operators, ordered duplicate clauses, ordinary and tenant selectors, empty-filter omission, serialized null versus string-null, false existence, escaped wildcards, and pre-request rejection of invalid filters and malformed/non-string membership arrays.
- Added the adapter-local v8.8 mapper and attached its output only to `UserTaskFilter.LocalVariables`; generated clients and process-instance services were unchanged.
- `go test ./internal/services/usertask/v88 -count=1` passed.
- `go test -race ./internal/services/usertask/v88 -run 'TestService_SearchUserTasksPage_' -count=1` passed.
- `git diff --check` passed; fixture assertions prove exact native request construction, not live backend missing/null or parent-scope semantics.

### Iteration 5 — v8.9 native local-variable mapping (2026-09-19)

- Added v8.9 request coverage for all six operators, ordered duplicate clauses, ordinary and tenant selectors, empty-filter omission, serialized null versus string-null, false existence, escaped wildcards, and pre-request rejection of invalid filters and malformed/non-string membership arrays.
- Added the adapter-local v8.9 mapper and attached its output only to `UserTaskFilter.LocalVariables`; generated clients and process-instance services were unchanged.
- `go test ./internal/services/usertask/v89 -count=1` passed.
- `go test -race ./internal/services/usertask/v89 -run 'TestService_SearchUserTasksPage_' -count=1` passed.
- `git diff --check` passed; fixture assertions prove exact native request construction, not live backend missing/null or parent-scope semantics.

### Iteration 6 — v8.10 native local-variable mapping (2026-09-19)

- Added v8.10 request coverage for all six operators, ordered duplicate clauses, ordinary and tenant selectors, empty-filter omission, serialized null versus string-null, false existence, escaped wildcards, and pre-request rejection of invalid filters and malformed/non-string membership arrays.
- Added the adapter-local v8.10 mapper and attached its output only to `UserTaskFilter.LocalVariables`; generated clients and process-instance services were unchanged.
- `go test ./internal/services/usertask/v810 -count=1` passed.
- `go test -race ./internal/services/usertask/v810 -run 'TestService_SearchUserTasksPage_' -count=1` passed.
- `git diff --check` passed; fixture assertions prove exact native request construction, not live backend missing/null or parent-scope semantics.

### Iteration 7 — user-task command integration and US1 validation (2026-09-19)

- Registered `--var-exists`, `--var`, and `--var-like` as task-owned repeatable flags; validation and request construction both propagate parser errors without cached parsed state.
- Added execution coverage for every user-task alias across Camunda 8.8, 8.9, and 8.10, combined ordinary/tenant selectors, exact native `localVariables` placement and clause order, malformed and unknown clauses, and explicit/implicit stdin key conflicts.
- Added direct request-construction coverage and a filtered Camunda 8.7 unsupported regression that proves rejection occurs before transport use.
- `go test ./cmd -count=1` passed after flag registration.
- `go test ./cmd -run 'Test.*(VariableFilter|PIVariable|UserTask.*Filter|GetUserTask.*Variable|NewGetUserTaskSearchRequest|RejectsInvalidInputBeforeReads)' -count=1` passed.
- `go test ./c8volt/task -run 'Test.*(Search|VariableFilter)' -count=1` passed.
- `go test ./internal/services/usertask/... -run 'Test.*(Search|VariableFilter|Native)' -count=1` passed for the version-neutral service and v8.7–v8.10 adapters.
- `go test ./internal/services/processinstance/... -run 'Test.*Variable' -count=1` passed; packages without matching tests reported `[no tests to run]` and were not counted as direct evidence.
- Focused race variants of the command, task facade, and user-task service/adapter commands above passed.
- Request fixtures prove exact native request construction and absence of added endpoints. No live backend scope, missing/null, parent-only, or shadowing validation was attempted.
- `git diff --check` and the touched-command declaration inventory passed before coordinated persistence.

### Iteration 12 — US2 integration and validation (2026-09-19)

- Reviewed `cmd/get_usertask_search.go` and `internal/services/usertask/search.go`; the complete query already flows unchanged through command/facade traversal and every service page, while total mode intentionally clears only `Limit`. No production edit was required.
- `go test -v ./internal/services/usertask -run 'TestFilteredSearchUserTasks' -count=1` passed and visibly executed the filtered cursor, offset, sparse, limit, stop, exact/capped total, cancellation, malformed-metadata, and later-page failure cases.
- `go test -v ./cmd -run 'TestGetUserTask(Output_FilteredNonemptyModes|Output_FilteredEmptyModes|PagingTerminal|Error_FilteredSearchFailuresNeverClaimSuccess|Error_FilteredTotalConflictsFailBeforeRequests)$' -count=1` passed and visibly executed the intended filtered output/error suites plus real-terminal paging across Camunda 8.8, 8.9, and 8.10.
- `go test -race ./internal/services/usertask -run 'TestFilteredSearchUserTasks' -count=1` passed.
- `go test -race ./cmd -run 'TestGetUserTask(Output_FilteredNonemptyModes|Output_FilteredEmptyModes|Error_FilteredSearchFailuresNeverClaimSuccess|Error_FilteredTotalConflictsFailBeforeRequests)$' -count=1` passed.
- Failure cases remained errors with no empty-success envelope, false final summary, or numeric total. No live backend validation was attempted; request fixtures remain the evidence for native predicate propagation rather than server scope semantics.

### Iteration 14 — filtered display selection (2026-09-19)

- Added filtered command regressions proving identical selected task identities with and without `--with-vars`, zero filter-only effective-variable reads, within-page limit trimming before enrichment, sparse-page continuation with predicates retained, and terminal-decline enrichment only for accepted pages.
- `go test ./cmd -run 'TestGetUserTaskCommand_FilteredDisplay(PreservesSelection|StopsBeforeUnreadPages)$' -count=1` passed.
- `go test -race ./cmd -run 'Test(GetUserTaskCommand_(FilteredDisplayPreservesSelection|FilteredDisplayStopsBeforeUnreadPages|SearchVariablesEnrichOnlySelectedTasks)|FilteredUserTaskVariablesFixture)$' -count=1` passed.
- Request-fixture results prove selection/display request boundaries and native predicate propagation; no live backend scope validation was attempted.

### Iteration 16 — filtered display integration and validation (2026-09-19)

- Reviewed `cmd/get_usertask_vars.go` and `cmd/get_usertask_search.go`; service-selected and limit-trimmed tasks reach the command before the shared enrichment gate, so incremental and collected paths each enrich only eligible selected tasks once. No production edit was required.
- `go test -v ./cmd -run 'Test(GetUserTaskVariable|GetUserTaskCommand_(FilteredDisplay|SearchVariables))' -count=1` passed and visibly executed the filtered display selection/output suites plus existing user-task variable display regressions.
- `go test -race -v ./cmd -run 'Test(GetUserTaskVariable|FilteredUserTaskVariablesFixture|GetUserTaskCommand_(KeyedVari|VariableModeGates|SearchVariables|FilteredDisplay))' -count=1` passed, including keyed display, exclusions, limits, sparse and stopped paging, effective-variable pagination, truncation, JSON fidelity, and failure behavior.
- Filter-only execution and keys-only, total, and empty display exclusions performed zero effective-variable reads; eligible display read only selected tasks. The fixtures expose only task reads, task searches, and effective-variable searches, so successful runs also prove no mutation requests were issued.
- Native request fixtures, rather than a live backend, remain the evidence for local predicate propagation and request boundaries; no live scope or missing/null validation was attempted.

### Iteration 19 — final focused implementation validation (2026-09-19)

- Ran `gofmt` across every Go file changed from `develop`; it produced no diff.
- `go test ./cmd -run 'Test.*(VariableFilter|PIVariable|UserTask.*Filter|GetUserTask.*Variable)' -count=1` passed.
- `go test ./c8volt/task -run 'Test.*(Search|VariableFilter)' -count=1` passed.
- `go test ./internal/services/usertask/... -run 'Test.*(Search|VariableFilter|Native)' -count=1` passed across the version-neutral service and v8.7–v8.10 adapters.
- `go test ./internal/services/processinstance/... -run 'Test.*Variable' -count=1` passed across packages with matching PI variable regressions; traversal, waiter, and walker reported no matching tests and were not counted as direct evidence.
- `go test ./cmd -run 'TestGetUserTaskPagingTerminal' -count=1` and `go test ./cmd -run 'TestCommandCapabilityForCommand_UserTaskReadContract' -count=1` passed.
- The `-race` variants of all six focused commands above passed, covering changed parser/command/facade/service paths, PI regressions, terminal paging, and command metadata.
- `make test` was not run: targeted ordinary and race coverage passed, and the implementation adds no concurrency, dependency, generated-client, or unresolved shared-runtime risk requiring the full suite under constitution principle III.
- No live backend validation was attempted because no authorized fixture or credentials were provided; deterministic request fixtures remain the evidence for exact native mapping, while backend missing/null, parent-only, shadowing, and negative-existence semantics remain intentionally unclaimed.

### Iteration 20 — final contract and scope review (2026-09-19)

- Reviewed the complete `develop...HEAD` diff against `spec.md`, `contracts/cli.md`, and `contracts/facade-service.md`; the implementation and documented behavior satisfy the additive local-variable filtering, validation, paging/output, display-independence, supported-version, and compatibility contracts.
- `git diff --check develop...HEAD` passed.
- No files under `internal/clients/camunda/` changed. The only public model addition is the search-input `VariableFilters` field and aliases; task result records, command view files, and shared result envelopes are unchanged.
- `cmd/get_usertask_search.go`, `cmd/get_usertask_vars.go`, and `internal/services/usertask/search.go` are unchanged, confirming no new production paging, filtering, total, or enrichment loop. The versioned adapters only validate/map predicates into native `filter.localVariables` before the existing search request.
- No mutation command, mutation endpoint, or task lifecycle behavior changed. The shared process-instance parser change only parameterizes its existing orchestration with explicit slices; lower-level grammar helpers and PI behavior remain unchanged.
- Prior focused ordinary and race results remain valid because this iteration changed only feature evidence and completion records; constitution principle III does not require repeating runtime tests for this documentation-only review.
