# Ralph Memory

Feature: 301-preserve-json-errors
Started: 2026-09-11T13:45:51Z

## Codebase Patterns

- Command paths that can call `os.Exit` are tested with `testx.RunCmdSubprocessInDirWithSeparateOutputs`; bind the subprocess to the exact test through `testx.CmdSubprocessNameEnv` and send real CLI arguments through `Execute()`.
- Successful subprocess helpers that return from `Execute()` must call `os.Exit(0)` afterward so Go's helper-test `PASS` marker cannot contaminate captured command stdout.
- Full-contract execution failures must use `handleCommandError` with the actual Cobra command. It preserves JSON precedence, normalized classification, human stderr, and `--no-err-codes` while terminating immediately.
- `readKeysIfDash` removes blank lines before `validateKeys`, so invalid stdin indexes describe the processed key slice rather than physical input lines.

## Decisions

- Shared error-envelope assertions use independent test-only wire structs and raw JSON field maps so tests do not derive expectations from production rendering.
- US1 retains every stdin caller's existing `.Unique()` placement and changes only command context/error dispatch.
- The final production audit uses commit `6afb3cbe` as the pre-feature baseline; the contract schema, renderer, error normalization/classification, capability metadata, and exit policy are unchanged from that baseline.

## Gotchas

- `--no-err-codes` exits zero but still must terminate; request-count assertions are required to catch accidental continuation.
- JSON errors omit `payload`, `suggestion`, and unavailable `tenantContext`; their absence is part of the contract.
- Read-only GET fixtures may emit transient retry context on stderr before the final failure; JSON assertions must prohibit the final human diagnostic without rejecting that established context.
- The root mode resolver gives JSON precedence over keys-only, while keys-only and quiet alone retain the ordinary stderr error path; unsupported automation is checked before stdin validation in commands that call `requireAutomationSupport` first.
- Non-full fallback coverage should use synthetic limited/unsupported commands for shared stdin validation and real config/embed commands for production compatibility; an HTTP 400 config fixture is deterministic because it avoids transient retry counts.

## Reusable Commands

- `go test ./cmd -run '^TestCommandErrorEnvelope(DeleteValidation|Stdin)' -count=1`
- `go test ./cmd -run '^TestCommandErrorEnvelope(Cluster|ProcessDefinition|EmbedList)|TestGetCluster|TestGetProcessDefinition|TestEmbedList' -count=1`
- `go test ./cmd -run '^TestCommandErrorEnvelopeModes$' -count=1`
- `go test ./cmd -count=1`

## Do Not Repeat

- Do not route eligible full-contract execution errors directly through `ferrors.HandleAndExit`; that bypasses the JSON envelope.
- Do not use a production `ResultEnvelope` to decode or construct expected regression results.

## Current Handoff
- Continue with T022 in Phase 6: update README machine-contract guidance within the documented compatibility boundary.
