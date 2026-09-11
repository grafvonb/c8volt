# Research: Confirmation Prompts on Stderr

## Writer propagation

**Decision**: Add an explicit first `io.Writer` argument to `confirmCmdOrAbort` and `confirmCmdOrAbortDefaultYes`. Pass `cmd.ErrOrStderr()` from every production caller, including the `confirmCmdOrAbortFn` and `confirmProcessDefinitionSelectorListVisibleFn` seams. Use `os.Stderr` for a nil writer so direct helper use has a defined fallback.

**Rationale**: Both helpers currently use `fmt.Print`, which bypasses command output configuration. All identified production call sites have a command in scope. Explicit writer propagation supports inherited Cobra stderr destinations without a process-global writer or new public API. Replace only the prompt write with `fmt.Fprint`; retain formatting, terminal checks, scanner input, and decisions. Preserve current treatment of prompt-write errors rather than adding a new failure policy in this routing correction.

**Alternatives considered**: Writing directly to `os.Stderr` fixes the stream but ignores configured command writers. A mutable global writer risks cross-command leakage. Passing a whole command works but couples the helper to more context than it needs. A variadic compatibility shim hides unmigrated callers; an explicit signature makes omissions compile errors.

**Evidence**: `cmd/cmd_cli.go`, `cmd/process_definition_selector_validation.go`, and production references to both confirmation seams. Direct `delete_processdefinition.go` invocation must be migrated too. Existing tests replacing the seams need mechanical signature updates while preserving their behavioral assertions.

## Preserve distinct caller behavior

**Decision**: Keep all caller-side skip checks and continuation policies in place. Default-yes remains scoped to selector recovery.

**Rationale**: Shared confirmation checks stdin only. Selector recovery additionally requires terminal stdout and forbids auto-confirm, JSON, keys-only, and automation modes. Its full command therefore must still suppress recovery with redirected stdout, even though the default-yes helper can be tested directly with terminal stdin and captured stdout. Paging callers turn a declined confirmation into a normal stop where they already do so; helper aborts must not be imposed as new command failures.

**Alternatives considered**: Unifying both terminal policies or changing paging defaults would expand the issue and alter compatibility.

**Evidence**: `processDefinitionSelectorPromptAllowed`, `processDefinitionSelectorInteractiveTerminal`, `processDefinitionSelectorRecovery`, and `cmd/get_job_search.go`.

## Real terminal acceptance testing

**Decision**: Extend the existing test subprocess convention with a narrowly scoped terminal-input runner under `testx`. Use Linux and Darwin allocator files backed by the existing `golang.org/x/sys/unix` module. Attach only child stdin to a pseudo-terminal (PTY), with separate stdout/stderr capture. Use build constraints and an explicit unsupported-platform stub for the terminal-specific facility; retain portable tests everywhere.

**Rationale**: `testx/cmd_subprocess_runner.go` already re-executes the test binary with exact test selection and helper environment markers. Its string-reader input is non-terminal and cannot reproduce this bug. Isolating stdin and command globals in the child avoids races from changing process globals in the parent. Linux CI and local macOS need actual terminal coverage. The existing module provides Linux terminal ioctls and Darwin PTY grant/unlock/name constants; keep allocation mechanics confined to small platform files.

**Test discipline**: Require `term.IsTerminal` in the child. Wait for each complete stderr prompt before sending the next response, because separate scanners may read ahead if answers are preloaded. Keep canonical input for end-of-input tests and send the terminal EOF control character at an empty line. Disable echo or drain the PTY master. Bound runs with deadlines, close descriptors, kill and reap on timeout, and synchronize output observation. Verify both standard stderr and a custom/inherited command destination; capture any independent result/control information separately. Failure to allocate a PTY on Linux or macOS is a failed acceptance prerequisite, not a silently passing skip. Windows retains portable regressions and compile coverage; the Unix PTY scenarios are explicitly unsupported there.

**Alternatives considered**: Pipe input or mocking terminal detection cannot prove FR-008. Python or `script` adds executable prerequisites and platform differences. A new PTY module would reduce allocator code but is unnecessary for the two required validation platforms. No general terminal framework is proposed.

**Evidence**: `testx/cmd_subprocess_runner.go`, `go.mod`, `.github/workflows/go.yml`, `.goreleaser.yaml` (release platform configuration), and the installed `golang.org/x/sys` source. Research delegated a bounded inspection of terminal facilities and subprocess conventions; no existing PTY helper was found.

## Documentation and completion

**Decision**: Update the README output-contract guidance and root command long description, then regenerate references with `make docs-content`. Run targeted confirmation/paging/selector checks before the full `make test` race suite.

**Rationale**: The constitution requires documentation for user-visible behavior and full race validation before implementation completion. Root command metadata provides a shared location without modifying every command description.

**Alternatives considered**: Editing generated Markdown directly would violate repository guidance. Non-TTY-only testing would miss the defect. No live Camunda environment is required for automated acceptance when existing command/service fakes supply results.

All planning unknowns are resolved. Runtime validation remains implementation work.
