# Implementation Plan: User-Task Integration Coverage

**Branch**: `codex/324-user-task-integration` | **Date**: 2026-09-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/324-user-task-integration/spec.md`

## Summary

Extend the existing baseline and volume user-task integration scenarios for #309/#310 using unchanged C88/C89/C810 embedded models. Use `SimpleUserTaskWithIncident` with `hasIncident=false` to obtain local `incident=1`, contrasted with process `incident=99`. Verify standalone and called-process scope behavior, independent effective display, and filtered paging. Keep C87 as an expected-unsupported check. No product changes, BPMN changes, new runner, or dependencies.

## Technical Context

**Language/Version**: Go 1.26; repository toolchain go1.26.2.
**Primary Dependencies**: Existing testing/testify, CLI subprocess harness, embedded model selector, and JSON evidence helpers. No additions.
**Storage**: Existing disposable cluster state and local integration evidence directories.
**Testing**: Existing integration build tag, focused test selectors, baseline and volume Make targets; deterministic seed-payload checks only where useful.
**Target Platform**: Existing local development/CI host with configured disposable Camunda profiles.
**Project Type**: CLI integration harness extension.
**Performance Goals**: Bounded readiness polling; baseline seeds only the cases needed; volume uses the existing count with at least three instances. No arbitrary latency benchmark or exhaustive live flag matrix.
**Constraints**: Default-local profile configuration; version gate before setup; immutable embedded models; no direct API setup; preserve old seed defaults and run markers; no t.Parallel for live scenarios sharing model IDs.
**Scale/Scope**: Three supported minors (8.8/8.9/8.10) plus unsupported 8.7. Two standalone model shapes, one parent-child shape, and a volume dataset.

## Constitution Check

| Principle | Before research | After design |
| --- | --- | --- |
| Operational proof | Require real task identities, variable scopes, readiness and explicit outcomes | Pass: readiness precedes assertions; unexecuted checks cannot count as verified |
| CLI-first/script-safe | Existing command grammar, output contracts and entry points | Pass: no public interface changes; JSON and keys assertions remain strict |
| Proportionate validation | Planning changes require lightweight checks only | Pass: focused baseline/volume runs during implementation; no blanket full suite or repeated checks just to commit |
| Documentation fidelity | Harness behavior belongs in integration documentation | Pass: update integration README only as needed; product help and generated CLI docs unchanged |
| Small repository-native changes | Reuse helpers and versioned models | Pass: three cohesive delivery units, no generic scenario engine or cross-version model rewriting |

No gate failures or exceptions. Read `specs/ralph-implementation-rules.md` before any later Ralph implementation, alongside the feature artifacts and integration-specific guidance.

## Project Structure

### Documentation (this feature)

```text
specs/324-user-task-integration/
  spec.md
  plan.md
  research.md
  data-model.md
  quickstart.md
  contracts/integration.md
  checklists/requirements.md
```

`tasks.md` is produced by the subsequent tasks workflow, not by this plan.

### Source Code (repository root)

```text
integration/cli/
  get_usertask_test.go             existing basic task cases
  get_usertask_variables_test.go   focused new variable scenarios and small seed helpers
  deploy_embed_run_test.go         existing seed/run/deploy helpers, minimal extension if needed
  volume_seed_test.go              existing volume start helper, preserve default callers
  volume_get_test.go               invoke focused filtered-volume scenarios
  all_commands_test.go             update user-task flag manifest
integration/README.md              focused usage/coverage notes
embedded/processdefinitions/      read-only C87/C88/C89/C810 models
Makefile                          reuse unchanged selectors and aggregate targets
```

**Structure Decision**: Keep new cohesive variable coverage beside the existing user-task tests. Reuse command execution, profile readiness, fixture selection, JSON decoding, and evidence. Add only scenario-specific helper functions and, if necessary, one small dataset struct; no new abstraction layer. Do not import the C89-only real-state scenario engine into generic get coverage.

## Phase 0: Research Decisions

See [research.md](research.md). Repository inspection resolves model availability, payload defaults, native filter encoding, result shapes, target selection, and dirty-cluster boundaries. Live backend results remain implementation validation, not claimed research evidence.

## Phase 1: Design

### Cohesive delivery units

1. **Standalone scope and display**: Extend seed payload handling without breaking existing defaults; seed ordinary and incident models; add readiness, operator, shadowing, display and unsupported-version assertions; update the manifest in the same unit.
2. **Called-process scope**: Deploy matching child then parent, start with explicit values, discover and verify child ownership through existing commands, and reuse scope assertions against the child PI. Do not add optional topology permutations unless existing assertions need them.
3. **Volume and integration documentation**: Seed multiple incident-model matches, prove filtered traversal/count/limit/empty contracts, connect the existing volume entry point, document focused invocation, and record actual version outcomes.

Each unit includes its relevant checks and evidence; do not split reading context, fixture self-tests, repeated validation, or completion bookkeeping into standalone implementation iterations.

### Seed and observation lifecycle

Select profile -> verify actual version -> select matching embedded definitions -> deploy only required definitions -> snapshot relevant preexisting volume matches if needed -> start owned instances -> discover task/child keys -> wait for expected process and effective variable scopes -> execute bounded assertions -> record results and existing retained/cleanup outcomes.

Use existing `runSelectorArgs` to start exact deployed definitions. Parent calls bind according to their unchanged model; verify the actual child's definition/version instead of assuming a deployment implies ownership. Preserve existing default-local authentication and selected tenant behavior.

### Scope and paging

Standalone queries use `--pi-key`; called-process queries use the discovered child's key. Read process values independently with existing PI variable display, then check task effective scope keys against the task element-instance key and owner PI key.

Volume queries use the actual deployed definition key and state, plus the local predicate. Do not assume re-deployment always provides an unused definition. Snapshot preexisting matching keys in that bounded scope, record them as preexisting, and wait for the expected union with newly discovered seeded keys. Assert every seeded key occurs once and JSON/keys/total agree on that full bounded set. Do not apply a process-only run marker as a task-local filter. Concurrent external changes invalidate a stable count comparison and must produce explicit evidence, not a silently adjusted expected count. Mutation/cleanup remains limited by existing suite rules; no new broad cleanup is introduced.

### Evidence and output contracts

Use existing `evidenceRecord`/command records and JSON files, with unique scenario labels per profile, model, case, and polling attempt. Ensure assertion failures mark their records failed. Include new volume records in the existing family report rather than only writing a disconnected side report. Preserve stdout/stderr separation and decode one envelope with EOF. See [contracts/integration.md](contracts/integration.md).

### Validation and documentation

Planning: document/link review and whitespace checks only. Implementation: targeted helper checks if helper semantics change, then focused integration baseline and volume on selected profiles. Existing get cases are the regression check for any shared seed change. Run inventory independently to verify manifest drift; making the aggregate execute extra inventory/runner phases is out of scope. Do not run full `make test` absent a demonstrated wider impact. Product source metadata and generated CLI documentation do not change.

## Complexity Tracking

No constitution violations or added architectural complexity require justification.
