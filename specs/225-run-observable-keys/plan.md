# Implementation Plan: Run Confirmation Observes Real Process Instance States

**Branch**: `225-run-observable-keys` | **Date**: 2026-05-23 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/225-run-observable-keys/spec.md`

**Issue**: [#225](https://github.com/grafvonb/c8volt/issues/225)

**Implementation Context**: Ralph planning, task generation, and every implementation iteration must read and apply `specs/ralph-implementation-rules.md`. Ralph must be launched only with `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Run-style process-instance creation must confirm success when the created process instance is observable in a real lifecycle state, not only `ACTIVE`. The change belongs in the process-instance service/wait ownership boundary for v8.7, v8.8, and v8.9, then in the `run pi` command rendering path for keys-only and normal output. `deploy --run` and `embed deploy --run` should benefit from the shared creation confirmation behavior without gaining new expectation flags. User-facing examples and generated docs must show the pipeline pattern where `run pi --keys-only` feeds `expect pi --state <state> -`.

## Technical Context

**Language/Version**: Go module using the repository's current Go toolchain from `go.mod`

**Primary Dependencies**: Cobra command tree, existing c8volt facade packages, internal process-instance services, generated Camunda clients, repository `toolx` and `testx` helpers

**Storage**: N/A; process instance facts remain observed from Camunda

**Testing**: Go tests with `testify`, command tests under `cmd`, service tests under `internal/services/processinstance/{v87,v88,v89}`, docs generator tests under `docsgen`

**Target Platform**: Local or CI CLI binary on supported release platforms

**Project Type**: CLI

**Performance Goals**: Preserve existing wait/backoff behavior and worker controls; do not add extra remote lookup loops beyond the confirmation wait already performed for creation

**Constraints**: Preserve strict `expect pi` semantics; do not add `--expected-status` to `run`; keep generated-client types below service boundaries; regenerate CLI docs from command metadata/source

**Scale/Scope**: One issue-scoped CLI behavior fix across `run pi`, shared process-instance creation used by `deploy --run` and `embed deploy --run`, plus docs/tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature improves operational proof by confirming successful creation against observable Camunda lifecycle facts.
- **CLI-First, Script-Safe Interfaces**: Pass. `run pi --keys-only` is an explicit script-safe output mode; strict lifecycle assertions remain in `expect pi`.
- **Tests and Validation Are Mandatory**: Pass. The plan requires closest service, command, contract, and docs validation.
- **Documentation Matches User Behavior**: Pass. README/help/generated CLI docs must be updated with the pipeline pattern.
- **Small, Compatible, Repository-Native Changes**: Pass. The design reuses existing process-instance waiters, output rendering helpers, command contracts, and generated-doc workflows.

## Project Structure

### Documentation (this feature)

```text
specs/225-run-observable-keys/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-output-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── run_processinstance.go
├── deploy_processdefinition.go
├── embed_deploy.go
├── cmd_views_get.go
├── command_contract.go
├── command_contract_test.go
├── capabilities_test.go
└── *_test.go

c8volt/process/
├── api.go
├── client.go
├── model.go
└── state.go

internal/domain/
├── processinstance.go
└── state.go

internal/services/processinstance/
├── waiter/
│   └── waiter.go
├── v87/
│   ├── service.go
│   └── service_test.go
├── v88/
│   ├── service.go
│   └── service_test.go
└── v89/
    ├── service.go
    └── service_test.go

README.md
docsgen/
docs/cli/
```

**Structure Decision**: Implement shared confirmation semantics in the internal process-instance service/wait boundary, keep command rendering in `cmd`, keep public state/result shape in `c8volt/process`, and regenerate derived docs through the existing docs workflow.

## Phase 0: Research Summary

See [research.md](./research.md).

## Phase 1: Design Summary

See [data-model.md](./data-model.md), [contracts/cli-output-contract.md](./contracts/cli-output-contract.md), and [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Pass. The design confirms creation only after Camunda observes an accepted real state and keeps absent/not-found as failure for run confirmation.
- **CLI-First, Script-Safe Interfaces**: Pass. Keys-only output is constrained to one key per line; JSON remains envelope-compatible for full-contract commands.
- **Tests and Validation Are Mandatory**: Pass. Required tests cover service wait states, run rendering, command contract metadata, docs examples, and strict `expect pi` behavior.
- **Documentation Matches User Behavior**: Pass. README and generated CLI docs are part of the implementation scope.
- **Small, Compatible, Repository-Native Changes**: Pass. No new top-level package, command family, or parallel output path is required.

## Complexity Tracking

No constitution violations require justification.
