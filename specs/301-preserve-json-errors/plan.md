# Implementation Plan: Preserve JSON Error Envelopes

**Branch**: `codex/301-preserve-json-errors` | **Date**: 2026-09-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/301-preserve-json-errors/spec.md`

## Summary

Repair command-execution failures that currently bypass an already-advertised shared JSON contract. Reuse `handleCommandError` at eligible validation/runtime failures, pass the actual command into shared stdin-key validation, and preserve existing normalization, classification, termination, human diagnostics, and `--no-err-codes`. The audit includes the issue's known paths plus process-definition retrieval/validation and embedded-file listing. No contract expansion or service-layer change is needed.

## Technical Context

**Language/Version**: Go 1.26; repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, Viper 1.21.0, standard-library JSON/logging/process tools, and repository `c8volt/ferrors`, `testx`, and command views. No new dependencies.

**Storage**: No persistence changes; tests use temporary configuration and local fixtures.

**Testing**: Go testing and testify; subprocess execution with separately captured streams, local HTTP fixtures, focused command tests followed by `make test` (`go test ./... -race -count=1`).

**Target Platform**: Existing supported CLI platforms; subprocess checks run in the existing development/CI environment without a real Camunda cluster.

**Project Type**: CLI with public facades and versioned internal services; this correction stays in the command layer.

**Performance Goals**: One result per eligible failed command; no new backend calls, retries, or discovery introduced by rendering.

**Constraints**: Preserve schemas, error wrappers/classes, exit-code policy, JSON precedence, quiet/automation behavior, success results, and non-full contract support. Keep raw XML, bootstrap, parsing, prompts, tenant formatting, and empty-result semantics unchanged.

**Scale/Scope**: Four known direct handler corrections, two shared stdin validation branches across 12 callers, four additional process-definition branches, and two embedded-list execution branches. Exact inventory and exclusions are in [research.md](research.md).

## Constitution Check

| Gate | Before research | After design |
| --- | --- | --- |
| I. Operational proof | PASS: retain failure meaning and termination | PASS: no accepted/succeeded result for failures, even with exit suppression; no operational workflow change |
| II. CLI-first and script-safe | PASS: repair existing contract only | PASS: one envelope, separate diagnostics, existing modes and exit codes retained |
| III. Tests mandatory | PASS: command-level subprocess coverage planned | PASS: each reachable correction has command-path coverage, deterministic embedded failure seam, targeted checks then full race suite required |
| IV. Documentation aligned | PASS: user-visible correction requires docs | PASS: README and affected help metadata updates followed by generated CLI references |
| V. Small compatible changes | PASS: existing handler reuse | PASS: command-context parameter and one narrow local listing seam; no framework, dependencies, facade/service/client changes |

No gate failures or exceptions. These are design compliance checks, not claims that implementation tests have already passed.

## Project Structure

### Documentation (this feature)

```text
specs/301-preserve-json-errors/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/cli-error-envelope.md
└── tasks.md                         # Future speckit-tasks output
```

### Source Code (repository root)

```text
cmd/
├── cmd_cli.go                      # Command context for shared key validation
├── delete_processinstance.go       # Search validation and helper caller
├── get_cluster_topology.go
├── get_cluster_version.go
├── get_cluster_license.go
├── get_processdefinition.go        # Eligible validation/retrieval/search errors
├── embed_list.go                   # Local runner/listing injection and errors
├── <11 other stdin helper callers> # Enumerated in research.md
├── cmd_views_contract.go           # Existing handler: reuse, no redesign
├── command_contract.go             # Existing schema and eligibility: unchanged
├── cmd_error_envelope_test.go      # Proposed scoped subprocess regression matrix
├── get_test.go                     # Existing cluster and definition regressions
├── get_processdefinition_test.go
├── embed_test.go
└── command_contract_test.go
testx/cmd_subprocess_runner.go       # Existing separate-stream runner: reuse
c8volt/ferrors/errors.go             # Existing classification/exit policy: unchanged
README.md
docs/cli/                           # Generated with make docs-content
```

**Structure Decision**: Keep ordinary error dispatch in its existing command files and final rendering in existing command views. The helper signature change is internal to `cmd`. The embedded listing seam stays command-local as ordinary execution, not a new lifecycle or mode. Preserve the facade/service/versioned-adapter boundaries and avoid generated clients.

## Phase 0: Research Completed

[research.md](research.md) resolves eligibility, all direct-exit scope, shared-helper callers, testing seams, mode precedence, documentation, and alternatives. The independent research audit confirmed all 12 current helper callers are full-contract. The setup script reports a normalized feature label `301-preserve-json-errors`; the verified Git branch is `codex/301-preserve-json-errors`, and `.specify/feature.json` resolves this feature directory.

## Phase 1: Design

### Error dispatch

1. Replace only the eligible validation/runtime direct exits enumerated in research with `handleCommandError`, preserving the original error expression and wrappers.
2. Add `cmd *cobra.Command` to `mergeAndValidateKeys` and pass the real caller at all 12 sites. Leave validation order, key merging, indexes, and deduplication unchanged.
3. Keep the handler's immediate exit even with `--no-err-codes`; neither a second diagnostic nor subsequent command work may occur.
4. Extract an ordinary embedded-list runner taking a listing function explicitly. Production passes `embedded.List`; subprocess tests pass deterministic listing errors or an empty list. Preserve bootstrap handling and success output.
5. Leave unreachable renderer-error branches and raw XML retrieval/write paths unchanged, as documented by the audit. Do not alter `renderResultEnvelope` write-failure semantics.

### Acceptance and regression matrix

Use proposed test prefix `TestCommandErrorEnvelope` for the new matrix. Reuse `testx.RunCmdSubprocessInDirWithSeparateOutputs`, exact helper-process scoping, isolated config, and local HTTP fixtures. Run actual command execution wherever inputs can reach the error; the embedded failure cases use the production runner with injected listing results. Do not run flag-global cases in parallel within one process.

| Slice | Required cases and evidence |
| --- | --- |
| Delete search validation | Invalid search combination accepted by argument parsing; verify invalid envelope, precise error, no mutation/discovery after validation |
| Shared stdin | All 12 commands × filter-text and malformed-key forms × JSON/human × default/suppressed exit codes; preserve bad-key index after a valid key and blank input line |
| Cluster | Topology/version/license runtime errors; 503 unavailable and representative malformed response; existing endpoint and normalized context retained |
| Process definition | By-key retrieval, selector lookup, paged search failure; XML with JSON missing key and with key; invalid validation outcome and no XML retrieval on rejection |
| Embedded list | Listing failure and empty version scope through production runner; failed/local-precondition preserved for empty scope; real command success and version filtering unchanged |
| Output combinations | JSON plus quiet, JSON plus keys-only where available, and supported automation; preserve precedence and existing unsupported-automation rejection |
| Fallback | A command fixture with limited/unsupported contract metadata through shared validation; real non-full config/embed regression tests; no claim that such a helper caller currently exists |
| Success and existing failures | Cluster JSON/human successes, valid merged keys and caller deduplication, definitions list/key/XML, embed list; an already-correct error must still emit only one envelope |

For every corrected error branch, cover ordinary JSON and human output and both exit-code settings. Decode one JSON value then require EOF; assert command, outcome, class, detail, existing omission rules, absence of duplicate diagnostic text, and exit status. Human mode requires zero stdout bytes and the existing diagnostic on stderr; allow existing unrelated stderr context without relabeling it as duplicate error output. Assert no post-error work, especially when exit status is zero. Account for tenant context only where already attached; do not add requests to enrich it.

No terminal prompt behavior is modified. Pipe-based stdin is the real input under test here; existing terminal prompt tests remain in the full regression suite.

### Documentation and validation delivery

Update README machine-contract guidance and affected command source descriptions to explain command-execution errors, preserved exit suppression, and the pre-execution exclusion. Preserve support annotations and command identifiers. Regenerate `docs/cli/` using `make docs-content`; do not hand-edit generated pages. Inspect generated changes for unrelated churn.

Run focused tests from [quickstart.md](quickstart.md), format touched Go files, regenerate docs, and run `make test` before implementation completion or commit. Plan artifacts do not require application test execution; only artifact consistency is checked at this phase.

### Requirement traceability

- FR-001–FR-005: dispatch corrections and per-path envelope assertions.
- FR-006–FR-009: fallback, mode, exit, success, and non-full compatibility matrix.
- FR-010–FR-011: separate-stream subprocess checks, request counts, focused checks, full race suite.
- FR-012: README/help metadata and generated documentation.
- SC-001–SC-005: matrix assertions measure single-result consumption, failure precision, exit compatibility, human streams, and unchanged regressions.

## Complexity Tracking

No constitutional violations. The only testability accommodation is a command-local explicit listing-function parameter for compiled-in fixture errors; it avoids global mutation and does not introduce a new public abstraction.
