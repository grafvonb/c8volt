# Validation Guide: Delete and Cancel Logging

## Prerequisites

- Use branch `codex/316-delete-cancel-logging` with the implementation completed.
- Use the Go toolchain declared by `go.mod` (Go 1.26 / go1.26.2) and existing dependencies.
- Allow local loopback HTTP fixtures and test subprocesses. Linux/macOS is required for the existing real-terminal tests; inspect skips on other hosts.
- Run commands below from the repository root. These tests use local fixtures; a live Camunda cluster or real mutation credentials are not required.

The `TestProcessInstanceLoggingTranscript` and `TestProcessInstancePollingRecordBudget` test names below are planned acceptance test entry points, not tests delivered by this planning command. Implementation must add them (or update this guide to the final names). A successful `go test` with no matching tests is not validation: verify test discovery first.

## 1. Verify acceptance-test discovery

```sh
git branch --show-current
go version
go test ./cmd -list 'TestProcessInstance(LoggingTranscript|PollingRecordBudget)'
```

Expected: both named tests are listed after implementation. If absent, stop this guide and complete implementation.

## 2. Run the complete transcript scenarios

```sh
go test ./cmd -run '^TestProcessInstanceLoggingTranscript$' -count=1 -v
go test ./cmd -run '^TestProcessInstancePollingRecordBudget$' -count=1 -v
```

The transcript fixture must execute actual command/facade/service/waiter/progress/error handling against local HTTP responses: active child, deletion conflict (409), accepted root cancellation (204), continuing ACTIVE family observations, and short deterministic confirmation timeout. Use the existing temporary config and subprocess helpers so exit handling and configured stderr are exercised. Configure short fixed backoff through the existing config mechanism; no production default changes or two-minute sleeps.

Expected outcomes:

- Default output identifies phase, root/scope, timeout, last observations, accepted cancellation and unreached resumed deletion.
- Verbose emits each actual transition explanation once; DEBUG-only does not emit those explanations, and verbose-only emits no HTTP diagnostics.
- Each completed polling check has one observation; actual observed exchanges each have one unchanged HTTP diagnostic. Routine cache pairs and nested polling chatter are absent.
- Original JSON detail, class, exit outcome, and result shapes are preserved. stdout/stderr are captured separately; JSON is decoded once and followed by EOF.
- The fixture request ledger proves mutation order and no resumed deletion after timeout. Companion success fixtures prove existing waits and resumed deletion still occur.

The record-budget fixture separately guarantees exactly 36 successful single-exchange cached-token checks, using a deterministic attempt bound or terminal response sequence. It must assert 36 observations + 36 HTTP diagnostics in the polling scope, excluding token bootstrap and phase records. Do not infer 36 checks from elapsed wall time. Match requests/checks by key and fixture ledger rather than requiring concurrent log completion order.

See [the transcript contract](contracts/cli-logging.md) and [internal data notes](data-model.md) for exact expectations.

## 3. Validate closest packages

```sh
go test ./internal/services/processinstance/waiter -run 'TestWaitForProcessInstance' -count=1
go test ./internal/services/processinstance/... -run 'Test.*(Wait|Cancel|Delete|State|Logging|Observation)' -count=1
go test ./internal/services/auth/oauth2 -run 'Test(RetrieveTokenForAPI|APIDiagnosticsOAuth)' -count=1
go test ./c8volt/ferrors ./c8volt/foptions -count=1
go test ./c8volt/process -run 'Test.*(Cancel|Delete|Wait|Error)' -count=1
```

Verify all four adapters are exercised, including v87 search-based state lookup. New cases must cover scoped suppression/direct diagnostics, timeout evidence, terminal first/repeated check, max attempts, canceled entry/sleep/lookup, absence, lookup errors, concurrent keys, and unchanged worker/fail-fast behavior. Facade tests must prove exact prior error text/classification and inspectable original evidence via `errors.Is`/`errors.As`, including joined/mixed classes and repeated normalization.

## 4. Validate output and real-terminal compatibility

```sh
go test ./cmd -run 'Test(APIDiagnostics|ProcessInstanceConfirmationTerminal|ConfirmationEmptySelectorResults|CommandErrorEnvelope|DeleteProcessInstance|CancelProcessInstance)' -count=1
go test ./internal/services/processdefinition/... -count=1
```

Retain separately captured stdout/stderr with configured and inherited stderr. Confirm actual terminal stdin tests ran, not only pipe-input helpers or skipped terminal cases. Require prompt-free empty completion; cover acceptance, abort, EOF, machine output, quiet combinations, and auto-confirm/automation/no-wait where supported. Verify zero-byte empty keys output and no confirmation/mutation calls for completed empty scopes. Preserve sparse-page continuation and existing warning policy.

Protect nested process-definition-delete callers from shared progress/error changes. Use existing HTTP/auth diagnostics tests to verify unchanged sequence fields and redaction.

## 5. Format, regenerate documentation, and run the full suite

Run `gofmt` on each touched Go file. Then:

```sh
make docs-content
git diff --check
make test
git status --short
```

Expected: CLI docs generated from updated command metadata, README aligned with actual logging behavior, no unintended files changed, and the full race-enabled suite passes. Investigate failures before implementation acceptance or commit; report any blocked validation explicitly. No runtime tests or implementation completion are claimed by the planning artifacts themselves.
