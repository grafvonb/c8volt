# Quickstart: Audit and Fix CLI Config Precedence

## Planned Behavior

- Every config-backed setting resolves with one shared order: `flag > env > profile > base config > default`.
- Profiles act as a lower-precedence overlay over base config instead of replacing whole config sections after flags or env vars have already been applied.
- Command-local config-backed flags participate in the same effective resolver as root persistent flags.
- Explicit higher-precedence zero-like or empty values stay authoritative unless the setting’s own validation rules reject them.
- Ambiguous precedence cases fail explicitly instead of preserving legacy behavior or silently choosing a winner.
- The shared audit baseline covers `tenant`, active profile selection, API base URLs, auth mode, and auth credentials/scopes everywhere they appear.
- The same precedence contract is documented in internal guidance and the relevant user-facing CLI/config docs.

## Verification Focus

1. Start from the shared bootstrap path in [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go) and [`config/config.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go); do not patch individual commands first.
2. Verify that command-local config-backed bindings, especially shared backoff flags from [`cmd/cmd_flagpacks.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_flagpacks.go), are not stranded on `viper.GetViper()` while root bootstrap uses a separate `viper.New()` instance.
3. Use existing config tests and command/subprocess tests as the first validation seams before adding any new harness. The current baseline is `cmd/config_test.go`, `cmd/bootstrap_errors_test.go`, `config/app_test.go`, `config/errors_test.go`, and the existing command-family tests under `cmd/`.
4. Add the missing config-level precedence seam in `config/config_test.go` instead of forcing overlay assertions into unrelated test files.
5. Keep user-facing docs aligned: update [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md), regenerate [`docs/index.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md) via the repo workflow, and regenerate affected `docs/cli/` pages from Cobra help text.
6. Treat every config-backed command path as in scope, even if the baseline settings appear to be the highest-risk subset.

## Shared Audit Matrix

| Audit seam | What it proves |
|------------|----------------|
| `config/config_test.go::TestResolveEffectiveConfig_CriticalBaselineSettingsShareOneContract` | The authoritative resolver applies one precedence contract to active profile selection, tenant, API base URLs, auth mode, and auth credentials/scopes |
| `cmd/config_test.go::TestRetrieveAndNormalizeConfig_CriticalBaselineSettingsStayAlignedAcrossCommands` | `config show`, `get`, `cancel`, `delete`, `deploy`, `expect`, `run`, and `walk` all bootstrap the same effective values for the critical baseline |
| Existing command-family tests under `cmd/` | Command-specific behavior still matches the shared baseline where individual commands add request shaping, waiting, paging, or delete/cancel semantics |

## Audit Baseline Captured In Setup

- Root persistent config-backed settings currently enter through `cmd/root.go` and should remain the only authoritative effective-config bootstrap.
- Profile application currently replaces whole config sections in `config.Config.WithProfile()`, so later implementation must change it to a field-aware lower-precedence overlay.
- Shared backoff flags are the clearest command-local drift case because `get`, `cancel`, `delete`, `deploy`, `expect`, and `run` bind them via the global Viper singleton instead of the root bootstrap instance.
- Existing regression coverage is strongest around bootstrap failure handling and command execution helpers, but there is no dedicated config-level precedence test file yet.

## Suggested Verification Commands

```bash
go test ./config -count=1
go test ./cmd -run 'TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfig|Test.*Profile.*|Test.*Tenant.*|Test.*Config.*' -count=1
go test ./cmd -run 'Test.*Backoff.*|Test.*Completion.*|Test.*Subprocess.*' -count=1
make test
```

## Manual Smoke Checks

Use one base config file with at least two profiles and then exercise the same setting through multiple source combinations:

```bash
./c8volt --config /tmp/c8volt.yaml --profile prod config show
C8VOLT_APP_TENANT=tenant-env ./c8volt --config /tmp/c8volt.yaml --profile prod config show
C8VOLT_AUTH_MODE=none ./c8volt --config /tmp/c8volt.yaml --profile prod get cluster topology
./c8volt --config /tmp/c8volt.yaml --profile prod --tenant tenant-flag get cluster topology
./c8volt --config /tmp/c8volt.yaml --profile prod --backoff-timeout 5s get cluster topology
```

## Verification Notes

- Confirm `config show` reflects the same effective values the command bootstrap path actually uses.
- Confirm a higher-precedence tenant value from `--tenant` wins over env, profile, and base config in every command path where tenant is consumed.
- Confirm profile selection changes the effective profile-sourced values without overriding explicit env or flag winners.
- Confirm API base URL and auth settings follow the same precedence rules as tenant and do not diverge between commands.
- Confirm command-local shared flag packs, such as backoff settings, do not resolve through a different registry than the root bootstrap path.
- Confirm ambiguity and invalid-value cases fail explicitly through the shared CLI error model instead of silently selecting a lower-precedence value.
- Confirm the generated CLI help and operator-facing docs describe the same `flag > env > profile > base config > default` order that the shared tests enforce.
