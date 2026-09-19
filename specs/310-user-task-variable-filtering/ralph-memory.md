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
- `git diff --check`

## Do Not Repeat

- Do not broaden the existing PI grammar, hand-edit generated clients, add client-side variable matching, or introduce new paging/filter loops.

## Current Handoff
- Continue US1 with T013 and T014: add command execution/request-construction coverage first, then register and parse the three task-owned flags while preserving key conflicts and request counts.
