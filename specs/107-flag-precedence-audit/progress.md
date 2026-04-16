# Ralph Progress Log

Feature: 107-flag-precedence-audit
Started: 2026-04-16 21:31:31

## Codebase Patterns

- `cmd/root.go` constructs effective config from a fresh `viper.New()` instance during `PersistentPreRunE`; any command-local config-backed bindings that use `viper.GetViper()` are outside the shared bootstrap resolver.
- `config.Config.WithProfile()` currently replaces `App`, `Auth`, `APIs`, and `HTTP` wholesale, then selectively backfills a few auth fields; follow-up work should preserve field-level winners instead of relying on section replacement.
- Existing CLI bootstrap/failure-model coverage lives primarily in `cmd/config_test.go` and `cmd/bootstrap_errors_test.go`; config precedence coverage does not yet have a dedicated `config/config_test.go` seam.

---

## Iteration 1 - 2026-04-16 22:01:00
**User Story**: Phase 1 Setup - inventory precedence seams and existing regression coverage
**Tasks Completed**:
- [x] T001: Inventory config-backed command paths and current precedence seams
- [x] T002: Inspect existing config/bootstrap regression seams for tenant, profile, and shared failure handling
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/107-flag-precedence-audit/research.md
- specs/107-flag-precedence-audit/quickstart.md
- specs/107-flag-precedence-audit/tasks.md
- specs/107-flag-precedence-audit/progress.md
**Learnings**:
- The clearest mixed-source drift is shared backoff flags, because the command families bind them through the global Viper singleton while root bootstrap reads from a fresh Viper instance.
- Current tests are strong on failure normalization and command execution scaffolding but weak on explicit precedence assertions, so the next iteration should add shared config/bootstrap tests before command-by-command cleanup.
---
