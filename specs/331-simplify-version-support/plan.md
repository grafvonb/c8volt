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
| Small, repository-native changes | No new layering or abstractions. | Pass: three cohesive units; coupled runtime work and tooling/assets work each share one validation pass. |

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

1. **Runtime and shared compatibility (US1)**: Remove the retired identity, aliases, adapters, wiring, affected tests, tenant defaults and capability guards in one compiling unit. Delete the now-exclusive legacy dry-run path with its sole caller. Preserve generic deployment fallback. Update current guidance, command metadata, README and generated docs together. Run focused checks once.
2. **Generation and validation assets (US2)**: Delete retired generated output with its targets/protection/mock entries, then remove retired fixtures/integration cases. Preserve retained component paths and unrelated clients. Run generation checks, true compile-only tagged integration checks, one full suite and build for the complete executable result.
3. **Remaining tracked support content (US3)**: Remove obsolete historical requirements, PRD examples and guidance from a single candidate inventory. Preserve unrelated context. Complete the final tracked-path/content audit and document checks; rerun generators only if their source changes.

Each unit includes fixtures/test-double updates, formatting, relevant validation and progress. No standalone context-reading, branch-check or bookkeeping tasks. The identity removal remains broad but compilation-coupled. Shared dry-run deletion is small enough to finish with its caller; generator/assets removal has a common final validation boundary. Historical cleanup remains separate because hundreds of candidate documents need focused diff review. Three tasks do not guarantee three turns: Ralph is bounded by one story per iteration and practical validated progress.

## Validation Strategy

See [quickstart.md](quickstart.md) for task-owned commands and prerequisites. Reuse passing evidence until relevant edits invalidate it; do not repeat broad package tests or docs generation in each unit. Use generic invalid identities in unsupported-version tests. Capture request counts and prompts; retain real terminal stdin where prompt eligibility matters. Preserve accepted/confirmed mutation distinctions, exact empty JSON envelopes, zero-byte keys output, sparse pages, explicit keys, abort/error paths and quiet/machine-output precedence.

Run generation harnesses without contacting a live cluster. The repository-boundary script checks Git cleanliness as well as content; execute it once in a temporary clean snapshot containing reviewed changes before task completion (see quickstart). Never reset/discard changes or weaken protections to make it pass. Full upstream refresh is unnecessary if deterministic generation tests prove target exclusion and unchanged retained outputs.

Compile integration-tagged packages with `go test -c` so their `TestMain` does not run or rebuild the binary. Do not run destructive suites. Live-cluster execution is additional evidence only with disposable configured clusters; do not infer passing live validation from mocked tests. Existing harness excludes experimental runtime coverage; this change does not expand it.

## Complexity Tracking

No new complexity or constitution exception. The explicit support break is recorded above. No unresolved clarification remains.

## Accepted closeout — 2026-10-06

The user approved finishing #331 with the pre-existing repository-boundary conflict deferred to the separate stable Camunda 8.10 upgrade. Keep the restored guard and existing integration coverage. The guard's failure was reproduced at pre-implementation commit `c9949f11` and is not reported as passing. This accepted exception supersedes the all-boundary-checks-pass completion requirement for this feature only.

No stable 8.10 client refresh, baseline/maturity change, integration expansion or guard redesign belongs to this closeout. Default remains 8.9. Runtime tests, build and generation evidence from the implementation are retained; review corrections received targeted command tests, integration compilation, shell syntax and whitespace checks. See `progress.md` for evidence and corrections to earlier completion claims.
