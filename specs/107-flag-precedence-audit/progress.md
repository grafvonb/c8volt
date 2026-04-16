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

## Iteration 3 - 2026-04-16 21:52:58 CEST
**User Story**: User Story 1 - Resolve Effective Values Consistently
**Tasks Completed**:
- [x] T007: Add root bootstrap precedence tests for tenant, profile selection, and config-file loading
- [x] T008: Add config-level precedence and overlay tests for active profile, API base URLs, auth mode, and auth credentials/scopes
- [x] T009: Add command regression tests that verify baseline settings resolve consistently in representative get, deploy, run, and walk flows
- [x] T010: Implement the authoritative effective-config resolution flow for env-backed oauth2 scope maps and per-entry profile overlay
- [x] T011: Align baseline setting resolution with the shared resolver for root/profile/env combinations
- [x] T012: Verify and normalize baseline setting consumption across bootstrap and representative command execution paths
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- config/config.go
- config/config_test.go
- cmd/bootstrap_errors_test.go
- cmd/config_test.go
- cmd/get_test.go
- cmd/deploy_test.go
- cmd/run_test.go
- cmd/walk_test.go
- specs/107-flag-precedence-audit/tasks.md
- specs/107-flag-precedence-audit/progress.md
**Learnings**:
- Env-backed nested map entries such as `auth.oauth2.scopes.<api>` do not populate through the existing struct-only env binding pass, so the shared resolver needs an explicit env overlay step for known scope keys.
- Profile overlay for map-backed settings should resolve per entry, not as a whole-map replacement, otherwise a profile scope map can stomp higher-precedence env winners for individual keys.
- Fresh helper-process command tests should avoid the generic shared-flag reset before `SetArgs()` when `StringSlice` flags are involved; clearing the specific globals is safer than round-tripping `"[]"` defaults into argv parsing.
---

## Iteration 4 - 2026-04-17 00:12:00 CEST
**User Story**: User Story 2 - Preserve Correctness Across Command Types and Edge Cases
**Tasks Completed**:
- [x] T013: Add command-local precedence regression tests for shared backoff/config-backed flags
- [x] T014: Add edge-case tests for explicit empty/zero values and non-fallback behavior
- [x] T015: Add subprocess or shared-failure-model tests for ambiguous-precedence validation failures
- [x] T016: Implement shared command-local binding and precedence handling for config-backed flag packs
- [x] T017: Implement explicit ambiguity and invalid-value failure handling
- [x] T018: Normalize zero/empty-value preservation and non-fallback behavior
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/bootstrap_errors.go
- cmd/bootstrap_errors_test.go
- cmd/cmd_errors.go
- cmd/cmd_subprocess_scope_test.go
- cmd/config_test.go
- cmd/get_test.go
- cmd/cancel_test.go
- cmd/delete_test.go
- cmd/deploy_test.go
- cmd/expect_test.go
- cmd/root.go
- cmd/run_test.go
- config/api.go
- config/app.go
- config/app_test.go
- config/auth.go
- config/backoff.go
- config/config.go
- config/config_test.go
- specs/107-flag-precedence-audit/tasks.md
- specs/107-flag-precedence-audit/progress.md
**Learnings**:
- Empty environment variables only participate in Viper precedence when `AllowEmptyEnv(true)` is enabled; without that, explicit empty env inputs silently fall back to lower-precedence config.
- Source-aware normalization needs to distinguish unset values from explicitly configured zero/empty values before applying defaults or derived fallbacks such as inherited API base URLs.
- Unknown active profiles should bypass the blanket bootstrap local-precondition wrapper so the shared bootstrap mapper can classify them as invalid input.
---
