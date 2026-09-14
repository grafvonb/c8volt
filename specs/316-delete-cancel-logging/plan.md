# Implementation Plan: Delete and Cancel Logging Consolidation

**Branch**: `codex/316-delete-cancel-logging` | **Date**: 2026-09-13 | **Spec**: [spec.md](spec.md)

## Summary

Fix the existing delete/cancel transcript in place: one polling observation per check, no routine OAuth cache chatter, the required verbose explanations, and concise failures that distinguish submission from confirmation. Bundle tests with each change. The user requested a pragmatic revision of the earlier design; [tasks.md](tasks.md) now contains seven work units.

## Technical Context

Go 1.26 / go1.26.2, existing Cobra/Viper, slog, local HTTP fixtures and command/terminal test helpers. Existing Linux/macOS/Windows targets; terminal proof runs on supported Linux/macOS hosts. No dependencies, flags, storage, mutation behavior or result-schema changes.

The measurable budget remains 36 observations plus 36 HTTP diagnostics for 36 successful single-exchange cached-auth checks, excluding bootstrap/phase records. Preserve tenant logging and HTTP formatting/redaction from #303/#305.

## Approach

1. **Reproduce first.** One command test file exercises the actual child-conflict/root-cancel/ACTIVE/timeout path using short fixed wait settings. Reuse existing fixture and subprocess helpers. Add success and mode cases to that fixture rather than building new test infrastructure.
2. **Consolidate polling locally.** Replace waiter pre-fetch/result chatter with one observation. Keep one last observation per key and at most one pending record until the existing continuation/stop decision is known. Preserve lookup-completion elapsed time; omit next delay on stop. Gate nested adapter messages narrowly while retaining direct lookup diagnostics. Keep all timers, retries, state matching, activity and requests intact, including v87 traversal search.
3. **Remove the OAuth cache pair.** Delete routine cache lookup/hit logging; leave token behavior and useful diagnostics intact.
4. **Repair verbose suppression selectively.** Use existing service transition sites and verbose logging helpers. Ensure effective verbose options honor command human/quiet/automation/machine-mode policy. Keep broad lifecycle suppression and existing semantic progress; restore only prerequisite, escalation, accepted submission, wait policy/scope, and actual resumption explanations. Each explanation has one owner. Do not add a workflow event kind, callback DTO or stage ledger.
5. **Keep failure handling small.** Retain last-state details from existing reads and annotate failure branches with the phase/root and known submission facts. Use a small wrapped error carrying a concise summary or necessary facts plus its original cause only where current errors cannot convey both. Keep original Error() text for machine results. Preserve inspectable causes through the narrow affected ferrors conversion path and freeze existing class selection. Use existing completion FailureDetail for concise warning presentation without changing serialized reports; human formatting belongs in command views. The command error boundary emits the full chain once at DEBUG and uses the original error for JSON and exit status. No public projection model or new completion payload is planned.
6. **Verify and document.** Add the exact 36-check budget case separately from wall-clock timeout assertions. Run existing compatibility coverage, extending only gaps caused by the fix. Update README/source help, regenerate docs and run the full race suite.

## File Ownership

- `internal/services/processinstance/waiter/waiter.go`: observations and locally retained timeout evidence.
- `internal/services/processinstance/v87/service.go`, `v88/service.go`, `v89/service.go`, `v810/service.go`: targeted log gates and existing mutation transition sites.
- `internal/services/auth/oauth2/service.go`: routine cache logging removal.
- `internal/services/common/logging.go`: existing verbose helper; a small shared helper may live in this package if genuinely reused. Adapters must not import their parent factory package.
- `internal/services/processinstance/bulk.go`: existing completion detail path.
- `c8volt/ferrors/errors.go`: only the cause-preservation change needed by affected errors, with exact-text/class tests.
- `cmd/processinstance_mutation_progress.go`, `cmd/cmd_views_contract.go`, proposed `cmd/cmd_views_processinstance_logging.go`: mode gating and presentation. No wait/mutation machinery in commands.
- Proposed `cmd/processinstance_logging_transcript_test.go`: central execution regression; nearby existing tests cover lower-level changes.

Do not create all proposed helper files/types up front. Add a helper only when the implemented fix needs it; keep version-neutral error data in the existing domain ownership if it must cross package boundaries.

## Constitution Check

| Gate | Evidence | Status |
| --- | --- | --- |
| Operational proof | Submission and unconfirmed outcome remain distinct; no wait or mutation change. | Pass |
| Script-safe CLI | Original result schemas/text, exit classes, prompt streams and mode policy preserved. | Pass |
| Validation | Reproduction, targeted regressions, real-terminal coverage and `make test` remain required. | Pass |
| Documentation | README/source metadata updated, docs regenerated. | Pass |
| Repository-native scope | Existing logging/progress/error paths; minimal wrapper only if necessary. | Pass |

Design gates pass; runtime validation is not yet performed. No constitutional exception is requested.

## Risks and Limits

Preserving original causes can expose nested sentinels previously hidden by conversion: prove unchanged class precedence and exact text. Shared waiter changes must preserve direct expect and process-definition-delete callers. Pending observations must flush once on stop without extra requests. Existing family and later single-key waits remain even if they look redundant.

Scope and acceptance remain in [spec.md](spec.md) and [contracts/cli-logging.md](contracts/cli-logging.md). Internal design notes are in [data-model.md](data-model.md); validation steps are in [quickstart.md](quickstart.md). This revision replaces the earlier mandatory workflow/event/evidence architecture.
