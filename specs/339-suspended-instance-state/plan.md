# Implementation Plan: Suspended Process-Instance State

**Branch**: `339-suspended-instance-state` | **Date**: 2026-10-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/339-suspended-instance-state/spec.md`; issue #339.

## Summary

Represent `SUSPENDED` consistently in existing process-instance reads, filters, expectations and creation visibility. Extend the shared/public state vocabulary and timestamp conversions, enforce older-version capability errors inside services, and preserve existing CLI output contracts. Reuse the generated v810 client unchanged; validate against server 8.10.4, distinct from client baseline 8.10.3. No new mutation commands, mutation eligibility rules, purge behavior, definition statistics or watch mode.

## Technical Context

**Language/Version**: Go 1.26; repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, testify 1.11.1 and generated Camunda clients; no new dependencies.

**Storage**: No new persistence; current backend instance facts and transient command results.

**Testing**: Go package tests, deterministic fake clients/HTTP servers and waiter observations, existing build-tagged `integration/cli` harness, generated documentation checks.

**Target Platform**: Existing supported CLI/library platforms and configured Camunda 8.8, 8.9, 8.10 environments; live feature validation on 8.10.4.

**Project Type**: Go CLI and public library with versioned service adapters.

**Performance Goals**: Preserve existing pagination, worker bounds and polling timing. State capability checks and metadata rendering add no discovery calls or new polling loops.

**Constraints**: Keep generated clients/provenance unchanged; keep facades thin and version policy in adapters. Preserve aliases, unknown/absent behavior, exact ACTIVE selection, ALL selection, empty-result/output precedence, and older creation defaults.

**Scale/Scope**: One added lifecycle state and one optional timestamp across existing process-instance paths; three version adapters with two rejecting explicit suspension. No new endpoints, resources, or scale requirements.

## Constitution Check

| Principle | Before research | After design |
| --- | --- | --- |
| Operational proof | Pass: suspension is observable existence, not completion | Pass: creation/terminal expectations are separated and acceptance tests prove each |
| CLI-first/script-safe | Pass: additive state and metadata | Pass: envelope, exit classification, keys/quiet modes and compact rows retained |
| Proportionate validation | Pass: planning uses document checks only | Pass: targeted checks per slice; final race suite justified by shared state/interface changes across packages |
| Documentation matches behavior | Pass: documentation work included | Pass: README, source metadata, completions and generated CLI docs in scope |
| Small repository-native changes | Pass: existing domain/facade/adapters own behavior | Pass: one narrow capability method, existing waiters/fixtures, no dependencies or framework |

No constitutional or Ralph-rule conflict identified. Pre/post-design gates pass. This is a design assessment, not implementation validation.

## Project Structure

### Documentation (this feature)

```text
specs/339-suspended-instance-state/
├── spec.md
├── planning-context.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/process-instance-state.md
└── checklists/requirements.md
```

`tasks.md` will be produced by the next workflow, not this planning phase.

### Source Code (repository root)

```text
internal/domain/{state.go,processinstance.go}
internal/services/processinstance/{api.go,search.go,wait.go}
internal/services/processinstance/v88/{contract.go,service.go,bulk.go}
internal/services/processinstance/v89/{contract.go,service.go,bulk.go}
internal/services/processinstance/v810/{contract.go,service.go,bulk.go,convert.go}
internal/services/processinstance/waiter/
c8volt/process/{state.go,model.go,convert.go,client.go,bulk.go}
c8volt/ferrors/errors.go
cmd/{get_processinstance.go,get_processinstance_validation.go,expect_processinstance.go}
cmd/{cmd_views_processinstance.go,completion_test.go,run.go,run_processinstance.go}
integration/cli/
README.md
docs/cli/                  # regenerated through make docs-content
```

**Structure Decision**: Extend current ownership. Source paths identify owners and review surfaces, not a requirement to modify every listed file. Keep declaration-heavy command-specific helpers in focused files; no new mode is introduced. Do not alter generated clients.

## Design and Execution Boundaries

1. **State and metadata:** Extend domain/public states and valid parsing, retaining both existing terminal sets. Add `SuspendedDate string` with existing empty-value and formatting conventions, mapping both facade directions and v810 responses. Test reports as well as instances so ParseState-based conversions cannot produce UNKNOWN.
2. **Capability enforcement:** Add `ValidateProcessInstanceStates(domain.States) error` to current shared/versioned service contracts. Older adapters reject explicit suspended membership, including mixed desired-state sets; v810 permits it. Validate shared page/total search before direct-incident branches, adapter page searches, and single/bulk explicit waits before reads or worker pools. Update interface consumers/test doubles atomically. Facades continue ordinary error conversion through `ferrors`.
3. **Creation visibility:** Expand the shared observable-state list. Keep the generic waiter independent of explicit-request capability validation, because creation uses this implicit list on all versions. Preserve timeout/error/no-wait behavior and prove older defaults still work.
4. **CLI:** Update get/expect state metadata, validation and shell completion using the public vocabulary and command-specific allowed sets. Value completion must not contact the backend. Audit other selectors without expanding mutation eligibility, definition purge discovery or cleanup. Keep compact rows unchanged and expose timestamp through existing structured instance payloads.
5. **Refresh:** No current process-instance watch exists; definition watch does not contain this metadata. Use sequential present-to-absent instance responses and renderings to prove no stale timestamp survives. Do not add a new watch mode.
6. **Integration:** Add reusable suspension setup/read/resume fixture behavior within the existing integration package. Resume held instances before existing cleanup so tests do not depend on #341's suspended-hierarchy cleanup changes. Verify server 8.10.4 before mutations; retain client provenance and tested revision in existing evidence outputs. Add the focused scenario to an existing get/expect slice so it is not stranded outside normal integration runs.

## Validation Strategy

| Requirements | Evidence required |
| --- | --- |
| FR-001–003 | Case/whitespace parsing, nonterminal classification, alias and UNKNOWN/ABSENT regressions, public lookup and expectation report conversion |
| FR-004–008 | Serialized v810 equality filter, exact ACTIVE and ALL behavior, older-version unsupported errors with zero backend calls, direct-incident empty path, totals, mixed expectations, empty-key waits |
| FR-009 | Deterministic first-observation suspension, unchanged old-version creation, errors/timeouts/terminal observations and no-wait behavior |
| FR-010–012 | UTC RFC3339Nano timestamp conversion, nil/zero omission, two-way facade mapping, sequential refresh, human/JSON/keys/quiet and successful empty command paths |
| FR-013 | Help/validation/completion tests without bootstrap, README and regenerated documentation review |
| Live proof | State lookup, search, expectations and metadata on observed 8.10.4; resume/cleanup fixtures, explicit unavailable/failure evidence |

Use targeted package tests with each implementation unit and gofmt on touched Go files. Run `make test` once after the integrated implementation because domain state and service-interface changes cross commands, facades, adapters and concurrent waits. Do not run runtime tests for these planning documents. Refer to [quickstart.md](quickstart.md) for commands and exact result checks.

## Work-Unit Guidance for Task Generation

Keep cohesive implementation, tests, formatting and targeted validation together. Suggested boundaries are (a) state representation and report conversions (Story 1), (b) selection/expectation capability validation across all contracts/adapters/shared paths, test doubles and relevant CLI metadata/completion (Story 2), (c) creation visibility and waiter regressions (Story 3), (d) timestamp representation/conversions and structured refresh output (Story 4), and (e) reusable integration validation/evidence. Keep each story’s affected documentation and command acceptance coverage with its behavior. These are planning boundaries, not task checkboxes.

Representation precedes behavior. The capability unit deliberately spans all versions to keep compilation intact; do not split interface changes from implementations or stubs. Creation remains separate because implicit visibility has a different correctness boundary from explicit requested-state validation. CLI acceptance covers existing stories without rebuilding common mechanisms. Integration is separate because it requires disposable live infrastructure; deterministic acceptance coverage stays with implementation. Review adjacent tasks for consolidation; do not split for hypothetical parallel execution, and respect Ralph's one-story/work-unit scope when writing tasks.

## Complexity Tracking

No gate violations. The narrow service validator is justified by the direct-incident search bypass and public-call coverage; it extends the existing contract rather than adding a new architecture layer. No optional type-assertion fallback is allowed to bypass it silently.
