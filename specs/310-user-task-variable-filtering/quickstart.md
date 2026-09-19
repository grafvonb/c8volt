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
