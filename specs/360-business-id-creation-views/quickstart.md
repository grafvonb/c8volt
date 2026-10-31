# Quickstart: Isolated Business-ID Validation

This guide is for the implementation phase. Planning created design artifacts only; none of the runtime checks below has been run or claimed passed.

## Prerequisites and safety boundary

Use the feature branch `360-business-id-creation-views`, Go 1.26.2, and the repository's existing dependencies. Run commands from the repository root. Tests must own their service dependencies through mocks or HTTP fake servers, create explicit temporary command configuration with `testx.WriteTestConfigForVersion`, and capture output separately.

Do not use real Camunda services, integration build tags/targets, local clusters, smoke workflows, deployment, target discovery, or resource cleanup. No endpoint or credentials are required. The normal Makefile test target uses `go test ./... -race -count=1`; live suites are gated by the `integration` tag. `docs-content` invokes only the docs generator.

Before each run, inspect the selected tests for fake ownership. Remove inherited live settings and test-helper activation, and override persisted `GOFLAGS` so tags cannot enable live tests. The following shell helper sanitizes the child environment without printing credentials or changing the parent shell:

```sh
isolated() {
  python3 - "$@" <<'PY'
import os, subprocess, sys
blocked = ('C8VOLT_', 'CAMUNDA_', 'ZEEBE_', 'OPERATE_', 'TASKLIST_')
env = {k: v for k, v in os.environ.items() if not k.startswith(blocked)}
for key in ('GO_WANT_HELPER_PROCESS', 'MAKEFLAGS', 'MFLAGS', 'GNUMAKEFLAGS', 'GOENV', 'GOFLAGS'):
    env.pop(key, None)
env['GOFLAGS'] = ''
sys.exit(subprocess.call(sys.argv[1:], env=env))
PY
}
```

Tests themselves must create explicit fake-only config; environment sanitation is not permission to execute environment-backed tests. Do not run example CLI mutation commands from the creation contract against an operator configuration.

## Targeted implementation checks

Name new acceptance tests with `BusinessID` in the test name across affected packages. These are planned tests, not claims that they already exist. Inventory first; require nonzero matching tests in each changed area and reject a successful command that reports no tests. Retain the existing regression tests below as well.

```sh
isolated go test ./internal/domain ./internal/services/processinstance/... ./internal/services/job/... ./internal/services/usertask/... ./internal/services/ops/... ./internal/services/processdefinition/... -list 'Test.*BusinessID'
isolated go test ./c8volt/process ./c8volt/job ./c8volt/task ./c8volt/element ./c8volt/ops -list 'Test.*BusinessID'
isolated go test ./cmd -list 'Test.*BusinessID'
```

Run the narrow group corresponding to the current work unit, not every group after each small edit:

```sh
isolated go test ./internal/domain -run 'Test.*BusinessID' -count=1
isolated go test ./internal/services/processinstance/... -run 'Test.*BusinessID|Test.*Create.*ProcessInstance' -count=1
isolated go test ./internal/services/job/... ./internal/services/usertask/... ./internal/services/ops/... ./internal/services/processdefinition/... -run 'Test.*BusinessID' -count=1
isolated go test ./c8volt/process ./c8volt/job ./c8volt/task ./c8volt/element ./c8volt/ops -run 'Test.*BusinessID|Test.*Create.*ProcessInstance' -count=1
isolated go test ./cmd -run 'TestRunProcessInstance|TestRunHelp|Test.*BusinessID' -count=1
```

Existing test homes and regression baselines include `cmd/run_test.go`, `cmd/cmd_views_processinstance_test.go`, `cmd/cmd_views_processinstance_activity_test.go`, `cmd/walk_test.go`, `cmd/get_job_test.go`, `cmd/update_job_plan_test.go`, `cmd/get_usertask_output_test.go`, listener/element/slow-analysis view tests, and `cmd/ops_report_{json,markdown}_test.go`. Existing logging matrices in job completion and task assignment provide fixtures to extend, not substitutes for testing changed creation paths.

## Required scenarios and expected results

| Scenario | Fixture and expected proof |
| --- | --- |
| Native creation | Both generated payload variants target an owned HTTP fake; exact supplied field or complete omission; existing selector/tenant/variables unchanged |
| Unicode and input presence | Run every boundary in [research.md](research.md), including empty, whitespace, commas, BMP, supplementary characters, and malformed UTF-8 public input; invalid inputs submit no mutation |
| Version policy | v88/v89 input with ID fails unsupported before mutation through public and command paths; absent ID preserves existing creation and query behavior |
| Waiting | Native response ID differs from subsequent observation, including observed nil; normal wait returns observed value, no-wait returns native value with no added lookup |
| Count and partial failure | Sequential first success then conflict; bounded concurrent mixed outcomes; fail-fast/cancel cases; identical IDs in all submitted requests, retained successful siblings, no zero-key placeholders, original error class/exit |
| Mapping and ownership | Round trips across process/operations facades and completion plans; mutate caller pointer after snapshot and verify frozen value; related resources retain old/nil IDs while parent has new ID |
| Every output surface | Cover every row of [resource-views.md](contracts/resource-views.md), including bespoke dry-run structs, slow timelines and saved Markdown escaping |
| No extra reads | Count requests for key/status-only results and local exact-key previews; absent observed ID stays absent; rendering causes no backend calls |
| Empty/no-op | Existing empty JSON payload, one envelope plus EOF, zero-byte keys, quiet+JSON retained, prompt-free completion under real terminal stdin, no mutation |
| Failure logging | Repeated failures with distinctive raw URL/method/body/key; compact single ERROR on configured/inherited logger, no raw transport evidence on either stream under normal/verbose/quiet |
| Success logging | Creation/mutation/dry-run/verbose INFO records in configured/inherited plain and structured handlers, empty human stdout, suppression at WARN/quiet |
| Machine output | Exactly one envelope plus EOF on success/validation/partial failure; unchanged payload/enrichment shape, original exits, keys-only only actual successful keys |
| DEBUG and terminal | Flag and configured debug separately, existing redaction, no stdout diagnostics; real terminal stdin with separate captures, existing prompt wording and destination, approve/decline/EOF and redirected stdout |

Keep nonempty, sparse-page, explicit-key, abort, and error regressions when touching shared views. Validate supported output combinations according to existing flag precedence; do not force incompatible flags into a new behavior. Assert severity and destination, not merely a substring in stderr. Tests should fail if logging is replaced by a direct print or if raw aggregate errors are exposed.

## Formatting, documentation, and final validation

Run `gofmt` on the explicit set of touched Go files. Update command metadata/examples and README, then regenerate documentation:

```sh
isolated make docs-content
git diff --check
```

After targeted checks pass, run the full default race suite once for the final cross-package change:

```sh
isolated make test PKG=./...
```

This broader run is justified by shared resource fields, copied plan pointers, and sequential/concurrent creation error propagation across services, facades, and commands. Do not repeat it solely for later prose edits. Record actual test names/results, request-count assertions, generated-doc review, and any terminal platform skips in implementation progress. No live-engine behavior is claimed verified.
