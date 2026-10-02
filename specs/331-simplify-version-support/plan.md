# Implementation Plan: Simplify Supported Camunda Versions

**Branch**: `codex/331-simplify-version-support` | **Date**: 2026-10-02 | **Spec**: [spec.md](spec.md)

**Input**: [Issue #331](https://github.com/grafvonb/c8volt/issues/331)

## Summary

Keep Camunda 8.8, 8.9, and 8.10 as the supported set, with 8.9 as default. Remove the retired release identified by the issue from runtime selection, adapters, generated clients, fixtures, tests, generation workflows, and tracked support documentation. Remove exclusive compatibility behavior while preserving dependencies still used by retained versions.

## Technical Context

**Language/Version**: Go 1.26, toolchain go1.26.2; existing shell and Python generation checks.

**Primary Dependencies**: Cobra, Viper, testify, oapi-codegen runtime and existing generated clients. No new dependencies.

**Storage**: Existing file-based configuration and tracked artifacts; no persistence migration.

**Testing**: Targeted Go package tests, HTTP fixtures, terminal/subprocess command tests, generation shell/Python checks; full race-enabled suite and build for the broad runtime removal.

**Target Platform**: Existing CLI platforms and retained Camunda releases.

**Project Type**: CLI and public Go library.

**Performance Goals**: Reduce maintained code and duplicated tests; no unsupported claim of measured speed or binary-size improvement.

**Constraints**: Preserve supported-version semantics, component authentication, output envelopes, tenant isolation, real-terminal prompt coverage, version capability restrictions, and generated-client provenance. No new compatibility shim or retired-release rejection suite.

**Scale/Scope**: Eleven service families; approximately 55 dedicated tracked files identified in the initial audit, plus shared code and documentation. Counts are estimates for scoping, not deletion quotas. Unrelated unused clients remain outside this feature.

## Constitution Check

| Principle | Before design | After design |
| --- | --- | --- |
| Operational proof | Preserve mutation acceptance, waiting and confirmation semantics. | Pass: retained regression paths and targeted transport checks specified. |
| CLI-first, script-safe | Intentional support-policy break is authorized by issue and user. | Pass: generic unsupported errors, existing exit classes, output streams and configuration precedence retained. |
| Proportionate validation | Planning receives document checks only. | Pass: targeted checks accompany implementation; full suite justified by shared version selection and generated-client removal. |
| Documentation matches behavior | Support messaging changes with runtime selection. | Pass: README, command source metadata and regenerated docs included in the first runtime unit; historical guidance cleanup follows. |
| Small, repository-native changes | No new layering or abstractions. | Pass: five compiling units, with coupled references consolidated and separate tooling/assets work. |

**Compatibility note**: This intentionally removes the retired release's public version identity and all accepted aliases, including source compatibility for callers using that identity. No shim, deprecation period, migration document or dedicated rejection suite is introduced. Generic unsupported-version behavior remains. User-facing documentation states the resulting support set and default.

The user's explicit removal instruction supersedes the old version inventories in repository guidance and Ralph rules. Their architecture, UX, validation and work-unit rules remain binding. Update the version inventories with the first runtime unit so later iterations do not carry contradictory context. There are no unresolved architecture or UX conflicts.

## Project Structure

### Documentation (this feature)

```text
specs/331-simplify-version-support/
  spec.md
  plan.md
  research.md
  data-model.md
  quickstart.md
  contracts/support-policy.md
  checklists/requirements.md
  tasks.md
```

### Source Code (repository root)

```text
toolx/version.go, fixture_compatibility.go
config/app.go
cmd/root.go, root_config.go, get_processinstance_validation.go
cmd/cmd_deploy_run_data.go
c8volt/client.go, diagnostics_test.go, ferrors/errors.go
internal/services/<area>/{api.go,factory.go,factory_test.go}
internal/services/<area>/<retired-adapter>/
internal/services/processinstance/dryrun.go
internal/services/ops/slow_process_analysis.go
internal/clients/camunda/<retired-client>/
api/{3-generate-clients-from-fetched-specs.sh,generate-v810-client.sh,tests/}
embedded/processdefinitions/, testx/, integration/
README.md, docs/, docsgen/, AGENTS.md, specs/, tasks/
```

`<area>` expands to batchoperation, cluster, element, incident, job, processdefinition, processinstance, resource, tenant, usertask and variable. The retired directory is identified from issue #331 and the current inventory; placeholder paths describe deletion targets without reintroducing obsolete support strings into these artifacts.

## Design and Delivery Boundaries

1. **Runtime cutover (US1)**: Delete the retired identity, aliases, adapter directories, wiring and all references that would fail compilation or tests. Normalize supported discovery tenants uniformly; remove obsolete capability guards. Change version-sensitive command metadata, README and generated docs in the same unit. Update current version inventories in AGENTS and Ralph rules. Retained generated clients and unreferenced assets may remain until their dedicated cleanup unit.
2. **Shared compatibility cleanup (US2)**: Delete the unreachable legacy dry-run dispatch/interface/path and its tests. Trace other exclusive helpers. Preserve the version-independent deployment identifier fallback and its generic tests because it also handles incomplete supported responses; remove only retired-specific tests and comments. Keep normal traversal, uncertain totals and key guards.
3. **Generation cleanup (US2)**: Delete the retired generated-client tree together with generator targets, mockery entries and protection lists. Verify retained generation output contracts. Do not regenerate unrelated stable clients merely to delete one target.
4. **Fixture and integration cleanup (US2)**: Delete retired embedded and integration assets and remove harness exceptions; preserve retained restrictions. Refresh fake topology values and tests where required. Run the full suite and build after the final executable change.
5. **Tracked support-content cleanup (US3)**: Remove remaining obsolete support requirements from historical specs, PRDs, guidance and prose. Preserve unrelated history and context; do not delete whole historical documents just to remove a sentence. Audit paths and content, including these artifacts. Update any generated documentation through its owner.

Each unit includes fixtures/test-double updates needed for compilation, formatting, nearby validation and progress recording. No separate context-reading, branch-check or bookkeeping task. The runtime cutover is necessarily broad but predominantly mechanical deletion; splitting constant removal from its consumers would break compilation. Remaining units have distinct observable outcomes.

## Validation Strategy

See [quickstart.md](quickstart.md) for commands and prerequisites. Use generic invalid identities in unsupported-version tests. Capture request counts and prompts; retain real terminal stdin where prompt eligibility matters. Preserve accepted/confirmed mutation distinctions, exact empty JSON envelopes, zero-byte keys output, sparse pages, explicit keys, abort/error paths and quiet/machine-output precedence.

Run generation harnesses without contacting a live cluster. The repository-boundary script checks Git cleanliness as well as content; execute it after authorized cleanup commits in a clean relevant tree, or in a temporary snapshot containing the reviewed changes. Never reset/discard changes or weaken protections to make it pass. Full upstream refresh is unnecessary if deterministic generation tests prove target exclusion and unchanged retained outputs.

Compile integration-tagged packages without running destructive suites. Live-cluster execution is additional evidence only with disposable configured clusters; do not infer passing live validation from mocked tests. Existing harness excludes experimental runtime coverage; this change does not expand it.

## Complexity Tracking

No new complexity or constitution exception. The explicit support break is recorded above. No unresolved clarification remains.
