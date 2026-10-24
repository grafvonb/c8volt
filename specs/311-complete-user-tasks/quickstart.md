# Isolated Validation Guide: User-Task Completion

**Issue**: #311 | **Contracts**: [CLI](contracts/cli.md), [facade/service](contracts/facade-service.md) | **Models**: [data-model.md](data-model.md)

This guide is for implementation verification after the planned code and tests exist. It is not evidence that those checks have run. Planning itself requires document review and whitespace/link checks only.

## Prerequisites and test isolation

- Use the repository Go 1.26 / go1.26.2 toolchain and installed dependencies.
- Every command test must pass an explicit temporary config pointing exclusively to its test-owned fake server. A default user profile or actual Camunda URL is never a fallback.
- Use generated-client mocks and `testx.NewIPv4Server`/test-owned HTTP servers. Command exit tests use `testx.RunCmdSubprocess`; prompts use `testx.NewCmdTerminalRunner` with real PTY stdin and independently captured stdout/stderr.
- Before each target, inspect the selected tests/target and prohibit `integration` or other live build tags, live helpers, smoke scripts, cluster discovery and external test profiles. Untagged `testx/integration_env_require_stub.go` skips environment-backed helpers; the tagged implementation must remain excluded.
- Remove inherited Go flags, application credentials/profile settings, helper-process markers and Make overrides. The following wrapper runs under a clean environment while retaining local toolchain lookup and home/cache access. Tests must still supply their explicit fake-only config.

From the repository root:

```sh
isolated() {
  env -i PATH="$PATH" HOME="$HOME" TMPDIR="${TMPDIR:-/tmp}" \
    GOENV=off GOFLAGS= GOWORK=off "$@"
}
```

Do not enable missing coverage by contacting real services. If a compiler/cache dependency is unavailable, report that limitation instead. Keep subprocess/terminal environments clean as their runners inherit the parent environment.

## Focused checks

Proposed new command tests use the prefix `TestUpdateUserTaskCompletion`; service/facade tests include `Complete` or `Completion`. Ensure the named tests actually exist before claiming coverage; a successful “no tests to run” is not validation.

```sh
isolated go test ./cmd -list 'TestUpdateUserTaskCompletion'
isolated go test ./cmd -run 'TestUpdateUserTaskCompletion' -count=1
isolated go test ./internal/services/usertask/... -run 'Complete|Completion' -count=1
isolated go test ./c8volt/task -run 'Complete|Completion' -count=1
```

Use narrower named cases while implementing each owner; broaden to the full relevant group after that behavior is complete. No executable CLI invocation in this guide targets a real backend. CLI examples in the contract describe scenarios that these tests execute with fakes.

## Acceptance matrix

| Group | Fixture and execution | Required evidence |
|---|---|---|
| Keys and aliases | Each alias; single/repeated/comma/stdin/combined keys; duplicates; invalid token late in input | Stable unique order, one logical operation per unique key; flags before stdin; zero calls for invalid input |
| Empty selection | Missing keys; empty/blank stdin; valid flags plus empty stdin | First two fail before requests; valid flag selection succeeds; no discovery/no-op fallback |
| Variables | Omitted, `{}`, nested typed values, equivalent file/inline data | Correct native body for every task; nullable omission vs empty object; no variable reads/diff/scope writes/polling |
| Invalid controls | Both sources, unreadable file, invalid/non-object/trailing JSON, invalid workers, complete+no-wait | Classified failure with zero completion requests; variable no-wait still supported |
| Preview | Human/JSON/keys/quiet and supported automation combinations | Zero native mutation and read calls; no prompt; preview never claims acceptance |
| Terminal | Real terminal stdin; accept, decline, default, EOF; inherited and configured stderr; redirected stdout | Exactly one whole-selection prompt on effective stderr; no logger prefix; no mutation on abort; stdout contains results only |
| Worker policy | Synchronization-controlled fakes; workers one/two; automatic; explicit bound plus no-worker-limit | Sequential input order at one; actual peak active calls respects selected bound; deterministic result order |
| Fail-fast | One rejecting task while another is already submitted; queued remainder | No new submission after observed failure; in-flight acceptance survives; queued tasks appear not_attempted; original error retained |
| Interruption | Cancellation before submit/during request/during retry delay, timeout, ambiguous attempt then rejection | Accurate submission uncertainty, preserved earlier status evidence and accepted outcomes, no false rejection/completion |
| Output | Success/partial/all failure/validation/preview in human/JSON/keys/quiet, quiet+JSON/keys, JSON precedence, automation | Exact result assertions, accepted keys only, zero-byte keys when none accepted, exactly one envelope then EOF, original exit class/code |
| Supported adapters | v88/v89/v810 mock/native fake responses, foreign-tenant explicit keys, unsupported factory input | Native endpoint and body mapping, no content required for 204, no tenant filter/assignment/fallback, generic unsupported handling |
| Compatibility | Existing variable planning/no-op/shared scopes/wait/no-wait and read/resolver tests | Existing schemas, acceptance-vs-confirmation and output behavior unchanged |

For timing-sensitive scheduling, use gates/channels and counters instead of brittle wall-clock sleeps. Fixtures should fail on any unexpected task/variable/read/assignment/poll request, so “no forbidden workflow” assertions cannot pass accidentally.

## Diagnostic regression matrix

Build actual command execution cases using the structural patterns in `cmd/update_job_batch_logging_test.go`, `cmd/root_api_diagnostics_test.go`, and `cmd/get_processdefinition_watch_timeout_test.go`.

1. Return repeated failures for multiple distinct keys containing distinctive HTTP method, URL, body and secret markers. Capture result stdout, configured logger destination, and stderr separately where appropriate.
2. At normal and verbose-only levels, assert one compact aggregate ERROR with counts/grouped reasons and no raw transport markers on either stream. Verbose may add safe per-key statuses. Verify true logger framing/severity, not just a substring. Exercise configured and inherited writers plus the active plain and structured formats.
3. Quiet keeps the compact ERROR and suppresses informational output. Quiet+JSON preserves the requested envelope. Confirm JSON error evidence is retained and that no second error renderer or human aggregate duplicates it.
4. Enable DEBUG separately through `--debug` and configured DEBUG with verbose off; also test existing quiet precedence. Assert actual completion exchanges and permitted retries, redaction, no diagnostic records on stdout, and result bytes unchanged from the corresponding non-debug execution.
5. Cover workers one and bounded concurrency. Use 429/RESOURCE_EXHAUSTED responses with controlled `Retry-After` and fake responses, plus interruption during backoff. Do not generate synthetic requests merely for measurement.
6. Prove the assertions detect regressions: temporarily replace the new aggregate logger call with a direct stderr print, then temporarily expose a raw joined error at normal level; the relevant tests must fail. Restore the implementation immediately and rerun only affected cases. Record this evidence in implementation progress, never commit the intentional violations.

Do not require response bodies to appear in DEBUG: assert the existing diagnostics contract and redaction, not an invented payload dump. Existing variable-mode debug substring assertions are insufficient for completion.

## Regressions and documentation

```sh
isolated go test ./cmd -run 'UpdateUserTask|GetUserTask|HasUserTasks|UserTaskHTTPFallback|APIDiagnostics' -count=1
isolated go test ./c8volt/task ./internal/services/usertask/... -count=1
isolated go test ./internal/services/httpc -run 'APIDiagnostics' -count=1
```

Ensure the existing `get pi --has-user-tasks` cases are selected; inspect actual test names and broaden the pattern only if needed. Run changed interface consumers as compilation requires. Keep variable-mode output assertions unchanged; completion has its own stdout result contract.

Format touched Go files with targeted `gofmt -w` using their explicit paths. After source help, parent metadata, examples and README are updated:

```sh
isolated make docs-content
isolated go test ./docsgen -count=1
git diff --check
```

`docs-content` generates references from command metadata; it does not run a user command against Camunda. Review generated changes for both modes, optional variables, explicit keys, acceptance-only semantics and no-wait rejection. Do not hand-edit generated CLI docs.

For the completed cross-package runtime change, run the repository race target once after targeted checks:

```sh
isolated make test
```

`make test` is untagged `go test ./... -race -count=1`. The broader check is justified by concurrent mutation scheduling and interface changes affecting three adapters and their consumers. Do not repeat it after documentation-only edits or merely to commit. Never use integration targets/tags, smoke tests, cluster tools, deployments or live resources in this workflow.

## Recording results

Record actual commands and results, covered matrix cases, unsupported terminal coverage, and relevant failure-path review. Distinguish implemented/passed cases from planned cases or skipped checks. Isolated fixtures verify the command/service contract; actual engine task-listener timing, assignment eligibility, and eventual task/process state remain unverified unless a separate target-specific live workflow is explicitly requested.
