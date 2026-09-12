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

## Iteration 6 foundational validation

Validation ran on 2026-09-11 with `go1.26.2 darwin/arm64`. Terminal test discovery was confirmed before running the required checks:

```text
$ go test ./testx -list 'Terminal'
TestDarwinCmdTerminalAllocator
TestDarwinCmdTerminalAllocatorClosesMasterWhenSlaveOpenFails
TestCmdTerminalRunnerSequencesPrompts
TestCmdTerminalRunnerTimesOut
TestCmdTerminalRunnerReportsAllocatorFailure
TestCmdTerminalRunnerReportsUnsupportedPlatform
ok github.com/grafvonb/c8volt/testx 0.430s
```

Both post-migration focused suites passed:

```text
$ go test ./testx -run 'Terminal' -count=1
ok github.com/grafvonb/c8volt/testx 0.510s

$ go test ./cmd -run 'Confirm|Paging|Selector' -count=1
ok github.com/grafvonb/c8volt/cmd 1.294s
```

The native Darwin allocator tests and isolated terminal runner tests executed; the focused command suite also remained green after the writer-signature migration. Story-level prompt routing remains intentionally unimplemented until US1 and US3.

## Iteration 7 US1 validation

Validation ran on 2026-09-11 with `go1.26.2 darwin/arm64`. Before T010, both new terminal suites failed by reaching their bounded deadlines while waiting for the complete prompt on stderr; the helper still wrote that prompt to stdout. This supplied red routing evidence without treating the timeout as a passing regression.

After routing the default-no prompt through the supplied writer with a standard-stderr fallback, the required US1 filter passed:

```text
$ go test ./cmd -run 'TestConfirmOrAbortTerminal|TestConfirmationCommand|FormatConfirmationPrompt|DeleteProcessDefinition' -count=1
ok github.com/grafvonb/c8volt/cmd 0.845s
```

`TestConfirmOrAbortTerminal` exercised real PTY stdin for acceptance, decline, empty input, and canonical EOF, including the nil-writer fallback and byte-exact plain prompt. `TestConfirmationCommand` exercised the real delete-process-definition command with a fake backend: configured and inherited stderr both received the prompt, accepted JSON results stayed on stdout, decline retained the error exit, and the backend received no deletion request after decline. The broader `go test ./cmd -count=1` suite also passed in 35.253s, followed by `git diff --check`.

## Iteration 8 US2 validation

Validation ran on 2026-09-11 with `go1.26.2 darwin/arm64`. The new terminal parent test was first run against the temporarily restored pre-T010 `fmt.Print` write. All three scenarios reached their bounded deadlines waiting for the complete prompt on stderr, proving the old route fails the regression; the corrected `fmt.Fprint` writer route was restored immediately afterward.

The required US2 filter then passed:

```text
$ go test ./cmd -run 'TestGetProcessInstanceKeysOnlyPagingTerminal|Paging|GetIncident|GetJob|GetElement' -count=1
ok github.com/grafvonb/c8volt/cmd 1.403s
```

`TestGetProcessInstanceKeysOnlyPagingTerminal` exercised real PTY stdin through the actual keys-only command path. Repeated acceptance fetched all three pages, decline after one continuation retained two keys, and canonical EOF retained the first key; stderr contained the exact one-or-two prompts, stdout contained one key per line, and request counts proved no page was fetched after stop. Process-instance, incident, job, and element caller regressions also verified their command stderr writer and normal paging-stop interpretation. The broader `go test ./cmd -count=1` suite passed in 35.837s, followed by `git diff --check`.

## Iteration 9 US3 validation

Validation ran on 2026-09-11 with `go1.26.2 darwin/arm64`. Before T017, all seven default-yes terminal scenarios reached their bounded deadlines waiting for the complete prompt on stderr because the helper still wrote to stdout. This supplied the required red routing evidence.

After routing the default-yes prompt through the supplied writer with standard-stderr fallback, the required US3 filter passed:

```text
$ go test ./cmd -run 'TestConfirmOrAbort.*Terminal|Confirmation.*Skip|ProcessDefinitionSelector|Automation' -count=1
ok github.com/grafvonb/c8volt/cmd 0.911s
```

`TestConfirmOrAbortDefaultYesTerminal` and the expanded default-no matrix exercised real PTY stdin for `y`, `yes`, mixed case, surrounding whitespace, empty input, other answers, and canonical EOF. Both helpers retained byte-exact choice labels and abort behavior, honored custom writers and nil fallback, and kept result stdout separate. Selector recovery tests proved configured and inherited stderr wiring for visible and near-match prompts while preserving accepted listings and declined diagnostics.

`TestConfirmationSkipPolicies` held no-input stdin open behind a blocking confirmation seam and completed within its deadline for auto-confirm, supported and unsupported automation, non-terminal input, JSON, and keys-only modes. `TestConfirmationRedirectedStdoutSkip` used real terminal stdin with captured non-terminal stdout and confirmed recovery remained suppressed. Test discovery was verified with the same filter, the broader `go test ./cmd -count=1` suite passed in 36.580s, and `git diff --check` passed.

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

## Iteration 10 final validation

Validation ran on 2026-09-11 with `go1.26.2 darwin/arm64`. README guidance and root help now state that results use stdout, plain confirmation and continuation questions use stderr, prompt consumers must capture stderr, and selector-recovery/auto-confirm policies remain unchanged. `make docs-content` regenerated `docs/cli/c8volt.md` and the README-derived `docs/index.md`; inspection found the same routing contract in all three sources and only the expected current-build provenance refresh in the homepage.

The focused checks passed and explicitly discovered all three real-terminal parent tests:

```text
$ go test ./cmd -list 'TestConfirmOrAbort.*Terminal|TestGetProcessInstanceKeysOnlyPagingTerminal'
TestConfirmOrAbortTerminal
TestConfirmOrAbortDefaultYesTerminal
TestGetProcessInstanceKeysOnlyPagingTerminal
ok github.com/grafvonb/c8volt/cmd 0.503s

$ go test ./cmd -run 'TestConfirmOrAbort.*Terminal|TestGetProcessInstanceKeysOnlyPagingTerminal' -count=1
ok github.com/grafvonb/c8volt/cmd 0.747s

$ go test ./cmd -run 'Confirm|Paging|Selector' -count=1
ok github.com/grafvonb/c8volt/cmd 1.704s

$ go test ./testx -run 'Terminal' -count=1
ok github.com/grafvonb/c8volt/testx 0.536s

$ go test ./cmd -run 'TestRootHelp_PreservesHumanTaxonomyAndDiscoveryCommand|TestRootHelpAndGeneratedMarkdownShareDiscoveryAnchors' -count=1
ok github.com/grafvonb/c8volt/cmd 0.555s
```

The first race-enabled full-suite run exposed only a test-harness deadline issue: all six `TestConfirmationSkipPolicies` subprocesses exceeded the former one-second startup deadline under race instrumentation. The deadline was increased to five seconds, retaining bounded failure for an accidental stdin read. The focused race regression then passed in 7.887s, and the required full suite passed:

```text
$ go test ./cmd -race -run '^TestConfirmationSkipPolicies$' -count=1
ok github.com/grafvonb/c8volt/cmd 7.887s

$ make test
go test ./... -race -count=1
ok github.com/grafvonb/c8volt/cmd 173.353s
[all remaining packages passed or reported no test files]

$ git diff --check
[no output]
```

Platform evidence:

- macOS Darwin/arm64: the three terminal parent tests executed natively and passed; no PTY skip occurred.
- Windows amd64: `GOOS=windows GOARCH=amd64 go test -c -o /tmp/c8volt-cmd-298.test.exe ./cmd` produced a PE32+ x86-64 test executable. This is compilation evidence only; Windows runtime terminal behavior was not exercised.
- Linux arm64: `GOOS=linux GOARCH=arm64 go test -c -o /tmp/c8volt-cmd-298-linux.test ./cmd` produced a statically linked ELF aarch64 test executable. A Docker runtime attempt stalled before creating a visible container and was terminated after more than five minutes, so no new Linux runtime claim is made in this iteration. Earlier iteration 2 Linux PTY allocator runtime evidence remains recorded separately.
- `.github/workflows/go.yml` runs `make cover`; the Makefile target invokes `go test ... -race -covermode=atomic`, so the new command and test-support tests are included in existing Linux race-enabled package coverage.

Final review matched the T001 production inventory exactly: the same 22 files contain the two helpers/seams and their callers, every production caller supplies `cmd.ErrOrStderr()`, and both helpers retain the existing terminal checks, formatting, scanning, decisions, and error conversion while using the supplied writer with `os.Stderr` fallback. No facade, internal service, generated client, backend call, flag, alias, default, or unrelated output path changed. The task coverage table and iteration 7–10 evidence jointly satisfy FR-001–FR-010.
