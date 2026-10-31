# Quickstart: Isolated Validation of the Advisory Version Check

This guide is for implementation validation after the planned code/tests exist. It does not claim those tests have already run. See [CLI contract](contracts/cli-version-check.md) for expected output and [data model](data-model.md) for state/timing rules.

## Prerequisites and boundary

- Use branch `222-camunda-version-check`, the repository Go toolchain from `go.mod`, and existing project dependencies.
- Inspect test names and helpers before running them. All endpoint addresses/configs must belong to the tests. No real Camunda, Docker/Kind/Kubernetes, integration target, live build tag, or environment-backed helper is permitted.
- The planned new test prefix is `TestVersionCheck`. Adjust filters only to actual equivalent test names after implementation. Confirm the selector lists tests; a successful command with no matching tests is not validation.
- Subprocess helpers inherit the environment. Remove inherited application configuration, credentials/config paths, helper-process markers, GOFLAGS, and make overrides before compilation. Test setup must also isolate `C8VOLT_*` and explicitly select test-owned config/server files; setting GOFLAGS inside an already running test cannot remove build tags used to compile it.
- Do not run the commands below solely for specification/plan changes. Planning checks are document review, link/structure validation, and `git diff --check`.

## Clean validation shell

From the repository root, the following starts a child shell with the relevant inherited overrides removed. It does not discover or contact any environment. Preserve the ordinary toolchain PATH; never print inherited credentials.

```sh
python3 - <<'PY'
import os
import subprocess

env = os.environ.copy()
for key in list(env):
    if key.startswith(('C8VOLT_', 'GO_WANT_')) or key in {
        'GOFLAGS', 'MAKEFLAGS', 'MFLAGS', 'GNUMAKEFLAGS', 'MAKEOVERRIDES',
        'PKG', 'LDFLAGS', 'GORACE', 'BASH_ENV', 'ENV',
    }:
        env.pop(key, None)
env['GOFLAGS'] = ''
raise SystemExit(subprocess.call(['/bin/bash', '--noprofile', '--norc'], env=env))
PY
```

Run the following inside that shell after reviewing changed test sources. These are isolated test commands, not manual commands against a configured gateway.

## Focused validation

```sh
go test ./internal/services/cluster/... -list 'TestVersionCheck'
go test ./internal/services/httpc/... -list 'TestVersionCheck'
go test ./cmd -list 'TestVersionCheck'
go test ./toolx -run 'TestCamundaRelease' -count=1
go test ./internal/services/cluster/... -run 'TestVersionCheck' -count=1
go test ./internal/services/httpc/... ./internal/services/common/... -run 'Test(VersionCheck|PrepareServiceDeps)' -race -count=1
go test ./cmd -run 'TestVersionCheck' -count=1
```

Expected proof:

1. Numeric major/minor comparison accepts ordinary release, patch, and prerelease forms, rejects malformed/multiline/overlong values, and produces fixed bounded reasons.
2. One attempt precedes the first operation and no second attempt occurs for pages, workers, polling, failed checks, or operation retries. Standalone consumers without a gate pass through unchanged; distinct invocations each check.
3. A stalled probe with a short configured HTTP timeout warns once; a subsequent fast operation succeeds using its own normal timeout. An earlier parent cancellation stops dispatch and preserves its exit class. Concurrent waiters can cancel promptly.
4. Auth bootstrap/refresh is not gated. Probe bypass prevents recursion; shared pacing counts its actual start; probe admission failure cannot contaminate original mutation evidence. No probe retries or redirect-following occur.
5. Help, capabilities, local application version/config inspection, local invalid input, local dry-run/empty scope, and a declined local-only mutation perform zero checks. Discovery dry-run and discovered emptiness check once. Use existing real-terminal fixtures for eligible abort/empty prompt cases.
6. Matching, mismatched, denied, timed-out, malformed, and unavailable responses are exercised across v88/v89/v810 and the relevant legacy-client wiring.

## Diagnostic and output regression

```sh
go test ./cmd -run 'Test(APIDiagnostics|RequestRate|ConfigTestConnection|BulkCompletionDiagnosticLogging|CapabilityDocument|AllCommandHelp)' -count=1
go test ./internal/services/httpc/... -run 'Test(VersionCheck|ReadRetry|APIDiagnostics|RequestRate)' -race -count=1
```

Use existing helpers in `cmd/root_api_diagnostics_test.go`, `cmd/root_request_rate_test.go`, `cmd/request_rate_mutation_test.go`, and `testx` to assert separate stdout/stderr and configured/inherited destinations. Reuse complete logger records, not warning substrings alone. Exercise active plain and structured formats; normal, verbose-only, quiet, WARN/ERROR levels, JSON and keys-only including quiet combinations; DEBUG by flag and config independently of verbose.

The multi-resource fixture must include a failed advisory followed by repeated backend failures and one successful operation, with distinctive URL, method, body, key, and credential markers. Require one advisory WARN and the established compact aggregate ERROR, original failure exit, partial results, and absence of raw transport details on both normal/verbose streams. Quiet suppresses WARN but retains ERROR. JSON decoding must require one envelope followed by EOF; keys must be exact. DEBUG evidence must remain redacted and never enter stdout.

For `config test-connection`, verify actual-result JSON warnings and legacy classification remain unchanged (including historically permissive values such as `8.9.garbage`), duplicate human diagnostics are suppressed after an advisory warning, a changed actual observation after an advisory match remains diagnosable, and explicit compatibility evaluation still works with automatic checking disabled.

Update strict fake handlers to serve matching topology and account for the extra request/DEBUG counter where relevant. Explicit `--no-version-check` is appropriate in a narrowly unrelated fixture; global opt-out in shared resets is forbidden. Assertions must detect direct-print substitution, raw-error leakage, extra checks, duplicate JSON, and lost submission evidence.

## Formatting, documentation, and final scope check

After targeted checks pass and changed Go files are formatted with `gofmt`, run:

```sh
make docs-content
make test PKG=./...
git diff --check
```

`docs-content` runs the local Cobra docs generator; inspect generated CLI references and the README-synced index. `make test` runs the race-enabled unit suite. The full suite is justified here by common request wiring across all service areas and concurrent invocations; it is not a blanket requirement for future documentation edits. Do not invoke integration targets or add integration tags. Repeat a successful check only after a relevant subsequent change or failure.

Record actual commands/results and any gaps. Real-engine version reporting, authentication deployments, and network/proxy behavior beyond isolated fixtures remain unverified; they are not implementation completion gates.
