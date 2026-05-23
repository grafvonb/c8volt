# Progress: Run Confirmation Observes Real Process Instance States

**Issue**: [#225](https://github.com/grafvonb/c8volt/issues/225)
**Branch**: `225-run-observable-keys`
**Implementation Context**: Use `--implementation-context specs/ralph-implementation-rules.md` for every Ralph iteration.

## Status

- Specification generated: complete
- Clarification gate: complete; no critical ambiguities detected worth formal clarification
- Architecture grounding: reused existing architecture memory; no refresh needed
- Planning artifacts: complete
- Task generation: complete
- Ralph launch: pending explicit budget confirmation

## Codebase Patterns

- `internal/services/processinstance/v87`, `v88`, and `v89` own version-specific process-instance creation and wait behavior.
- `internal/services/processinstance/waiter` owns shared polling, absent/not-found handling, backoff, fail-fast worker fan-out, and state expectation checks.
- `cmd/run_processinstance.go` owns `run pi` flags, validation, facade calls, and result rendering dispatch.
- `cmd/cmd_views_get.go` already renders `process.ProcessInstances` in normal, JSON, and keys-only modes through shared helpers.
- `cmd/cmd_views_contract.go` renders shared JSON envelopes for full-contract commands and intentionally does not render non-JSON command results.
- `README.md`, `cmd/*` help text, `docsgen`, and generated `docs/cli/*` must stay aligned for user-facing command changes.

## Validation Log

- Not run yet; implementation tasks are pending.

## Residual Risks

- The exact v8.7 behavior when creation returns no usable process instance key must be preserved.
- Keys-only mode must not emit human summaries or activity text on stdout.
- `expect pi` strictness must remain separate from run confirmation semantics.
