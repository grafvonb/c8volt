# Implementation Plan: Empty Selector Result Output

**Branch**: `codex/299-empty-selector-output` | **Date**: 2026-09-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/299-empty-selector-output/spec.md`

## Summary

Issue #299 fixes successful zero-match selector-based process-instance delete/cancel output. Replace three unconditional human-output writes with a small process-instance view helper that emits one successful JSON envelope, no keys-only bytes, or the existing human summary subject to quiet mode. Reuse existing report and dry-run types and preserve discovery counts, confirmation eligibility, mutation behavior, and nonempty rendering.

## Technical Context

**Language/Version**: Go 1.26; repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, shared command envelopes, process facade report types, and current dry-run summary constructor. No new dependency.

**Storage**: No new storage or persistence. Existing configuration and remote process-instance data remain unchanged.

**Testing**: Go testing, testify, local HTTP fixtures, existing facade stubs, and `testx.NewCmdTerminalRunner`; full race validation via `make test`.

**Target Platform**: Existing c8volt CLI platforms; terminal and redirected stdout/stderr. Real-terminal checks run on platforms supported by the existing terminal harness.

**Project Type**: Go CLI with public library facade and internal versioned services.

**Performance Goals**: Zero additional discovery requests, zero mutation requests, and no new polling, retries, waits, or confirmation for an empty scope; bounded single-result rendering.

**Constraints**: Only successful empty selector results change. Preserve shared payload shapes and mode precedence, no-wait no-op success, exit codes, tenant behavior, and command file cohesion. No generated-client edits or broad renderer refactoring.

**Scale/Scope**: Two commands, two execution types, three existing zero-match branches, and one focused view helper; test and documentation updates in existing package ownership.

## Constitution Check

*Initial gate before research: PASS. Post-design gate: PASS. These are design checks; implementation validation remains required.*

| Principle | Initial assessment | Post-design evidence |
| --- | --- | --- |
| I. Operational Proof Over Intent | Success only after successful zero-match discovery | Keep `planned.RequestedCount == 0` checks after error handling; explicitly render `succeeded`, including no-wait; no submitted mutation claim |
| II. CLI-First, Script-Safe Interfaces | Preserve existing flags, envelope, exit codes | JSON/keys-only/human/quiet contract in `contracts/cli-output.md`; unchanged precedence and nonempty paths |
| III. Tests and Validation Are Mandatory | Command regressions and full race suite required | Mode matrix, exact requests, terminal stdin, error/nonempty guards, targeted tests then `make test` in quickstart |
| IV. Documentation Matches User Behavior | User-facing correction requires docs | README and command metadata updates, `make docs-content`, help regression checks |
| V. Small, Compatible, Repository-Native Changes | Reuse established helpers and ownership | Three call-site substitutions and one view helper; no service, facade, schema, or dependency change |

**Compatibility note**: Empty JSON stdout changes from invalid human text to the existing successful result envelope. Empty keys-only stdout changes from `found: 0` to zero bytes. Quiet suppresses that informational line. Ordinary human stdout retains `found: 0` exactly once. Existing nonempty and explicit-key contracts are preserved.

## Project Structure

### Documentation (this feature)

```text
specs/299-empty-selector-output/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/cli-output.md
```

`tasks.md` is deferred to `$speckit-tasks`.

### Source Code (repository root)

```text
cmd/
├── delete_processinstance_selector.go       # Two authoritative empty-scope call sites
├── cancel_processinstance_selector.go       # One authoritative empty-scope call site
├── cmd_views_processinstance.go             # Focused empty-result view helper
├── cmd_views_processinstance_dryrun.go      # Reuse summary constructor/result rendering
├── cmd_views_contract.go                   # Reuse succeeded envelope rendering
├── delete_processinstance_selector_test.go # Delete command output/request matrix
├── cancel_processinstance_selector_test.go # Cancel command output/request matrix
├── cmd_views_processinstance_test.go        # Focused view coverage
├── cmd_confirmation_terminal_test.go        # Real-terminal prompt-free regressions
├── cmd_processinstance_test.go              # Shared HTTP harness and help assertions
├── delete_processinstance.go                # Help/example metadata
└── cancel_processinstance.go                # Help/example metadata
c8volt/process/model.go                      # Existing report types; no schema changes
testx/cmd_terminal_runner.go                 # Reuse terminal harness
README.md                                   # Empty-result behavior documentation
docs/cli/                                   # Regenerate from metadata
```

**Structure Decision**: Keep all behavior changes under `cmd`. The existing selector mode files retain lifecycle ownership; final output belongs in the process-instance view file. No new lifecycle or service mechanics are introduced. Existing test files may be extended with locally named cases; no parallel command architecture is needed.

## Phase 0: Research

Completed in [research.md](research.md). Resolved: trustworthy no-op detection, report and preview wire shapes, explicit success despite no-wait, quiet precedence, test harness selection, request counts, and documentation ownership. No unresolved clarification or gate violation remains.

## Phase 1: Design

### Rendering flow

1. Existing selector validation and planning run unchanged.
2. At each current successful `planned.RequestedCount == 0` branch, call a shared view helper with command identity/operation and execution type. Do not infer emptiness from reports, individual pages, or abort state.
3. The helper chooses JSON, keys-only, or human using existing mode resolution. JSON constructs the existing operation-specific report or aggregate dry-run summary and emits a successful envelope. Human mode writes the unchanged summary only when not quiet.
4. Propagate helper errors through each function's existing error return. Keep upper-runner empty report/preview early returns, so the result is not rendered twice.
5. Retain existing attached tenant context and planning-activity cleanup. Do not add tenant discovery, logger rewrites, or new progress output.

### Payload design

Normal execution uses the existing delete/cancel report container, whose empty `items` field is omitted, producing `payload: {}`. Dry-run uses the existing aggregate summary with zero counts, complete scope, no warning, empty collections following current serialization, and `mutationSubmitted: false`. Do not synthesize one empty preview or introduce a no-op schema. See [data-model.md](data-model.md) and [CLI contract](contracts/cli-output.md).

### Validation design

Test both commands and both execution types in human, JSON, keys-only, and quiet modes. Add quiet+JSON/keys-only, auto-confirm, automation with explicitly selected machine output, and supported no-wait variants. Decode stdout once and require EOF on a second decode; verify outcome, command, and payload. Check exactly one human summary or zero keys-only bytes, and capture stderr independently.

Use real command HTTP fixtures to assert unchanged one-request state/date-filter discovery and unchanged two-request BPMN selector validation/discovery where applicable, with unexpected mutation/expansion paths rejected. Add confirmation spies and real-terminal stdin runs with no input exchanges: empty scopes must exit successfully without a prompt or timeout. Preserve sparse-page continuation, nonempty, explicit-key, invalid-selector, discovery-failure, and abort tests. Global flag mutations in package tests must be reset and must not run in parallel.

Update README and command metadata; regenerate docs and verify help output. Run targeted tests first, then full race-enabled validation. See [quickstart.md](quickstart.md) for runnable commands. Planning itself does not claim implementation tests pass.

## Complexity Tracking

No constitution violations or additional complexity require exceptions. A focused shared view helper avoids duplicating mode logic while preserving all three existing discovery guards.
