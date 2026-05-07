# Implementation Plan: Job Lookup And Updates

**Branch**: `180-job-update-lookup` | **Date**: 2026-05-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/180-job-update-lookup/spec.md`

## Summary

Add job-focused runtime commands for inspecting a job by key and updating supported job parameters on Camunda 8.8 and 8.9. Introduce `c8volt get job --key <job-key>` and `c8volt update job --key <job-key>` under the existing command tree; add a dedicated `internal/services/job` service package with versioned 8.7, 8.8, and 8.9 implementations; use generated job search/update capabilities; confirm requested retries through job lookup when retries are supplied; and treat timeout updates as accepted/submitted after mutation without deadline-based confirmation.

## Technical Context

**Language/Version**: Go, repository current module toolchain  
**Primary Dependencies**: Cobra command tree, existing command metadata helpers, generated Camunda v8.8/v8.9 clients, versioned internal service factories, dedicated job facade package, shared rendering/error helpers, command context logger/activity plumbing, waiter/backoff helpers for retry confirmation  
**Storage**: N/A  
**Testing**: Go tests through targeted `go test ./cmd ./c8volt/job ./internal/services/job ./internal/services/job/v87 ./internal/services/job/v88 ./internal/services/job/v89 -count=1`, docs generation checks, regression checks for related process-instance behavior, and final `make test`  
**Target Platform**: CLI on the repository's supported platforms  
**Project Type**: Go CLI  
**Performance Goals**: Single-key lookup and update should make only the required generated client calls; retries confirmation should poll only the target job until requested retries are observed or waiter exhaustion occurs; timeout-only updates should avoid unnecessary polling  
**Constraints**: Preserve existing `get pi --with-incidents` and `update pi --vars` behavior; require `--key`; require at least one update flag; reject invalid retry/timeout values before calling Camunda; reject unsupported 8.7 job mutation before mutation; do not expose `retryBackOff`; do not confirm timeout through deadline comparison; keep job behavior out of process-instance and incident services  
**Scale/Scope**: One new `get job` command, one new `update job` command under the existing update root, a new job facade/domain/service path, versioned service implementations, waiter behavior for retries confirmation, human/JSON rendering, command metadata, tests, README/help/generated docs, and Speckit/Ralph artifacts

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Retry updates are confirmed through job lookup before confirmed success is reported. Timeout-only updates explicitly report submitted/accepted output and do not claim deadline confirmation.
- **CLI-First, Script-Safe Interfaces**: PASS. The command surface uses stable Cobra flags, validation, human/JSON output, and existing metadata conventions.
- **Tests and Validation Are Mandatory**: PASS. The plan requires command tests, facade/domain tests, versioned service tests, waiter tests for retries confirmation, docs generation checks, targeted Go tests, and final `make test`.
- **Documentation Matches User Behavior**: PASS. The new commands, flags, examples, unsupported-version behavior, no-wait behavior, retries confirmation, and timeout submitted-only behavior require README/help/generated docs updates.
- **Small, Compatible, Repository-Native Changes**: PASS. The design adds a dedicated job service package while reusing existing command, facade, versioned service, waiter, rendering, metadata, and docs patterns.

## Project Structure

### Documentation (this feature)

```text
specs/180-job-update-lookup/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-job-lookup-update.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get.go
├── get_job.go
├── get_job_test.go
├── update.go
├── update_job.go
├── update_job_test.go
├── cmd_views_get.go
├── cmd_views_job.go
├── command_contract_test.go
└── root.go

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
        ├── v87/
        │   ├── contract.go
        │   ├── service.go
        │   └── service_test.go
        ├── v88/
        │   ├── contract.go
        │   ├── convert.go
        │   ├── service.go
        │   └── service_test.go
        └── v89/
            ├── contract.go
            ├── convert.go
            ├── service.go
            └── service_test.go

README.md
docs/
docsgen/
```

**Structure Decision**: Add `cmd/get_job.go` under the existing get root and `cmd/update_job.go` under the existing update root. Add a dedicated `c8volt/job` facade package for command-facing job lookup/update orchestration, while keeping generated-client ownership and confirmation inside `internal/services/job`. Implement v8.8/v8.9 with generated job search and update calls, make v8.7 return unsupported-version errors, and add a job waiter only for retries confirmation. Reuse existing rendering, command metadata, docs generation, and error mapping paths instead of adding parallel infrastructure.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-job-lookup-update.md](./contracts/cli-job-lookup-update.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract distinguishes submitted, confirmed, confirmation-failed, and mutation-failed outcomes, and explicitly limits confirmation claims to retry checks where an observable reliable predicate exists.
- **CLI-First, Script-Safe Interfaces**: PASS. The contract defines valid/invalid invocations, JSON behavior, human output, required flags, no-wait behavior, and unsupported-version outcomes.
- **Tests and Validation Are Mandatory**: PASS. The task list will include failing-first command/service/waiter tests, regression tests for related existing commands, docs generation, targeted Go tests, and final `make test`.
- **Documentation Matches User Behavior**: PASS. Documentation work is part of the feature contract and task list, including the timeout submitted-only caveat.
- **Small, Compatible, Repository-Native Changes**: PASS. The design follows existing service package and command patterns, with a new job package only because the issue explicitly requires service ownership boundaries.

## Complexity Tracking

No constitution violations or additional complexity exceptions are required.
