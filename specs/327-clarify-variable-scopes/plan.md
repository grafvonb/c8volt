# Implementation Plan: Clarify Variable Destination Scopes

**Branch**: `codex/327-clarify-variable-scopes` | **Date**: 2026-10-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/327-clarify-variable-scopes/spec.md`; [issue #327](https://github.com/grafvonb/c8volt/issues/327).

## Summary

Make every detailed planned addition and change identify its destination in PI and UT update previews. Carry the already fetched owning process key through the user-task plan as non-serialized metadata, then format `pi:<key>` and `element:<key>` annotations in command views. Preserve local/inherited distinctions, compact bulk inherited-target visibility, summary wording, all machine results, and existing update workflows. No extra backend calls, new dependencies, generated-client edits, or new commands are needed.

## Technical Context

**Language/Version**: Go 1.26; repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, standard formatting/JSON libraries, existing domain/service/facade models; no additions or upgrades.

**Storage**: Existing in-memory plans only; no persistent schema change.

**Testing**: Go testing, testify, existing HTTP fixtures, command subprocess helpers, and real-terminal runner in `testx`.

**Target Platform**: Existing supported CLI platforms; terminal tests use the repository's platform-aware harness.

**Project Type**: CLI and public Go facade library.

**Performance Goals**: Zero additional requests; one string field per planned task and existing linear traversal for rendering. No new latency or throughput promise.

**Constraints**: Existing JSON and keys contracts, update scopes, confirmation/status behavior, and summary wording remain stable. Planning facts stay in services/models; human labels stay in views. No inference of intermediate BPMN element type.

**Scale/Scope**: Existing single and multi-key PI/UT commands on supported v88, v89, v810 adapters. Extend existing fixtures rather than introduce another workflow or compatibility matrix.

## Constitution Check

Pre-research and post-design gates both pass:

| Principle | Design evidence | Result |
| --- | --- | --- |
| Operational proof over intent | Rendering metadata cannot affect accepted/confirmed/failure transitions; existing confirmation remains authoritative | PASS |
| CLI-first, script-safe | Only planned-write human annotations change; public owner metadata uses `json:"-"`; machine contract tests retained | PASS |
| Proportionate validation | Documentation checks for this plan; focused service/facade/command checks during implementation; real terminal tests for prompts | PASS |
| Documentation matches behavior | Update both command help sources and README with each user-visible slice; regenerate CLI docs | PASS |
| Small, compatible, repository-native | One metadata field, mechanical conversion, existing view helpers and test harnesses; no new backend interfaces | PASS |

Compatibility note: human addition/change text and bulk inherited scope labels intentionally gain destination notation. Scripts parsing human prose must use existing machine modes for a stable contract. Unknown legacy metadata retains existing untyped wording rather than causing a new error or lookup.

Current Ralph rules and AGENTS layering/UX requirements are compatible with this design. Existing PI command-side planning is not expanded or refactored; this feature only consumes its existing selected key for formatting. No new command lifecycle is introduced, so the mode-file split gate does not apply.

## Project Structure

### Documentation (this feature)

```text
specs/327-clarify-variable-scopes/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/preview-output.md
```

`tasks.md` is produced by the later task-generation command, not this planning phase.

### Source Code (repository root)

```text
internal/domain/usertask_update.go          # Add owner metadata
internal/services/usertask/update.go        # Copy already fetched task identity
internal/services/usertask/update_test.go   # Metadata + unchanged request/target evidence
c8volt/task/model.go                        # Public non-serialized metadata
c8volt/task/convert.go                      # Both conversion directions
c8volt/task/update_test.go                  # Round-trip and JSON compatibility
cmd/cmd_views_usertask_update.go            # Detailed + compact inherited labels
cmd/cmd_views_processinstance_update.go     # PI addition/change labels
cmd/cmd_views_usertask_update_test.go        # Exact view contracts
cmd/update_usertask*_test.go                # Command, output, and terminal regressions
cmd/update_processinstance_test.go          # PI preview/output regressions
cmd/update_usertask.go                      # Help/examples only
cmd/update_processinstance.go               # Help/examples only
README.md                                  # Operator guidance
docs/cli/                                  # Regenerate via make docs-content
```

**Structure Decision**: Extend existing owners. Do not add a scope service, new endpoint, new facade method, or general-purpose formatting package. A tiny pure helper within the UT view can serve both detailed and bulk rendering; PI already knows its root key.

## Phase 0: Research Decisions

See [research.md](research.md). Repository research resolved metadata provenance, serialization compatibility, display eligibility, stable bulk deduplication, incomplete-plan fallback, and focused validation. No unresolved clarification or gate failure remains.

## Phase 1: Design

1. Add process identity to internal and public per-task plans; populate at existing task-plan construction and map both directions. Never derive mutation scopes from the new field.
2. Render UT additions/changes with relationship and typed destination. Render bulk inherited destinations using the same classification while preserving raw-key deduplication/order. Keep unchanged/untouched and final results unchanged.
3. Render PI additions/changes using the existing plan key. Reuse current normal/dry-run dispatch and bulk-detail eligibility.
4. Preserve the old untyped inherited annotation only when legacy/incomplete metadata cannot establish kind. No new validation or requests.
5. Extend exact view assertions and real command scenarios. Update help and README guidance with delivered behavior and regenerate docs.

Detailed fields/invariants are in [data-model.md](data-model.md); user and library contracts are in [contracts/preview-output.md](contracts/preview-output.md); execution checks are in [quickstart.md](quickstart.md).

## Delivery and Validation Strategy

Organize later tasks around two cohesive slices: UT metadata plus all UT display modes, then PI display. Include each slice's fixtures, regression assertions, help/README updates, generated docs, formatting, and focused checks in that same work unit. Story 3 supplies compatibility acceptance coverage to both slices; do not postpone bulk support or rebuild a single-item solution later. Keep each Ralph iteration within its chosen primary story. Split only if a distinct compilable, independently verifiable result warrants it; no standalone context-reading, routine review, or artificial parallelism tasks.

Validate metadata with existing service and facade tests, including sparse discovery and shared targets. Assert JSON equality with owner metadata present/absent. At command level verify root/intermediate/local additions and changes, both preview entry paths, verbose and compact bulk, stable inherited ordering, exact summaries, unchanged/untouched entries, quiet/machine precedence, and one-envelope EOF behavior. Retain tests for no-op, explicit keys, errors, aborts, accepted/confirmed results, and all supported versions. Use real terminal stdin and separate streams for normal confirmation and prompt-free no-ops; auto-confirm alone cannot exercise normal preview rendering. Track all relevant requests so a new scope lookup cannot escape detection.

Run targeted checks listed in quickstart after executable changes. Use focused `-race` coverage if shared concurrent fixtures are touched. Run `make test` only if implementation expands into shared runtime behavior, concurrency, dependencies, or unresolved regressions warrant it. This documentation-only planning phase performs no runtime tests or docs regeneration.

## Complexity Tracking

No constitution violations or new architectural complexity require exceptions.
