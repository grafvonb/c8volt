# Isolated Validation Guide: Standardize Flag Intent

This guide is for implementation validation after the corresponding code/tests exist. No command below designates or contacts a live Camunda target. The planning run performed document checks only.

## Prerequisites

- Use branch `369-standardize-flag-intent` and the Go toolchain pinned in `go.mod`.
- Read [CLI contract](contracts/cli-flags.md) and [data model](data-model.md).
- Inspect each selected test before execution: dependencies must be test-owned fakes, mock clients, or isolated terminal/subprocess fixtures.
- Clear inherited `GOFLAGS` containing build tags and any environment/profile/credential configuration that might enable live helpers. Test subprocesses must explicitly select temporary fixture config and fake endpoints; never rely on the user's config or default profile. No integration build tags, `make integration-*`, live helpers, recording scripts, cluster inspection, or provisioning.
- New test names below are proposed implementation names, not claims that tests currently exist. Require named cases to exist before using `-run`, so a no-tests-to-run result cannot be reported as validation.

## Targeted implementation checks

From the repository root, after each related work unit:

```sh
go test ./cmd -run 'TestFlagIntentMigration' -count=1
go test ./cmd -run 'TestFlagIntentJob' -count=1
go test ./cmd -run 'TestFlagIntentVariables|TestFlagIntentCreationRepair' -count=1
go test ./cmd -run 'TestFlagIntentEnrichment' -count=1
```

Use existing neighboring test families as well; update their invocation syntax rather than replacing meaningful coverage. Each work unit includes touched Go formatting and its relevant command metadata/docs. Use test-owned HTTP fixtures for cross-layer behavior, never a local engine.

### 1. Canonical spelling and native payloads

Cover every rename-table row in every affected supported mode. Compare captured requests, defaults, omissions and response evidence with existing assertions. Include element/process/task inline and file variables, task assignment/completion, run BID/variables, both repair paths, job scalar/collection/selector completion, retry/timeout update, technical failure and BPMN error. Preserve task-completion exact numeric values, BID exact opaque values, assignment exclusions, and optional versus required payloads. Repair explicit zero still skips retry restoration.

Expected: same targets and payloads, with no new mode combinations or requests. Use unchanged facade/service test doubles. No variable-file support is added to run or job commands.

### 2. Legacy rejection and parser order

Cover all removed spellings with canonical replacement guidance, with and without conflicting new input and in both argument orders. Include equal values and explicit zero; aliases of the command noun must behave identically. Reject legacy mutation retries, keyed-completion retries and attached/separated BPMN-error values, including codes `true`/`false`. Validate that only the bare BPMN action is canonical.

Place rejected syntax before later `--config`, `--profile`, `--json`, `--quiet`, logger-level/format flags and inherited controls. Include attached flag values, supported short forms, payload values beginning with flag-looking text, and `--`. Do not consume stdin or read mutation files to report these failures.

Expected: invalid-input classification and existing no-err-codes behavior, no confirmation, no authentication/version request, no discovery or mutation, one error emission. The reporting context still uses configured/inherited logger destination and the actual selected command.

### 3. Timeout and retries are independent intents

Test actual effective `http.timeout`, inherited flag identity and captured job timeout separately. Include default, file/profile and environment HTTP settings, explicit flag override before/after command words, job-only timeout, both timeouts and a key with HTTP timeout alone. A valid canonical mutation plus HTTP timeout remains valid; mutation exclusions must not reject that execution control.

For retry predicates, include exact zero on get and keyless completion; absence remains omission. Mutation `set-retries 0` stays explicit zero. Keyed completion and previously unsupported combinations remain rejected.

Expected: no job-local `timeout` shadow; no retry-selector value leaks into a mutation request.

### 4. Output and terminal execution

Capture stdout and stderr separately through command execution. Test configured and inherited destinations with plain and structured logging, quiet, WARN level, and supported JSON/keys modes. Verify successful mutation/dry-run/verbose details as INFO logger output with empty human stdout, independently of prompts and machine results. Under JSON decode exactly one existing envelope and require EOF, including partial failure; retain original exit classification.

For repeated multi-resource backend failures, use distinctive fake URL, HTTP method, body and resource key. Assert compact aggregated diagnostics and no raw details on either stream under normal/verbose; quiet retains the compact error. Isolated DEBUG cases through both flag and configured level retain redacted evidence and keep stdout clean. Do not assume structured attributes are present in plain logger text.

Use existing real-terminal stdin helpers to verify prompt wording/routing and prompt-free completed empty selection. Cover normal/dry-run and supported quiet/machine/automation/auto-confirm/no-wait combinations. Empty keys output is zero bytes; empty JSON retains established payload conventions. Keep sparse-page continuation, incomplete discovery, nonempty, explicit-key, abort and error coverage.

Expected: replacing logger output with direct stdout/stderr, routing through the default logger, exposing a raw aggregate, or rendering a second envelope must make assertions fail.

## Shared-boundary and documentation checks

Because the implementation touches early error preparation shared with root/configuration, run the affected package checks after targeted cases:

```sh
go test ./cmd ./config ./docsgen -count=1
make docs-content
git diff --check
```

Before running, verify the selected packages and Makefile target still obey the isolated boundary. The current docs target runs local docsgen only. If implementation changes shared runtime bootstrap beyond the affected migration path, run `make test` for the resulting broader race-sensitive blast radius after checking its invocation/environment; do not run it merely because planning or a commit completed.

Inspect help, capabilities and generated shell completion using isolated CLI/subprocess tests. Assert canonical names present, removed names absent, unaffected flags unchanged. Audit README, regenerated `docs/cli` and homepage, maintained scripts, demo tape text, and integration invocation fixtures. Static syntax/text checks are allowed on live-suite files; do not execute those suites or recording scripts. Historical feature specs remain unchanged.

## Evidence to report

Record actual test names and outcomes, generation/diff checks and any skipped coverage. Missing proposed tests or a no-tests-to-run result is not a pass. No real-engine behavior is established by these checks; report that limitation without making live validation an implementation gate.
