# Ralph Progress Log

Feature: 107-flag-precedence-audit
Started: 2026-04-16 21:31:31

## Codebase Patterns

- The authoritative precedence seam now runs through `config.ResolveEffectiveConfig(...)`; bootstrap should pass source-awareness callbacks into that resolver instead of re-implementing merge logic in `cmd/root.go`.
- Shared config-backed flag packs should only define flags locally; `cmd/root.go::initViper` is responsible for binding those flags and defaults into the bootstrap-scoped Viper instance.
- Profile overlays can preserve explicit flag and env winners by checking canonical config keys against `v.InConfig("profiles.<name>.<key>")` for profile presence and tracked flag/env sources for higher-precedence winners.
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

## Iteration 2 - 2026-04-16 23:10:00
**User Story**: Phase 2 Foundational - shared precedence resolver and profile overlay
**Tasks Completed**:
- [x] T003: Define the authoritative precedence contract and effective-config resolver seams
- [x] T004: Refactor profile application into a lower-precedence field overlay
- [x] T005: Normalize shared command-local config-backed bindings into the same resolver path
- [x] T006: Add or update shared config/bootstrap helper coverage for the normalized resolver
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel.go
- cmd/cmd_flagpacks.go
- cmd/config_test.go
- cmd/delete.go
- cmd/deploy.go
- cmd/expect.go
- cmd/get.go
- cmd/root.go
- cmd/run.go
- config/app_test.go
- config/config.go
- config/config_test.go
- specs/107-flag-precedence-audit/contracts/config-precedence.md
- specs/107-flag-precedence-audit/tasks.md
- specs/107-flag-precedence-audit/progress.md
**Learnings**:
- A command-local config-backed flag can stay repository-native without the global singleton by letting command init add only the flag definitions and deferring all binding to `initViper`.
- In-process Cobra tests need explicit flag reset cleanup after mutating shared root flags; otherwise later executions inherit stale bootstrap values such as `--profile`.
- The foundational phase is green with `go test ./config ./cmd -count=1` and `make test`, so follow-up user-story work can build on the shared resolver instead of reopening bootstrap mechanics.
---
