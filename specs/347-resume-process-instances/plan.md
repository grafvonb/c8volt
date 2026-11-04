# Implementation Plan: Resume Process Instances

**Branch**: `347-resume-process-instances` | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/347-resume-process-instances/spec.md`; authoritative issue [#347](https://github.com/grafvonb/c8volt/issues/347).

## Summary

Add `resume process-instance` / `resume pi` for exact keys and explicit `--tree` scope. Reuse generated Camunda 8.10 single-instance resumption, existing descendant traversal, worker policy, retry/request-rate support, configured logging, activity, and result envelopes. Internal services own planning, collection execution, submission, and bounded observation; public facades only map/delegate/convert errors. One collection path handles singleton, multiple-key, and frozen tree targets.

Only a successful submission response establishes acceptance. Default confirmation requires a subsequent matching ACTIVE or COMPLETED observation; CANCELED/TERMINATED is a distinct conflict, and no-wait reports acceptance only. Tree discovery selects SUSPENDED members after traversing all reachable states, fails closed on incomplete discovery, and freezes targets before approval. Invalid input remains an error; a complete empty eligible scope is a prompt-free no-op.

## Technical Context

**Language/Version**: Go 1.26.0, repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, generated oapi-codegen Camunda client, standard `slog`, repository `toolx/pool`, HTTP retry/rate helpers, and descendant traversal. No new dependency.

**Storage**: No persistent storage or migration. Per-invocation plans and ordered outcome collections only.

**Testing**: Go testing/testify 1.11.1, generated-client mocks, test-owned HTTP fakes, `testx` CLI subprocess/terminal runners, and activity sinks. Integration source may change; live execution is excluded.

**Target Platform**: Existing Go CLI/library platforms; real-terminal checks on supported Unix environments. Unsupported PTY checks are reported as coverage gaps.

**Project Type**: CLI plus public Go facade library with version-specific internal adapters.

**Performance Goals**: Preserve bounded worker concurrency and existing request-rate control. Default workers follow `min(count, 2*GOMAXPROCS, 32)`; explicit positive workers/no-worker-limit follow current policy. Stable deduplication and linear accumulation in discovered members; no second discovery after approval. No new latency SLA.

**Constraints**: Configured 8.10 only; 8.8/8.9 fail before discovery/unsupported I/O. Existing backoff timeout defaults to two minutes and per-request timeout remains independent. Up to 1,000 direct children per parent lookup; no pagination changes. No native batches, root escalation, atomic guarantee, rollback, generic mutation framework, generated-client edits, or live validation.

**Scale/Scope**: One new verb with a process-instance command and alias; singleton/collection/tree inputs use the same collection executor. Existing suspension/cancel/delete behavior and public schemas remain stable. All research questions are resolved in [research.md](research.md).

## Constitution Check

*Gate evaluated before research and re-evaluated after Phase 1 design.*

| Principle / repository gate | Before research | Post-design evidence |
| --- | --- | --- |
| I. Operational proof | PASS: acceptance and observed outcome are separate | PASS: explicit state matrix, bounded read-only waiter, no-wait acceptance, partial evidence, no timestamp-only proof |
| II. CLI-first, script-safe | PASS: follow suspension contracts | PASS: exact/tree payloads, one envelope, failure exit precedence, stable key ordering, configured logging and plain prompts |
| III. Proportional validation | PASS: planning is documentation-only | PASS: lightweight checks now; isolated package/command/PTY checks during implementation; broader race suite only for demonstrated shared/concurrent impact |
| IV. Documentation matches behavior | PASS: new command requires docs | PASS: command source metadata, README, generated CLI docs, integration guidance/fixture source covered with relevant implementation work |
| V. Small compatible changes | PASS: existing ownership and helpers fit | PASS: narrow resumption types/files, shared key/traversal validation only where identical; no framework or new dependency |
| Layering and file cohesion | PASS: services own backend mechanics | PASS: separate command batch/tree runners and views; facade maps only; interfaces and affected test doubles change together |
| Output and no-op contracts | PASS: required by spec | PASS: command-execution diagnostic matrices, real-terminal no-op checks, accepted/confirmed/canceled distinctions, no duplicate rendering |
| Implementation testing boundary | PASS: isolated dependencies only | PASS: quickstart excludes all live profiles/targets and integration tags; source-only fixture updates explicitly distinguished |

No unjustified violation or unresolved clarification remains. `AGENTS.md` and `specs/ralph-implementation-rules.md` remain binding for implementation.

## Project Structure

### Documentation (this feature)

```text
specs/347-resume-process-instances/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli.md
│   └── service-api.md
└── tasks.md                     # next phase; not created by this command
```

### Source Code (repository root)

```text
cmd/
├── resume.go
├── resume_processinstance.go
├── resume_processinstance_batch.go
├── resume_processinstance_tree.go
├── cmd_views_processinstance_resume.go
├── cmd_views_processinstance_resume_tree.go
└── resume_processinstance*_test.go / command_contract_test.go
c8volt/process/
├── api.go, model.go, convert.go
└── resume.go, resume_plan.go + colocated tests
internal/domain/
└── processinstance_resumption.go, processinstance_resumption_plan.go
internal/services/common/
└── processinstance_keys.go + normalization tests  # narrow shared extraction
internal/services/processinstance/
├── api.go
├── resume.go, resume_plan.go + colocated tests
├── waiter/resumption.go + tests
├── v810/contract.go, resume.go, resumption_submission.go + tests
└── v88/ and v89/ contract.go, resume.go + capability tests
integration/cli/
├── get_processinstance_suspended_test.go         # source changes only
└── suspended_fixture_test.go                    # preserve ownership/cleanup evidence
README.md / integration/README.md / docs/cli/* / docs/index.md
```

**Structure Decision**: Extend existing `c8volt/process` facade and `internal/services/processinstance` area. The base command owns Cobra/validation/dispatch, the batch runner owns CLI approval/activity around a collection call, the tree runner owns CLI interaction around a service plan, and views own all final formatting. A mode's backend work never moves into its CLI runner. Preserve current suspension files apart from narrow shared normalization or structural traversal validation, with regression coverage. Existing generated client files are read-only dependencies.

## Phase 0 — Research Outcome

[research.md](research.md) resolves operation support, strict response/ambiguity handling, waiter semantics, target order, frozen scope, tenant behavior, evidence shapes, logging, and isolated validation. The generated endpoint already exists. The singular facade will mechanically route through the same internal collection as multiple-key callers; only the adapter owns one-target submission/confirmation. Existing suspension remains the behavioral baseline, with explicit resumption differences.

## Phase 1 — Design

### Ownership and execution sequence

1. Parse explicit keys and output/control flags using existing CLI conventions. Reject malformed/empty input before discovery. Capability and timing validation remain effective in public-library use and previews.
2. Exact-key mode calls collection dry-run to validate/freeze normalized keys without I/O. Tree mode calls `PlanProcessInstanceResumption`, which checks capability first, validates locally, resolves full descendant results, validates completeness/identity, and records eligible/excluded observations.
3. A plan error stops immediately. A successful tree preview or empty eligible scope renders mode-aware results without confirmation, mutation, or activity. Successful plans contain nonnil arrays; zero-value error plans cannot be interpreted as no-ops.
4. Clone frozen eligible keys before approval. Prompt once with the unique eligible count using configured stderr; preserve auto-confirm/automation/abort conventions. Do not rediscover.
5. Start shared activity when eligible, invoke one collection facade call, then stop activity before rendering. Services normalize/prevalidate before pool dispatch and preserve an ordered slot for every target, including unattempted ones.
6. Versioned adapter submits through existing mutation retry/rate primitives. Capture uncertainty across attempts; only valid empty 204 establishes acceptance. Default read-only waiter confirms ACTIVE/COMPLETED, stops distinctly on cancellation, and retains prior evidence on failure. No-wait omits this step entirely.
7. Map results and error through the facade, render one envelope or selected keys/INFO summary, emit one bounded human failure if appropriate, and exit with the original error class. Never call a second error renderer after a JSON envelope. A writer failure must not replace an existing operation error.

### Model and interface decisions

See [data-model.md](data-model.md), [CLI contract](contracts/cli.md), and [service contract](contracts/service-api.md). Resumption gets separate types preserving established field names, with only `eligible`/`excluded` planning classifications. No “already resumed” success. Concurrent cancellation is `failed` + `accepted` + `unconfirmed`, with CANCELED/TERMINATED observation and conflict error. Partial machine evidence remains intact; human diagnostics use bounded categories instead of raw backend strings.

### Cohesive implementation order for task generation

Use these as design dependencies, not a pre-created task list:

1. Domain and internal adapter/waiter contracts, 8.10 behavior, unsupported-version guards, all affected interface test doubles, and isolated protocol/state tests together. Narrow key normalization reuse belongs here.
2. Complete collection execution plus thin public singular/collection facade methods, converters, ordered partial evidence, worker/fail-fast/rate behavior, and tests. Implement collection behavior once; do not first ship a separate single-only workflow.
3. Internal tree planning plus public plan mapping, structural validation reuse, mixed-state/overlap/identity/failure tests, and no-I/O capability tests. This is independently verifiable before CLI exposure.
4. Exact-key CLI construction/runner/views, full diagnostic/prompt/activity/output tests, command metadata, README/generated docs, and source updates to intended CLI resumption integration coverage.
5. Tree CLI runner/views, frozen-count approval, exhaustive empty-scope/PTY/output/failure checks, help/docs updates, and relevant regressions.

Each resulting work unit includes its own fixtures, formatting, tests, and documentation rather than separate cleanup tasks. Keep compilation intact after interface expansion. These separations have distinct verifiable outcomes; tree planning and CLI tree interaction do not rebuild collection execution. Review adjacent tasks for consolidation during `$speckit-tasks`; do not create tasks solely for parallelism or context reading.

### Validation strategy

Follow [quickstart.md](quickstart.md). Start with targeted service, facade, or command tests; inspect build tags and environment before executing. Add every meaningful regression with its implementation. Required command coverage includes configured/inherited logger destinations and severities, plain/structured output, normal/verbose/quiet/DEBUG, JSON EOF and failure exits, successful INFO suppression, actual PTY stdin, redirected stdout, frozen scope, empty eligible scopes, and activity cleanup. Restore direct prints or raw error aggregation locally as a mutation check when needed to prove assertions detect those violations; never retain such mutations.

After relevant implementation checks pass, run `gofmt` on touched Go files and regenerate docs with `make docs-content`. Shared interface/concurrency changes across consumers justify a broader race-enabled isolated run once; do not repeat broad tests solely for documentation or commits. Integration fixtures remain source-only; actual engine behavior is an explicit coverage limitation.

### Documentation and fixture impact

Update source help/metadata, README, and generated CLI docs with permissions, eligibility, exact/tree scope, traversal limit, independently suspended descendants, frozen/non-atomic behavior, no rollback, output, and wait/no-wait evidence. Update `integration/README.md` and outdated direct-call fallback evidence in `integration/cli/suspended_fixture_test.go`. Route intended CLI resumption scenarios in `get_processinstance_suspended_test.go` through the new command; preserve fixture-owned keys, cleanup tracking before mutation, and cleanup evidence. Direct fixture calls may remain only where setup/cleanup intentionally is not CLI coverage, clearly documented. Do not broaden `integration/scripts/run-c810-suspended.sh`, which has no resumption scenario, merely because of its name.

## Complexity Tracking

No constitution exceptions required. Focused resumption types and waiter are necessary because success predicates differ from suspension. Narrow validation reuse avoids duplication without coupling independent lifecycle semantics or introducing a generic mutation engine.
