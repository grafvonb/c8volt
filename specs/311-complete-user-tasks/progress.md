# Ralph Progress Log

Feature: 311-complete-user-tasks
Started: 2026-10-24 08:09:58

---


---
## Iteration 1 - 2026-10-24 08:17
**Work Unit**: Phase 2 Foundational T001 — Native completion evidence across supported adapters; completed. T002 remains in the same phase for a subsequent iteration.
**Tasks Completed**:
- [x] T001: Implement native completion, domain evidence, all supported generated-client contracts, retry/cancellation handling and affected test doubles.
**Tasks Remaining in Work Unit**: 0 in selected T001; T002 is the next foundational work unit. Three feature tasks remain overall.
**Commit**: This work-unit commit
**Files Changed**:
- AGENTS.md
- c8volt/task/client_test.go
- c8volt/task/search_test.go
- c8volt/task/variables_test.go
- internal/services/common/job_completion.go
- internal/services/usertask/api.go
- internal/services/usertask/bulk_test.go
- internal/services/usertask/search_test.go
- internal/services/usertask/update_test.go
- internal/services/usertask/update_wait_test.go
- internal/services/usertask/v810/contract.go
- internal/services/usertask/v810/service_test.go
- internal/services/usertask/v88/contract.go
- internal/services/usertask/v88/service_test.go
- internal/services/usertask/v89/contract.go
- internal/services/usertask/v89/service_test.go
- internal/services/usertask/variables_test.go
- internal/services/usertask/workflow_test.go
- internal/domain/usertask_completion.go
- internal/services/common/usertask_completion.go
- internal/services/usertask/v810/completion.go
- internal/services/usertask/v810/completion_test.go
- internal/services/usertask/v88/completion.go
- internal/services/usertask/v88/completion_test.go
- internal/services/usertask/v89/completion.go
- internal/services/usertask/v89/completion_test.go
- specs/311-complete-user-tasks/checklists/requirements.md
- specs/311-complete-user-tasks/contracts/cli.md
- specs/311-complete-user-tasks/contracts/facade-service.md
- specs/311-complete-user-tasks/data-model.md
- specs/311-complete-user-tasks/plan.md
- specs/311-complete-user-tasks/progress.md
- specs/311-complete-user-tasks/quickstart.md
- specs/311-complete-user-tasks/ralph-memory.md
- specs/311-complete-user-tasks/research.md
- specs/311-complete-user-tasks/spec.md
- specs/311-complete-user-tasks/tasks.md
**Learnings**:
- Added explicit v88/v89/v810 generated mappings with shared evidence accumulation in common. Reused the existing structured resource-exhaustion parser without changing job behavior. Existing initial feature planning artifacts and AGENTS.md active-plan pointer are preserved in this transaction.
- Red gate: completion tests initially failed to compile because the new native methods and domain types did not exist. Green gate: `isolated go test ./internal/services/usertask/... -run 'Complete|Completion' -count=1` passed for every supported adapter.
- Regression/race gate: `isolated env GORACE=atexit_sleep_ms=0 go test ./internal/services/usertask/... ./c8volt/task ./internal/services/common -race -count=1` passed. Existing factory/default-version, read, variable and facade workflows remain covered.
- Shared-helper consumer gate: `isolated go test ./internal/services/job/... -run 'Completion|Complete' -count=1` passed (waiter had no matching tests). Repository compilation gate: `isolated go test ./... -run '^$'` passed; this is compilation evidence, not a full test-suite pass. Touched Go files were formatted and `git diff --check` passed.
- Adapter cases cover real generated wire mapping for omitted/empty/nested payloads and no-content acceptance; explicit backend rejection; exhausted 429/503/network retries; nonretrying 404/409/500/504; nil/malformed responses; pre-submission cancellation/deadline; canceled/deadline retry backoff; last-status preservation; ambiguity followed by rejection or acceptance; invalid payloads; snapshot ownership; and forbidden-call guards.
- Diagnostic review: new `err.Error()`/error formatting sites populate returned machine evidence only. No new direct prints or normal-level raw-error logging. Fresh subprocess tests assert one bounded configured INFO retry record for verbose in both plain/structured formats and zero records for compact/suppressed modes; distinctive URL/method/body/key markers are absent. Transport DEBUG instrumentation is unchanged. Completion facade/render/logger/exit integration and command-level diagnostic matrices remain T002–T004 obligations.
- No CLI behavior changed, so generated CLI docs were not regenerated. No real Camunda environment was contacted; engine listener/assignment/eventual-state behavior remains unverified. Durable reuse details and the T002 handoff are in ralph-memory.md.
---

---
## Iteration 2 - 2026-10-24 08:24
**Work Unit**: Phase 2 Foundational T002 — collection completion service and public facade; completed
**Tasks Completed**:
- [x] T002: Deliver the validated, ordered collection coordinator and thin public completion API with isolated service/facade tests.
**Tasks Remaining in Work Unit**: 0; T003 and T004 remain in the subsequent US1 phase.
**Commit**: This work-unit commit
**Files Changed**:
- c8volt/task/api.go
- c8volt/task/client.go
- c8volt/task/completion.go
- c8volt/task/completion_test.go
- c8volt/task/convert.go
- c8volt/task/model.go
- cmd/process_api_stub_test.go
- internal/services/usertask/completion.go
- internal/services/usertask/completion_test.go
- specs/311-complete-user-tasks/progress.md
- specs/311-complete-user-tasks/ralph-memory.md
- specs/311-complete-user-tasks/tasks.md
**Learnings**:
- Red gates: new service tests failed to compile before CompleteUserTasks existed; new facade tests failed before the public method/models existed. Both focused gates passed after implementation: `isolated go test ./internal/services/usertask/... -run 'Complete|Completion' -count=1` and `isolated go test ./c8volt/task -run 'Complete|Completion' -count=1`.
- Race/regression gate passed: `isolated env GORACE=atexit_sleep_ms=0 go test ./internal/services/usertask/... ./c8volt/task -race -count=1`. After adding a final acceptance/caller-error regression, `isolated env GORACE=atexit_sleep_ms=0 go test ./internal/services/usertask -race -run TestCompleteUserTasksCallerErrorAfterResult -count=1` also passed.
- Public-interface consumer gate passed: `isolated go test ./... -run '^$'` compiled all packages without executing their tests. Affected command regressions passed: `isolated go test ./cmd -run 'UpdateUserTask|GetUserTask|HasUserTasks|UserTaskHTTPFallback' -count=1`. All touched Go files were formatted; `git diff --check` passed.
- Service coverage includes whole-input validation before calls (including invalid late keys, cyclic/non-JSON data, empty input, no-wait, negative workers), dependency guard, nil/empty/nested snapshots, stable duplicate removal, sequential calls, reverse completion ordering, actual bounded/automatic/unlimited/explicit-precedence concurrency, continue-on-error, fail-fast queued slots, in-flight caller-context preservation, pre-start/mid-flight cancellation, pre-expired deadline and acceptance retained alongside final cancellation. Nil embedded test APIs reject forbidden reads, writes and polls.
- Facade coverage includes both constructors, worker/options propagation, nested input ownership, nil versus empty variables, all status/submission enums and JSON omission rules, initialized empty items, copied output slices, classified partial errors and unchanged failure exit codes. Existing task reads, resolution, variable planning/update/no-wait and schemas passed their package regressions.
- Failure-path review: adapters return structured machine evidence; coordinator joins real native failures and caller cancellation; facade maps all items before ferrors.FromDomain. No new direct print, raw error logger call, renderer or exit path was added. CLI renderer/logger/exit integration remains T003/T004; these tests do not claim command-level completion diagnostic compliance.
- Only a test-double field and rejecting method changed under cmd; no command mode or source metadata changed, so no CLI docs regeneration is due. No live environment was contacted; actual engine listener, assignment eligibility and eventual state remain unverified. Next work is T003, documented in Current Handoff.
---

---
## Iteration 3 - 2026-10-24 08:32
**Work Unit**: Phase 3 / US1 T003 — local completion input and preview/result contract completed; US1 delivery remains partial
**Tasks Completed**:
- [x] T003: Freeze and validate local completion input, build typed preview/execution reports, and render safe selected-format results with bounded failure formatting.
**Tasks Remaining in Work Unit**: T004 remains in US1; no additional task was started in this iteration.
**Commit**: This work-unit commit
**Files Changed**:
- cmd/update_usertask_complete_input.go
- cmd/update_usertask_complete_input_test.go
- cmd/cmd_views_usertask_complete.go
- cmd/cmd_views_usertask_complete_test.go
- specs/311-complete-user-tasks/tasks.md
- specs/311-complete-user-tasks/ralph-memory.md
- specs/311-complete-user-tasks/progress.md
**Learnings**:
- Prerequisite/context review selected T003 after completed T001/T002. T003/T004 separation remains deliberate: this work establishes local parsing/report behavior without registering a partially implemented flag. No CLI source metadata changed, so docs generation remains T004.
- Red/green evidence: input tests initially failed compilation before the parser existed, and view tests initially failed before the report/render helpers existed. Final focused gate passed: `isolated go test ./cmd -run 'TestUpdateUserTaskCompletionInput|TestUpdateUserTaskCompletionView' -count=1`.
- Regression gate passed: `isolated go test ./cmd -run 'TestUpdateUserTask|TestUpdateProcessInstanceCommand_InvalidVarsFailBeforeMutation' -count=1`. This includes existing variable-mode source validation, native-version command fixtures, scope/no-op/output behavior and real-terminal confirmation regressions using test-owned dependencies. New completion command execution/terminal assertions remain T004.
- Focused race gate passed: `isolated env GORACE=atexit_sleep_ms=0 go test ./cmd -run 'TestUpdateUserTaskCompletion' -race -count=1`. All four new Go files were formatted. Tests ran with the quickstart clean environment and no integration build tags; selected existing subprocess/terminal fixtures use explicit fake configurations. No live test target was discovered or contacted.
- Input tests cover flag/stdin merge order, duplicate collapse, whitespace/blank lines, empty combined selection, invalid late keys, ASCII digit length, read failures, opt-in configured stdin, omitted/empty/nested/file variables, conflicting/unreadable/empty/malformed/non-object/trailing sources, worker/no-wait validation, and JSON confirmation/verbose guardrails. pflag drops empty repeated --key occurrences: the tested wrapper records them while delegating legacy values/reset behavior. T004 must register that wrapper on the production flag and test re-registrations.
- Parser-to-preview tests feed frozen inputs directly into the report/view without constructing a backend dependency; human/verbose/JSON/keys/quiet combinations preserve selection order, payload presence/count and preview-only intent. Execution tests cover accepted/failed/timed-out/canceled/not-attempted evidence, partial JSON with original class/detail, exact human and accepted-only keys output, zero-byte no-accepted keys, quiet machine modes, JSON precedence, initialized arrays/zero counts, evidence omission, defensive result copying and write failures. Single-envelope tests require EOF.
- Failure-path review: native adapters retain typed submission/status and raw machine error evidence; the coordinator preserves ordered partial results and real errors; the facade maps before conversion. New views pass the original error only to resultEnvelopeForError, attach the report once, and never interpolate item Error or unknown enum text into human output. All three new direct print sites are selected-format results on configured stdout; aggregate failure formatting emits nothing itself and bounds reason groups to three plus other failures. Configured logger emission, original exit handling, DEBUG/redaction and diagnostic mutation-sensitivity tests are explicitly T004 work; renderer tests do not claim that compliance.
- The source files keep input handling and final formatting separate, with no CLI-owned workers/retries/polling, no service changes, and no shared variable-parser change. Real-engine task-listener, assignment eligibility and eventual state remain unverified. Current Handoff points to T004.
---

---
## Iteration 4 - 2026-10-24 08:46
**Work Unit**: Phase 3 / US1 T004 — complete command delivery, operator documentation and final isolated validation completed
**Tasks Completed**:
- [x] T004: Expose native user-task completion with command-level safety, output/diagnostics, real-terminal confirmation, compatibility coverage and generated documentation.
**Tasks Remaining in Work Unit**: 0; all feature tasks are complete.
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- cmd/command_contract_test.go
- cmd/update.go
- cmd/update_usertask.go
- cmd/update_usertask_complete.go
- cmd/update_usertask_complete_test.go
- cmd/update_usertask_complete_execution_test.go
- cmd/update_usertask_complete_output_test.go
- cmd/update_usertask_complete_terminal_test.go
- cmd/update_usertask_complete_logging_test.go
- docs/cli/c8volt_update.md
- docs/cli/c8volt_update_user-task.md
- docs/cli/command-tree.md
- docs/index.md
- specs/311-complete-user-tasks/tasks.md
- specs/311-complete-user-tasks/ralph-memory.md
- specs/311-complete-user-tasks/progress.md
**Learnings**:
- Selection rechecked T001–T003 completion and their explicit dependency chain; only T004 was selected. No task expansion, second story, agent delegation or live-environment work occurred. Commit policy resolves to conventional / ralph / auto issue, with subject suffix #311.
- The command registers --complete and the empty-key occurrence tracker, dispatches before mandatory variable parsing, freezes one request, previews locally or confirms once, and delegates to the existing collection facade. Explicit-key tenant context carries configured provenance without invented resolved tenants or reads. The dedicated completion mode file contains the lifecycle; the base command contains only flag/metadata/dispatch changes. Backend pools, retries and evidence remain in their established service owners.
- Red/green evidence: the first actual selection test failed on the absent --complete flag. Final focused checks passed for completion selection/payload/validation/output, failure classes and no-error-code overrides, workers/fail-fast/cancellation, rendering failures, terminal execution, diagnostics and metadata. The complete focused gate was `isolated go test ./cmd -run 'TestUpdateUserTaskCompletion|TestCommandCapabilityForCommand_UpdateUserTaskContract' -count=1`; subsequently added output-write failure and strengthened tenant assertions passed their targeted command gate and the final full race suite.
- Actual command coverage includes every alias and key source across 8.8/8.9/8.10; stable uniqueness; flags plus empty stdin; invalid late/empty input; optional/empty/matching/nested/file variables; invalid/conflicting sources; workers/no-wait/JSON guardrails; explicit false variable dispatch; unchanged unsupported-version bootstrap handling; exact preview/accepted/partial/all-rejected results in human/verbose/quiet/JSON/keys combinations; one envelope followed by EOF; original rejection/timeout classes; zero-byte no-accepted keys; and no forbidden read, scope-write, assignment or polling request. Barriers prove workers one/two and explicit-worker precedence, fail-fast retains in-flight acceptance and serial queued slots, and a separate test control handshake cancels a known native request while preserving accepted and unattempted outcomes.
- Real-terminal checks passed on this macOS host using testx.NewCmdTerminalRunner: real terminal stdin, redirected output files, command-configured and inherited stderr, one exact plain whole-selection prompt, yes/no/default/EOF decisions, preview, auto-confirm, automation and quiet machine output. Abort/preview submitted no native requests. No terminal coverage was skipped.
- Diagnostic matrix passed with repeated distinctive URL/method/body/key/secret failures, both logger formats and effective destinations, normal/verbose/quiet, JSON partial results, DEBUG flag/config independently of verbose, and quiet precedence. Assertions check actual logger framing, ERROR severity, exact single grouped diagnostic, no normal-level raw details on either stream, preserved machine evidence, redaction, actual request/retry counts and unchanged result stdout. Sequential/bounded retries cover 429 and RESOURCE_EXHAUSTED 503. Deliberately replacing the aggregate logger call with a direct stderr print failed the framing assertion; deliberately logging operationErr.Error() also failed. Both mutations were restored immediately and the real implementation passed subsequent checks.
- Failure-path audit: native adapter evidence flows through the ordered coordinator and facade conversion into one completion report; results render on stdout, one safe non-JSON aggregate is logged through NewCli's configured logger, and exit uses the original converted error. Local validation/abort retain existing helpers. The new runner has no direct result/diagnostic print and never formats a raw operation error. Render failures emit one constant logger message, never invoke another renderer, and preserve an existing mutation failure code. Tests detect duplicate rendering and raw/error-routing regressions. --no-err-codes retains its existing zero-exit override; unsupported versions retain bootstrap local-precondition diagnostics.
- Regression gates passed: `isolated go test ./cmd -run 'UpdateUserTask|GetUserTask|HasUserTasks|UserTaskHTTPFallback|APIDiagnostics|TestCommandCapabilityForCommand_UpdateUserTaskContract|TestGetJobAndUpdateJobHelp_DocumentsDiscoveryAndMutationGuards|TestUpdateProcessInstanceHelp_DocumentsVariableUpdateDiscovery' -count=1`; `isolated go test ./c8volt/task ./internal/services/usertask/... -count=1`; and `isolated go test ./internal/services/httpc -run 'APIDiagnostics' -count=1`. Patterns include existing variable scope/shared-target/no-op/wait/no-wait, task read/resolver and get-pi has-user-tasks cases.
- Updated both command and parent help/examples, control wording, metadata and README; regenerated references/homepage using `isolated make docs-content` and passed `isolated go test ./docsgen -count=1`. All touched Go files were formatted. Final untagged `isolated make test` passed (`go test ./... -race -count=1`, cmd package 171.883s); git diff --check passed. Tests used the clean-environment wrapper, no integration build tags, test-owned configurations and isolated fakes. Real-engine listener timing, assignment eligibility and eventual task/process state remain unverified. Current Handoff is terminal.
---

## Pragmatic simplification and T005 — 2026-10-24

- Moved all ten shared HTTP outcome/retry/cancellation test functions from the three adapter suites to `internal/services/common/usertask_completion_test.go`. Every original test function remains represented in shared or version-specific coverage; adapters retain wire mapping, nullable payloads, nested ownership, invalid payloads, logging options, and added response/body/error delegation checks. Combined adapter/common test source is 532 lines shorter.
- Removed the CLI's copy of its freshly decoded map and the facade's redundant JSON/key copies. The coordinator still validates and snapshots before scheduling; native adapters still isolate generated-client input and direct callers. Existing caller-mutation and concurrency tests remain intact.
- Fixed T005 by separating shared source selection from decoding. Completion uses `json.Decoder.UseNumber`, rejects outer null/non-objects and trailing input, and preserves nested integers/decimals exactly. Existing variable-update decoding is unchanged. Actual-request fixtures now decode with `UseNumber`, so assertions cannot conceal rounding.
- Before the fix, parser and native-request tests failed for both inline/file sources: `9007199254740993` became `9007199254740992`, and `0.123456789012345678901` became `0.12345678901234568`. After the fix, exact values reach both selected tasks; invalid/trailing inputs still submit zero requests.
- Focused numeric/invalid-input checks: 56 tests/subtests passed. Facade, coordinator and all three adapters passed their completion groups; the relocated common group passed 43 tests/subtests. A relocation-only compile reference and logger-fixture mismatch were corrected before final validation.
- Final validation: clean-environment `make test` (`go test ./... -race -count=1`, JSON test reporting only, no integration tags) passed 8,049 tests/subtests in 86 tested packages, including 417 user-task completion cases, with no test skips or failures. Another 26 packages contained no tests. `make docs-content`, Go formatting and `git diff --check` also passed.
- Existing machine envelopes, status/submission identifiers, counts, keys-only output and exit classification remain covered. No polling, backend workflow changes, or live-environment validation was added. Help, README and generated CLI documentation now mention exact completion numbers.

## Completed-task and assignee preflight — T006 — 2026-10-24

- Added one keyed native read per unique task before CLI approval. Service preparation freezes inputs and observations; the thin facade exposes prepare/execute so warnings precede approval without putting backend loops in cmd. Direct CompleteUserTasks composes the same path. No new service dependency, generated-client changes, persistent plan, assignment, or polling.
- Observed COMPLETED tasks receive no POST or variables. Existing failed/not_submitted evidence carries a safe conflict explanation and omits statusCode because no submission occurred. Other tasks continue unless fail-fast is set, in which case preflight conflicts prevent all submissions. The prompt counts only eligible submissions and is absent when none remain. Dry-run and invalid inputs remain request-free.
- Eligible unassigned tasks produce an advisory WARN. Failed/malformed/mismatched reads produce a bounded aggregate WARN and leave eligibility to the backend; they never imply completion or unassignment. Caller cancellation stops preparation. Reads can be stale, so submitted outcomes still use the native completion response.
- The initial service preflight tests failed before implementation. The later actual-command diagnostic tests exposed shared transport GET retries on 503 (six reads for three unique keys). A request-scoped WithoutReadRetry context now disables retries only for advisory preflight; the same transport still retries ordinary reads. The repaired diagnostic/transport/help gate passed 49 tests/subtests. Final service/facade completion checks passed 113 tests/subtests across five packages.
- Isolated command tests cover all-completed, mixed, unassigned, unavailable and stale observations, deduplication, fail-fast, supported versions, human/verbose/quiet/JSON/keys modes and quiet machine combinations. JSON retains its eight payload fields, existing status/submission enums, one envelope followed by EOF, ordered partial results and conflict exit classification. Completed tasks never appear in accepted-only keys output. Fixtures reject repeated task reads, reads after approval, and unrelated variable/assignment/poll requests.
- Real PTY tests exercise configured/inherited stderr with plain and structured logging, warning-before-prompt ordering, eligible-count approval/decline, prompt-free all-completed selections, local preview and invalid input. The failure diagnostic matrix combines a completed task, two failed reads containing distinct URL/method/body/key/secret markers, successful submission and backend conflict. Normal/verbose/quiet/JSON diagnostics use the configured logger; DEBUG flag and configured level expose actual exchanges with redaction. Temporarily routing the new unavailable-check warning directly to stderr failed the structured logger assertion; the mutation was restored immediately.
- Failure-path review: service builds safe preflight conflict evidence and joins actual submission failures; facade preserves results and converts the original error; views derive human reasons from typed statuses, not raw error strings; runner emits bounded warnings before approval, one non-JSON aggregate ERROR after rendering, and preserves the original exit class. Lookup error text is discarded from ordinary diagnostics; HTTP exchange visibility remains in the configured DEBUG path. No new result fields or duplicated envelopes were introduced.
- Updated spec, plan, data model, contracts, research, validation guide, task tracking, source help and README. Regenerated CLI docs/homepage with clean-environment make docs-content; formatting, feature links and whitespace checks passed. Historical T001–T005 zero-read execution expectations are explicitly superseded by T006; local-preview and scalar-adapter restrictions remain.
- Final validation: clean-environment `make test` (`go test ./... -race -count=1`, JSON reporting only, no integration tags) passed 8,161 tests/subtests in 86 tested packages, with no failures or test skips. This includes command preflight/terminal/diagnostic tests, metadata and docsgen tests, ordinary-read retry regressions and existing variable workflows. No live services were contacted; real-engine eventual visibility and assignment eligibility remain outside this isolated coverage. Changes are uncommitted.

## Human wording refinement — T007 — 2026-10-24

- Human summaries now omit zero outcome counters. Failed-only results no longer append the generic missing-response sentence. Errors lead with bounded grouped reasons instead of repeating result counters; completed-task conflicts state that the task is already completed and the completion request was not sent, with plural wording for multiple tasks.
- Human conflict/verbose remarks mention unapplied variables only when the existing VariablesProvided flag is true, including `{}`. Preview omits absent variables and describes an explicitly empty object without a zero variable counter. No service, request, retry, polling, exit or machine serialization behavior changed; original JSON error evidence and all zero count fields remain intact.
- Added exact command execution checks for one/two completed tasks, omitted/empty/nonempty variable payloads, human/verbose/JSON, conflict exit 6, one GET per key, no POST, configured logger severity and single-envelope EOF. Existing execution/terminal/diagnostic expectations now assert compact counters and reason-first errors; quiet, partial failures, keys-only, DEBUG redaction and original error classifications remain covered.
- Targeted `go test ./cmd -run TestUpdateUserTaskCompletion -count=1` passed 396 tests/subtests using isolated fakes and a clean environment. Temporarily replacing the aggregate logger call with direct stderr printing failed the structured framing assertion; temporarily logging the raw operation error failed the exact bounded diagnostic assertion. Both changes were restored before final regression validation.
- Updated source help and its metadata test, spec/plan/contracts/model guidance, research, quickstart, README and task tracking. Regenerated CLI reference/homepage through make docs-content. Historical machine evidence and earlier validation results remain preserved.
- Final validation: command and docsgen tests passed 4,903 tests/subtests with no failures or skips (`go test ./cmd ./docsgen -run "TestUpdateUserTaskCompletion|TestCommandCapabilityForCommand_UpdateUserTaskContract|Test.*" -count=1`, clean environment, isolated dependencies). Documentation regeneration, Go formatting and git diff --check passed. The previous T006 full race result remains historical; this rendering-only refinement used command/docs validation and did not run live checks. Changes remain uncommitted.

## Consistent update rhythm — T008 — 2026-10-24

- Ordinary human approval now follows configured INFO selection scope and plan, WARN advisories, a plain stderr confirmation, then selected-format stdout results. A single-key plan/prompt identifies the key; bulk plans show requested, eligible and already-completed counts, and prompts use the eligible count with singular/plural grammar. Mixed fail-fast plans explain zero submissions without prompting; all-completed selections retain the short conflict result/error path. Auto-confirm/automation skip the approval plan; quiet and machine modes suppress its INFO messages. Dry-run remains local and shows known scope plus a stdout intent preview.
- Single success is `updated user-task <key>: completion confirmed`; batches/failures use `update user-task: N requested; ...`, omit zero counters and call successful responses `completed`. Plans/previews add `variables: submit` only for a supplied source, including `{}`, without values. Assignee and unavailable-check warnings use correct singular/plural wording. Human confirmation still means the successful native response, with no independent state read-back or downstream waiting.
- Added CompletedCount to the existing private-data preparation plan and thin facade mapping, populated during the same reads. It is advisory and is not added to serialized results. No additional calls, polling, retries, tenant inference, scheduling changes or machine schema/status/count/error/exit changes were introduced. Failure tracing remains service evidence → facade conversion → stdout result/one JSON envelope → configured bounded non-JSON ERROR → original exit.
- Updated source help, README, spec FR-021, plan, contracts, data model, research, validation guide and task tracking. Regenerated CLI docs and homepage with clean-environment `make docs-content`. Feature links, Go formatting and `git diff --check` passed.
- Targeted completion, preparation/facade, metadata and docs tests passed 567 tests/subtests. New real-terminal tests exercise plain/structured configured/inherited logging, exact scope/plan/warning/prompt order, request counts, abort, fail-fast, preview, variables, quiet/keys/JSON, auto-confirm and automation. Existing bulk, conflict, partial-result, DEBUG and redaction tests remain in place. Temporarily printing the plan directly to stderr failed structured logger parsing; direct error printing and raw aggregate logging also failed their diagnostic assertions. All three mutations were restored before race validation.
- Race validation: clean-environment `make test` exercised all packages and found only the action-focused help test rejecting stream terminology in command help (8,235 passing tests/subtests; two failed events for the case and its parent). Removed the implementation wording from command help, retained routing documentation in README/contracts, regenerated docs, and passed the focused help checks. Then `go test ./cmd ./docsgen -race -count=1 -json` passed all 4,960 tests/subtests. Combined latest results are 8,237 passing tests/subtests across 86 tested packages; 26 other packages have no tests, and no test was skipped. All tests used clean environments and isolated fixtures; no live services were contacted. Changes remain uncommitted.

## INFO routing for all human completion output — T009 — 2026-10-24

- Replaced the completion renderer's direct human summary and verbose-row writes with the existing `renderHumanLine` INFO helper. Preview, single success, batch/partial/failure summaries and verbose rows now honor the configured logger format, destination and level. Human stdout is empty. JSON/keys-only branches are unchanged; plain prompts, WARN advisories, bounded ERROR diagnostics, original exits and quiet/machine precedence remain intact. No separate logger or synthetic prefix was introduced.
- The completed-task path now emits INFO `update user-task: 1 requested; 1 failed`, then one ERROR explaining the completed state/no-submission consequence, with exit 6, one GET and no POST. No backend, public facade, lookup, scheduling, retry or polling code changed in this refinement. Failure tracing remains service evidence → facade conversion → INFO human summary or one machine envelope → bounded non-JSON ERROR → original exit; human rows still use typed safe facts rather than raw error strings.
- Updated exact execution, real-terminal, preflight, partial-failure and DEBUG tests to assert INFO framing/severity/order, configured/inherited destinations in plain and structured formats, empty human stdout, unchanged JSON/envelope EOF and keys-only bytes, quiet suppression, prompt preservation and request counts. Added configured WARN/ERROR-level suppression cases. View tests now install a command-context logger; a failing stdout destination still fails machine writes but is irrelevant to human logger output.
- Targeted completion/help checks passed 521 tests/subtests. Temporarily reverting the summary to direct stdout or stderr printing, verbose rows to direct stderr printing, or the aggregate ERROR to raw operation-error logging failed the corresponding routing/framing/safe-message assertions. All four mutations were restored before regression validation.
- Revised FR-015/FR-021, output contracts, plan, model, research, quickstart, task coverage and README to supersede the human-stdout exception. Help documents configured log-level behavior without stream implementation details. Regenerated CLI references and homepage with `make docs-content`; Go formatting, local feature links and `git diff --check` passed. Historical T001–T008 evidence remains preserved.
- Final validation: clean-environment `go test ./cmd ./docsgen -race -count=1 -json` passed 4968 tests/subtests in both affected packages, without failures or skips. The prior full-repository race run remains historical; this renderer-only change used targeted and affected-package race validation. All dependencies were isolated fixtures; no live services were contacted. Changes remain uncommitted.
