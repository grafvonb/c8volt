# Ralph Memory

Feature: 369-standardize-flag-intent
Started: 2026-11-03T05:40:31Z

## Codebase Patterns

- `Execute` now calls `executeCommandWithMigrationContext`, preserving original argv and the leaf returned by Cobra. Commands opt into early reporting with `setFlagMigrations(cmd, map[string]flagMigration{...})`; this registers only diagnostic annotations, never accepted legacy flags. No production command opts in yet.
- Each removed name records static `Replacement` guidance (include leading `--`) and `TakesValue`. Register metadata together with the corresponding canonical rename. Registration also silences Cobra usage, which otherwise contaminates stdout before an Args failure returns.
- `migrationReportingInput` uses registered local/inherited arity and removed-name arity, respects opaque values and `--`, and recovers only configuration/output controls into fresh flag values. Unknown arity stops recovery conservatively. It cannot execute input or read variable files/stdin.
- `resolveLocalCommandConfig` and `localCommandLogContext` share normal configuration precedence and configured stderr/activity logger setup without credential validation or remote services. The root pre-run marker keeps ordinary bootstrap/runtime errors out of migration recovery.
- `migrationReportingError` carries isolated render mode/no-error-code state and preserves the private original cause. `handleBootstrapError` sends prepared failures once through `handleCommandErrorWithMode`; ordinary rendering continues to use `pickMode`. Static migration errors cannot disclose supplied values; invalid local configuration uses the existing bootstrap logger fallback.

## Decisions

- T001 is the only completed work unit in iteration 1. Production flag names, metadata/help, README, generated docs, facade/service schemas and backend behavior remain unchanged; no documentation refresh was needed for this internal foundation.
- Use subprocess `Execute` tests (or the execution-boundary helper) for migration reporting. Bare Cobra parsing does not itself preserve invocation argv or prepare local reporting context.
- Early parser/Args errors without a recognized removed token receive a bounded generic argument diagnostic rather than exposing arbitrary positionals. Future semantic migration validation must retain its specific static guidance and run before remote bootstrap; the local setup and reporting carrier are reusable for that integration.
- Configuration recovery includes locally bound execution controls such as version/HTTP timeout: an explicit override must be able to repair an otherwise invalid file setting. Recovery never changes the original parsed flags or globals.

## Gotchas

- Profiles overlay app/auth/API/HTTP settings, not logging. Test logging precedence with base config, environment and explicit flags; test profile selection through supported app fields.
- `--no-err-codes` intentionally converts failure exit status to zero. It does not suppress the invalid envelope/error diagnostic.
- Quiet overrides debug when resolving the logger. JSON selection still wins over keys-only, independently of quiet, matching existing error rendering.
- The complete cmd package suite took 406.115s in this iteration. It contains isolated subprocess/terminal tests; do not mistake its quiet execution period for a stalled launcher.

## Reusable Commands

- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentMigration|TestBootstrap' -count=1`
- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentMigration|TestBootstrap|TestRoot|TestRetrieveAndNormalizeConfig|TestExecute_Config' -count=1`
- `GOFLAGS='' go test ./cmd ./config ./docsgen -count=1`
- Run targeted `gofmt` and `git diff --check`. Inspect inherited test/config environment and use only isolated dependencies; no live validation.

## Do Not Repeat

- Do not register removed flags as hidden/deprecated aliases or recover flags into command-global storage.
- Do not rerun Cobra to recover reporting, install remote services for a rejected invocation, print raw argv/parser values, or render another error after the existing exit boundary.
- Do not emit usage before Args diagnostics or assume profile logging overlays exist.

## Current Handoff

- Recheck tasks.md dependency readiness. T002 (US1 shared variable sources and user-task assignment dispatch) is earliest next; T001 is complete. T003 is independently ready after T001, but document order selects T002. T004 requires T002. No blockers. Keep the next iteration within US1 and register each command's migration metadata as its flags are renamed.
