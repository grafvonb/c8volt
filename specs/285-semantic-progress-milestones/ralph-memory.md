# Ralph Memory

Feature: 285-semantic-progress-milestones
Started: 2026-08-31T17:14:26Z

## Codebase Patterns
- Active feature artifacts live under `specs/285-semantic-progress-milestones/`; Ralph iterations for this feature must include `--implementation-context specs/ralph-implementation-rules.md`.
- Branch is `285-semantic-progress-milestones`; issue-backed commit subjects for this feature use Conventional Commits with scope `ralph` and end with `#285`.

## Decisions
- Phase 1 setup was completed as a tracking-only work unit before any source implementation.
- The canonical completion fact is `OpsCompletionProgress` / `CompletionProgress` with `submitted`, `confirmed`, and `failed` dispositions; `AffectedCount *int` distinguishes unavailable from trustworthy zero.
- Completion facts are mirrored mechanically through `c8volt/foptions` and `c8volt/ops`; command wording remains out of domain, services, and facades.
- The first command reporter scaffold is isolated in `cmd/ops_semantic_progress.go`; it owns one workflow-priority activity, mutex-protected completion aggregation, affected-count invalidation, family vocabulary, and idempotent close.
- Completion output policy stays command-local in `cmd/ops_progress_mode.go`: default human allows transient and future paced aggregate progress, verbose/debug allow per-item durable lines, quiet allows failure warnings only, and JSON/keys-only/automation stay silent.
- Process-instance bulk services now emit one completion fact per executed create/cancel/delete worker. Phases are `create`, `cancel`, and `delete`; cancel/delete facts count `process-instance tree(s)` and use `affected process instances` only when a per-worker delta is trustworthy.
- Process-instance bulk legacy ticker progress is disabled when a structured progress callback is installed, but existing frozen-scope callback events and final service summaries are unchanged.
- Process-instance direct-key, stdin-key-equivalent, and search-selected cancel/delete command paths now start a post-confirmation semantic reporter and pass a completion-only callback to the mutation facade call; planning preflight/page/frozen-scope progress remains on the existing planning callback.
- Process-instance command affected-count rendering is initially enabled only for scopes that can be proven from the frozen command impact (`one root` or `affected == roots`); the reporter still permanently invalidates affected output if any completion arrives without a trustworthy delta.
- Process-definition service deletion completion facts use phase `delete process definitions`, core resource `process definition(s)`, identity = process-definition key, and submitted/confirmed/failed disposition based on `--no-wait`, response OK, and errors; affected counts are intentionally unavailable for this scope.
- All-process-definition purge now reattaches request-owned progress to the destructive delete options so APD request progress receives process-definition deletion facts through the reused delete service path.

## Gotchas
- `progress.md` and `ralph-memory.md` started untracked in this worktree; include them with the coordinated task commit.
- `c8volt/foptions` previously had no test file; `c8volt/foptions/options_test.go` now covers service-to-facade progress callback mapping.
- `cmd/ops_analyse_slow_process_instances_progress_test.go` participates in broader `Progress|Activity` runs; apply output-mode globals after `resetOpsSlowProcessAnalysisTestFlags(t)` because that helper now clears shared mode flags for isolation.
- `cmd/ops_semantic_progress_test.go` now asserts all 64 concurrent reporter updates produce the exact completed-count sequence under `-race`; use `requireOpsSemanticProgressCompletedSequence` for similar aggregate-sequence checks.
- `toolx/logging/activity_test.go` has a workflow-priority regression proving lower-priority HTTP/wait updates cannot replace the visible workflow aggregate.
- `internal/services/processinstance/bulk_test.go` filters callback events by kind because structured callbacks now receive both frozen-scope and completion events.
- Delete affected counts are nil for expanded multi-root scopes when only an aggregate affected count is available; cancel can use the returned affected process-instance slice length as a trustworthy per-root delta.
- `cmd/processinstance_mutation_progress_test.go` has shared helpers for simulating process-instance completion facts and asserting workflow-priority semantic activity across cancel/delete direct and stdin-key-equivalent paths.
- In `cmd/cancel_processinstance_selector.go`, close the per-page semantic reporter immediately after each page mutation call; using `defer` inside the visitor would keep prior page activities alive until traversal completes.
- Process-definition deletion uses a serial first delete before the worker pool to catch Camunda delete-history request-shape errors; emit a failed completion for that first attempted key and no facts for the unscheduled remainder.
- APD force cleanup progress includes nested process-instance cancel/delete completion facts plus process-definition deletion facts; filter by phase `delete process definitions` in APD service tests that only care about the process-definition completion scope.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./internal/domain ./c8volt/foptions ./c8volt/ops ./toolx/logging -race -count=1`
- `go test ./internal/domain ./c8volt/foptions ./c8volt/ops ./internal/services -race -count=1`
- `go test ./cmd -run 'Progress|Activity' -race -count=1`
- `go test ./cmd -run 'TestOpsSemanticProgress' -race -count=1`
- `go test ./toolx/logging -race -count=1`
- `go test ./internal/services/processinstance/... -run 'Progress|Cancel|Delete|CreateNProcessInstances' -race -count=1`
- `go test ./internal/services/processinstance -race -count=1`
- `go test ./cmd -run 'TestProcessInstanceMutationDirectAndStdinKeysUseSemanticCompletionActivity|TestCancelProcessInstanceSearchSelectedUsesSemanticCompletionActivity|TestDeleteProcessInstanceSearchSelectedUsesSemanticCompletionActivity' -race -count=1`
- `go test ./cmd -run 'ProcessInstance.*(Progress|SearchSelected|SearchProgress|WorkflowImportance|DryRun_Search|WithPlan)|CancelProcessInstanceSearch|DeleteProcessInstanceSearch' -race -count=1`
- `go test ./cmd -run 'ProcessInstance' -race -count=1`
- `go test ./internal/services/processdefinition ./internal/services/ops -run 'DeleteProcessDefinitionResources.*Completion|DeleteProcessDefinitionResourcesStopsOnDeleteHistoryRequestShapeError|PurgeAllProcessDefinitionsForceCleanupDeduplicatesProcessInstanceRoots' -race -count=1`
- `go test ./internal/services/processdefinition ./internal/services/ops -run 'Progress|Delete|PurgeAllProcessDefinitions' -race -count=1`
- `go test ./internal/services/processdefinition/... ./internal/services/ops/... -race -count=1`
- `git diff --check`

## Do Not Repeat
- Do not reintroduce semantic progress wording into services or facade converters; completion facts remain wording-free and command renderers choose verbs.

## Current Handoff
- Next iteration should continue User Story 1 at T011 by adding per-definition deployment visibility and no-wait acceptance progress tests in `internal/services/resource/payload/payload_test.go` and `internal/services/resource/v87/service_test.go`, `internal/services/resource/v88/service_test.go`, `internal/services/resource/v89/service_test.go`, and `internal/services/resource/v810/service_test.go`; T019 still needs command reporter wiring for basic/APD process-definition deletion.
