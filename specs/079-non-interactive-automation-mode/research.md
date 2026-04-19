# Research: Define Non-Interactive Automation Mode

## Decision 1: Expose automation mode through one dedicated root flag

- **Decision**: Add one dedicated root flag, planned as `--automation`, as the canonical opt-in for non-interactive execution.
- **Rationale**: The clarified spec explicitly selected a new dedicated flag rather than a documented bundle. `--automation` is short, intention-revealing, and aligns with the issue’s goal of giving AI agents and CI callers one easy-to-remember contract entry point.
- **Alternatives considered**:
  - `--non-interactive`: rejected because it describes a behavior but not the intended caller context, and it reads more like a generic terminal flag than a repository-level automation contract.
  - Keep only documented existing flags: rejected because the clarification chose a dedicated flag and the issue is specifically about removing command-specific guesswork.
  - `--ci` or `--headless`: rejected because they are narrower than the intended audience of AI agents, scripts, and CI.

## Decision 2: Keep automation mode layered on top of the existing `#78` machine contract

- **Decision**: Reuse the existing shared result-envelope helpers, discovery command, and command metadata from `#78` rather than building a separate automation-only output path.
- **Rationale**: The repo already has `capabilities --json`, `ResultEnvelope`, and command metadata annotations. Extending those seams keeps the automation feature repository-native and avoids creating two competing machine-facing contracts.
- **Alternatives considered**:
  - Introduce a second automation-only output schema: rejected because it would duplicate the current machine contract and increase maintenance cost.
  - Limit the feature to help text and docs only: rejected because the issue requires a testable runtime contract, not just guidance.

## Decision 3: Add explicit automation support metadata per command

- **Decision**: Extend command metadata so each command can report automation-mode support explicitly, with discovery exposing truthful support instead of assuming that shared-envelope support automatically means full automation support.
- **Rationale**: The current `contractSupport` annotation captures JSON-envelope support, not whether prompts, paging continuation, or destructive confirmations are automation-safe. A separate automation support signal lets the CLI reject unsupported commands intentionally and keeps discovery honest.
- **Alternatives considered**:
  - Reuse only `contractSupport`: rejected because JSON-envelope support and automation-safety are related but not identical concerns.
  - Infer support from mutation type: rejected because some read-only commands still have continuation or output-mode nuances that matter for automation.

## Decision 4: Unsupported commands must reject automation mode explicitly

- **Decision**: If a command has not opted into supported automation-mode behavior, `--automation` must fail explicitly rather than falling back to the current interactive path or inventing a best-effort non-interactive default.
- **Rationale**: The clarified spec selected explicit failure as the safety rule. This prevents unattended callers from mistaking an interactive fallback or partial behavior for a supported automation contract.
- **Alternatives considered**:
  - Fall back to interactive behavior: rejected because it reintroduces the hang risk the feature is meant to remove.
  - Auto-apply a guessed safe behavior per command: rejected because it makes the contract inconsistent and hard to test.

## Decision 5: Supported automation mode implicitly confirms supported prompts

- **Decision**: For commands that explicitly support automation mode, the automation flag should implicitly confirm supported confirmation prompts and paging continuation prompts.
- **Rationale**: The clarified spec selected implicit confirmation. The repo already centralizes confirmation through `confirmCmdOrAbort` and `confirmCmdOrAbortFn`, which makes this behavior implementable without scattering new checks across every command.
- **Alternatives considered**:
  - Require both `--automation` and `--auto-confirm`: rejected because it weakens the “one clear contract” goal.
  - Auto-confirm only non-destructive prompts: rejected because it would make support rules harder to explain and test than an explicit per-command support model.

## Decision 6: Automation mode does not imply asynchronous execution

- **Decision**: Keep `--no-wait` as the explicit selector for accepted-but-not-yet-complete work even when automation mode is active.
- **Rationale**: The clarified spec selected `accepted` outcomes only when callers combine automation mode with `--no-wait`. This preserves the current operational-proof standard that commands report confirmed completion unless a caller explicitly opts out of waiting.
- **Alternatives considered**:
  - Make all supported state-changing commands asynchronous in automation mode: rejected because it weakens current verification guarantees and would surprise existing scripted use.
  - Ignore `--no-wait` in automation mode: rejected because the repo already uses it as the explicit no-confirmation execution contract.

## Decision 7: JSON output in automation mode keeps stdout machine-safe

- **Decision**: When automation mode is used together with `--json`, stdout remains reserved for the machine-readable result, while human-oriented logs and progress go to stderr or are suppressed.
- **Rationale**: The clarified spec chose stdout isolation for machine-readable results. The root command already separates `stdout` and `stderr`, and current logging goes through the configured logger, so this can be enforced without redefining the entire rendering stack.
- **Alternatives considered**:
  - Allow mixed stdout as long as JSON appears somewhere: rejected because machine parsing becomes fragile.
  - Suppress all logs in automation mode: rejected because stderr diagnostics remain useful for operators and CI troubleshooting.

## Decision 8: Build on the current repository-native seams for rollout

- **Decision**: Use `cmd/root.go` for the flag and config binding, `cmd/command_contract.go` and `cmd/capabilities.go` for discovery and support metadata, `cmd/cmd_cli.go` plus paging helpers for prompt control, and `cmd/cmd_views_contract.go` for JSON-envelope behavior.
- **Rationale**: These seams already hold the behavior that the feature needs to extend. Reusing them keeps the change incremental and consistent with the repo’s current patterns.
- **Alternatives considered**:
  - Add a new automation package under `internal/`: rejected because the behavior is CLI-surface policy, not a separate backend subsystem.
  - Push automation logic directly into each command only: rejected because it would duplicate behavior and drift across families.

## Decision 9: Start with representative supported command families and explicit unsupported reporting

- **Decision**: Prioritize representative command families that already expose the shared machine contract or prompt seams: `capabilities`, `get process-instance`, `run process-instance`, `deploy process-definition`, `delete process-instance`, `cancel process-instance`, plus explicit supported or unsupported treatment for `expect` and `walk`.
- **Rationale**: The spec requires representative read and write coverage rather than a whole-repo conversion. These commands already sit on the right seams for prompting, waiting, and JSON behavior.
- **Alternatives considered**:
  - Convert every command in one feature: rejected as unnecessarily large for the initial automation contract.
  - Limit the feature to one destructive command only: rejected because the issue calls for one coherent contract, not a single-command exception.

## Decision 10: Documentation must describe the automation flag as the canonical invocation anchor

- **Decision**: Update README guidance, relevant command help text, and generated CLI docs so automation callers see `--automation` as the canonical entry point and understand how it interacts with `--json`, `--no-wait`, and current human-oriented modes.
- **Rationale**: The constitution requires docs to match user-visible behavior, and this feature changes the recommended way to invoke the CLI for AI agents and CI.
- **Alternatives considered**:
  - Update only the new flag help text: rejected because the issue is about a repository-level automation contract, not just one more flag.
  - Delay docs until after implementation: rejected because the repo requires docs and behavior to move together.

## Representative Implementation Anchors

| Area | Current anchors | Why they fit the feature |
|--------|-----------------|--------------------------|
| Root flag and config | `cmd/root.go` | owns persistent flags, config binding, and root-level operator guidance |
| Command metadata and discovery | `cmd/command_contract.go`, `cmd/capabilities.go`, `cmd/capabilities_test.go` | already define the machine contract and can be extended with automation support metadata |
| Prompt confirmation | `cmd/cmd_cli.go`, `confirmCmdOrAbort`, `confirmCmdOrAbortFn` | central confirmation seam for state-changing commands and paging continuation |
| Paging continuation | `cmd/get_processinstance.go`, `cmd/get_processinstance_test.go`, `cmd/delete_test.go`, `cmd/cancel_test.go` | already model prompt versus auto-continue behavior and have focused regression coverage |
| Shared JSON envelopes | `cmd/cmd_views_contract.go`, `cmd/cmd_views_rendermode.go` | already decide when to emit shared `succeeded`, `accepted`, `invalid`, and `failed` envelopes |
| State-changing accepted outcomes | `cmd/run_processinstance.go`, `cmd/deploy_processdefinition.go`, `cmd/delete_processinstance.go`, `cmd/cancel_processinstance.go` | already use `--no-wait` and current contract helpers |

## Validation Baseline

- Focused automation-mode validation should start with `go test ./cmd -count=1`.
- Generated docs should be refreshed with `make docs`.
- If README guidance changes, homepage content should be refreshed with `make docs-content`.
- Final repository validation remains `make test`.

## Final Implementation Snapshot

- The repository-native rollout stayed inside the planned seams: `cmd/root.go` owns the flag and config binding, `cmd/command_contract.go` plus `cmd/capabilities.go` own automation discovery truthfulness, `cmd/cmd_cli.go` owns shared prompt gating, and the representative command families opt in individually.
- The final supported automation boundary is intentionally narrow and explicit: `capabilities`, `get process-instance`, `run process-instance`, `deploy process-definition`, `delete process-instance`, and `cancel process-instance` support the contract; `expect process-instance` and `walk process-instance` remain explicit rejections.
- JSON automation behavior did not require a parallel output model; the existing shared envelope from `#78` remained sufficient once tests proved stdout stayed machine-readable and `--no-wait` continued to map to `accepted`.
- The documentation rollout confirmed the issue goal is best served by one canonical invocation pattern, `--automation` plus `--json` when machine-readable stdout is required, rather than preserving multiple equally recommended non-interactive recipes.
