# Isolated Validation Guide

This guide is for implementation validation after the feature exists. This planning turn ran document checks only. No command below designates, discovers, provisions or uses a live Camunda target. Product usage examples are in the [CLI contract](contracts/cli.md), not executable validation steps here.

## Prerequisites and isolation

Use the Go toolchain pinned in `go.mod`, repository dependencies, and test-owned generated-client doubles/HTTP servers. CLI fixtures must create their own temporary config, isolated HOME/config paths, owned endpoint and credentials; avoid user profiles, inherited credentials, live-test helpers or service URLs. Real terminal stdin fixtures may spawn a subprocess but must still connect only to an owned fake.

Before executing a test target, inspect its selected tests/helpers and ensure it remains isolated. Clear inherited Go flags so build tags cannot enable live suites. Never invoke integration targets, live tags, cluster tools or existing live smoke scripts. Tests absent before implementation must be added with the relevant behavior; a regex matching no tests is not validation success.

From the repository root, inspect the existing isolated suites and implementation-added test names:

```bash
rg -n '^func Test|^//go:build' internal/domain/business_id_test.go internal/services/job/completion internal/services/job/v810/completion_test.go internal/services/job/v810/completion_payload_test.go cmd/update_job_batch_logging_test.go cmd/update_processinstance_terminal_test.go
```

## Targeted checks

Start with the closest changed package. Extend existing tests and use `BusinessIDAssignment` in new direct-assignment test names so the following commands select real assertions. The commands specify no integration build tags; verify inherited flags and fixture isolation first.

```bash
env GOFLAGS= go test ./internal/domain -run 'TestValidateBusinessID' -count=1
env GOFLAGS= go test ./internal/services/processinstance/... -run 'BusinessIDAssignment' -count=1
env GOFLAGS= go test ./c8volt/process -run 'BusinessIDAssignment' -count=1
env GOFLAGS= go test ./internal/services/job/completion -run 'Test.*(Prepare|Execute|Reconcil|BusinessID)' -count=1
env GOFLAGS= go test ./internal/services/job/... -run 'Test.*(Completion|BusinessID|BusinessId|WorkerOutcome)' -count=1
env GOFLAGS= go test ./c8volt/job -run 'Test.*(Completion|BusinessID|BusinessId|WorkerOutcome)' -count=1
env GOFLAGS= go test ./cmd -run 'Test.*(BusinessIDAssignment|UpdateJob|BulkCompletion|UpdateProcessInstance)' -count=1
```

Use gofmt on touched Go files. Exact selected names may be refined during implementation, but do not accept an empty test selection. Require these scenarios through real execution paths:

| Scenario | Required evidence |
| --- | --- |
| Value/version rules | Empty/blank/invalid UTF-8/overlength fail before requests through CLI and public calls; padded/comma/case values are exact; BMP and supplementary Unicode at 256/257 units; creation whitespace behavior unchanged; 8.8/8.9 guards also reject forged prepared batches |
| Direct targets | Exact key and body; duplicate keys submit once; child never writes parent; full invalid stdin submits nothing; frozen keys/value; ordered mixed outcomes, fail-fast and interruption preserve accepted/not-started evidence |
| Direct observation | Known acceptance plus exact matching observation confirms; mismatch/read timeout keeps accepted-unconfirmed failure; no-wait performs no required confirmation reads; lost response submits once and remains unknown |
| Atomic wire | Scalar and bulk carry BID and variables together once; omission excludes businessId; retain supported fields; rejected assignment never falls back; listener variable restrictions remain |
| Prepared payload | Mutate caller BID pointer, nested variables, key slices and plan attachments after preparation; approved payload and selection remain stable at public/domain/adapter boundaries |
| Completion satisfaction | Uniqueness enabled, identical reassignment, two jobs/one process, and preexisting completed job retain correct per-job failures; lost scalar/bulk response followed only by COMPLETED submits once, remains unknown, retains original error class |
| Discovery | Tenant scope, exact retries zero, limits, duplicates and sparse pages preserved; late failure or incomplete selection sends zero mutations; no rediscovery after approval |
| Input shape | Scalar remains scalar; repeated/comma/stdin/selector collection remains bulk after deduplication to one; assignment alone never becomes a job selector |
| Empty/preview | Deliberate empty stdin and authoritative empty discovery succeed without prompt/mutation; no-key without deliberate input remains invalid; dry-run never mutates; no-wait never claims accepted work when nothing submitted |

## Output and terminal matrix

Extend existing configured-logger, subprocess and terminal fixtures rather than asserting only renderer substrings. Cover both workflows, success, dry-run, partial failure, pre-submit error and empty result. Run supported human/verbose/quiet/JSON/keys combinations with auto-confirm, automation and no-wait; retain existing rejection of bulk job keys-only and JSON+verbose guardrails.

- Capture stdout, configured stderr, inherited logger destination and a separate configured logger destination. Human stdout is empty for mutation summaries/previews; INFO records follow configured format/destination and disappear at WARN or quiet. Plain prompts remain separate from logs.
- Inject repeated backend failures carrying distinctive URL, method, body and key. Require one bounded grouped ERROR under normal, verbose and quiet, without raw details on either stream. Check plain and structured loggers. Assertions must fail if routing becomes a direct print or raw joined error.
- Enable DEBUG separately by flag and configured level; prove redacted diagnostic visibility without verbose and keep stdout free of diagnostic records.
- Decode one JSON envelope then require EOF. Preserve partial evidence and original exit classification; quiet+JSON still returns the envelope. Keys output retains documented semantics and zero bytes for an empty supported mode.
- Use real terminal stdin for nonempty confirmation, decline, EOF, configured/inherited prompt routing and redirected stdout. Empty eligible scopes must emit no prompt. Pipe-only stdin or a mocked terminal check is insufficient.

## Documentation and broader checks

Once metadata and README changes exist, regenerate docs without contacting an engine:

```bash
env GOFLAGS= make docs-content
git diff --check
```

After targeted checks pass, one full race suite is justified by changes to shared completion coordination, copied payloads, adapter interfaces and concurrent execution:

```bash
env GOFLAGS= make test
```

The inspected Makefile test target runs `go test ./... -race -count=1`; it does not opt into integration tags. Recheck before executing if the target changed. No blanket rerun is needed for later prose-only edits.

## Completion evidence

Record actual test names, commands and outcomes, separately from planned coverage. Mocked eligibility proves client response handling; it does not verify real-engine rules, durability, timing or uniqueness configuration. Keep those limitations explicit without blocking implementation on a live target.

## Iteration 4 isolated validation evidence (2026-11-02)

- Initial `TestUpdateJobAssignment` assertions failed on the absent `--set-bid`
  flag before implementation, then passed after CLI wiring.
- `env GOFLAGS= go test ./cmd -run 'Test.*(UpdateJob|BulkCompletion)' -count=1`
  passed (24.992s). The final selection contains 97 top-level tests, including
  helper entry points and eight new assignment test functions. The last-added
  `TestUpdateJobAssignmentInputGuards` also passed independently (1.013s).
- `env GOFLAGS= go test ./c8volt/job -run 'Completion|WorkerOutcome' -count=1`
  passed (0.667s). No facade interface or backend workflow was added in T004;
  the new public value validator delegates to the existing domain validator.
- New command coverage includes scalar/collection/selector wire payloads in
  normal and no-wait modes, nonempty stdin deduplication, distinct jobs in one
  process, native conflict, precompleted targets, lost response followed by
  COMPLETED, separate requested/observed identity, unsupported versions, and
  scalar approval despite flag edits. Existing paging cases now also run with
  assignment, including sparse pages and failed/incomplete discovery.
- Assignment variants exercise the bulk success/failure logger and real-terminal
  matrices. Scalar coverage verifies configured/inherited destinations, plain
  and structured formats, INFO/WARN/quiet behavior, DEBUG by flag/config without
  verbose, redaction, one envelope plus EOF, preserved exit classes, and actual
  terminal approval/decline/EOF with redirected stdout. Empty terminal discovery
  and stdin retain exact human wording and successful non-accepted envelopes.
- Negative regression checks temporarily replaced the scalar configured error
  logger with direct stderr output, exposed `runErr.Error()` at normal level,
  and wrote its human summary to stdout. The command logger tests rejected all
  three mutations. Production code was restored before final validation.
- `env GOFLAGS= make docs-content`, targeted gofmt, `git diff --check`, and
  feature Markdown local-file link checks passed. Generated `docs/index.md`
  also incorporates the T003 library documentation from README.
- Failure routing was reviewed from native classification through service
  evidence, facade conversion, scalar/bulk view, configured logger, and
  `exitAfterRenderedResult`: JSON retains the original error and partial payload
  exactly once. Human output consumes known evidence and bounded reasons.
- The inspected `make test` target uses `go test ./... -race -count=1`, clears
  GOFLAGS for this run, and adds only the race-detector exit-delay optimization.
  No inherited C8VOLT/MAKE test settings were present; integration suites require
  explicit build tags. All exercised services are test-owned fakes. Real-engine
  eligibility, uniqueness configuration, durability and timing remain unverified.
- Final `env GOFLAGS= make test` passed with race detection: 87 packages passed,
  19 had no tests, and `cmd` completed in 560.531s. Untagged tests under
  `integration/cli` are fixture/example-validation unit tests; tagged live-engine
  suites were not selected. No live-engine check is claimed.
