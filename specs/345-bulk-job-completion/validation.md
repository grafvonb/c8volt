# Bulk Job Completion Validation

## Current scope and status

2026-10-20 user correction: real-environment tests are prohibited during implementation. The original SC-006/T009 live-recovery gate is superseded by isolated three-listener/control acceptance. T009 is complete after the focused tests and artifact checks recorded below; no environment designation is required.

This validation proves CLI/adapter behavior using test-owned fakes, not real-engine process termination. No real-environment tests, deployment, mutation or cleanup are part of the corrected completion gate.

## Historical evidence from iterations 9–11

The previous planning error caused iteration 9 to inspect local Kubernetes context `kind-c8.10`, namespace `camunda`, and perform a read-only `get cluster version --with-brokers --json` through `c810local`. The recorded response was gateway/broker 8.10.4. This contact was not needed for implementation validation and must not be repeated. The logs report no fixture deployment, start, cancellation, job completion or deletion. Iterations 10–11 made no further environment contact and stopped waiting for designation; those requests are now obsolete.

Two BPMN examples and the old live guide were prepared and statically checked. The examples remain as static references with an explicit do-not-deploy note; the live commands have been removed from the current quickstart. No created environment resources require cleanup. Historical progress entries are retained unchanged.

## Delivery checks

| Check | Iteration 9 evidence |
| --- | --- |
| Build | `go build -o /tmp/c8volt-345 .` passed. |
| Test discovery | Required service, facade and command patterns matched existing tests; lists saved under `/tmp/c8volt-345-iteration9-*-tests.txt`. |
| Full race suite | Default `make test` failed at the command package's ten-minute timeout (603.928s), while `TestUpdateProcessInstanceCommand_MultipleRepeatedKeysApplyOneVarsPayloadToEachUniqueKey` was running. No assertion failure or race report preceded it. The adjusted full `GORACE=atexit_sleep_ms=0 make test` passed, exit 0; the command package completed in 150.346s. Race detection remained enabled. Logs: `/tmp/c8volt-345-iteration9-test.log` and `/tmp/c8volt-345-iteration9-test-no-exit-delay.log`. |
| Vet | `make vet` passed, exit 0 (`go vet ./...`). |
| Documentation | `make docs-content` passed; generated CLI pages were unchanged. Only the generated build provenance in `docs/index.md` changed. |
| Help and README | Reviewed bulk selection, explicit collections/stdin, retries zero, tenant scope, unique limit, listener consequence, partial outcomes, no replay and process-termination limits against contracts. No behavior correction was needed. |
| Capabilities | `update job` advertises full contract/automation support, human and JSON output, and optional repeated `key` of type `stringArray`; no keys-only support. |
| Static artifact checks | Bash syntax, XML checks and `git diff --check` passed. No Go source was changed. |

T007/T008 evidence remains in the prior coordinated commits and progress records:
the HTTP fixture covers v88/v89/v810 with 1,000 selected jobs, 12 pages, 500
one-shot submissions and peak concurrency four; terminal acceptance covers 51
scenarios. These are automated adapter/CLI checks, not live-engine recovery runs.

## Historical baseline and CI

- Branch: `345-bulk-job-completion`.
- Iteration starting HEAD: `6ca82f4a47e502b097d0109b321a9002867e6cbd`.
- Local `main`, peeled `v4.3.4^{commit}`, and `git merge-base main HEAD` all resolve
  to `019e4c0bd865b9cf675422b4d8c97c489c36f694`.
- The annotated tag object's own SHA differs from its peeled commit; the pinned
  baseline is the peeled commit. No ref was moved.
- Initial `gh run list` failed with HTTP 401 using an environment token. A
  read-only retry using the CLI's stored authentication succeeded:

  ```text
  env -u GH_TOKEN -u GITHUB_TOKEN gh run list \
    --branch 345-bulk-job-completion --limit 5 \
    --json databaseId,headSha,status,conclusion,url,workflowName,event
  ```

  It returned `[]`, exit 0: no feature-branch workflow runs were available.
  There is no green CI run to cite. `.github/workflows/go.yml` excludes `main`
  from both push and pull-request branch filters; local checks are not
  PR-triggered CI evidence.
- No merge, push, release, publication, or commit occurred during iterations 9–11.

## Corrected isolated completion evidence

The following checks passed on 2026-10-20 during manual completion of T009 after the stopped loop:

| Check | Result |
| --- | --- |
| Revised SC-006 | `go test ./cmd -run '^TestBulkCompletionAcceptanceListenerHierarchy$' -race -count=1 -v` passed: nine scenarios across adapters 8.8/8.9/8.10, each covering preview, execution and no-wait. Three matching canceling listeners retained distinct process keys; the control was excluded. Preview sent zero mutations; execution submitted each selected key once and reported accepted submission with observation skipped, never inferred completion. Log: `/tmp/c8volt-345-isolated-listener.log`. |
| Shared fixture regression | `go test ./cmd -run '^TestBulkCompletion(Acceptance|Terminal)' -race -count=1 -v` passed, command package 17.096s. Includes the new listener scenario, existing scale/paging/reconciliation/interruption/output cases, and terminal acceptance. Log: `/tmp/c8volt-345-final-isolated-regression.log`. |
| Static checks | All 17 relative Markdown links in current feature docs resolved; all three quickstart shell blocks passed `bash -n` without execution. Both BPMN files parsed as XML; parent/child references, three parallel instances and the native user-task canceling listener matched. All 17 functional requirements and seven success criteria remain present. `gofmt -l` was empty for both touched test files; `git diff --check` passed. |

Both test runs cleared `GOFLAGS`, disabled Go environment-file overrides, dependency downloads and toolchain downloads (`GOENV=off`, `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`), removed inherited `C8VOLT_*`, helper-process and proxy variables, and set `GORACE=atexit_sleep_ms=0`. Each command subprocess used a temporary configuration pointing to its test-owned fake server. The first sandbox attempt could not access the Go build cache and did not run tests; the rerun with local cache/loopback access passed.

No real environment was contacted during this corrected work unit. Prior passing full race-suite, vet and generated-documentation evidence remains applicable because production behavior is unchanged; this unit changes instructions, feature documentation and isolated tests only. The generated `docs/index.md` provenance from iteration 9 is retained.

All nine implementation tasks are complete under the user-corrected scope. Real-engine listener execution and eventual process termination remain unverified by these tests; no live pass or external CI result is claimed. Historical progress is preserved, and the completion records and new checks belong to one final work-unit commit.

## Follow-up: bulk failure logging

The failure-path review found that the service correctly returned per-job evidence and joined errors, and the facade preserved classification, but the human renderer printed the joined transport text directly to stderr before the summary. The existing output tests did not check logger routing or forbid HTTP details on stderr.

The corrected command renders the partial result first, then emits one compact ERROR through the configured command logger. The formatter uses bounded outcome reasons and HTTP status labels, groups repeated causes and limits distinct reasons. It never formats raw error or diagnostic text. Quiet retains the error; verbose retains per-job outcome facts. JSON still uses the original error and partial payload in one envelope, and the original error reaches exit-code resolution. HTTP diagnostics continue through the existing redacted DEBUG transport.

- Before the fix, the new isolated regression failed on the direct stderr print containing the fake URL and response body (`/tmp/c8volt-output-red.log`).
- `TestBulkCompletionDiagnosticLogging` covers 72 scenarios: three adapters, plain/structured log formats, normal/verbose/quiet modes, debug flag/configuration and quiet precedence, JSON partial failures, preparation failure and exit-code suppression. Real terminal subprocesses verify inherited/configured stderr destinations; both output streams are checked. Backend body markers remain absent from logs, including DEBUG, while JSON preserves existing evidence.
- `TestBulkCompletionFailureSummaryBounds` checks mixed-reason grouping, bounded summaries, interruptions and exclusion of unknown/untrusted diagnostic strings. Terminal abort assertions now require one configured ERROR after the unchanged plain prompt.
- The affected command suite passed with race detection: `go test ./cmd -run 'Test.*(BulkCompletion|UpdateJob|CommandContract|CommandCapability|AllCommandHelp)' -race -count=1` (23.630s). Log: `/tmp/c8volt-output-regressions.log`.
- `make docs-content` passed after updating source help and README (`/tmp/c8volt-output-docs.log`). Go formatting and `git diff --check` passed.

All tests used isolated fakes with the same sanitized, download-disabled environment described above. No real environment was contacted. That correction changed output only; the separately identified 8.10 `leaseToken: null` serialization issue was addressed in the subsequent adapter correction below.

## Follow-up: v8.10 optional completion fields

The pinned generated completion request serializes unset optional `leaseToken` and `businessId` pointers as null. An isolated strict server reproduced rejection of this payload for both scalar and bulk completion before the fix (`/tmp/c8volt-lease-red.log`).

Both paths now use `internal/services/job/v810/completion_request.go`: it shadows only those two fields with `omitempty`, serializes the body and calls the generated custom-body endpoint. Generated client/provenance files, response parsing, error classification, scalar retry policy and bulk one-shot behavior are unchanged. This adapter correction does not refresh the upstream API baseline. Explicit values (including empty strings), result data and nulls inside variables are preserved; required variables still default to an empty object.

Validation used only isolated fakes and the sanitized environment described above:

- The focused completion suite passed with race detection (1.758s; `/tmp/c8volt-lease-focused.log`). New HTTP tests cover both entry points, nil/empty/nested-null variables, backend 400/401/500 failures and malformed responses. Encoding failures retain their errors and dispatch no HTTP requests. Explicit optional values and result data survive serialization. Backend rejections requiring a lease remain failures, with status/diagnostic evidence retained and no token-removal fallback.
- All job services and the public job facade passed: `go test ./internal/services/job/... ./c8volt/job -race -count=1` (`/tmp/c8volt-lease-services.log`).
- Affected CLI regressions passed with race detection (24.344s): `go test ./cmd -run 'Test.*(BulkCompletion|UpdateJob|CommandContract|CommandCapability|AllCommandHelp)' -race -count=1` (`/tmp/c8volt-lease-cli.log`). The shared acceptance server now rejects unexpected completion fields rather than accepting the defective null payload.
- Go formatting and `git diff --check` passed. The v8.10 generated client still matches its recorded provenance checksum.

No real environment was contacted or mutated. No assertion of successful live process cancellation follows from these tests.
