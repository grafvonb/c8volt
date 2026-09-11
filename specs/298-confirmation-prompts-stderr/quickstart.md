# Validation Guide: Confirmation Prompts on Stderr

## Prerequisites

Use the repository root on branch `298-confirmation-prompts-stderr`, with the Go toolchain from `go.mod`. Linux or macOS must permit PTY allocation for terminal acceptance. Existing test fixtures/fake services provide command results; a live Camunda cluster is not required for automated validation.

The terminal tests below are planned additions, not tests already implemented by this planning command. Confirm that they are listed before treating a passing filtered run as evidence.

## Iteration 1 migration inventory and baseline

Audit performed on 2026-09-11 with `go1.26.2 darwin/arm64`.

Production declarations and callers containing `confirmCmdOrAbort`, `confirmCmdOrAbortFn`, or `confirmProcessDefinitionSelectorListVisibleFn`:

```text
cmd/cancel_processinstance.go
cmd/cancel_processinstance_selector.go
cmd/cmd_cli.go
cmd/delete_processdefinition.go
cmd/delete_processinstance.go
cmd/delete_processinstance_selector.go
cmd/get_element_search.go
cmd/get_incident_search.go
cmd/get_job_search.go
cmd/get_processinstance_search.go
cmd/ops_analyse_slow_process_instances_progress.go
cmd/ops_execute_retention_policy.go
cmd/ops_execute_smoketest.go
cmd/ops_purge_all_processdefinitions.go
cmd/ops_purge_orphan_processinstances.go
cmd/ops_purge_processinstances_with_incidents.go
cmd/ops_repair_incident.go
cmd/ops_repair_processinstance.go
cmd/process_definition_selector_validation.go
cmd/resolve_processinstance.go
cmd/update_job.go
cmd/update_processinstance.go
```

Test files containing either confirmation seam or helper:

```text
cmd/cancel_processinstance_selector_test.go
cmd/cancel_processinstance_test.go
cmd/delete_processinstance_selector_test.go
cmd/delete_processinstance_test.go
cmd/get_incident_test.go
cmd/get_processinstance_paging_test.go
cmd/get_processinstance_test.go
cmd/ops_analyse_slow_process_instances_progress_test.go
cmd/ops_execute_retention_policy_test.go
cmd/ops_execute_smoke_test_test.go
cmd/ops_purge_all_processdefinitions_test.go
cmd/ops_purge_orphan_processinstances_test.go
cmd/ops_purge_processinstances_with_incidents_test.go
cmd/ops_repair_incident_test.go
cmd/ops_repair_processinstance_test.go
cmd/ops_tenant_context_timing_test.go
cmd/process_definition_selector_validation_test.go
cmd/processinstance_mutation_progress_test.go
cmd/resolve_processinstance_test.go
cmd/update_job_plan_test.go
cmd/update_job_test.go
cmd/update_processinstance_test.go
```

Baseline focused validation passed:

```text
$ go test ./cmd -run 'Confirm|Paging|Selector' -count=1
ok github.com/grafvonb/c8volt/cmd 1.615s
```

## Focused validation after implementation

```sh
git branch --show-current
go test ./cmd -list 'TestConfirmOrAbort.*Terminal|TestGetProcessInstanceKeysOnlyPagingTerminal'
go test ./cmd -run 'TestConfirmOrAbort.*Terminal|TestGetProcessInstanceKeysOnlyPagingTerminal' -count=1
go test ./cmd -run 'Confirm|Paging|Selector' -count=1
```

Expected new parent tests: `TestConfirmOrAbortTerminal`, `TestConfirmOrAbortDefaultYesTerminal`, and `TestGetProcessInstanceKeysOnlyPagingTerminal`. Helper subprocess entrypoints may also appear. Verify these parent tests actually execute, with no PTY skips on Linux/macOS.

The terminal runner supplies input only after observing each prompt. It exercises `yes`, mixed-case/whitespace-padded acceptance, decline, empty answers, and canonical EOF, and fails on timeout. Inspect exact stream assertions against [the contract](contracts/confirmation-streams.md): stdout has no prompt text, stderr has the complete unchanged question, and result keys need no filtering. Also exercise configured/inherited stderr separately from standard-stderr fallback.

Run the existing selector-recovery guard and automation tests. Redirected stdout must still suppress selector recovery; direct default-yes terminal testing does not authorize changing this guard. Paging decline/EOF must retain the normal-stop behavior of the tested caller and retain keys already emitted.

## Full validation and documentation

```sh
make docs-content
make test
git diff --check
git diff --stat
```

Before these checks, run `gofmt` on the specific Go files changed by implementation. Review README and root help changes together with generated CLI references; do not edit generated references manually. The full suite must pass with the race detector. Build the Windows cmd test binary from a Unix checkout to verify platform isolation:

```sh
GOOS=windows GOARCH=amd64 go test -c -o /tmp/c8volt-cmd-298.test.exe ./cmd
```

Cross-compilation is a build check, not Windows execution or terminal acceptance evidence. Run terminal acceptance on Linux CI and macOS when available, and record any platform not exercised.

## Optional manual smoke check

With a configured read-only Camunda environment containing more than one page of process instances, build the CLI and run from an interactive terminal:

```sh
go build -o /tmp/c8volt-298 .
/tmp/c8volt-298 get process-instance --keys-only --batch-size 1 --limit 3 > /tmp/c8volt-298-keys.txt
cat /tmp/c8volt-298-keys.txt
```

Use an environment with at least three matching instances and leave auto-confirm/automation disabled. If a continuation question is reached under the existing paging rules, accept once and decline the next question. The question stays visible on stderr, and the file contains only the fetched keys. Consult command help for configuration flags required by your environment. This read-only smoke check supplements the automated terminal matrix.

Implementation completion requires actual recorded targeted and full-suite results; the plan alone supplies no runtime proof.
