# Implementation Plan: CLI Debt Refactor

**Branch**: `254-cli-debt-refactor` | **Date**: 2026-07-24 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/254-cli-debt-refactor/spec.md`

**Issue**: [#254](https://github.com/grafvonb/c8volt/issues/254) - refactor(cli): reduce command layering, progress, and performance debt

**Mandatory Implementation Context**: Ralph and any implementation agent MUST receive `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Refactor c8volt CLI command debt by first checking in a full 55-command assessment, then moving repeated paging, discovery, query-strategy, progress, and high-volume planning mechanics to the correct service/facade ownership boundaries. The approach is incremental: classify everything, refactor basic read paging from lower-risk commands toward process-instance search, then address cancel/delete planning, selectively review ops workflows, define a CLI-wide progress policy, and keep tests, help text, generated docs, and command capability metadata aligned.

## Technical Context

**Language/Version**: Go 1.26 with the repository's current Go toolchain (`go.mod` declares `toolchain go1.26.2`)

**Primary Dependencies**: Cobra and pflag command layer, Viper-backed configuration, c8volt public facade packages, internal service interfaces and versioned adapters, generated Camunda clients, `toolx`, `toolx/pool`, `toolx/logging`, docs generator, and existing `testx` helpers

**Storage**: N/A. c8volt remains a stateless CLI; Camunda is the runtime fact source and generated docs/spec artifacts are repository files

**Testing**: Targeted Go tests in `cmd`, affected `c8volt/<area>` facade packages, affected `internal/services/<area>` packages, command contract tests, docs generator tests, fake-latency or benchmark-style tests where high-volume behavior changes, then broader `make test`

**Target Platform**: Cross-platform CLI binary for local operator shells and automation environments

**Project Type**: Go CLI application with public facade and internal service layering

**Performance Goals**: High-volume workflows must process thousands of process instances and related resources without slower targeted validation; independent enrichment, lookup, discovery, planning, and mutation phases should use bounded concurrency where it materially improves throughput

**Constraints**: Preserve current user-visible command behavior unless a tested compatibility note is planned; keep JSON, keys-only, quiet, and automation output clean; keep generated Camunda clients isolated; respect existing `--workers`, `--batch-size`, `--limit`, `--fail-fast`, `--no-worker-limit`, `--automation`, `--quiet`, and `--no-indicator` semantics

**Scale/Scope**: 55 command nodes; basic read commands for jobs, elements, incidents, and process instances; process-instance cancel/delete mutation planning; high-level ops workflows for slow-process analysis, retention policy, smoke tests, all-process-definition purge, orphan-process-instance purge, process-instance-with-incidents purge, incident repair, and process-instance repair

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The plan preserves observable confirmation, dry-run, partial-completion, deterministic exit, and mutation verification behavior; changes that alter operational claims require tests.
- **CLI-First, Script-Safe Interfaces**: PASS. Work stays within existing commands, flags, output modes, exit behavior, command contract metadata, and generated docs.
- **Tests and Validation Are Mandatory**: PASS. Every refactor slice must add targeted tests at the closest useful level and broaden validation based on blast radius.
- **Documentation Matches User Behavior**: PASS. Command help, generated CLI docs, README/docs examples, and capability metadata are explicit deliverables when behavior or wording changes.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses existing `cmd`, facade, internal service, `toolx`, `toolx/pool`, and `toolx/logging` patterns instead of introducing a universal pager or a parallel architecture.

## Project Structure

### Documentation (this feature)

```text
specs/254-cli-debt-refactor/
+-- plan.md
+-- research.md
+-- data-model.md
+-- quickstart.md
+-- contracts/
|   +-- cli-debt-refactor-contract.md
+-- checklists/
|   +-- requirements.md
+-- tasks.md
```

### Source Code (repository root)

```text
cmd/
+-- command_contract.go
+-- command_contract_test.go
+-- capabilities.go
+-- cmd_views_*.go
+-- cmd_paging_totals.go
+-- get_job.go
+-- get_job_search.go
+-- get_element.go
+-- get_element_search.go
+-- get_incident.go
+-- get_incident_search.go
+-- get_processinstance.go
+-- get_processinstance_search.go
+-- get_processinstance_paging.go
+-- cancel_processinstance.go
+-- delete_processinstance.go
+-- ops_*.go
+-- *_test.go

c8volt/
+-- element/
+-- incident/
+-- job/
+-- ops/
+-- process/
+-- foptions/

internal/domain/
internal/services/
+-- element/
+-- incident/
+-- job/
+-- ops/
+-- processdefinition/
+-- processinstance/
+-- */v87|v88|v89/

toolx/
+-- logging/
+-- pool/
+-- poller/

docsgen/
docs/cli/
README.md
```

**Structure Decision**: Keep CLI-only concerns in `cmd`, public input/output mapping in `c8volt/<area>`, backend traversal and workflow mechanics in `internal/services/<area>`, and bounded worker/activity helpers in existing `toolx` packages. The command assessment should live under this feature directory until implementation tasks choose whether a durable project-wide audit document belongs elsewhere.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

See [data-model.md](./data-model.md), [contracts/cli-debt-refactor-contract.md](./contracts/cli-debt-refactor-contract.md), and [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. Contracts require destructive confirmation, partial-completion reporting, and observable outcome proof to remain covered for changed workflows.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI behavior contract makes JSON, keys-only, quiet, automation, prompts, progress, and docs behavior explicit.
- **Tests and Validation Are Mandatory**: PASS. Quickstart defines targeted, docs, fake-latency/benchmark-style, and broad validation expectations.
- **Documentation Matches User Behavior**: PASS. Help text, generated CLI docs, README/docs examples, and capability metadata are part of the acceptance contract.
- **Small, Compatible, Repository-Native Changes**: PASS. The design rejects a universal pager and keeps ops workflow semantics domain-specific unless mechanics are identical.

## Complexity Tracking

No constitution violations requiring justification.
