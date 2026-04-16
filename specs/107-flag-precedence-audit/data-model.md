# Data Model: Audit and Fix CLI Config Precedence

## Config-Backed Setting

- **Purpose**: Represents any user-facing setting whose effective value may come from CLI flags, environment variables, the selected profile, base config, or a built-in default.
- **Key attributes**:
  - Canonical config key, such as `app.tenant` or `auth.mode`
  - Owning command scope, such as root persistent, shared subcommand pack, or command-local
  - Allowed source set
  - Validation rules for accepted values
  - Documentation surface where the setting is explained to operators
- **Invariants**:
  - Every config-backed setting must have one authoritative precedence outcome per command execution.
  - The same canonical setting key must not resolve through conflicting precedence rules in different command paths.

## Precedence Source

- **Purpose**: Represents the ordered source categories that may supply a value for a config-backed setting.
- **States**:
  - `Flag`
  - `Environment`
  - `Profile`
  - `BaseConfig`
  - `Default`
- **Resolution order**:
  - `Flag > Environment > Profile > BaseConfig > Default`
- **Invariants**:
  - A higher-precedence explicit value wins over all lower-precedence values unless that value is invalid for the setting.
  - Source ordering must be identical for root persistent settings and command-local settings.

## Effective Setting Value

- **Purpose**: Represents the final winner chosen for one config-backed setting during a command execution.
- **Required fields**:
  - Canonical setting key
  - Final value
  - Winning source
  - Whether the winning value was explicitly provided or inherited
  - Validation status
- **Behavioral rules**:
  - Explicit zero-like or empty values remain authoritative if the setting allows them.
  - A final value may be rejected after source resolution if the setting’s validation rules disallow it.

## Profile Overlay

- **Purpose**: Represents the selected profile’s contribution to the effective config.
- **Required fields**:
  - Active profile name
  - Profile-scoped values by canonical config key
  - Overlay result against base config
- **Behavioral rules**:
  - Profile values may override base config values.
  - Profile values must not override explicit flag or environment values.
  - Missing profile fields fall back to lower-precedence sources without requiring command-specific exceptions.

## Binding Registry

- **Purpose**: Represents the registry that knows which command flags and environment names map to which canonical config keys.
- **Required fields**:
  - Canonical setting key
  - Flag name, if any
  - Environment variable name, if any
  - Owning command or shared flag pack
  - Default value, if any
- **Invariants**:
  - Config-backed flags must participate in the same authoritative resolution path rather than splitting between unrelated Viper registries.
  - Shared flag packs must not create precedence behavior different from directly bound root flags.

## Critical Baseline Setting

- **Purpose**: Represents a high-risk config-backed setting that must be verified everywhere it appears across the audited command surface.
- **Members**:
  - `app.tenant`
  - `active_profile`
  - API base URLs under `apis.*.base_url`
  - `auth.mode`
  - Auth credentials and scopes under `auth.*`
- **Behavioral rules**:
  - Every appearance of a critical baseline setting must be covered by the audit.
  - Command-specific settings may add extra checks beyond the baseline, but may not weaken it.

## Precedence Validation Outcome

- **Purpose**: Represents the result of applying source ordering and setting validation.
- **States**:
  - `Resolved`: one safe winner was chosen and validated
  - `Invalid`: a winner was chosen, but the value violates the setting’s rules
  - `Ambiguous`: the shared rules cannot determine a safe winner and the command must fail explicitly
- **Invariants**:
  - `Ambiguous` is a hard failure, not a warning-only state.
  - `Invalid` and `Ambiguous` must not silently fall through to a lower-precedence source.

## Documentation Surface

- **Purpose**: Represents the user-facing and maintainer-facing places where the precedence contract is described.
- **Members**:
  - Shared internal implementation guidance and tests
  - `README.md`
  - `docs/index.md`
  - Generated `docs/cli/` pages for affected commands
- **Invariant**:
  - All documentation surfaces must describe the same precedence order and ambiguity behavior.

## Audit Traceability Record

- **Purpose**: Represents the small set of shared artifacts reviewers can inspect to confirm the baseline audit is real and current.
- **Required fields**:
  - Resolver-level proof location
  - Command-surface proof location
  - Operator-facing documentation surfaces refreshed from the same contract
- **Current repository mapping**:
  - Resolver proof: `config/config_test.go::TestResolveEffectiveConfig_CriticalBaselineSettingsShareOneContract`
  - Command-surface proof: `cmd/config_test.go::TestRetrieveAndNormalizeConfig_CriticalBaselineSettingsStayAlignedAcrossCommands`
  - Operator-facing docs: `README.md`, generated `docs/index.md`, and generated `docs/cli/`
- **Invariant**:
  - The traceability record should point to executable proof and generated operator docs, not only planning notes.
