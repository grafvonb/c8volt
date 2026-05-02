# Research: Define Machine-Readable CLI Contracts

## Decision 1: Add one dedicated top-level discovery command

- **Decision**: Add one dedicated top-level machine-readable discovery command, with `c8volt capabilities --json` as the planned contract surface.
- **Rationale**: The clarified spec explicitly selected one top-level entry point. This fits the current Cobra tree cleanly, avoids overloading human help text, and gives automation a stable place to discover supported commands, flags, output modes, and mutation behavior.
- **Alternatives considered**:
  - Add discovery to `help --json`: rejected because it mixes human-oriented help concerns with machine contract concerns and makes the contract depend on documentation formatting.
  - Add only per-command `--schema` modes: rejected because automation still needs one aggregated entry point before it knows which command to inspect.
  - Add both aggregated and per-command discovery immediately: rejected for the initial feature slice because one aggregated command is enough to define the contract foundation.

## Decision 2: Reuse existing command payload models inside a shared result envelope

- **Decision**: Wrap existing JSON payloads from current command families in one shared top-level result envelope rather than replacing each command’s domain payload schema.
- **Rationale**: The repository already exposes stable JSON models such as `process.ProcessInstances`, `process.CancelReports`, `process.DeleteReports`, `resource.Resource`, and build info maps. Reusing those models minimizes churn and lets the machine contract standardize only the shared metadata that automation needs across commands.
- **Alternatives considered**:
  - Keep each command-family JSON completely independent and document semantics separately: rejected because it leaves automation to infer too much from family-specific shapes.
  - Replace each family’s payload with a brand-new contract-specific schema: rejected because it duplicates existing public output models and introduces unnecessary migration risk.

## Decision 3: Standardize the shared envelope around four explicit outcomes

- **Decision**: Standardize the machine result envelope on exactly four outcomes: `succeeded`, `accepted`, `invalid`, and `failed`.
- **Rationale**: The clarified spec selected those four categories. They cleanly separate confirmed success, accepted-but-not-yet-confirmed work, caller-correctable validation/input issues, and non-validation execution failures without overfitting the contract to one command family.
- **Alternatives considered**:
  - Use only `success` and `error`: rejected because it hides the accepted-work distinction that matters for `--no-wait` flows and automation retries.
  - Treat validation as a subtype of generic error: rejected because the issue explicitly calls for actionable validation behavior for non-human callers.
  - Add more granular states such as `warning` or `partial`: rejected for the initial contract because they add ambiguity without a confirmed cross-family need.

## Decision 4: Keep exit codes authoritative and align the envelope to `ferrors`

- **Decision**: Preserve the current process exit code as the authoritative coarse-grained signal and align the JSON envelope with the existing `c8volt/ferrors` failure model.
- **Rationale**: The repository already uses `ferrors.Classify`, `ExitCode`, and `ResolveExitCode` as stable CLI behavior. Keeping exit codes authoritative preserves script compatibility and lets the JSON envelope add detailed machine-readable semantics without creating a second conflicting signaling system.
- **Alternatives considered**:
  - Make JSON mode always exit zero and rely only on the envelope: rejected because it would break current script-safe behavior and contradict repository conventions.
  - Let exit codes and JSON outcomes disagree by command family: rejected because automation would have to guess which signal is real.

## Decision 5: Map existing failure classes into the shared outcome vocabulary

- **Decision**: Map current `ferrors` classes into the clarified outcome vocabulary as follows:
  - no error + confirmed completion -> `succeeded`
  - no error + intentionally unconfirmed state-changing work (`--no-wait` style flows) -> `accepted`
  - `ClassInvalidInput` -> `invalid`
  - `ClassLocalPrecondition`, `ClassUnsupported`, `ClassNotFound`, `ClassConflict`, `ClassTimeout`, `ClassUnavailable`, `ClassMalformedResponse`, and `ClassInternal` -> `failed`
- **Rationale**: This preserves the existing class granularity for machine detail while keeping the shared envelope vocabulary small and stable. `invalid` remains reserved for caller-correctable invocation issues, while the broader execution and environment failures stay under `failed`.
- **Alternatives considered**:
  - Treat local preconditions as `invalid`: rejected because missing config, authentication bootstrap issues, and other local-precondition problems are not the same as flag or input validation errors in the current CLI model.
  - Add one envelope outcome per `ferrors.Class`: rejected because it would make the top-level vocabulary larger and less predictable than the clarified spec allows.

## Decision 6: Represent contract support explicitly in discovery

- **Decision**: Every command listed in discovery should expose a contract support state: `full`, `limited`, or `unsupported`.
- **Rationale**: The clarified spec selected visibility over hiding unsupported commands. This lets agents explore the real command tree while safely preferring commands that fully support the shared machine contract.
- **Alternatives considered**:
  - Hide unsupported commands from discovery: rejected because the agent would get an incomplete picture of the CLI.
  - Allow undocumented ad hoc JSON on unsupported commands: rejected because it weakens the contract and makes support impossible to test consistently.

## Decision 7: Use repository-native metadata sources for discovery

- **Decision**: Build the discovery command from the existing Cobra command tree plus small command-local metadata additions for output modes, contract support, and state-changing/read-only classification.
- **Rationale**: Cobra already owns command names, hierarchy, help text, and flags. Using it as the discovery backbone keeps the feature aligned with the repo’s existing command structure and reduces the chance that help text and machine metadata drift apart.
- **Alternatives considered**:
  - Maintain a completely separate manual registry of commands: rejected because it would drift from the Cobra tree and duplicate command metadata.
  - Infer everything from docs files: rejected because generated docs lag the code and are the wrong source of truth for command behavior.

## Decision 8: Start with representative command-family coverage, not universal rollout

- **Decision**: Implement and prove the contract on at least one representative command from each required family: `get`, `run`, `expect`, `walk`, `deploy`, `delete`, and `cancel`.
- **Rationale**: The feature spec explicitly selected representative coverage as the acceptance target. This keeps the initial slice incremental while still defining a reusable contract that can expand later.
- **Alternatives considered**:
  - Convert every command in one feature: rejected as too large and unnecessary for the initial contract foundation.
  - Limit the feature to one or two families: rejected because the issue explicitly names all seven representative families.

## Decision 9: Treat existing JSON helpers as the main implementation seam

- **Decision**: Use `cmd/cmd_views_rendermode.go`, existing `itemView`/`listOrJSON` patterns, and command-local view/render calls as the main seam for introducing the shared machine envelope.
- **Rationale**: Current `--json` behavior is already centralized around a small number of helpers and command-local render functions. Extending those seams is the smallest repository-native way to add the shared envelope without rewriting each command family from scratch.
- **Alternatives considered**:
  - Add one entirely separate output pipeline only for machine mode: rejected because it duplicates current rendering behavior.
  - Inject the envelope in `toolx.ToJSONString`: rejected because `ToJSONString` is a generic serialization helper, not a command-contract boundary.

## Decision 10: Update both operator docs and generated command docs

- **Decision**: Update README automation guidance and any relevant human-facing docs, then regenerate `docs/cli/` with `make docs`; if README content changes that should flow into docs homepage content, also run `make docs-content`.
- **Rationale**: The constitution and AGENTS guidance require documentation updates for user-visible CLI behavior changes. A new top-level discovery command and a recommended automation contract are directly user-visible.
- **Alternatives considered**:
  - Update only the new discovery command help text: rejected because the feature also changes the recommended automation usage guidance.
  - Delay docs until implementation stabilizes: rejected because the repo requires user-facing docs to move with behavior changes.

## Decision 11: Preserve the final verification sequence as the implementation handoff baseline

- **Decision**: Treat `go test ./c8volt/ferrors -count=1`, `go test ./cmd -count=1`, `make docs`, `make docs-content`, and `make test` as the settled validation sequence for this feature.
- **Rationale**: The final contract spans shared outcome mapping, representative CLI behavior, generated docs, and repository-wide compatibility. Keeping the exact sequence in the feature artifacts makes later follow-up work validate the same contract surface instead of inventing a narrower gate.
- **Alternatives considered**:
  - Rely on only `make test`: rejected because it would not explicitly capture the docs regeneration and focused contract checks that define this feature's rollout boundary.
  - Keep the validation guidance only in `tasks.md`: rejected because the handoff notes should survive task completion and remain visible in the design artifacts.

## Representative Contract Anchors

| Family | Current anchors | Why they are the right starting seams |
|--------|-----------------|----------------------------------------|
| `get` | `cmd/get_processinstance.go`, `cmd/get_resource.go`, `cmd/get_processdefinition.go`, `cmd/get_test.go` | already use JSON render helpers and expose both list and item payloads |
| `run` | `cmd/run_processinstance.go`, `cmd/run_test.go` | already distinguishes `--no-wait` behavior and exercises state-changing success paths |
| `expect` | `cmd/expect_processinstance.go`, `cmd/expect_test.go` | provides a confirmed-completion waiting command that should always resolve to `succeeded` or `failed` |
| `walk` | `cmd/walk_processinstance.go`, `cmd/cmd_views_walk.go`, `cmd/walk_test.go` | exercises a read-only tree/list command family with custom render modes |
| `deploy` | `cmd/deploy_processdefinition.go`, `cmd/embed_deploy.go`, `cmd/deploy_test.go` | exercises deployment plus optional run/no-wait behavior |
| `delete` | `cmd/delete_processinstance.go`, `cmd/delete_processdefinition.go`, `cmd/delete_test.go` | already returns report-style payloads and no-wait variants |
| `cancel` | `cmd/cancel_processinstance.go`, `cmd/cancel_test.go` | already returns report-style payloads and no-wait variants |
| shared exit/error alignment | `c8volt/ferrors/errors.go`, `c8volt/ferrors/errors_test.go` | protects the process-level signal the new machine contract must preserve |

## Setup Inventory: Current Cobra Tree and Machine-Facing Seams

### T001: Root command tree, inherited flags, and shared render seams

- `cmd/root.go` is the authoritative top-level command registry and root metadata source. It already registers stable top-level families through `rootCmd.AddCommand(...)`, including `get`, `run`, `expect`, `walk`, `deploy`, `delete`, `cancel`, `config`, `embed`, and `version`.
- The root command already exposes operator-safe concise metadata that can seed a discovery surface: `Use`, `Short`, `Long`, aliases on child commands, and real flag definitions through Cobra.
- The machine-relevant inherited root flags are already centralized as persistent flags on `rootCmd`: `--json`, `--keys-only`, `--quiet`, `--verbose`, `--debug`, `--auto-confirm`, `--tenant`, `--config`, `--profile`, `--log-level`, `--log-format`, `--log-with-source`, and `--no-err-codes`.
- Completion behavior is already normalized at the root with `CompletionOptions.HiddenDefaultCmd = true`, which matters for keeping any future discovery command out of ad hoc completion filtering logic.
- Bootstrap and config normalization happen centrally in `PersistentPreRunE`, so any machine-contract command added at the root needs to decide whether it should participate in full bootstrap or bypass it the same way other metadata-style commands do.
- `cmd/cmd_views_rendermode.go` is the existing shared output-mode seam. `pickMode()` maps global flags to `json`, `keys-only`, `tree`, and the default `one-line` mode, while `itemView(...)` and `listOrJSON(...)` already centralize the JSON vs. line-oriented branching for many read flows.
- `cmd/cmd_views_get.go` shows the current repository-native JSON wrapping pattern: most `get` flows reuse public payload models directly, but some JSON responses already add command-local metadata wrappers such as the `with-age` JSON meta structure. That makes `cmd/` the right place to add a shared machine envelope without replacing public payload types.

### T002: Representative command-family payload and outcome seams

- `cmd/get_processinstance.go` is the strongest read-only contract anchor. It already supports inherited `--json` and `--keys-only`, uses shared paging/search helpers, and renders via `listProcessInstancesView(...)`.
- `cmd/run_processinstance.go` is the clearest accepted-versus-succeeded anchor. It already models `--no-wait`, validates machine-relevant input errors through `ferrors.HandleAndExit(...)`, and delegates success semantics to shared service behavior without custom rendering.
- `cmd/expect_processinstance.go` is a confirmed-completion wait seam. It has required `--key` and `--state` arguments, routes validation failures through `ferrors`, and currently reports success through logs rather than a structured payload.
- `cmd/walk_processinstance.go` is the custom render-mode outlier. It supports `--tree` in addition to the inherited modes and switches among ancestry, descendants, and family views through command-local render functions.
- `cmd/deploy_processdefinition.go` already exposes `--no-wait` and an optional `--run` follow-up, making it the natural deployment-family anchor for `accepted` outcomes when work is intentionally not yet confirmed.
- `cmd/delete_processinstance.go` and `cmd/cancel_processinstance.go` already reuse the process-instance search filters from `get`, share the dry-run dependency-expansion flow, prompt before destructive execution, and expose `--no-wait`. They are strong anchors for state-changing contract rollout because their operational payloads and confirmation semantics are already explicit.
- Across all representative families, caller-correctable validation paths already terminate through `ferrors.HandleAndExit(...)` with repository-native helpers such as `invalidFlagValuef`, `missingDependentFlagsf`, `mutuallyExclusiveFlagsf`, and `forbiddenFlagCombinationf`. That existing error-class vocabulary should drive the machine envelope rather than a new parallel classification layer.
- The representative family commands do not yet share one explicit machine-result envelope. Current machine-facing behavior is split between direct JSON payload rendering on read flows and implicit success through logs / exit codes on state-changing flows, which confirms the need for Phase 2 foundational work.

### T003: Current automation-facing documentation and generated anchors

- `README.md` is the main operator-and-automation narrative source. It already advertises automation-friendly flags such as `--json`, `--keys-only`, `--auto-confirm`, `--workers`, and `--no-wait`, so it is the correct place to describe the future machine-contract recommendation.
- `docs/index.md` is generated from the README content and therefore should not be hand-edited for this feature; any homepage-level machine-contract guidance must land in `README.md` first and then flow through `make docs-content`.
- `docs/index.md` already contains the strongest script/CI positioning and explicitly calls out automation-friendly flags. It is the best user-facing anchor for explaining why `capabilities --json` and the shared result envelope are the supported machine surface.
- `docs/cli/index.md` is the generated entry page for Cobra reference output and already frames the CLI by workflow families and common flags. It is the right generated anchor for surfacing any new top-level discovery command once Cobra metadata is updated.
- The current generated CLI reference already has per-command anchors for every representative family needed by the spec: `c8volt_get_process-instance.md`, `c8volt_run_process-instance.md`, `c8volt_expect_process-instance.md`, `c8volt_walk_process-instance.md`, `c8volt_deploy_process-definition.md`, `c8volt_delete_process-instance.md`, and `c8volt_cancel_process-instance.md`, plus the family index pages and root reference page. This means the docs pipeline already covers the feature’s discovery scope without introducing a parallel documentation structure.

## Resolved Planning Unknowns

- No unresolved technical-context clarifications remain for planning.
- The initial discovery surface is a dedicated top-level command, not an extension of help output.
- The initial machine contract uses one shared envelope and existing family payloads.
- Commands without full support remain visible in discovery with explicit support status.
- Exit codes remain authoritative even in machine-readable mode.
- The feature's final implementation and verification baseline has been exercised successfully on 2026-04-17 through the focused suites, docs regeneration, and `make test`.
