# Tasks: Standardize Flag Intent

**Input**: Design artifacts in `specs/369-standardize-flag-intent/`.
**Prerequisites**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli-flags.md), [quickstart.md](quickstart.md), `AGENTS.md`, `.specify/memory/constitution.md`. Ralph must also read `specs/ralph-implementation-rules.md` and existing feature progress.
**Policy**: Issue #369; branch `369-standardize-flag-intent`; v5 breaking change with no accepted legacy aliases or deprecation period.
**Organization**: P1 stories US1, US2, US4 precede P2 US3. Each checkbox is a complete compiling work unit with implementation, fixtures, tests, formatting, metadata and affected docs. Shared foundational support has no story label. Requirements shared across stories have one implementation owner and mapped acceptance coverage below.

## Execution Rules for Every Work Unit

1. Inspect local patterns and reuse existing request types, `testx` subprocess/terminal helpers, configured logger and result renderers. No production facade, domain, service or generated-client schema changes are needed. Keep each mode in its existing focused file; final formatting stays in `cmd/cmd_views_*.go`.
2. Add/adapt relevant tests first and observe meaningful failures before changing behavior. Preserve existing tests rather than deleting inconvenient assertions. Every new or materially modified function/test gets a purpose or invariant comment. Run `gofmt` on touched Go files, targeted tests with `-count=1`, then `git diff --check`. A proposed test pattern must actually match tests; no-tests-to-run is not a pass.
3. All tests use test-owned HTTP fakes, mocks or isolated CLI/subprocess/real-terminal fixtures. Inspect targets and inherited GOFLAGS/config/profile/credentials before execution; never use live services, integration tags/targets, cluster discovery, provisioning, recordings or live smoke scripts. Updating invocation text in live-suite files is static maintenance only.
4. Every command change includes source help/examples, relevant `cmd/command_contract.go` metadata/capability/completion assertions, affected maintained invocation fixtures/scripts/examples, README and `make docs-content`. Do not hand-edit `docs/cli/` or `docs/index.md`; preserve historical feature evidence. Apply only the work unit's syntax changes, not future ones.
5. Through execution, capture stdout/stderr separately. Human mutation/dry-run/verbose details use configured INFO logging with empty human stdout, plain/structured formats, configured/inherited destinations and WARN/quiet suppression. Prompts stay plain configured stderr; selected-format results stay stdout. Trace changed error flow from service/facade through renderer/logger/exit and inspect direct prints/raw error formatting. No new direct diagnostic prints/default logger bypass.
6. Preserve supported modes/defaults, exact values, authorization, tenant/discovery scope, confirmation, dry-run/no-wait, original failure classes/exit behavior and partial evidence. Shared JSON emits exactly one existing envelope with EOF afterward, without second rendering, including quiet machine results and established empty/null rules. Supported empty keys output has zero bytes. No new output-mode combinations.
7. Retain empty-scope, sparse-page continuation, incomplete-discovery, nonempty/explicit-key, abort/error regressions for affected paths. Eligible completed empty scopes require real terminal stdin coverage: no confirmation or mutation, including supported automation/auto-confirm/no-wait/dry-run combinations. Prompts require real terminal stdin, separate streams and configured/inherited stderr assertions.
8. Use targeted checks listed under each task. Shared bootstrap changes additionally need `go test ./cmd ./config ./docsgen -count=1`; use `make test` only if actual shared runtime blast radius justifies the full race suite. Do not repeat successful runtime checks for documentation-only follow-ups. Record actual checks/limitations in progress; no live-engine behavior is claimed. Issue-associated work-unit commits, when the workflow reaches its commit step, use Conventional Commits ending with ` #369`.

### Quoted Data-Model Constraints

These are execution requirements, assigned to task owners rather than deferred design choices:

- Input presence: “Presence tracked separately from decoded value. Explicit zero is never omission.”
- Flag metadata: “Existing scalar/object/file/duration/Boolean contract; BPMN action is strictly valueless.” “Diagnostic replacement metadata only; not accepted aliases or flag registrations.” “Existing restrictions remain unchanged.”
- Values: “Preserve operation-specific numeric decoding, optional payloads, and non-null-object validation.” “Optional exact BID and variables map to existing creation fields.”
- Job: “Optional supplied retry count comes only from `set-retries`; existing nonnegative/range validation and pointer semantics remain.” “Optional supplied positive duration comes only from `set-timeout`, with existing duration/unit validation.”
- Error action: “Active valueless BPMN action, required code, optional message and supported variables map to unchanged worker outcome fields. Technical failure keeps its existing retry/backoff requirements.”
- Repair: “Zero retry count means skip restoration.”
- Reporting: “Known configuration, profile, output and logging controls recovered according to registered token arity into temporary state.” “Existing exit-after-render contract prevents duplicate rendering.”

## Phase 1: Setup

The Go project, dependency versions and feature context already exist. No initialization, dependency installation, context-reading or branch-check checkbox is needed. Perform those responsibilities inside each work unit.

## Phase 2: Foundational — Local Migration Error Reporting

**Goal**: Prepare configured error context before remote bootstrap, including parse/Args failures. Do not remove production flags prematurely.
**Independent test**: Isolated command trees/leaf invocations with a synthetic rejected flag prove configured destinations, command identity, JSON EOF and zero remote calls before any feature command adopts new syntax.

- [x] T001 Implement and validate local-only migration error preparation in `cmd/flag_migration.go`, `cmd/flag_migration_input.go`, `cmd/root.go`, `cmd/root_config.go`, `cmd/bootstrap_errors.go`, `cmd/cmd_bootstrap_failure.go` and the existing flag-error hook in `cmd/get_processinstance_validation.go`; add focused tests in `cmd/flag_migration_test.go` and `cmd/bootstrap_errors_test.go`. Preserve leaf identity and original argv, build a command-scoped diagnostic replacement map recording removed flags' value arity without legacy registrations, extract/reuse configured local context setup, and route one classified error through existing command rendering without remote services or a second renderer; recovery state is diagnostic-only and cannot execute input. Follow quoted metadata/presence/reporting constraints. Test known flag arity, inherited flags, short/attached forms, `--`, flag-looking opaque payload values, and reporting controls after the bad token; preserve precedence, no-err-codes and bootstrap fallback for invalid local config without printing raw argv. Assert plain/structured configured/inherited logger routing, quiet compact errors and exactly one supported JSON envelope; no prompt, stdin consumption, variable-file read, auth/version/discovery/mutation. Preserve unrelated command behavior and current accepted flags. Add tests before implementation, then `go test ./cmd -run 'TestFlagIntentMigration|TestBootstrap' -count=1`, affected root/config regressions and shared cmd/config/docsgen checks under execution rules. Update affected bootstrap/help guidance in `README.md` and regenerate docs only if guidance changes.

**Checkpoint**: Support is reusable and compiled; production canonical renames occur with their consumers below.

## Phase 3: User Story 1 — Distinguish Selection from Submitted Values (P1)

**Goal**: Canonical submitted-value flags preserve selection, mutation and initialization behavior.
**Independent test**: Captured fake requests prove exact values/payloads and unchanged target scope across update/run/repair; job HTTP config is checked independently of job payload.
**Depends on**: T001. Story acceptance for completion error exclusions also receives coverage from US2.

- [x] T002 [US1] Deliver canonical variable-source and task-assignment dispatch together: migrate shared variable-source flags and all helper callers in `cmd/update_variables_payload.go`, `cmd/update_element.go`, `cmd/update_element_variables.go`, `cmd/update_processinstance.go`, `cmd/update_processinstance_variables.go`, `cmd/update_processinstance_assignment.go`, `cmd/update_usertask.go`, `cmd/update_usertask_complete_input.go`, `cmd/update_usertask_variables.go`, `cmd/update_usertask_assignment_input.go`, `cmd/ops_repair_variables.go`, `cmd/ops_repair_incident.go` and `cmd/ops_repair_processinstance.go`; adapt corresponding `cmd/update_element_test.go`, `cmd/update_usertask_complete_input_test.go` and existing variable/repair execution, logging and terminal tests, adding `cmd/flag_intent_variables_test.go`. Register/read only `--set-vars` and supported `--set-vars-file`, update Changed checks, source labels, validation/exclusions and diagnostic mappings; preserve current scalar/collection/stdin handling, task-completion exact numbers, process BID exclusions and optional repair payloads. Follow quoted presence/value constraints and existing accepted-value rules; reject old names and equal/mixed old/new inputs through T001 before mutation. Job/run inline variables are owned by T003/T004. Write failing assertions first, implement, format and run `go test ./cmd -run 'TestFlagIntentVariables|TestElementScopeVariablesUpdate|TestUpdateUserTaskCompletionCommandVariables|TestOpsRepair' -count=1` plus affected neighboring payload/output/terminal cases. Update affected source metadata, `README.md`, maintained fixtures and generated docs within this unit per execution rules.
  - Assignment portion of this same dispatch outcome: Rename task assignment input from `--assign` to `--set-assignee` in `cmd/update_usertask.go`, `cmd/update_usertask_assignment_input.go`, `cmd/update_usertask_variables.go` and affected assignment/completion dispatch helpers; update `cmd/update_usertask_assignment_command_test.go`, `cmd/update_usertask_assignment_input_test.go`, `cmd/update_usertask_assignment_logging_test.go` and `cmd/update_usertask_assignment_terminal_test.go`. Keep opaque identity, force/replacement confirmation, --unassign/--complete exclusions, no-wait and JSON guardrails unchanged; get-task `--assignee` remains a filter and result operation names remain unchanged. Reject removed and conflicting spellings before confirmation/mutation, including equal identities. Write failing command-path assertions, implement, format and run `go test ./cmd -run 'TestUpdateUserTaskAssignment|TestFlagIntentVariables' -count=1` and affected terminal/force/abort cases; update source help/metadata, `README.md`, fixtures and generated docs in the same unit. Apply these assignment changes in the same compiling unit as shared variable/exclusion changes, sharing fixtures and one documentation refresh.


- [x] T003 [US1] Separate job retry selection/mutation, restore inherited HTTP timeout and rename supported inline variables in `cmd/update_job.go`, `cmd/update_job_request.go`, `cmd/update_job_outcome.go`, `cmd/update_job_batch.go` and `cmd/root_config.go`; adapt `cmd/update_job_test.go`, `cmd/update_job_batch_test.go`, `cmd/update_job_outcome_test.go`, `cmd/root_config_test.go` and payload/logging/terminal fixtures, adding `cmd/flag_intent_job_test.go`. Use independent storage/presence for `retries` and `set-retries`; preserve exact zero filters on get/keyless completion and reject mutation retries on completion/keyed retry predicates. Reject bare retries in retry/failure mutation modes with replacement guidance, including both spellings together. Remove local `timeout`, use `set-timeout` only for job payload/exclusions and retain `timeout` only for HTTP execution. Follow quoted presence/job/value constraints; test actual inherited flag identity, effective default/file/profile/env/explicit HTTP precedence, overrides before/after command words, job-only/both timeouts, canonical mutations with HTTP timeout and key-plus-HTTP-timeout-only rejection. Migrate job `--vars` to `--set-vars` in scalar/collection/selector completion and BPMN-error paths without adding file support or altering atomic BID completion. Preserve current worker action/code spelling until US2. Write failing cases first, implement and format; run `go test ./cmd -run 'TestFlagIntentJob|TestUpdateJob' -count=1` plus affected config/payload/empty-scope/terminal cases; update source help/metadata, `README.md`, static maintained invocations and generated docs.

- [x] T004 [US1] Deliver canonical creation/repair write inputs using existing request mappings: rename process initialization inputs to `--set-bid` and `--set-vars` in `cmd/run_processinstance.go`; adapt creation execution and BID/number/count/output fixtures and add `cmd/flag_intent_creation_repair_test.go`. Preserve quoted exact creation fields and presence semantics, existing BID validation/version restrictions, exact values and count reuse, optional variables, tenant/destination and observable-run behavior; no new variable-file input. Reject `--bid`/`--vars` and mixed equal old/new values with replacement guidance. Add failing tests, implement, format and run `go test ./cmd -run 'TestFlagIntentCreationRepair|TestRun' -count=1` plus affected subprocess/confirmation/error cases; update command source metadata, `README.md`, `demos/vhs/fast-start.tape`, maintained invocation fixtures/scripts and regenerate docs. Do not run recordings or live scripts.
  - Repair portion of this same write-input outcome: Rename remaining repair mutation values to `--set-retries` and `--set-job-timeout` in `cmd/ops_repair_incident.go` and `cmd/ops_repair_processinstance.go`; adapt `cmd/ops_repair_incident_test.go`, `cmd/ops_repair_processinstance_test.go`, BID/report fixtures and `cmd/flag_intent_creation_repair_test.go`. Shared variable/file rename from T002 remains intact. Preserve optional timeout validation, discovery/tenant scope, reports, confirmation, existing repair native payloads and the quoted rule “Zero retry count means skip restoration.” Reject removed/mixed spellings before mutation and keep execution backoff controls unchanged. Add failing fixtures first, implement, format and run `go test ./cmd -run 'TestFlagIntentCreationRepair|TestOpsRepair' -count=1`, affected isolated terminal/logging/no-op cases and static maintained script syntax checks; update help/metadata, `README.md`, invocation text and generated docs, explicitly documenting the zero exception. Complete creation and both repair payload/default assertions in one work unit after T002; share migration fixtures and the documentation refresh.


**Checkpoint**: US1 canonical writing/initialization paths preserve native semantics; no legacy mutation names remain on these migrated inputs.

## Phase 4: User Story 2 — Separate Worker Actions from Error Values (P1)

**Goal**: A strictly valueless BPMN action and explicit submitted error values preserve worker-outcome semantics.
**Independent test**: Isolated scalar worker outcome execution captures native failure/error bodies and proves legacy code tokens never become a Boolean action.
**Depends on**: T003; all job retry, timeout and variable checks are canonical before error-action changes.

- [x] T005 [US2] Split the BPMN-error action from `--set-error-code`, rename `--message` to `--set-error-message` and failure `--retry-backoff` to `--set-retry-backoff` in `cmd/update_job.go`, `cmd/update_job_request.go`, `cmd/update_job_outcome.go`, `cmd/update_job_batch.go` and token-aware migration validation in `cmd/flag_migration.go`; adapt `cmd/update_job_outcome_test.go`, `cmd/update_job_outcome_guard_test.go`, `cmd/update_job_batch_test.go` and `cmd/flag_intent_job_test.go`. Apply the quoted worker-action constraint: only a bare active action with required valid code dispatches; reject code without action, missing/empty/blank code, incompatible actions, legacy positionals and every attached action value including `=true`, `=false` and `=` even alongside a canonical code. Accepted code values `true`/`false` via `--set-error-code` stay opaque; flag-looking strings in another value or after `--` are not action tokens. Preserve technical-failure required retries/backoff, message/variable exact values, completed-job guard, existing BID exclusions and unsupported batch modes; no backend field/schema change. Add failing parse/execution tests first, implement, format and run `go test ./cmd -run 'TestFlagIntentJob|TestUpdateJob|TestScalarJobCompletedGuard' -count=1` and affected payload/terminal/error cases. Update metadata, `README.md`, maintained BPMN-error/failure invocation fixtures and generated docs together; normal/quiet/verbose/JSON failure routes follow execution rules.

**Checkpoint**: US2 action/value grammar is complete; legacy error-code syntax cannot execute as a new action.

## Phase 5: User Story 4 — Migrate Automation with Clear Guidance (P1)

**Goal**: Prove production migration failures and runtime outcomes obey the configured reporting contract, using support built once in T001.
**Independent test**: Production write commands in isolated subprocess/real-terminal fixtures reject old syntax with the effective logger and exactly one shared envelope, preserving mutation results/exit classes on backend failures.
**Depends on**: T002–T005. US3 owns query-specific legacy rejection; the final consistency checkpoint includes those later consumers.

- [x] T006 [US4] Complete production migration/output acceptance in `cmd/flag_migration_test.go`, `cmd/command_contract_test.go`, `cmd/capabilities_test.go`, `cmd/completion_test.go`, `cmd/cmd_error_envelope_modes_test.go` and the existing affected command logging/terminal test families; fix only necessary integration of T001 support in `cmd/root.go`, `cmd/bootstrap_errors.go` and `cmd/cmd_bootstrap_failure.go`, without reimplementing the scanner or changing error schemas. Cover every migrated write spelling and command alias, old/new conflicts in both orders including equal/zero values, bad token before later config/profile/JSON/quiet/logger flags, no-err-codes, no remote bootstrap/prompt/mutation for migration failures, plain/structured configured/inherited destinations and local-config fallback. Through real command execution preserve successful/dry-run/verbose INFO routing and WARN/quiet suppression, existing output precedence, exact empty payloads/zero-byte keys, real-terminal prompts and prompt-free empty scopes, sparse/incomplete discovery, nonempty/explicit-key/abort cases, original exit class and partial results. Add repeated multi-resource fake failures with distinctive URL/method/body/key: normal/verbose must compactly group causes without raw details on either stream, quiet retains compact error, and DEBUG via flag and configured level independently shows existing redacted evidence with clean stdout. Assertions must fail on direct prints/default-logger bypass, raw normal-level aggregates or duplicate envelopes. Write missing assertions before fixes, format and run `go test ./cmd -run 'TestFlagIntentMigration|TestCommandContract|TestCapabilities|TestCompletion|TestCommandError' -count=1` plus the exact modified logging/terminal families, then shared cmd/config/docsgen checks if this task changes bootstrap. Update relevant migration guidance in `README.md` and regenerated docs if behavior guidance changes; query/enrichment-specific rejection assertions are added with US3, not preemptively against still-supported old names.

**Checkpoint**: Production write migration/reporting acceptance passes; remaining query spellings are migrated and certified in US3 and the final checkpoint.

## Phase 6: User Story 3 — Identify Returned Detail and Suppression (P2)

**Goal**: Canonical enrichment/suppression names retain query/watch behavior and current output support.
**Independent test**: Isolated query and embed-list execution compares returned stats/paths/suppression, rejects removed names and preserves existing watch/output exclusions.
**Depends on**: T001; scheduled after P1 work to serialize shared metadata/docs. Does not require new backend behavior.

- [x] T007 [US3] Rename `get pd --stat` to `--with-stats`, `embed list --details` to `--with-paths` and `get incident --with-no-error-message` to `--no-error-message` in `cmd/get_processdefinition.go`, affected `cmd/get_processdefinition_watch.go`/lifecycle validation, `cmd/embed_list.go` and `cmd/get_incident.go`; adapt `cmd/get_processdefinition_test.go`, query/watch/terminal/envelope tests, incident validation/output tests and add `cmd/flag_intent_enrichment_test.go`. Update all presence/exclusion checks, migration diagnostics, help/capabilities/completion tests and `demos/vhs/fast-start.tape`. Preserve statistics/path content, default/truncation behavior, incident JSON/keys/pi-keys exclusions and watch restrictions; existing with-* enrichments and hidden log-with-source remain unchanged. Prove old/new/equal conflicts fail using configured reporting and one supported envelope, with no query/backend call for migration rejection. Add failing command-path cases first, implement and format; run `go test ./cmd -run 'TestFlagIntentEnrichment|TestGetProcessDefinition|TestGetIncident|TestEmbed' -count=1` and exact modified watch/terminal/metadata cases. Update `README.md`, maintained scripts/fixtures/current command-matrix text and regenerate CLI docs without executing live suites.
  - Complete the canonical-surface inventory with this last consumer: Complete feature-wide canonical-surface contract coverage in `cmd/flag_migration_test.go`, `cmd/command_contract_test.go`, `cmd/capabilities_test.go`, `cmd/completion_test.go` and `docsgen/main_test.go`, alongside `README.md` and source metadata fixes identified by the inventory. Add any missing table-driven assertion covering all 16 rename rows and the error-action split; assert obsolete registrations/help/capability/completion entries absent, preserved filters/controls/public fields/result schemas unchanged, and all error classes/no-op/partial-output expectations retained. Statically audit `scripts/`, `demos/vhs/`, `integration/cli/`, `integration/assets/command-matrix.md` and current docs for maintained invocations; preserve historical feature evidence, never run live helpers. Format touched Go files and run missing targeted assertions first, regenerate via `make docs-content` before assertions that read generated references, then `go test ./cmd ./config ./docsgen -count=1` and `git diff --check`; reuse earlier successful runtime evidence when no executable change invalidates it. Run `make test` only for justified broader shared-runtime blast radius, never merely for finalization. Record actual checks and real-engine coverage limitation in `specs/369-standardize-flag-intent/progress.md`. Perform this final surface coverage in the same US3 work unit after its query consumers exist; no separate polish lifecycle or unrelated implementation is introduced.

**Checkpoint**: All rename rows and action/value rules are implemented; US4 query-surface acceptance now has its production consumers.

## Phase 7: Polish & Cross-Cutting Consistency

No separate checkbox: final canonical-surface assertions and remaining maintained-invocation alignment are included with the last query consumer. Every earlier work unit already owns its own tests, docs, formatting and progress; this phase introduces no postponed implementation.

## Dependencies & Execution Order

```text
T001 → T002
     → T003 → T005
T002 → T004
T002,T003,T004,T005 → T006
T001,T006 → T007
```

Default execution is numerical order: T001, T002, T003, T004, T005, T006, T007. No [P] markers: these work units overlap shared metadata, README/generated references or command-global test state. US1 owns T002–T004; US2 owns T005; US4 owns T006 and receives final query-surface acceptance from US3's T007. The US1 job exclusions also receive validation in T005. Story acceptance dependencies share existing implementation rather than duplicate it. Full US4 acceptance completes after T007; T006 independently establishes production write/reporting behavior.

### Parallel Examples per Story

- **US1**: After T001, an isolated reviewer could inspect run payload assertions while another examines repair fixtures; implementation remains serial because README/generated artifacts overlap. Shared variable consumers must stay coordinated.
- **US2**: Within T005, independent read-only inspection of the legacy-token matrix and captured worker payload expectations is possible; one owner changes job parser/state/tests serially.
- **US4**: Within T006, review plain/structured destination expectations separately from the DEBUG/redaction expectations; do not run tests concurrently if they mutate Cobra globals/environment.
- **US3**: Within T007, independent read-only review of stats/path/suppression expectations is possible; one owner integrates changes and regenerates shared docs.

These examples identify safe review opportunities, not authorized agent launches or separate implementation tasks.

## Coverage Ownership

| Acceptance / requirement | Implementation and validation owner |
| --- | --- |
| Shared diagnostic recovery, leaf identity, token boundaries, local fallback | T001; production acceptance T006/T007 |
| Update element/process/task variables, task-assignment dispatch, repair variable/file sources | T002 |
| Job mutation/selector retry split, exact zero, HTTP/job timeout independence, job variables/atomic BID completion | T003 |
| Process creation BID/variables, repair retry/job timeout and zero exception | T004 |
| Worker action/code/message/backoff and legacy Boolean-looking codes | T005 |
| Query stats/path/suppression, final canonical-surface inventory | T007 |
| FR-001 full rename table; FR-004 BID rules; FR-006 defaults/values | T002–T005, T007 |
| FR-002/FR-005 selectors/timeouts | T003, existing get-job regression coverage |
| FR-003 action/value grammar | T005 |
| FR-007/FR-008 breaking policy/conflicts | T001 support; T002–T007 consumers/acceptance |
| FR-009/FR-014 unchanged roles/layering/schemas | Every task; T007 final surface comparison |
| FR-010–FR-012 output/logging/JSON/partial evidence | Every affected task's execution checks, T006 matrix, T007 query checks |
| FR-013 documentation/current fixtures | Every command task; T007 complete inventory |
| SC-001–SC-006 | Above owners plus combined evidence at T007 |

## Implementation Strategy

Deliver T001 first, then the US1 writes with their tests/docs; that is the initial operator-value increment. Finish US2 to complete job grammar before treating the entire US1 acceptance set as complete. US4 certifies production migration/output behavior and US3 adds query-surface naming. The final consistency checkpoint closes full-feature acceptance. A partial increment is not the complete v5 migration and must not be described as one.

Do not implement another story or polish section within a Ralph iteration. Task responsibilities include context reading, branch checks, appropriate tests, docs, progress and commit discipline; none are standalone setup chores. Each work unit must compile and pass its closest checks. The optional Ralph hook does not launch automatically.

## Consolidation and Final Counts

Seven unchecked work units: one foundational, US1 three, US2 one, US4 one and US3 one. No standalone setup, registration-only, documentation-only or routine review task. Production contract acceptance remains a cohesive US4 outcome, including necessary integration fixes. See [consolidation-review.md](consolidation-review.md) for original-to-final ID mapping, code evidence, adjacent/nonadjacent merge review and retained scope boundaries. Proposed quickstart test names are introduced with their owning tasks; existing related test families remain required.

MVP: T001 plus US1's T002–T004 provides an initial write-input increment; complete T005 before claiming the entire US1 job-mode acceptance set, and T006/T007 before claiming the full v5 migration. There is no automatic launch, commit, release or live validation step.
