# Contract: CLI Config Precedence

## Precedence Contract

All config-backed settings must resolve using the same ordered sources:

| Source order | Required behavior |
|-------------|-------------------|
| `Flag` | Explicit CLI flag wins over every other source when the value is valid for the setting |
| `Environment` | Environment value wins when no explicit flag value overrides it |
| `Profile` | Selected profile value wins when neither flag nor environment provides a higher-precedence value |
| `BaseConfig` | Base config value wins when no higher-precedence source is set |
| `Default` | Built-in default applies only when no other source provides the setting |

## Profile Contract

| Rule | Required behavior |
|------|-------------------|
| Active profile selection | Determines which profile overlay is eligible to contribute values |
| Profile overlay | Applies field-by-field over base config, not as a whole-section replacement that can stomp higher-precedence winners |
| Missing profile field | Falls back to lower-precedence sources without command-specific special cases |
| Profile vs flag/env | Profile values must never override explicit flag or environment winners |

## Binding Contract

| Surface | Required behavior |
|---------|-------------------|
| Root persistent flags | Participate in the authoritative shared resolver |
| Command-local config-backed flags | Participate in the same authoritative shared resolver |
| Shared flag packs | Must not resolve through a separate precedence path from root bootstrap |
| Environment binding | Must surface the same canonical config keys used by flags, profiles, base config, and defaults |

The repository implementation seam for this contract is:

- `cmd/root.go::initViper` binds both root persistent flags and command-local config-backed flags into the same bootstrap `viper.New()` instance.
- `config.ResolveEffectiveConfig(...)` is the authoritative effective-config resolver used by bootstrap.
- `config.Config.WithProfile()` applies the selected profile as a field-level overlay instead of replacing whole config sections.

## Critical Baseline Contract

The audit must verify the following settings everywhere they appear:

- `app.tenant`
- `active_profile`
- `apis.*.base_url`
- `auth.mode`
- Auth credentials and scopes under `auth.*`

Additional command-specific config-backed settings remain in scope beyond this baseline.

## Validation Contract

| Condition | Required behavior |
|-----------|-------------------|
| Higher-precedence explicit value is valid | Use it even if the value is zero-like or empty |
| Higher-precedence explicit value is invalid | Fail through the shared validation/error path |
| Shared rules cannot determine a safe winner | Fail explicitly; do not preserve legacy ambiguity and do not choose a best-effort winner |
| Lower-precedence value exists when higher-precedence value fails | Do not silently fall back to it |

## Command Audit Contract

| Scope | Required behavior |
|-------|-------------------|
| Every config-backed command path | Must be reviewed against the shared precedence contract |
| Baseline settings | Must be verified everywhere they appear |
| Command-specific settings | Must be checked where the command exposes additional config-backed behavior |
| Shared bootstrap + command-local flags | Must behave consistently for the same canonical setting |

## Documentation Contract

- Shared internal implementation guidance and tests must describe the same precedence contract used in code.
- [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md) and [`docs/index.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md) must explain the operator-visible precedence and override behavior.
- Generated `docs/cli/` pages for affected commands must be regenerated from Cobra help text rather than edited by hand.
