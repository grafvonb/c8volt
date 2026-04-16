# Research: Audit and Fix CLI Config Precedence

## Decision 1: Use one authoritative precedence resolver in the shared bootstrap path

- **Decision**: Keep one authoritative effective-config resolution path rooted in [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go) and [`config/config.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go), rather than adding command-specific precedence patches.
- **Rationale**: The issue describes a whole-app correctness problem, and the current bootstrap path already constructs the effective config used by most commands. Fixing precedence at the shared entry point prevents repeated special cases and makes the contract testable in one place.
- **Alternatives considered**:
  - Patch only the currently observed commands: rejected because the issue explicitly requires an exhaustive audit of all config-backed command paths.
  - Let each command normalize its own precedence: rejected because it would multiply drift and produce inconsistent operator behavior.

## Decision 2: Treat profile values as a lower-precedence overlay, not a whole-section replacement

- **Decision**: Rework profile application so the selected profile contributes field-level overrides over base config only where higher-precedence flag or env values have not already won.
- **Rationale**: The current [`Config.WithProfile()`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go) flow replaces `App`, `Auth`, `APIs`, and `HTTP` wholesale, which is exactly the kind of late profile overwrite that can stomp explicit flag or env inputs.
- **Alternatives considered**:
  - Keep whole-struct replacement and add more root-level fallback exceptions: rejected because it preserves the underlying precedence bug class and encourages more ad hoc exceptions.
  - Apply profiles after normalization as a final override: rejected because profiles are supposed to rank below flags and env vars, not above them.

## Decision 3: Unify command-local config-backed bindings with the same Viper instance used for bootstrap

- **Decision**: Eliminate precedence drift between the fresh `viper.New()` instance created during root bootstrap and command paths that currently bind config-backed flags through the global `viper.GetViper()` singleton, especially shared flag packs such as [`cmd/cmd_flagpacks.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_flagpacks.go).
- **Rationale**: A command-local binding that lands on a different Viper registry is not participating in the same effective-config resolution path. The plan should normalize those bindings into one authoritative registry or an equivalent shared abstraction.
- **Alternatives considered**:
  - Leave global-Viper bindings in place and document exceptions: rejected because precedence exceptions are the bug, not the contract.
  - Copy global values into the bootstrap config after the fact: rejected because it would create another hidden merge layer instead of simplifying the existing one.

## Decision 4: Preserve explicit higher-precedence zero and empty values intentionally

- **Decision**: Treat explicit higher-precedence values as authoritative even when they are zero-like, empty, or false-like, unless a specific setting’s validation rules reject that value outright.
- **Rationale**: The spec explicitly calls out empty/zero-value edge cases. The precedence contract is about source ordering, not about silently guessing that an explicit value should be ignored because it looks “empty.”
- **Alternatives considered**:
  - Treat empty values as automatically unset: rejected because it would make source precedence depend on value shape, creating surprising behavior.
  - Ban all zero/empty values globally: rejected because some settings may legitimately use them, and validation belongs to the setting’s own rules.

## Decision 5: Fail explicitly when precedence remains ambiguous after shared rules are applied

- **Decision**: Introduce explicit validation failures for precedence cases that cannot be resolved safely through the shared contract.
- **Rationale**: This is an operational CLI; hidden best-effort winners or compatibility heuristics would reintroduce the unpredictability this feature is meant to remove.
- **Alternatives considered**:
  - Preserve legacy behavior in ambiguous cases: rejected because the current bug is that legacy behavior is not predictably aligned with the intended contract.
  - Choose a best-effort winner and log it: rejected because logging does not make a dangerous or misleading winner safe for automation.

## Decision 6: Use a named critical baseline to drive exhaustive audit coverage

- **Decision**: Anchor the exhaustive audit around a named cross-command baseline of high-risk settings: `tenant`, active profile selection, API base URLs, auth mode, and auth credentials/scopes.
- **Rationale**: These settings are the most likely to misroute requests, break authentication, or send operators to the wrong environment if precedence is wrong. Naming them in the plan keeps the full audit concrete without reducing its scope.
- **Alternatives considered**:
  - Leave the audit fully open-ended with no named baseline: rejected because it makes task planning and verification too vague.
  - Require identical direct regression coverage for every config key regardless of risk: rejected because the audit should still prioritize the settings with the biggest operational blast radius.

## Decision 7: Cover every config-backed command path, then add command-specific checks beyond the baseline

- **Decision**: Audit every config-backed command path in the CLI, verifying the named baseline settings everywhere they appear and layering command-specific coverage on top where commands expose additional config-backed behavior.
- **Rationale**: The clarified spec requires exhaustive audit coverage, not representative sampling. The baseline gives the audit a reusable spine, while command-specific checks keep the rest of the surface honest.
- **Alternatives considered**:
  - Audit only representative commands after fixing shared logic: rejected by clarification and too risky for a whole-app precedence bug.
  - Audit only the baseline settings everywhere and ignore the rest: rejected because the spec explicitly covers every config-backed flag path.

## Decision 8: Document the contract both internally and in user-facing CLI/config docs

- **Decision**: Keep the shared implementation contract in internal code/tests and update the relevant user-facing docs in [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md), [`docs/index.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md), and generated CLI references.
- **Rationale**: Maintainers need one authoritative internal contract, and operators need the same precedence order explained in the places they already learn about flags, profiles, env vars, and config files.
- **Alternatives considered**:
  - Internal docs only: rejected because operators should not need source inspection to understand override behavior.
  - User-facing docs only: rejected because reviewers also need a durable internal source of truth close to the code.

## Decision 9: Reuse existing config and command test seams before introducing new test scaffolding

- **Decision**: Extend existing tests in `config/` and `cmd/`, including subprocess/bootstrap tests where exit behavior matters, before adding any new testing harness.
- **Rationale**: The repository already has tests around config normalization, command execution, and shared failure handling. Extending those seams keeps the feature repository-native and aligned with the constitution’s validation rules.
- **Alternatives considered**:
  - Add a separate configuration test harness first: rejected because it duplicates existing coverage patterns and increases maintenance cost.
  - Rely on manual smoke testing for precedence correctness: rejected because this feature exists specifically to prevent regressions in a subtle shared contract.

## Audit Inventory: Current Precedence Surface

### Shared bootstrap and normalization seam

- [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go) is the authoritative bootstrap entrypoint today. `PersistentPreRunE` creates a fresh `viper.New()` instance, binds root persistent flags in `initViper`, reads the config file, and then calls `retrieveAndNormalizeConfig`.
- [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go)`::initViper` currently binds the root config-backed flags for `config`, `active_profile`, `log.*`, `app.tenant`, `app.camunda_version`, `app.no_err_codes`, and `app.auto-confirm`, and also sets the shared defaults for logging and `app.process_instance_page_size`.
- [`config/config.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go)`::BindConfigEnvVars` recursively binds all config fields to `C8VOLT_*` environment variables before `v.Unmarshal(&base)`.
- [`config/config.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go)`::WithProfile` currently applies the selected profile by replacing `App`, `Auth`, `APIs`, and `HTTP` wholesale, then selectively copying a few auth credentials back from the root config. This is the primary profile-overwrite hazard against the intended `flag > env > profile > base config > default` contract.

### Command-local config-backed seams

- [`cmd/cmd_flagpacks.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_flagpacks.go) defines the shared backoff flags and binds them as `app.backoff.timeout` and `app.backoff.max_retries`.
- [`cmd/get.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get.go), [`cmd/cancel.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel.go), [`cmd/delete.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete.go), [`cmd/deploy.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy.go), [`cmd/expect.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect.go), and [`cmd/run.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run.go) all call `addBackoffFlagsAndBindings(..., viper.GetViper())`.
- That command-local use of the global Viper singleton is distinct from the fresh `viper.New()` instance used in root bootstrap, so shared backoff settings currently do not flow through the same authoritative resolver as root persistent config-backed flags.

### Config-backed command paths to audit

- Root persistent baseline settings are consumed through the shared bootstrap path and then indirectly by commands that construct services or helpers from the normalized config context.
- The highest-risk config-backed command families in current tests and command wiring are `get`, `cancel`, `delete`, `deploy`, `expect`, `run`, `walk`, and `config show`, with `tenant` and API/auth settings flowing into service construction and request routing.
- Shared backoff settings are attached at the root command-family level for `get`, `cancel`, `delete`, `deploy`, `expect`, and `run`, making them the clearest mixed root-plus-local precedence surface.

## Audit Inventory: Existing Regression Seams

- [`cmd/config_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_test.go) currently covers only the `config show --validate` failure path through the shared CLI failure model; it does not yet assert precedence winners for flags, env vars, profiles, or config files.
- [`cmd/bootstrap_errors_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/bootstrap_errors_test.go) already exercises execute-time bootstrap normalization and helper-process exits, making it the right seam for root bootstrap precedence and ambiguity failures.
- [`config/app_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go) currently covers normalization defaults such as process-instance page size and tenant behavior by Camunda version, but not multi-source precedence or profile overlay semantics.
- [`config/errors_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/errors_test.go) only covers formatting of validation errors today.
- There is no existing [`config/config_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config_test.go); introducing it is consistent with the task plan and is the cleanest place for config-level precedence and overlay tests.
- Command regression coverage already exists for representative command families, but it is feature-specific rather than precedence-specific:
  - [`cmd/get_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go), [`cmd/deploy_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go), and [`cmd/walk_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go) are lightweight seams that can absorb baseline precedence tests without excessive fixture churn.
  - [`cmd/get_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go), [`cmd/cancel_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go), and [`cmd/delete_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go) already prove config-backed execution with temp configs and can be extended for mixed-source cases.
  - [`cmd/expect_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_test.go) currently focuses on validation failures, not config-backed precedence.
