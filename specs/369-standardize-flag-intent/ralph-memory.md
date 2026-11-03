# Ralph Memory

Feature: 369-standardize-flag-intent
Started: 2026-11-03T05:40:31Z

## Codebase Patterns

- `Execute` now calls `executeCommandWithMigrationContext`, preserving original argv and the leaf returned by Cobra. Commands opt into early reporting with `setFlagMigrations(cmd, map[string]flagMigration{...})`; this registers only diagnostic annotations, never accepted legacy flags. Element/process/user-task updates and both repair commands now opt in for their removed variable sources; user-task also maps removed assignment input.
- Each removed name records static `Replacement` guidance (include leading `--`) and `TakesValue`. Register metadata together with the corresponding canonical rename. Registration also silences Cobra usage, which otherwise contaminates stdout before an Args failure returns.
- `migrationReportingInput` uses registered local/inherited arity and removed-name arity, respects opaque values and `--`, and recovers only configuration/output controls into fresh flag values. Unknown arity stops recovery conservatively. It cannot execute input or read variable files/stdin.
- `resolveLocalCommandConfig` and `localCommandLogContext` share normal configuration precedence and configured stderr/activity logger setup without credential validation or remote services. The root pre-run marker keeps ordinary bootstrap/runtime errors out of migration recovery.
- `migrationReportingError` carries isolated render mode/no-error-code state and preserves the private original cause. `handleBootstrapError` sends prepared failures once through `handleCommandErrorWithMode`; ordinary rendering continues to use `pickMode`. Static migration errors cannot disclose supplied values; invalid local configuration uses the existing bootstrap logger fallback.

## Decisions

- T006 is complete. Production migration acceptance covers all removed write spellings across leaf and ancestor aliases, equal old/new conflicts in both orders (including repair zero), retained job retry/HTTP timeout meanings, and strict BPMN action syntax across job aliases. Reporting matrices cover later config/profile/output/logger controls, no-err-codes and invalid-config fallback without remote requests or mutation-input reads.
- `TestFlagIntentMigrationProductionHelper` uses the real command tree and Execute; its terminal mode retains actual terminal stdin and asserts it with `term.IsTerminal`. Migration terminal cases require exactly one configured/inherited diagnostic or JSON envelope, including JSON/quiet/keys precedence, with no prompts or remote requests.
- Existing canonical runtime logging/terminal tests provide partial-result, repeated-failure, redacted DEBUG, summary suppression, empty-scope and sparse-discovery coverage. Temporary direct-stderr and raw-aggregate mutations in element execution were both detected by `TestElementScopeVariablesUpdateDiagnostics/plain-time/configured/normal`; production source was restored byte-for-byte and the final tests passed. No production/bootstrap/schema/help change was needed for T006.

- T005 is complete. `throw-bpmn-error` is Boolean action storage, while `set-error-code` supplies the opaque code. `set-error-message` and `set-retry-backoff` are canonical; removed names remain diagnostic metadata only. Nonblank error codes retain exact spaces through the native body; Boolean-looking codes remain strings.
- `flagMigration.Valueless` extends the existing registered-arity scanner to reject every attached action value. `validateUpdateJobArgs` inspects preserved original argv before bootstrap and rejects positional legacy codes with static guidance; semantic validation requires an active action and a nonblank explicitly supplied code. Opaque flag values and tokens after `--` are never scanned as action flags. Do not rely on ordinary Boolean parsing to reject `=true` or `=false`.

- T004 is complete: creation uses `set-bid`/`set-vars`; both repair commands use `set-retries`/`set-job-timeout`. Repair retains default 1 and zero-as-skip, with timeout independently submitted. Existing repair variable migration mappings remain registered. Creation still has no file input; repair/get BID selectors remain unchanged.
- Creation failure wording depends on `cmd/cmd_views_processinstance_run.go` checking `Changed("set-bid")`; keep this aligned with registration so unsupported versions retain the useful bounded business-ID reason.

- T003 is complete. `flagUpdateJobRetries` remains selector-only; `flagUpdateJobSetRetries` owns submitted retries. Job mutation timeouts use `set-timeout`; removing local `timeout` restores the existing root HTTP binding without changing root_config.go. Job variables use `set-vars`; no file input was added. T005 now owns the completed worker action/code/message/backoff migration.
- `validateUpdateJobArgs` rejects retained-flag migration errors before bootstrap without reading stdin or constructing requests. `migrationSemanticError` preserves only static command-owned guidance through T001 local reporting; never wrap parser messages or supplied values in it. The command request parser also checks intent for direct callers. T005 can reuse this seam.

- T002 is complete: shared update/repair sources use `set-vars`/`set-vars-file`, task assignment uses `set-assignee`. Job inline variables are now `set-vars` (T003); run inline variables are now `set-vars` (T004). Task operation strings and report schemas still use `assign`; translate only the diagnostic flag label in assignment exclusions.
- Keep each command's existing migration entries when T004 adds repair retry/timeout entries. `setFlagMigrations` replaces the map rather than merging it.
- Shared completion/assignment test constructors and `assignmentFlags` now emit canonical input; their existing logging/terminal matrices cover the renamed production dispatch. Completion preserves exact JSON numbers; ordinary variable updates keep their existing decoder.

- T001 is the only completed work unit in iteration 1. Production flag names, metadata/help, README, generated docs, facade/service schemas and backend behavior remain unchanged; no documentation refresh was needed for this internal foundation.
- Use subprocess `Execute` tests (or the execution-boundary helper) for migration reporting. Bare Cobra parsing does not itself preserve invocation argv or prepare local reporting context.
- Early parser/Args errors without a recognized removed token receive a bounded generic argument diagnostic rather than exposing arbitrary positionals. Future semantic migration validation must retain its specific static guidance and run before remote bootstrap; the local setup and reporting carrier are reusable for that integration.
- Configuration recovery includes locally bound execution controls such as version/HTTP timeout: an explicit override must be able to repair an otherwise invalid file setting. Recovery never changes the original parsed flags or globals.

## Gotchas

- Native v8.8 job PATCH bodies use `changeset` with both nullable `retries` and `timeout` fields. Creation/repair payload fixtures assert this existing schema; zero repair retries emits no retry PATCH, while timeout still emits its own PATCH. Success subprocess helpers must `os.Exit(0)` after Execute to avoid the Go test harness appending PASS to JSON.
- docsgen/main_test.go also asserts repair flag spellings; update those expectations alongside generated docs. Maintained integration evidence CoveredFlags must match invocation changes, even though those suites must not be executed here.

- Config explicitly allows empty environment values. Timeout precedence fixtures must unset C8VOLT_HTTP_TIMEOUT to test omission; setting it to an empty string intentionally overrides file/profile input and normalization returns 30s.
- Listener batch completion rejects nonempty variables. Logging/terminal matrices use canonical `--set-vars '{}'` to retain listener eligibility; nonempty supported payloads and listener rejection remain covered elsewhere.
- `cmd/job_task_business_id_test.go` owns additional retry/no-op preview invocations outside the update_job test family; migrate these fixtures with retry syntax changes.

- Opting into T001 reporting also bounds unknown parse/Args errors on that command. The user-task `--allow-override` rejection now asserts the configured ERROR record `invalid input: invalid command arguments; see command help` and invalid-input exit, retaining zero-request and no-prompt checks.
- Mixed-family fixtures must migrate selectively: run `--vars` is rejected after T004; job/shared variable fixtures are canonical. Integration fixtures/scripts are edited statically only, never executed.

- Profiles overlay app/auth/API/HTTP settings, not logging. Test logging precedence with base config, environment and explicit flags; test profile selection through supported app fields.
- `--no-err-codes` intentionally converts failure exit status to zero. It does not suppress the invalid envelope/error diagnostic.
- Quiet overrides debug when resolving the logger. JSON selection still wins over keys-only, independently of quiet, matching existing error rendering.
- The complete cmd package suite took 406.115s in this iteration. It contains isolated subprocess/terminal tests; do not mistake its quiet execution period for a stalled launcher.

## Reusable Commands

- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentJob|TestUpdateJob|TestParseUpdateJob|TestScalarJobCompletedGuard|TestCommandContract' -count=1`
- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentJobWorkerSummaryLogging' -count=1` covers both worker actions, plain/structured INFO records, configured/inherited terminal writers, and WARN/quiet suppression.

- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentCreationRepair|TestRun|TestOpsRepair|TestBusinessIDOperationsRepair|TestFlagIntentVariables|TestCommandContract|TestCapabilities|TestCompletion|TestFlagParseErrors' -count=1`
- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentCreationRepair|TestFlagIntentMigration|TestFlagParseErrors' -count=1`; `GOFLAGS='' go test ./docsgen -count=1`

- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentJob|TestUpdateJob|TestParseUpdateJob|TestFlagIntentMigration|TestCommandContract|TestCapabilities|TestCompletion|TestScalarJobCompletedGuard|TestRetrieveAndNormalizeConfig' -count=1`
- `GOFLAGS='' go test ./cmd -run 'TestBulkCompletionDiagnosticLogging|TestBulkCompletionSummaryLogging|TestBulkCompletionTerminal|TestUpdateJobAssignmentScalarLogging' -count=1`
- `GOFLAGS='' go test ./cmd -run '^TestJobBusinessIDPlans$' -count=1`

- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentVariables|TestUpdateUserTaskAssignmentInput|TestUpdateUserTaskCompletionCommandVariables|TestElementScopeVariablesUpdate|TestOpsRepair' -count=1`
- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentVariables' -count=1` covers production legacy/mixed rejection, exact envelopes, human logger records and completion candidates.
- `GOFLAGS='' make docs-content`; `bash -n integration/scripts/run-suite.sh` (syntax only).

- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentMigration|TestBootstrap' -count=1`
- `GOFLAGS='' go test ./cmd -run 'TestFlagIntentMigration|TestBootstrap|TestRoot|TestRetrieveAndNormalizeConfig|TestExecute_Config' -count=1`
- `GOFLAGS='' go test ./cmd ./config ./docsgen -count=1`
- Run targeted `gofmt` and `git diff --check`. Inspect inherited test/config environment and use only isolated dependencies; no live validation.

## Do Not Repeat

- Do not register removed flags as hidden/deprecated aliases or recover flags into command-global storage.
- Do not rerun Cobra to recover reporting, install remote services for a rejected invocation, print raw argv/parser values, or render another error after the existing exit boundary.
- Do not emit usage before Args diagnostics or assume profile logging overlays exist.

## Current Handoff

- Recheck tasks.md dependencies. T007 (US3 query/enrichment names plus final canonical-surface inventory) is the next and only unfinished task; T001–T006 are complete. No blockers. This invocation was labeled iteration 1 by the orchestrator; earlier audit entries retain their original iteration labels.
- T006 validation passed: required migration/contract/capabilities/completion/error selection, new real-terminal rejection cases, job-alias semantics, canonical write surfaces, and affected logging/terminal/empty/sparse/partial-result families. Final restored-source regression selection passed. No production bootstrap changes required shared cmd/config/docsgen reruns or docs regeneration; prior shared-suite evidence remains applicable. No live-engine validation was performed.
