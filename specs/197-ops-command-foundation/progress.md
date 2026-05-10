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
