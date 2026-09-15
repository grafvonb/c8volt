# Quickstart Validation: User-Task Variables

This guide validates the implemented feature. The planning phase does not implement flags or run these runtime checks. See [CLI contract](contracts/cli.md), [service contract](contracts/facade-service.md), and [data model](data-model.md) for precise outcomes.

## Prerequisites

- Work from the repository root on `codex/309-user-task-variables` after implementation.
- Use the Go/toolchain versions pinned in `go.mod` and the existing test environment.
- For optional live checks, use an existing valid c8volt configuration and authorized native user tasks on Camunda 8.8, 8.9, or 8.10. Have tasks with known inherited/local variables, one without variables, and enough matching tasks to exercise limits. This feature does not create or mutate fixtures in a live environment.
- Use HTTP fixtures for sparse/capped pages, backend truncation, authorization failures, and request counts; live environments need not reproduce these conditions.

## Focused automated checks

Run checks relevant to the changed slice. Proposed test files should retain the following names/prefixes so these filters execute the new coverage. Confirm the output includes the intended tests rather than a no-tests-to-run result.

```sh
go test ./internal/services/usertask/... -run 'Test.*(Variable|Native|Resolve)' -count=1
go test ./c8volt/task -run 'Test.*(Variable|Enrich|Convert)' -count=1
go test ./cmd -run 'Test(GetUserTask|UserTasksView)' -count=1
go test ./cmd -run 'Test.*ProcessInstance.*(Variable|Vars)' -count=1
```

Expected: all targeted tests pass. New cases must prove:

- Native effective-variable route, `truncateValues=false`, name sort, offset-only requests, and all supported versions.
- All variable pages, sparse/exact/capped behavior (including continuation evidence after reaching a retained lower bound), raw body values and truncation flags, malformed response rejection, duplicate consistency, cancellation, and overflow.
- Input order, task/variable association, local effective scopes retained, empty arrays, options/error conversion, and no Tasklist fallback.
- Every key input form, filtered search, mid-page task limit, no enrichment beyond selected results, and zero requests in excluded modes.
- Exact stdout/stderr expectations, JSON decoding followed by EOF, quiet combinations, Unicode/value limits, later errors, and writer failures.

Run real-terminal coverage explicitly after command integration:

```sh
go test ./cmd -run 'TestGetUserTaskPagingTerminal' -count=1
```

Extend the existing `testx.NewCmdTerminalRunner` cases to enable variables. Expect the unchanged prompt on configured/inherited stderr, clean result stdout, yes/no/EOF behavior, redirected stdout compatibility, no prompt for empty completion or unattended modes, and no variable retrieval for a declined next page or effective keys-only output. Pipe-only stdin does not prove terminal behavior.

## Build and inspect existing tasks

```sh
go build -o /tmp/c8volt-309 .
export USER_TASK_KEY=2251799815391233
export PROCESS_INSTANCE_KEY=2251799813711967
```

Replace the sample keys with existing authorized fixture keys. Add the normal global `--config` argument when needed.

The four issue workflows:

```sh
/tmp/c8volt-309 get ut --key "$USER_TASK_KEY" --with-vars
/tmp/c8volt-309 get ut --assignee alice --limit 10 --with-vars
/tmp/c8volt-309 get ut --pi-key "$PROCESS_INSTANCE_KEY" --with-vars --var-value-limit 120
/tmp/c8volt-309 --json get ut --key "$USER_TASK_KEY" --with-vars
```

Expected: task rows retain their ordinary fields, each variable name appears once using the backend-selected effective value, values follow existing tree formatting, and JSON contains one enriched envelope. A human limit does not shorten JSON values. Backend-incomplete values remain explicitly identified even at limit zero.

Exercise stdin and combined keys:

```sh
printf '%s\n' "$USER_TASK_KEY" | /tmp/c8volt-309 get ut --with-vars
printf '%s\n' "$USER_TASK_KEY" | /tmp/c8volt-309 get uts --with-vars -
/tmp/c8volt-309 get user-tasks --key "$USER_TASK_KEY,$USER_TASK_KEY" --with-vars
```

Expected: existing input merging/deduplication behavior and one enriched entry per selected unique task.

## Output and flag checks

```sh
/tmp/c8volt-309 --keys-only get ut --key "$USER_TASK_KEY" --with-vars
/tmp/c8volt-309 get ut --assignee alice --total --with-vars
/tmp/c8volt-309 --quiet --json get ut --key "$USER_TASK_KEY" --with-vars --var-value-limit 3
/tmp/c8volt-309 --json --keys-only get ut --key "$USER_TASK_KEY" --with-vars
/tmp/c8volt-309 get ut --key "$USER_TASK_KEY" --var-value-limit 0
/tmp/c8volt-309 get ut --key "$USER_TASK_KEY" --with-vars --var-value-limit -1
```

Expected in order: pure key output without variable calls; numeric count without variable calls; full received JSON values despite the small human limit; enriched JSON because JSON wins; invalid dependency error before requests; invalid value error before requests. Assert no-request conditions with instrumented HTTP fixtures, not by inspecting live stdout alone.

Use a known unmatched filter in fixtures to assert human `found: 0` plus newline, successful empty JSON payload `{"total":0,"items":[]}`, zero-byte empty keys output, and no paging question or variable call. Separately use a returned task without variables to assert `variables: []` and no human variable subtree. Use quiet/automation/auto-confirm variants supported by the base command.

For an interactive live search, run with terminal stdin and a small task batch size:

```sh
/tmp/c8volt-309 get ut --batch-size 1 --with-vars
```

Declining the continuation prompt must stop after the displayed selected page. Request-count proof belongs to terminal HTTP-fixture tests. No mutation confirmation or mutation request is expected anywhere.

## Documentation and integrated validation

Format touched Go files with `gofmt`, update command metadata/help and README, then run:

```sh
make docs-content
git diff --check
make test
```

`make test` runs `go test ./... -race -count=1`. Use it for the integrated shared-interface/shared-formatter change; do not repeat it merely for a commit or documentation-only edit. Review generated docs to verify both flags and all issue examples. Record actual checks and any environment limitations. This plan adds no new dry-run/no-wait behavior and requires no live mutations.
