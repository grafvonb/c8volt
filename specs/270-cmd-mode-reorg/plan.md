# Implementation Plan: Command Mode And Concern Reorganization

**Branch**: `270-cmd-mode-reorg` | **Date**: 2026-08-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/270-cmd-mode-reorg/spec.md`

**Issue**: [#270](https://github.com/grafvonb/c8volt/issues/270) - refactor(cli): reorganize cmd package by command mode and concern

**Mandatory Implementation Context**: Ralph and any implementation agent MUST receive `--implementation-context specs/ralph-implementation-rules.md`.

## Summary

Reorganize the `cmd` package by command mode and concern so maintainers can review watch, rendering, paging/progress, mutation, root wiring, slow-analysis, ops report, and test ownership without crossing unrelated behavior. The technical approach is incremental and behavior-preserving: move cohesive concerns into focused files, align tests with production ownership, remove only confirmed dead helpers, preserve the single `cmd` package boundary, and validate representative CLI output plus targeted command tests after each slice.

## Technical Context

**Language/Version**: Go 1.26 with the repository's current Go toolchain (`go.mod` declares `toolchain go1.26.2`)

**Primary Dependencies**: Cobra and pflag command layer, Viper-backed configuration, c8volt public facade packages, internal service interfaces and versioned adapters, generated Camunda clients, `toolx`, `toolx/pool`, `toolx/logging`, docs generator, and existing `testx` helpers

**Storage**: N/A. c8volt remains a stateless CLI; this feature changes repository source/test organization and Spec Kit artifacts only

**Testing**: Targeted Go tests in `cmd` for each moved command mode, renderer, workflow, root/config path, and test helper area; affected facade/service tests only if ownership corrections move below `cmd`; `go test ./cmd -count=1`, `git diff --check`, generated-doc diff checks, then `make test`

**Target Platform**: Cross-platform CLI binary for local operator shells and automation environments

**Project Type**: Go CLI application with public facade and internal service layering

**Performance Goals**: No user-visible command path or high-volume workflow becomes slower due to reorganization; any moved progress/report code must preserve existing bounded worker, paging, and output ordering behavior

**Constraints**: Preserve all CLI flags, aliases, help text, generated docs, command metadata, prompts, output text, output fields, output ordering, exit codes, and backend calls; do not hand-edit generated clients; keep machine-output modes clean; keep implementation slices small and reviewable

**Scale/Scope**: `cmd` package reorganization across process-definition watch, resource views, process-instance dry-run/paging/progress support, job update, process-instance cancel/delete, root command support, slow-process analysis, ops progress/report files, and aligned test files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The plan is behavior-preserving and requires confirmation, watch, polling, mutation, progress, report, and exit behavior to remain covered where those flows are touched.
- **CLI-First, Script-Safe Interfaces**: PASS. Work stays within existing CLI command paths, flags, output contracts, prompt rules, metadata, and generated docs; machine-oriented modes must remain parseable and quiet where expected.
- **Tests and Validation Are Mandatory**: PASS. Each slice requires nearest targeted tests before broad `cmd` validation, then final `make test`.
- **Documentation Matches User Behavior**: PASS. Documentation is expected to have no intentional diff; if help, generated docs, examples, or command metadata change, that becomes an explicit compatibility item with regenerated docs.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses the current `cmd` package, facade/service boundaries, and local helpers; it rejects new packages or generic frameworks unless a separate follow-up proves ownership.

## Project Structure

### Documentation (this feature)

```text
specs/270-cmd-mode-reorg/
+-- plan.md
+-- research.md
+-- data-model.md
+-- quickstart.md
+-- contracts/
|   +-- command-mode-reorg-contract.md
+-- checklists/
|   +-- requirements.md
+-- tasks.md
```

### Source Code (repository root)

```text
cmd/
+-- get_processdefinition.go
+-- get_processdefinition_watch.go
+-- get_processdefinition_test.go
+-- get_processdefinition_watch_test.go
+-- cmd_views_get.go
+-- cmd_views_flat.go
+-- cmd_views_processinstance*.go
+-- cmd_views_processdefinition.go
+-- cmd_views_incident.go
+-- cmd_views_resource.go
+-- cmd_views_tenant.go
+-- cmd_views_*_test.go
+-- get_processinstance*.go
+-- get_processinstance_paging.go
+-- get_processinstance_*_test.go
+-- update_job.go
+-- update_job_*.go
+-- update_job*_test.go
+-- cancel_processinstance.go
+-- cancel_processinstance_*.go
+-- delete_processinstance.go
+-- delete_processinstance_*.go
+-- root.go
+-- root_*.go
+-- ops_analyse_slow_process_instances*.go
+-- ops_progress*.go
+-- ops_*report*.go
+-- ops_*_test.go

c8volt/
+-- process/
+-- resource/
+-- ops/
+-- foptions/
+-- ferrors/

internal/services/
+-- processinstance/
+-- processdefinition/
+-- ops/
+-- */v87|v88|v89/

toolx/
+-- logging/
+-- pool/
+-- poller/

docsgen/
docs/cli/
README.md
```

**Structure Decision**: Keep the existing single `cmd` package and split by file-level ownership only. Command files remain responsible for CLI construction, flags, validation, dispatch, prompts, render-mode choice, and presentation. Facades and internal services remain responsible for backend traversal, polling, retries, mutation planning, worker execution, and version-specific behavior.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

See [data-model.md](./data-model.md), [contracts/command-mode-reorg-contract.md](./contracts/command-mode-reorg-contract.md), and [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract requires every touched watch, mutation, polling, progress, and report flow to retain observable outcome proof and deterministic completion reporting.
- **CLI-First, Script-Safe Interfaces**: PASS. The command behavior contract covers flags, aliases, help, output modes, prompts, exit codes, metadata, generated docs, and automation safety.
- **Tests and Validation Are Mandatory**: PASS. The quickstart defines targeted command-mode, renderer, workflow, docs, and full-suite checks; tasks must attach validation to each slice.
- **Documentation Matches User Behavior**: PASS. The default expectation is no generated-doc diff; any intentional user-visible change requires source metadata and docs updates in the same work unit.
- **Small, Compatible, Repository-Native Changes**: PASS. The design keeps file moves small, retains cohesive large files when splitting would be artificial, and moves non-mechanical ownership corrections to follow-up work.

## Complexity Tracking

No constitution violations requiring justification.
