# Implementation Plan: Confirmation Prompts on Stderr

**Branch**: `298-confirmation-prompts-stderr` | **Date**: 2026-09-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/298-confirmation-prompts-stderr/spec.md`

## Summary

Route both existing confirmation helpers through an explicit stderr writer, supplied by each command's `ErrOrStderr()`. Preserve prompt bytes, input handling, terminal eligibility, defaults, skip policies, and paging outcomes. Prove the correction through terminal-input subprocess tests with independently captured outputs, update shared user guidance, and run the full race-enabled suite.

## Technical Context

**Language/Version**: Go 1.26; toolchain go1.26.2 from `go.mod`.

**Primary Dependencies**: Existing Cobra v1.10.2, `golang.org/x/term` v0.42.0, standard `io`/`fmt`/`os`/`bufio`; test helpers use the existing `golang.org/x/sys` v0.43.0 and testify v1.11.1. No new module is planned.

**Storage**: No persistence or schema changes; temporary test captures only.

**Testing**: Go package tests, existing helper subprocess conventions, real PTY stdin on Linux/macOS, `make test` (`go test ./... -race -count=1`).

**Target Platform**: Existing Linux, macOS, and Windows CLI releases. Terminal integration coverage on Linux/macOS; portable behavior tests and compilation remain available on Windows.

**Project Type**: CLI with public facade and internal service layers.

**Performance Goals**: No additional backend calls, paging work, retries, or polling. Replace the existing prompt write without introducing asynchronous production work.

**Constraints**: Preserve stdin terminal detection and scanner behavior; no stdout prompt bytes, logger decoration, new exit policy, or changes to selector recovery guards. Update internal test seams consistently.

**Scale/Scope**: Two confirmation helpers and all their existing direct/seam callers under `cmd/`; shared test support and documentation. No facade, domain, version adapter, or generated client changes.

## Constitution Check

*Pre-research and post-design gates both pass for this plan. Test results are required during implementation and are not claimed here.*

| Principle | Pre-research | Post-design evidence |
|---|---|---|
| Operational proof over intent | Pass | Mutation workflows and outcome verification remain untouched; confirmation decisions and paging-stop regressions are required. |
| CLI-first, script-safe interfaces | Pass | Contract separates prompts from results and preserves flags, defaults, skips, and exits. Intentional compatibility change: consumers capturing prompts from stdout must capture stderr instead. |
| Tests and validation mandatory | Pass | Real terminal helper and command paths, negative/skip matrix, closest package checks, then `make test`; PTY failure on supported validation platforms cannot silently pass. |
| Documentation matches behavior | Pass | README and root source metadata updates plus `make docs-content`; review generated references. |
| Small, compatible repository-native changes | Pass | Explicit writer propagation; no new production subsystem/module. Small platform PTY allocators are necessary test support because existing pipe helpers cannot reproduce the issue. |

## Project Structure

### Documentation (this feature)

```text
specs/298-confirmation-prompts-stderr/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── contracts/confirmation-streams.md
```

`tasks.md` is generated later by `$speckit-tasks`.

### Source Code (repository root)

```text
cmd/
├── cmd_cli.go                         # default-no helper and seam
├── process_definition_selector_validation.go  # default-yes helper/seam and two recovery calls
├── cmd_cli_test.go                    # existing formatting tests
├── cmd_confirmation_terminal_test.go  # proposed terminal acceptance tests
├── process_definition_selector_validation_test.go
├── get_*_search.go                    # existing paging caller writer propagation
├── delete_*.go, cancel_*.go, resolve_processinstance.go, update_*.go
├── ops_*.go                          # existing confirmation callers
└── root.go                           # shared help metadata
testx/
├── cmd_subprocess_runner.go           # existing subprocess conventions
├── cmd_terminal_runner.go            # proposed bounded terminal subprocess support
├── cmd_terminal_linux.go             # proposed PTY allocator
├── cmd_terminal_darwin.go            # proposed PTY allocator
└── cmd_terminal_unsupported.go        # proposed explicit unsupported-platform handling
README.md
docs/cli/                             # regenerated through docs-content
```

**Structure Decision**: Keep production routing in its current command ownership. All helpers/seams take `io.Writer` first; callers pass `cmd.ErrOrStderr()`. Update tests assigning the seams to accept this parameter. No new execution mode is added, so no lifecycle file split is required. PTY mechanics belong in shared test support, with platform build constraints preventing Unix-only imports on Windows.

## Implementation Design

1. Add writer input and standard-stderr fallback to both helpers; replace `fmt.Print` with `fmt.Fprint`. Preserve existing ignored write-error handling, prompt formatting, early returns, scanner construction, normalization, and error conversion.
2. Update every production reference to `confirmCmdOrAbort`, `confirmCmdOrAbortFn`, and `confirmProcessDefinitionSelectorListVisibleFn`, including direct process-definition deletion, mutation/selector paths, ops callbacks, and paging for process instances, elements, incidents, and jobs. Do not change their auto-confirm argument expressions or handling of `ErrCmdAborted`.
3. Update test seam signatures mechanically. Add assertions proving the configured or inherited writer reaches the real helper. Retain literal prompt expectations independent of the formatter under test.
4. Add terminal subprocess coverage for both defaults, accepted/declined/empty/normalized answers and EOF, plain exact prompt text, standard fallback, configured stderr, and uncontaminated results. Exercise actual keys-only paging with a fake service and synchronized per-prompt answers. Verify normal stop after decline/EOF and no extra page request.
5. Cover selector recovery at both levels: default-yes helper with captured stdout, plus real caller eligibility regressions proving redirected stdout still suppresses recovery. For an eligible recovery command path, provide terminal stdout as required by its existing guard or use the existing caller test seam strictly for wiring; do not claim a mocked guard is terminal-path proof.
6. Preserve auto-confirm, automation, non-TTY, JSON, and keys-only recovery suppression in existing command tests. Add timeout-backed evidence that skipped prompts do not wait for input.
7. Update shared documentation and metadata, regenerate CLI references, format touched Go files, run focused checks followed by `make test`, and inspect the final diff.

## Validation and Requirement Coverage

| Requirements | Evidence required |
|---|---|
| FR-001–FR-004 | Terminal subprocesses for both helpers; explicit/inherited stderr capture; exact stdout result comparison; actual keys-only paging. |
| FR-005–FR-007 | Answer/default/EOF matrix; unchanged caller abort/stop semantics; auto-confirm, automation and non-terminal regressions. |
| FR-008–FR-009 | Assert real terminal input; deadlines and isolated globals; Linux/macOS PTY execution; focused tests and full race suite. |
| FR-010 | README/root metadata and generated references agree on streams; existing prompt wording remains byte-for-byte. |

Run commands and scenario prerequisites are in [quickstart.md](quickstart.md). Platform-specific allocator tests must establish cleanup and successful terminal allocation. Compile Windows test targets to catch accidental Unix dependencies; do not claim Unix tests validate Windows console behavior.

## Complexity Tracking

No constitution violations require exceptions. Small platform-specific test allocation code is justified by mandatory terminal reproduction and the lack of an existing helper; all production changes remain writer propagation. No new module or general terminal abstraction is planned.
