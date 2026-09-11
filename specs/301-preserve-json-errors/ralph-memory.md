# Ralph Memory

Feature: 301-preserve-json-errors
Started: 2026-09-11T13:45:51Z

## Codebase Patterns

- Command paths that can call `os.Exit` are tested with `testx.RunCmdSubprocessInDirWithSeparateOutputs`; bind the subprocess to the exact test through `testx.CmdSubprocessNameEnv` and send real CLI arguments through `Execute()`.
- Full-contract execution failures must use `handleCommandError` with the actual Cobra command. It preserves JSON precedence, normalized classification, human stderr, and `--no-err-codes` while terminating immediately.
- `readKeysIfDash` removes blank lines before `validateKeys`, so invalid stdin indexes describe the processed key slice rather than physical input lines.

## Decisions

- Shared error-envelope assertions use independent test-only wire structs and raw JSON field maps so tests do not derive expectations from production rendering.
- US1 retains every stdin caller's existing `.Unique()` placement and changes only command context/error dispatch.

## Gotchas

- `--no-err-codes` exits zero but still must terminate; request-count assertions are required to catch accidental continuation.
- JSON errors omit `payload`, `suggestion`, and unavailable `tenantContext`; their absence is part of the contract.
- Read-only GET fixtures may emit transient retry context on stderr before the final failure; JSON assertions must prohibit the final human diagnostic without rejecting that established context.

## Reusable Commands

- `go test ./cmd -run '^TestCommandErrorEnvelope(DeleteValidation|Stdin)' -count=1`
- `go test ./cmd -run '^TestCommandErrorEnvelope(Cluster|ProcessDefinition|EmbedList)|TestGetCluster|TestGetProcessDefinition|TestEmbedList' -count=1`
- `go test ./cmd -count=1`

## Do Not Repeat

- Do not route eligible full-contract execution errors directly through `ferrors.HandleAndExit`; that bypasses the JSON envelope.
- Do not use a production `ResultEnvelope` to decode or construct expected regression results.

## Current Handoff
- Continue with T017 in US3: add corrected-path output-mode compatibility coverage before the final production audit.
