# Progress: Ops Command Foundation

## Codebase Patterns

- Grouping commands live as focused files in `cmd/` and attach to `rootCmd` or a parent command during `init`.
- Existing grouping commands such as `get` and `resolve` return `cmd.Help()` and set command mutation metadata through `setCommandMutation`.
- Machine-readable discovery metadata is owned by `cmd/command_contract.go` and verified in `cmd/capabilities_test.go`.
- Top-level help discovery expectations live in `cmd/root_test.go`.
- Generated CLI documentation is refreshed with `make docs-content`; generated docs should not be hand-edited.
- Ralph implementation iterations for this feature must read `specs/ralph-implementation-rules.md` and must be launched with `--implementation-context specs/ralph-implementation-rules.md`.

## Work Log

- 2026-05-10: Created issue-backed Speckit feature for GitHub issue #197.
- 2026-05-10: Completed clarification scan; no critical ambiguities required formal user questions.
- 2026-05-10: Generated plan, research, data model, command contract, quickstart, and tasks artifacts.
