# Progress: Ops Command Foundation

## Codebase Patterns

- Grouping commands live as focused files in `cmd/` and attach to `rootCmd` or a parent command during `init`.
- Existing grouping commands such as `get` and `resolve` return `cmd.Help()` and set command mutation metadata through `setCommandMutation`.
- Machine-readable discovery metadata is owned by `cmd/command_contract.go` and verified in `cmd/capabilities_test.go`.
- Top-level help discovery expectations live in `cmd/root_test.go`.
- Generated CLI documentation is refreshed with `make docs-content`; generated docs should not be hand-edited.
- Ralph implementation iterations for this feature must read `specs/ralph-implementation-rules.md` and must be launched with `--implementation-context specs/ralph-implementation-rules.md`.
- Grouping command examples in `cmd/get.go`, `cmd/resolve.go`, and `cmd/run.go` use package-level `*cobra.Command` values with `Use`, `Short`, `Long`, `Example`, aliases, optional suggestions, `Args` when needed, and `RunE` returning `cmd.Help()`.
- Existing top-level grouping commands call `addBackoffFlagsAndBindings` and then set read-only or state-changing metadata with `setCommandMutation`; contract support is normally inferred from discoverable children unless explicitly annotated.
- Capability discovery is generated from the live Cobra tree, excludes hidden/help/completion commands, clones aliases, resolves mutation/support annotations, and serializes visible command flags.
- Root help tests assert substrings with `assertHelpOutputContainsAll`/`assertHelpOutputOmitsAll`; generated markdown alignment tests render live Cobra commands with `renderMarkdownForCommand`.
- Docs generator tests create a temporary markdown tree with `doc.GenMarkdownTreeCustom` and inspect generated command files by filename.
- Grouping commands inherit global output modes such as `json` and `keys-only` in capability discovery, but unsupported contract support keeps JSON from being machine-preferred.
- Root grouping help should avoid example invocations for child commands until those children are registered, so help stays truthful at each independently delivered story.
- Help-path tests can prove config bypass by setting invalid runtime config environment values; `PersistentPreRunE` returns before config normalization when a help flag is present.
- Child grouping commands attach to their parent command during package init, set their own mutation metadata, and should avoid naming unavailable future playbook commands in help copy.
- Tests for absent top-level target flags should inspect the command's local and persistent flag sets directly when inherited global flags such as `--keys-only` would make help substring checks ambiguous.
- Shared ops workflow contracts live in `cmd/ops_contract.go` as command-layer report/status vocabulary only; resource-specific API traversal, mutation, polling, and generated-client behavior must stay below `cmd`.

## Work Log

- 2026-05-10: Created issue-backed Speckit feature for GitHub issue #197.
- 2026-05-10: Completed clarification scan; no critical ambiguities required formal user questions.
- 2026-05-10: Generated plan, research, data model, command contract, quickstart, and tasks artifacts.

---
## Iteration 1 - 2026-05-10 22:11:14 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Inspect grouping command patterns in `cmd/get.go`, `cmd/resolve.go`, and `cmd/run.go`
- [x] T002: Inspect command metadata helpers in `cmd/command_contract.go` and existing expectations in `cmd/capabilities_test.go`
- [x] T003: Inspect top-level help and generated markdown tests in `cmd/root_test.go` and `docsgen/main_test.go`
- [x] T004: Record any reusable implementation discoveries in `specs/197-ops-command-foundation/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/197-ops-command-foundation/tasks.md
- specs/197-ops-command-foundation/progress.md
**Learnings**:
- `ops` should follow the existing grouping-command shape and defer concrete behavior to children.
- Capabilities assertions should inspect `CommandCapability` values from the live command tree rather than hard-coding serialized JSON.
- Root help and generated markdown tests already provide shared helpers for future ops discovery assertions.
---

---
## Iteration 2 - 2026-05-10 22:15:04 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T005: Add `ops` root grouping command registration, help text, examples, aliases if warranted, and mutation metadata in `cmd/ops.go`
- [x] T006: Add base ops command help tests in `cmd/ops_test.go`
- [x] T007: Add ops root discovery metadata assertions in `cmd/capabilities_test.go`
- [x] T008: Update root help discovery expectations for `ops` in `cmd/root_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/ops.go
- cmd/ops_test.go
- cmd/capabilities_test.go
- cmd/root_test.go
- specs/197-ops-command-foundation/tasks.md
- specs/197-ops-command-foundation/progress.md
**Learnings**:
- `ops` follows existing root grouping command patterns: package-level Cobra command, `RunE` returning help, hidden backoff flags, and explicit state-changing mutation metadata.
- Root-level capability discovery inherits global output modes; unsupported contract support prevents those modes from being marked machine-preferred.
- Targeted validation passed with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapabilitiesCommand_JSONIncludesOpsRootMetadata|TestRootHelp' -count=1`.
---

---
## Iteration 3 - 2026-05-10 22:18:18 CEST
**User Story**: US1 Discover Ops Command Family
**Tasks Completed**:
- [x] T009: Add/extend tests proving `c8volt ops --help` succeeds without runtime config in `cmd/ops_test.go`
- [x] T010: Add/extend tests proving `ops` appears in root help while existing top-level commands remain discoverable in `cmd/root_test.go`
- [x] T011: Finalize `ops` help copy and grouping behavior in `cmd/ops.go`
- [x] T012: Run targeted validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestRootHelp' -count=1`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/ops.go
- cmd/ops_test.go
- cmd/root_test.go
- specs/197-ops-command-foundation/tasks.md
- specs/197-ops-command-foundation/progress.md
**Learnings**:
- Root ops help should mention future target-specific subcommands without listing unavailable child-command examples before those children exist.
- `ops --help` can be validated without usable runtime configuration by pairing a temporary config-free environment with an invalid Camunda version env var.
- Targeted validation passed with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestRootHelp' -count=1`.
---

---
## Iteration 4 - 2026-05-10 22:21:17 CEST
**User Story**: US2 Discover Execute Grouping Command
**Tasks Completed**:
- [x] T013: Add execute grouping help tests in `cmd/ops_test.go`
- [x] T014: Add capabilities assertions for `ops execute` in `cmd/capabilities_test.go`
- [x] T015: Add `ops execute` grouping command registration, help text, examples, and metadata in `cmd/ops_execute.go`
- [x] T016: Ensure no concrete execute playbook commands are registered in `cmd/ops_execute.go`
- [x] T017: Run targeted validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapability.*Ops' -count=1`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/ops_execute.go
- cmd/ops_test.go
- cmd/capabilities_test.go
- specs/197-ops-command-foundation/tasks.md
- specs/197-ops-command-foundation/progress.md
**Learnings**:
- `ops execute` stays grouping-only by registering no children and returning Cobra help for both `ops execute` and `ops execute --help`.
- Help text should describe the category of future playbooks without spelling out unavailable concrete workflow command names.
- Targeted validation passed with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapability.*Ops' -count=1`.
---

---
## Iteration 5 - 2026-05-10 22:24:49 CEST
**User Story**: US3 Discover Repair Grouping Command
**Tasks Completed**:
- [x] T018: Add repair grouping help tests, including no top-level `--key`, in `cmd/ops_test.go`
- [x] T019: Add capabilities assertions for `ops repair` in `cmd/capabilities_test.go`
- [x] T020: Add `ops repair` grouping command registration, help text, examples, and metadata in `cmd/ops_repair.go`
- [x] T021: Ensure `cmd/ops_repair.go` defines no ambiguous top-level `--key` flag
- [x] T022: Run targeted validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapability.*Ops' -count=1`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/ops_repair.go
- cmd/ops_test.go
- cmd/capabilities_test.go
- specs/197-ops-command-foundation/tasks.md
- specs/197-ops-command-foundation/progress.md
**Learnings**:
- `ops repair` follows the existing child grouping command pattern: parent registration during init, state-changing mutation metadata, no concrete children, and no workflow execution.
- Local flag-set assertions are the precise way to guard against a future ambiguous repair `--key` while allowing inherited global output flags.
- Targeted validation passed with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapability.*Ops' -count=1`.
---

---
## Iteration 6 - 2026-05-10 22:28:25 CEST
**User Story**: US4 Establish Shared Ops Workflow Contracts
**Tasks Completed**:
- [x] T023: Add tests for shared ops step status values and report-format behavior in `cmd/ops_contract_test.go`
- [x] T024: Add command contract tests proving grouping commands do not claim full automation support in `cmd/capabilities_test.go`
- [x] T025: Add lightweight shared ops workflow/report contract types and comments in `cmd/ops_contract.go`
- [x] T026: Keep resource-specific API logic out of `cmd/ops_contract.go` and record that boundary in `specs/197-ops-command-foundation/progress.md`
- [x] T027: Run targeted validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapability.*Ops' -count=1`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/ops_contract.go
- cmd/ops_contract_test.go
- cmd/capabilities_test.go
- specs/197-ops-command-foundation/tasks.md
- specs/197-ops-command-foundation/progress.md
**Learnings**:
- Shared ops step statuses are stable command-layer report tokens: `planned`, `skipped`, `submitted`, `confirmed`, `confirmation_failed`, `blocked`, and `failed`.
- Report format inference is intentionally narrow: explicit valid formats win, `.json` selects JSON, `.md`/`.markdown`/empty paths select Markdown, and unsupported extensions fail fast.
- Targeted validation passed with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*Ops|TestCapability.*Ops' -count=1`.
---
