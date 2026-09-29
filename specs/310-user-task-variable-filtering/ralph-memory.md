# Ralph Memory

Feature: 310-user-task-variable-filtering
Started: 2026-09-19T10:56:01Z

## Codebase Patterns

- Keep task CLI wiring in `cmd`, public request aliases/conversion in `c8volt/task`, shared query state in `internal/domain`, and generated request mapping in each supported versioned user-task adapter.
- Reuse the process-instance variable-filter grammar and domain validation without importing process-instance service adapters into user-task services.
- Public task filter types alias the existing process facade records; task conversion allocates the domain clause slice and copies `Exists` with `toolx.CopyPtr` so caller mutation cannot cross the facade boundary.
- The shared parser orchestration accepts explicit exists/value/like slices; PI and task wrappers retain independent package globals while reusing every lower-level grammar helper and diagnostic.
- Each supported user-task version owns a small local mapper mirroring its process-instance adapter: validate the shared domain set, decode membership arrays to `[]string`, and assign the generated slice only to `UserTaskFilter.LocalVariables`.
- The v8.8, v8.9, and v8.10 generated local-variable unions have matching shapes; their adapter-local request tests cover ordered duplicates, ordinary/tenant selectors, null text distinctions, false existence, escaped wildcards, empty omission, and pre-HTTP invalid input.
- User-task variable flags are repeatable `StringArray` inputs. Validation parses them before request dispatch, request construction parses again without cached state, and key conflicts use Cobra's `Changed` state so explicit and stdin keys reject all three flags.
- Command request fixtures accept only `POST /v2/user-tasks/search`; successful filtered execution therefore proves no task-variable or name-discovery request was added while asserting exact `filter.localVariables` placement.
- Version-neutral traversal retains the complete ordinary and variable-filter query on initial, cursor, offset, sparse, bounded, stopped, total, and failure paths. Total mode clears only `Limit`; the search-only fake API panics on any effective-variable read.
- Filtered output coverage uses the existing subprocess runner to capture stdout/stderr separately and a search-only fixture to prove every human, machine, total, quiet, auto-confirm, and automation mode retains the native local predicate with exactly one request.
- Real-terminal filtered paging uses `testx.NewCmdTerminalRunner`; representative configured/inherited stderr, default-no/continue/decline/EOF, sparse/empty, JSON, automation, and auto-confirm cases assert every native page retains the exact local predicate while stdout remains prompt-free.
- Filtered backend-failure coverage captures every attempted native search request: first-page failures emit no result, later keys output stays plainly partial without `found:`, collected JSON emits exactly one failed envelope, totals emit no number, and total/limit/output conflicts fail before HTTP.
- US2 integration needs no production changes: the command forwards one complete request to facade traversal, the service reuses the normalized query across cursor, offset, sparse, bounded, stopped, total, and failure paths, and total mode clears only `Limit`.
- The shared variable-display fixture now captures ordered native `localVariables` arrays separately from full search bodies, counts searches and effective-variable pages, and exposes backend-preselected local/inherited/shadowing examples; tests must choose returned task keys rather than evaluate predicates in the mock.
- Filtered display selection remains downstream of native search and service trimming: filter-only execution performs zero effective-variable reads, and display reads occur exactly once per rendered task across ordinary, limited, sparse, incremental, and terminal-stopped paths.
- Filtered output uses the same enrichment gate as unfiltered output: effective keys-only, total, and empty selections make zero variable reads; quiet human still performs requested enrichment; quiet JSON and JSON-over-keys preserve full received values while human limits remain rune-aware and retain API/CLI truncation labels.
- US3 integration requires no production changes: service-selected and limit-trimmed tasks reach the command before the shared enrichment gate, incremental pages enrich once before rendering, and collected results enrich once at final rendering; fixtures expose only read/search routes, so the passing display suites also exclude mutation requests.
- User-task source help and README now use the same six filter workflows and explicitly document shared operators, `$notin`, serialized string/array encoding, native local-only scope, backend-defined negative/existence behavior, and display enrichment as an independent option.
- `make docs-content` regenerates only `docs/cli/c8volt_get_user-task.md` and `docs/index.md` for this source-help change; the index build metadata refresh is expected alongside the matching flag, grammar, workflow, scope, and display text.
- The complete quickstart-focused ordinary and race matrix passes across command/parser, task facade, user-task service/adapters, PI variable regressions, terminal paging, and command metadata; targeted coverage is proportionate because the feature adds no concurrency, dependency, generated-client, or broader shared-runtime change.
- Final review against the specification and both contracts confirms the diff is limited to additive search input/mapping, task-owned CLI wiring, native local-variable adapters, tests, and aligned documentation; result views, production traversal/enrichment, generated clients, and mutation paths are unchanged.

## Decisions

- Filtering is native and local-task scoped for Camunda 8.8, 8.9, and 8.10; Camunda 8.7 retains its explicit unsupported behavior.
- Existing task traversal, total counting, output rendering, prompt routing, and optional effective-variable enrichment remain authoritative.

## Gotchas

- Preserve explicit false existence pointers and caller ownership when crossing facade/domain boundaries.
- Request fixtures prove native mapping, not live backend missing/null or parent-scope semantics.

## Reusable Commands

- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./c8volt/task -run 'Test.*(Search|VariableFilter)' -count=1`
- `go test -race ./c8volt/task -run 'Test.*(Search|VariableFilter)' -count=1`
- `go test -race ./cmd -run 'Test(UserTaskVariableFilterParser|ParsePIVariableFilters)' -count=1`
- `go test ./cmd -run 'Test.*(VariableFilter|PIVariable|UserTask.*Filter|GetUserTask.*Variable|NewGetUserTaskSearchRequest|RejectsInvalidInputBeforeReads)' -count=1`
- `go test ./cmd -run 'Test(GetUserTaskVariable|GetUserTaskCommand_FilteredDisplay)' -count=1`
- `go test ./internal/services/usertask/... -run 'Test.*(Search|VariableFilter|Native)' -count=1`
- `git diff --check`

## Do Not Repeat

- Do not broaden the existing PI grammar, hand-edit generated clients, add client-side variable matching, or introduce new paging/filter loops.

## Current Handoff
- Feature complete; no handoff required.
