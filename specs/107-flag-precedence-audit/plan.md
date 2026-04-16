# Implementation Plan: Audit and Fix CLI Config Precedence

**Branch**: `107-flag-precedence-audit` | **Date**: 2026-04-16 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/spec.md)
**Input**: Feature specification from `/specs/107-flag-precedence-audit/spec.md`

## Summary

Audit and harden the full CLI configuration-resolution path so every config-backed setting follows one shared contract: `flag > env > profile > base config > default`. The design keeps the existing Cobra and config package layout, fixes precedence at the root bootstrap and config-merge seams instead of patching individual commands, aligns command-local bindings with the same effective resolver, fails explicitly when ambiguous precedence cannot be resolved safely, and verifies the result across every config-backed command path with special focus on tenant, profile selection, API base URLs, auth mode, and auth credentials/scopes.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing config types under `config/`, shared command bootstrap/helpers under `cmd/`, existing environment-binding helpers under `internal/services/common`  
**Storage**: File-based YAML config plus environment variables; no persistent datastore changes  
**Testing**: `go test`, `make test`, config normalization tests under `config/`, command regression and subprocess tests under `cmd/`, targeted bootstrap/config-precedence tests for root persistent and command-local flags  
**Target Platform**: Cross-platform CLI for local and CI use against supported Camunda 8.7 and 8.8 environments  
**Project Type**: CLI  
**Performance Goals**: No user-visible startup regression, no additional network calls in bootstrap, deterministic precedence resolution on every invocation, and no silent fallback to lower-precedence sources when a higher-precedence value is explicitly provided  
**Constraints**: Preserve existing Cobra command surfaces and shared bootstrap error handling, reuse repository-native config and command patterns, audit every config-backed command path, fail explicitly when ambiguous precedence cannot be resolved safely, update relevant CLI/config docs together with implementation, and finish with `make test`  
**Scale/Scope**: Root bootstrap in [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go), config merge and profile handling in [`config/config.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go), app/auth/api config structs in `config/`, shared command flag packs in [`cmd/cmd_flagpacks.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_flagpacks.go), every config-backed command path under `cmd/`, and the relevant user-facing docs in [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md), [`docs/index.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md), and generated `docs/cli/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. This feature is about making configuration behavior predictable and observable; the plan removes hidden precedence drift and requires explicit validation failures when a safe winner cannot be determined.
- **CLI-First, Script-Safe Interfaces**: Pass. The design preserves existing command trees and flag names, keeps resolution inside the shared CLI bootstrap path, and treats precedence outcomes as deterministic command behavior suitable for scripts.
- **Tests and Validation Are Mandatory**: Pass. The plan requires exhaustive audit coverage, targeted config/bootstrap tests, command regression tests for root and command-local flags, and final `make test`.
- **Documentation Matches User Behavior**: Pass. The precedence contract will be documented in shared internal sources and the relevant user-facing CLI/config docs, with generated CLI docs refreshed from Cobra metadata.
- **Small, Compatible, Repository-Native Changes**: Pass. The design strengthens the existing `cmd` and `config` seams instead of introducing a parallel configuration subsystem or a new CLI hierarchy.

## Project Structure

### Documentation (this feature)

```text
specs/107-flag-precedence-audit/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── config-precedence.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── root.go
├── cmd_cli.go
├── cmd_flagpacks.go
├── config_show.go
├── get.go
├── cancel.go
├── delete.go
├── deploy.go
├── expect.go
├── run.go
├── walk.go
├── *_test.go
└── completion_test.go

config/
├── config.go
├── app.go
├── api.go
├── auth.go
├── http.go
├── log.go
└── *_test.go

internal/services/common/
└── envs.go

README.md
docs/
├── index.md
└── cli/
```

**Structure Decision**: Keep the feature inside the current root bootstrap and config-normalization flow. [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go) remains the single entry point for effective config construction, while [`config/config.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go) remains the canonical home for profile-aware normalization. Command-local flag packs and per-command bindings should be normalized into the same resolution path instead of adding command-specific precedence patches.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/research.md).

- Confirm the current precedence hazards in the root bootstrap path, especially the interaction between `viper.Unmarshal`, explicit env binding, and `Config.WithProfile()`.
- Confirm how command-local config-backed flags are currently bound and where global `viper.GetViper()` usage diverges from the fresh `viper.New()` instance created in root bootstrap.
- Confirm the lowest-risk repository-native approach for applying profile values as a lower-precedence overlay rather than a whole-struct replacement that can stomp explicit flag or env values.
- Confirm which command paths and tests already exercise tenant, profile selection, API base URLs, auth mode, auth credentials/scopes, and shared backoff/config settings.
- Confirm the exact user-facing documentation surfaces that must be updated when precedence rules become an explicit operator contract.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/quickstart.md)
- [contracts/config-precedence.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/contracts/config-precedence.md)

- Centralize precedence resolution in the shared bootstrap/config seam so one authoritative effective-config path decides winners for root flags, env vars, profile values, base config values, and defaults.
- Change profile application from whole-section replacement to a field-aware overlay model that preserves higher-precedence flag and env values while still allowing profiles to override base config.
- Normalize command-local config-backed bindings so they participate in the same effective resolver as root persistent flags, removing drift between the root bootstrap `viper.New()` instance and command-level global Viper bindings.
- Add explicit validation outcomes for precedence cases that remain ambiguous after the shared rules are applied, rather than preserving legacy implicit winners.
- Use the named critical baseline settings (`tenant`, active profile selection, API base URLs, auth mode, auth credentials/scopes) as the cross-command audit spine, then layer in command-specific checks for any additional config-backed settings.
- Update shared internal guidance and the relevant CLI/config documentation so operator-facing behavior matches the implementation contract exactly.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Inventory all config-backed command paths and map each one to the current precedence path, highlighting root-bootstrap, profile-overlay, and command-local binding drift.
2. Refactor the shared bootstrap/config merge path so profile values act as a lower-precedence overlay and one authoritative resolver can preserve explicit flag/env winners.
3. Bring command-local config-backed bindings into the same resolver path, especially commands using shared flag packs or global Viper helpers.
4. Add exhaustive regression coverage for the critical baseline settings everywhere they appear, plus command-specific tests for remaining config-backed settings and explicit ambiguity failures.
5. Update internal guidance, `README.md`, `docs/index.md`, and regenerate affected `docs/cli/` pages so the documented precedence contract matches shipped behavior.
6. Run targeted Go tests during implementation and finish with `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design makes precedence outcomes explicit, auditable, and failure-safe instead of relying on hidden merge side effects.
- **CLI-First, Script-Safe Interfaces**: Still passes. Existing commands and flags stay in place, while the underlying resolution logic becomes more predictable for both humans and automation.
- **Tests and Validation Are Mandatory**: Still passes with exhaustive config-backed audit coverage, shared bootstrap tests, command regressions, and final `make test`.
- **Documentation Matches User Behavior**: Still passes with shared internal documentation plus relevant user-facing CLI/config documentation updates planned in the same work.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design reuses the current `cmd` + `config` seams and existing docs generation paths instead of introducing a new configuration framework.

## Implementation Status

- Shared precedence now resolves through [`config.ResolveEffectiveConfig(...)`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go), with [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go) responsible for bootstrap-scoped Viper setup, source tracking, and error normalization.
- Profile application is implemented as a lower-precedence field overlay rather than whole-section replacement, preserving explicit flag and env winners for app, auth, API, and HTTP settings.
- Shared command-local config-backed flags, especially backoff settings from [`cmd/cmd_flagpacks.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_flagpacks.go), now participate in the same resolver path as root persistent flags.
- Shared failure handling now rejects ambiguous precedence and invalid effective-config outcomes through the repository’s `ferrors`-based bootstrap and command error mappings.
- Operator-facing precedence guidance was updated in [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md), [`docs/index.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md), [`cmd/config_show.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_show.go), and regenerated CLI reference docs under `docs/cli/`.

## Verification Status

- Shared resolver coverage now lives in `config/config_test.go`, including baseline precedence and explicit empty/zero-value preservation.
- Bootstrap and shared failure-model coverage now lives in `cmd/config_test.go` and `cmd/bootstrap_errors_test.go`.
- Command-surface regression coverage now spans `get`, `cancel`, `delete`, `deploy`, `expect`, `run`, `walk`, and `config show` paths to verify the shared baseline everywhere it appears.
- Final polish validation remains the last open step:
  - `T025`: targeted `go test ./config -count=1` and focused `go test ./cmd ... -count=1`
  - `T026`: full `make test`

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
