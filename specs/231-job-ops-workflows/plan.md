# Implementation Plan: Job Ops Workflow Primitives

**Branch**: `231-job-ops-workflows` | **Date**: 2026-05-25 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/231-job-ops-workflows/spec.md`

## Summary

Extend the existing job command surface from keyed lookup and retry/timeout updates into reusable ops workflow primitives. Preserve `get job --key`, `update job --retries`, and `update job --timeout`; add `get job` list/search mode when `--key` is omitted; add mutually exclusive worker outcome modes for technical failure, modeled BPMN error, and completion; and keep all new behavior inside the existing command, facade, domain, service, versioned-client, rendering, metadata, docs, and validation patterns. Planning, tasks, and Ralph implementation MUST include `--implementation-context specs/ralph-implementation-rules.md`.

## Technical Context

**Language/Version**: Go, repository current module toolchain
**Primary Dependencies**: Cobra command tree, existing command contract helpers, `c8volt/job` facade, `internal/domain` job models, `internal/services/job` API/factory/waiter, generated Camunda v8.8/v8.9 clients, shared render/error/confirmation helpers, docs generator
**Storage**: N/A
**Testing**: Targeted Go tests for `cmd`, `c8volt/job`, `internal/domain`, and `internal/services/job` packages; generated docs check through `make docs-content`; broader validation with `make test` when implementation reaches commit readiness
**Target Platform**: Multi-platform CLI binary
**Project Type**: Go CLI
**Performance Goals**: Keyed lookup remains one bounded job search; list/search uses explicit limits and repository pagination patterns; dry-run and confirmation perform only the required lookup/mutation calls; worker outcome mutations avoid confirmation claims that cannot be reliably observed

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Existing retry confirmation is preserved; timeout and worker outcomes report accepted/submitted outcomes only where no stable read-model predicate is specified. Dry-run and confirmation remain required for material mutations.
- **CLI-First, Script-Safe Interfaces**: PASS. The feature is exposed through existing Cobra commands, stable flags, validation, exit behavior, JSON output, command metadata, and automation guardrails.
- **Tests and Validation Are Mandatory**: PASS. The plan requires command, facade, service, version, rendering, contract, docs, and regression tests before implementation can be considered complete.
- **Documentation Matches User Behavior**: PASS. README, help, and generated CLI docs are in scope for every changed command and flag.
- **Small, Compatible, Repository-Native Changes**: PASS. The work extends the existing issue #180 job architecture rather than adding parallel job-specific verbs or alternate ops semantics.

## Project Structure

### Documentation (this feature)

```text
specs/231-job-ops-workflows/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-job-ops-workflows.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_job.go
├── get_job_test.go
├── update_job.go
├── update_job_test.go
├── cmd_views_job.go
├── cmd_views_job_test.go
└── command_contract_test.go

c8volt/
└── job/
    ├── api.go
    ├── client.go
    ├── client_test.go
    └── model.go

internal/
├── domain/
│   ├── job.go
│   └── job_test.go
└── services/
    └── job/
        ├── api.go
        ├── factory.go
        ├── factory_test.go
        ├── waiter/
        │   ├── waiter.go
        │   └── waiter_test.go
        ├── v88/
        └── v89/

README.md
docs/cli/
docsgen/
```


## Architecture Grounding

- Architecture extension status: installed.
- Architecture memory status: `.specify/memory/architecture.md` and all five 4+1 view files are present.
- Decision: reuse existing architecture memory without refresh. Issue #231 extends command/facade/service behavior inside already documented boundaries: command contract, service/generated-client isolation, mutation safety, and ops playbook composition.

## Ralph Implementation Context

- Every implementation iteration MUST receive `--implementation-context specs/ralph-implementation-rules.md`.
- Do not launch Ralph unless the launcher instructions include that implementation context.
- Each Ralph work unit must complete only one story or validation slice and must not stage or commit until validation passes.
- Commit subjects must use Conventional Commits and end with `#231`.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-job-ops-workflows.md](./contracts/cli-job-ops-workflows.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract separates dry-run plans, submitted outcomes, retry confirmation, mutation failure, validation failure, and unsupported capability errors.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI contract defines valid commands, invalid combinations, JSON behavior, automation behavior, output modes, and help/docs expectations.
- **Tests and Validation Are Mandatory**: PASS. The design artifacts require tests close to command, facade, service, generated-client adapter, renderer, contract, docs, and regression behavior.
- **Documentation Matches User Behavior**: PASS. The quickstart and task plan require README and generated CLI docs updates through existing generation paths.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends existing job packages and rejects parallel verbs or command-layer generated-client ownership.

## Complexity Tracking

No constitution violations or complexity exceptions are required.
