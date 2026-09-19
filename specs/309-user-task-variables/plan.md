# Implementation Plan: Display Effective User-Task Variables

**Branch**: `codex/309-user-task-variables` | **Date**: 2026-09-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/309-user-task-variables/spec.md`, backed by issue #309 and the implemented base command from #308.

## Summary

Add opt-in `--with-vars` and `--var-value-limit` to the existing user-task command. Extend the task facade with one selected-task enrichment method and the internal usertask adapters with native effective-variable page reads. Internal services own complete offset pagination, validation, sorting, and sequential task enrichment. The CLI enriches selected incremental pages or the final collected result once, preserving limits, output modes, and prompts. Reuse existing variable records, raw response decoding patterns, and the current process-variable presentation conventions without applying process-root scope filtering.

## Technical Context

**Language/Version**: Go 1.26; toolchain go1.26.2 pinned in `go.mod`.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag, Viper, standard context/HTTP/JSON, generated Camunda clients, shared service helpers, `toolx`, and `testx`. No new dependency or framework.

**Storage**: Camunda is authoritative; no local persistence or migrations. Selected tasks and complete variable collections are held in memory.

**Testing**: Go/testify, HTTP fixtures, facade/service stubs, command subprocesses, and real-terminal `testx.NewCmdTerminalRunner`. Use focused checks first, then integrated race validation for changed shared interfaces and shared formatter behavior. Planning-only changes use document checks, not runtime tests.

**Target Platform**: Existing CLI/library platform support, including Linux, macOS, and Windows builds. Terminal tests follow the existing supported-platform helpers.

**Project Type**: CLI plus public Go facade library.

**Performance Goals**: Zero variable requests for excluded tasks, absent enrichment, effective keys-only/count, and empty results. Retrieve each selected task's variable pages once. Preserve task order. Use fixed internal variable pages of 1000 and sequential enrichment matching the process-variable workflow; no new latency SLA or concurrency policy.

**Constraints**: Native effective-variable search on 8.8, 8.9, and 8.10; explicit unsupported behavior on 8.7. Generated response records omit values/truncation, requiring adapter-local raw decoding. Endpoint paging is offset-only. Always request `truncateValues=false` and mark any remaining truncation. Preserve tenant/authentication/error/interactive contracts and legacy task resolver behavior. No generated-client edits, generic enrichment framework, variable filters, mutations, forms, or audit logs.

**Scale/Scope**: Two CLI flags, one new public facade operation, one new adapter page operation across the version contracts, and task-specific wrappers. Memory grows with selected tasks and their variables; page requests are bounded, but complete JSON is not constant-memory. Existing task search and keyed worker controls remain unchanged. Backend reads are eventually consistent and do not provide a new snapshot guarantee.

## Constitution Check

*Pre-research gates passed; post-design review also passes. No exception is requested.*

| Principle | Pre-research gate | Post-design evidence |
| --- | --- | --- |
| Operational proof over intent | Successful reads must represent completed retrieval | Complete variable pagination; inconsistent metadata and later failures remain errors; no empty substitution or final success claim after failure |
| CLI-first, script-safe interfaces | Preserve existing flags, modes, and prompt streams | Explicit output matrix and CLI contract; one JSON envelope; pure keys/count; stderr prompt tests; value-limit compatibility |
| Validation proportional to change | Select checks by actual behavior and risk | Targeted adapter/service/facade/command checks; integrated race suite for shared interfaces/formatter; only document checks during this planning phase |
| Documentation matches behavior | Both flags change user-visible behavior | Update source help, examples, command contract tests, README, and regenerate with `make docs-content` |
| Small compatible repository-native changes | Extend existing ownership and reuse semantics | One facade method; existing task traversal; focused mode/view files; existing variable record alias and pure formatter; no new framework or dependency |

Compatibility is additive: ordinary task results are unchanged. The enriched JSON wrapper is selected only by `--with-vars`. Reusing a public process-variable record does not reuse process-scope filtering or process metadata. No unresolved architecture or constitution conflict was found.

## Project Structure

### Documentation (this feature)

```text
specs/309-user-task-variables/
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

`tasks.md` is generated by the next `$speckit-tasks` phase, not by this plan.

### Source Code (repository root)

```text
cmd/
├── get_usertask.go                   # Flags, validation, metadata, ordinary dispatch
├── get_usertask_search.go            # Existing paging; enrich selected incremental pages
├── get_usertask_vars.go              # New focused enrichment dispatch helpers
├── cmd_views_usertask.go             # Existing task rows/count/empty views
├── cmd_views_usertask_vars.go        # New enriched tree and JSON views
├── cmd_views_variable_values.go      # Narrow explicit-limit formatter extracted for reuse
├── cmd_views_processinstance_vars.go # Preserve PI wrapper and observable behavior
├── get_usertask_*test.go             # Extend execution/output/search/error/terminal coverage
└── command_contract_test.go          # Flags/help/capabilities and examples
c8volt/task/
├── api.go, client.go                 # Add thin selected-task enrichment method
├── model.go, convert.go              # Record alias, enriched wrappers, mechanical mapping
└── *_test.go                        # Options, mapping, ordering, empty/error coverage
internal/domain/
└── usertask.go                       # Enriched task wrappers and offset variable page types
internal/services/usertask/
├── api.go                           # Effective-variable page contract
├── variables.go                     # Complete pagination and sequential enrichment
├── variables_test.go                # Completion, scope/name integrity, errors, cancellation
├── v87/                             # Unsupported operation and tests
└── v88/, v89/, v810/
    ├── contract.go                  # Extend native generated interface and adapter API
    ├── variables.go                 # Effective-variable request and raw conversion
    └── variables_test.go            # Version-specific HTTP/decoding/option/error checks
README.md
docs/cli/                            # Generated using make docs-content
```

**Structure Decision**: Keep backend loops in `internal/services/usertask`. Keep the existing task search walker and legacy resolver unchanged. New cohesive enrichment dispatch belongs in `get_usertask_vars.go`, especially once it has three mode-specific declarations; formatting stays in user-task view files. Shared formatting is a narrow pure helper, not a new rendering framework. Existing domain/public variable record definitions and generated clients are reused without relocation. Update all affected interface stubs and assertions.

## Phase 0: Research Outcomes

[research.md](research.md) records decisions, rationale, alternatives, and evidence. Resolved questions include identical version contracts, offset-only paging, capped totals, raw response fields omitted by generated models, explicit truncation, exact empty shapes, sequential enrichment, and the selected-page integration point. No unresolved clarification remains.

## Phase 1: Design

### Data and service boundary

Use [data-model.md](data-model.md) and [contracts/facade-service.md](contracts/facade-service.md). The adapter owns one native effective-variable page and generated/raw conversions. The shared service owns all pages, checked offset advancement, exact/lower-bound completion with retained lower bounds and sparse-page continuation evidence, stable name sorting, duplicate consistency, cancellation, and task attachment. It must not filter variables to the process root. Preserve actual returned scope and tenant information.

Always disable backend truncation in the request and preserve reported remaining truncation. Do not perform additional variable GET recovery or substitute a process-variable search. Explicit offset metadata validation is necessary because absent generated primitive fields otherwise resemble an authoritative empty response.

### Command integration and rendering

Keyed input retains strict bulk task resolution before enrichment. For incremental human search, enrich the already-trimmed visitor page and render it before asking the existing paging question. For collected modes, complete the existing selection first, enrich once, and render the final result. The final streamed summary must not trigger another enrichment pass.

Gate retrieval by effective output mode, not raw flag presence. Count exits through its existing path; keys-only and empty collections never request variables. Quiet human still honors requested retrieval and errors, while its view emits nothing. JSON remains explicit even with quiet and wins over keys-only. Preserve command control streams and writer-error propagation.

Human output uses existing task rows plus a `vars:` subtree; the narrow shared formatter takes the value limit explicitly. Preserve the PI wrapper and regression tests. JSON uses task-specific enriched wrappers with initialized arrays, int64 returned counts, and no process age metadata. Full behavior is in [contracts/cli.md](contracts/cli.md).

### Validation and acceptance mapping

| Requirement group | Required coverage |
| --- | --- |
| FR-001–003 | Single/multiple/repeated/comma/stdin keys and aliases; effective scope values; multi-page reads; names unique per task; stable ordering; exact/capped totals and sparse variable pages |
| FR-004–005 | Task limit inside a page; declined paging; filtered tenants; explicit authorized cross-discovery-tenant keys; no variable requests for omitted flag, empty tasks, effective keys-only, count, or excluded tasks |
| FR-006–009 | Exact tree and summary; JSON decode plus EOF; empty `items`/`variables`; Unicode boundaries; compaction; default/zero/positive/negative and dangling limit; JSON unaffected by human limit; both raw truncation fields; request `truncateValues=false`; PI formatter regression |
| FR-010–012 | Later-page/task failures; malformed/missing raw payload fields; duplicate conflict; cancellation/overflow; 401/403/404/upstream errors; writer errors; no successful completion after failure; v88/v89/v810 parity and v87 unsupported |
| FR-011 | Quiet and quiet+machine; JSON+keys precedence; count conflicts; auto-confirm/automation; real-terminal stdin; configured/inherited stderr; continue/decline/EOF; redirected stdout; empty prompt-free completion |
| FR-013–014 | Source metadata/help no longer excludes variables; all four issue examples; README and generated docs; no filtering/mutation scope expansion; zero mutation requests |

Use existing task test helpers and stub servers. Command execution tests must prove absence of requests and output contamination; view-only tests are insufficient. Do not parallelize tests that modify global command state. Backend fixtures, rather than live mutation setup, cover pathological paging and error cases.

### Delivery sequence

1. Introduce domain paging/enriched records, public record alias, and adapter contracts with corresponding stubs.
2. Implement and test native v810 page requests/raw decoding, explicitly mirror/validate v89 and v88, and implement no-request v87 unsupported behavior.
3. Add complete offset traversal and selected-task enrichment with focused service tests; preserve task traversal/resolver regressions.
4. Add the thin facade operation and conversions, proving options, error categories, initialized empties, and unchanged task metadata.
5. Add flags, selected-page/final-result dispatch, explicit-limit formatter, and task views; cover keyed/search/output/terminal paths and process formatter compatibility.
6. Update source documentation/README, regenerate CLI docs, and complete integrated validation. This sequence guides task generation without creating an implementation task list here.

### Validation gates

This planning pass validates artifact completeness, links, whitespace, issue/branch consistency, and contract agreement only. During implementation use [quickstart.md](quickstart.md) for focused commands and end-to-end scenarios. Run `gofmt` for touched Go files and `make docs-content` after metadata edits. The integrated feature warrants `make test` because it extends shared facade/service interfaces and a formatter used across commands; focused slices do not independently require it. Reuse passing evidence until relevant changes invalidate it. No runtime test or CLI generator was run for these documentation-only artifacts.

## Complexity Tracking

No constitution violation or dependency exception. The small dedicated offset loop is necessary because the task walker follows cursors and is task-specific. A shared value formatter prevents state coupling to PI flags; its extraction is limited to behavior used by both commands. A public record alias avoids duplicate variable schemas without moving package ownership. Sequential enrichment avoids adding concurrency to the feature.
