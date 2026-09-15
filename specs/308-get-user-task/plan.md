# Implementation Plan: Get User Tasks by Key or Search

**Branch**: `codex/308-get-user-task` | **Date**: 2026-09-13 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/308-get-user-task/spec.md`, backed by issue #308.

## Summary

Add `c8volt get user-task` and aliases for strict keyed reads, backend-filtered search, bounded discovery, and exact counts on Camunda 8.8, 8.9, and 8.10. Extend the existing task facade and usertask service area. Add native adapter operations alongside the legacy resolver getter, keeping `get pi --has-user-tasks` unchanged. Shared service functions own bulk execution, cursor/offset advancement, sparse-page handling, result limits, and count fallback. Commands own input, interaction, and rendering through mechanical facade visitor adapters.

## Technical Context

**Language/Version**: Go 1.26; toolchain go1.26.2 as pinned in `go.mod`.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag, Viper, generated oapi-codegen Camunda clients, standard HTTP/context, `toolx/pool`, shared service and command helpers. No new module dependency or framework.

**Storage**: Camunda is authoritative. No local persistent storage or migration; in-memory collected results and page traversal state only.

**Testing**: Go tests with testify, HTTP fixtures, facade stubs, command subprocesses, and `testx.NewCmdTerminalRunner` for real terminal stdin with separate result/control streams. Start with checks covering the changed behavior. The integrated feature warrants `make test` (`go test ./... -race -count=1`) because it changes shared service/facade contracts and concurrent bulk execution; a commit alone does not trigger tests.

**Target Platform**: Existing Linux, macOS, and Windows CLI builds for amd64/arm64. Real-terminal integration tests use the repository's Linux/macOS support and existing unsupported-platform handling.

**Project Type**: CLI plus public Go facade library.

**Performance Goals**: Honor page-size and overall limits without extra rendering requests; use exact totals without full traversal when trustworthy; capped count traversal retains only the current page and metadata. Bulk reads use bounded existing worker policy and deterministic input order. No new latency SLA.

**Constraints**: Camunda 8.7 explicitly unsupported for new native reads, without breaking its existing resolver behavior. Keep generated clients unmodified. Reuse HTTP authentication, retries, cancellation, tenant options, errors, and output contract. Explicit-key reads bypass discovery tenant filtering. Do not expose forms, variables, dates, sorting, watch, or mutations.

**Scale/Scope**: One command with three aliases, nine filters, page-size/limit/count controls, three supported adapters and v87 unsupported stubs. Default/max page size follows `consts.MaxPISearchSize` (1000); count uses int64. Collected JSON/search memory grows with returned tasks; count memory is bounded by page size. No snapshot guarantee under concurrent backend changes.

## Constitution Check

*Pre-research and post-design gates both pass. No exception is requested.*

| Principle | Pre-research assessment | Post-design evidence |
| --- | --- | --- |
| Operational proof over intent | Reads must return backend facts; no partial success claims | Strict missing-key errors; authoritative traversal completion; capped totals counted to completion; failures do not render successful JSON or a numeric count |
| CLI-first, script-safe interfaces | Existing command grammar and output contract apply | Explicit CLI contract, JSON/keys purity, mode-aware empty views, terminal prompt tests, full contract/automation annotations |
| Validation proportional to the change | Constitution v2.0.0: select checks by behavior and risk | Preserve adapter, traversal, facade, command, real-terminal, and legacy-regression coverage; use targeted checks for focused slices and `make test` for integrated shared-contract/concurrency changes; documentation-only work uses lightweight checks |
| Documentation matches behavior | New command is user-visible | Help, parent command metadata, README, examples, and `make docs-content` included |
| Small compatible repository-native changes | Existing area and dependencies can own work | Native getter is separate from legacy resolver; shared service functions; no generated edits or new framework; dedicated command paging file |

The active specification and issue are consistent with repository layering and operator UX requirements. Paging continuation is the explicitly supported read interaction; this feature adds no mutation confirmation. Existing global error behavior and HTTP policy are retained.

## Project Structure

### Documentation (this feature)

```text
specs/308-get-user-task/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── cli.md
    └── facade-service.md
```

`tasks.md` is present. T001–T002 record the original setup baseline; T003 is the first incomplete task. The 2026-09-14 refresh follows the rebase onto `develop` at `a9aef2c3` and does not implement feature behavior.

### Source Code (repository root)

```text
cmd/
├── get_usertask.go                 # New command, flags, validation, normal dispatch
├── get_usertask_search.go          # New paging render/prompt policy only
├── cmd_views_usertask.go           # New row, collection, empty, count views
├── get_usertask_test.go            # New execution/input/output/error coverage
├── get_usertask_search_test.go     # New search/count/limit coverage
├── get_usertask_terminal_test.go   # New real-terminal paging coverage
├── get.go                         # Parent command help as needed
└── command_contract_test.go        # Capability/automation coverage
c8volt/task/
├── api.go, client.go               # Extend existing facade
├── model.go, convert.go            # New stable public models and mapping
└── client_test.go                  # New facade delegation/error coverage
internal/domain/
└── usertask.go                     # Extend identity model; add search/page types
internal/services/usertask/
├── api.go                         # Add native getter and page operation
├── workflow.go                    # Preserve existing process-instance resolver
├── bulk.go                        # New strict ordered native bulk workflow
├── search.go                      # New collection, visitor traversal, exact count
├── bulk_test.go, search_test.go    # New cross-version-independent mechanics tests
├── v87/                           # New-method unsupported stubs and tests
└── v88/, v89/, v810/               # Extend contracts, request/response mapping, tests
README.md
docs/cli/                          # Generated through make docs-content
```

**Structure Decision**: Use issue-mandated filenames `get_usertask.go` and `cmd_views_usertask.go`. Distinct paging policy lives in `get_usertask_search.go`; cursors, offsets, pools, accumulation, and total calculation never enter command code. A scoped implicit-stdin reader can live with command input validation or the existing stdin support file with an opt-in interface; it must not change other commands. `c8volt.API` already embeds task.API, so no second client construction path is needed. Update affected internal/public API stubs and compile-time interface assertions.

## Phase 0: Research Outcomes

[research.md](research.md) records decisions, rationale, alternatives, and source evidence. The main compatibility findings are the legacy getter's version-specific fallback behavior, native filter union differences in v810, the implicit-stdin gap in existing get commands, quiet rendering requiring explicit handling, and sparse-page termination hazards in neighboring code. No unresolved clarification remains.

## Phase 1: Design

### Adapter and workflow boundaries

1. Extend the domain task with only common fields listed in [data-model.md](data-model.md). Add native conversion functions while preserving legacy conversion/checks where they differ.
2. Extend the existing service API and each version contract with `GetNativeUserTask` and `SearchUserTasksPage`. Native getters perform direct GET with backend authorization and no tenant discovery check or Tasklist retry. Search maps all predicates plus `common.EffectiveTenant` into backend filters. Use `common.RequirePayload` and existing error mapping.
3. Add sibling service free functions for ordered bulk lookup, collected search, page visitor traversal, and exact totals. Bulk applies options and deduplicates before worker scheduling; any error keeps the CLI operation unsuccessful. Traversal applies the backend continuation rules in research and exposes only selected pages and decision metadata to visitors. Count reuses traversal mechanics without retaining the collection.
4. Extend public task models and facade methods. Convert options using existing `foptions`; convert errors with `ferrors.FromDomain`. Visitor adapters only map values/actions. Preserve the existing task facade constructor and resolver methods.
5. Add the command, views, and focused paging policy file. Select JSON/keys/human before quiet-human suppression. Numeric totals bypass list rendering and prompt policy. Both keyed CLI lookup and search use a collection payload, including a single key.

### Input and interaction

Use string-slice keys, explicit validation of both flag and stdin values, and existing 16-digit key conventions. Read piped stdin without requiring `-` only for this new command; explicit `-` retains the existing terminal/empty-stream errors. An empty implicit stream supplies no keys. Search filters explicitly supplied alongside keys are conflicts, including an explicit `--state all`; the default `all` is not a conflict. `--tenant` is inherited and not a keyed-mode conflict. Page-size bounds apply in every mode; explicit-key page size does not alter direct lookup.

Paging eligibility mirrors the existing get behavior: JSON, automation, and auto-confirm continue without asking; reaching the result limit stops without a prompt. Before the limit is reached, eligible terminal stdin can prompt even when stdout is redirected. Nonterminal stdin does not prompt. Empty intermediate pages continue silently when the service reports more; completed empty search renders once without asking. The command callback never computes the next request.

### Validation and acceptance mapping

| Requirement groups | Validation |
| --- | --- |
| FR-001–FR-003, FR-010 | All aliases; repeated/comma keys; implicit and explicit stdin; whitespace/dedup/order; malformed flag/stdin and conflicts with zero read requests; strict mixed-found/missing lookup; cross-tenant authorized key and denied access |
| FR-004–FR-009 | Each filter and combinations serialized correctly on all supported versions; nine states/all/case normalization; no `assigned`; tenant defaults/override/all-tenant options; page bounds; partial/empty intermediate pages; advancing cursor; offset fallback; limits mid-page; exact vs capped counts; repeated cursor and overflow errors |
| FR-011–FR-014 | Exact human rows/summary; technical IDs plus optional labelled name and final assignee with explicit unassigned marker; JSON decode plus EOF; keys exact bytes; empty payload; quiet+JSON/keys; JSON precedence; numeric total including quiet; automation/auto-confirm; separate stdout/stderr and activity suppression |
| FR-014, FR-017 | Real-terminal yes/no/EOF paging; configured and inherited stderr; redirected stdout; keys one per line; prompt-free empty completion; request counts and no mutation calls |
| FR-015–FR-016 | v88/v89/v810 direct/search results and errors; v87 unsupported with no new native requests; existing resolver's tenant and Tasklist fallback tests unchanged; failure after an earlier streamed page returns failure without complete-success claims |
| FR-018–FR-019 | Help/capability snapshots, runnable examples, README and generated docs; no out-of-scope flags or operations |

Tests must cover actual command paths, not only view helpers. Use fake HTTP servers for capped totals, sparse pages, malformed payloads, authorization failures, and request-count assertions. Do not run tests mutating global command state in parallel. New/touched test functions and declarations receive the comments required by repository rules.

### Delivery order

Start with domain and native contracts, implement v810 first and compare v89/v88 explicitly, then add v87 unsupported stubs. Preserve resolver regression tests throughout. Next deliver shared service workflows and facade mappings, then CLI keyed lookup, search/count interaction, output matrix, and documentation. Each slice gets its closest tests before broader integration validation. This ordering guides task generation without creating tasks in the planning phase.

### Validation gates

Follow [constitution v2.0.0](../../.specify/memory/constitution.md): select the smallest checks covering each changed behavior, using [quickstart.md](quickstart.md) as the acceptance matrix. Format touched Go files and regenerate CLI documentation when command metadata changes. Run `make test` for the integrated shared-contract and concurrent bulk changes, or when other concrete cross-package risk warrants it, and record the reason. Reuse passing evidence until relevant changes or failures invalidate it; do not rerun tests merely to commit or mark a task complete. Verify command declaration ownership before completion. Documentation-only refreshes require diff, consistency, and local-link checks, not runtime tests or CLI generation.

## Complexity Tracking

No constitution violations or additional framework/dependency complexity. A separate native getter is necessary to preserve the legacy tenant/fallback contract. A focused service traversal is necessary because nearby implementations' empty-page rules do not satisfy this feature; reuse primitives and local patterns without broad refactoring of unrelated commands.
