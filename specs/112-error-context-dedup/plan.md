# Implementation Plan: Preserve Concise CLI Error Breadcrumbs

**Branch**: `112-error-context-dedup` | **Date**: 2026-04-17 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/spec.md)
**Input**: Feature specification from `/specs/112-error-context-dedup/spec.md`

## Summary

Audit duplicated CLI error composition across the repository and refactor the affected wrappers so user-facing failures keep the existing shared error-class prefix, preserve concise breadcrumb context, and render the root failure detail only once. The design keeps `c8volt/ferrors` as the shared classification and exit boundary, fixes duplication where commands and service helpers build nested `fmt.Errorf` chains, allows equivalent breadcrumb shortening when it reduces noise, and adds representative regression coverage for each affected duplication-pattern family rather than only the originally reported process-instance walk path.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing `c8volt/ferrors`, command helpers under `cmd/`, versioned services under `internal/services/...`, walker/waiter helpers under `internal/services/processinstance/`  
**Storage**: N/A  
**Testing**: `go test`, `make test`, command subprocess and in-process regression tests under `cmd/`, focused shared-error tests under `c8volt/ferrors`, service/helper tests under `internal/services/...`  
**Target Platform**: Cross-platform Go CLI execution in local development, shell automation, and CI  
**Project Type**: CLI  
**Performance Goals**: No user-visible regression in command startup or normal command execution; error rendering changes must remain negligible compared with existing command and HTTP work while producing shorter, easier-to-scan failure output  
**Constraints**: Preserve existing Cobra command surfaces, keep `ferrors` classification/normalization/exit-code behavior unchanged, preserve shared class prefixes such as `resource not found:`, fix duplication at repository-native wrapping seams instead of inventing a parallel error framework, allow breadcrumb shortening only when meaning stays equivalent, add representative regression coverage for each affected error-pattern family, and finish with `make test`  
**Scale/Scope**: Shared error rendering in [`c8volt/ferrors/errors.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go), command wrappers under `cmd/`, process-instance traversal and mutation helpers under [`internal/services/processinstance/walker/walker.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go) and versioned services under `internal/services/processinstance/v87`, `v88`, and `v89`, plus any additional repository error paths discovered during the audit that share the same duplication pattern and feed user-facing CLI output

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature changes how failures are composed, not whether commands verify success, and it preserves the observable failure class while making the message easier to understand.
- **CLI-First, Script-Safe Interfaces**: Pass. Existing commands, flags, and exit codes stay stable; the only intended user-visible change is cleaner failure text that keeps the same shared class prefixes and command-safe semantics.
- **Tests and Validation Are Mandatory**: Pass. The plan requires representative regression coverage for each affected duplication-pattern family plus final `make test`.
- **Documentation Matches User Behavior**: Pass with explicit minimal-doc assumption. The feature is primarily an internal message-composition cleanup; operator docs stay unchanged unless the audit finds user-facing examples or scripting guidance that rely on the duplicated strings.
- **Small, Compatible, Repository-Native Changes**: Pass. The design keeps `ferrors` as the shared boundary and refactors existing command/service wrappers rather than introducing a new rendering subsystem.

## Project Structure

### Documentation (this feature)

```text
specs/112-error-context-dedup/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-error-rendering.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── cmd_cli.go
├── get_processinstance.go
├── walk_processinstance.go
├── cancel_processinstance.go
├── delete_processinstance.go
├── get_processdefinition.go
├── get_resource.go
├── get_cluster_license.go
├── get_cluster_topology.go
├── *_test.go
└── shared command error helpers

c8volt/
├── ferrors/
│   ├── errors.go
│   └── errors_test.go
├── process/
│   └── client.go
└── resource/
    └── client.go

internal/services/
├── processinstance/
│   ├── walker/
│   │   ├── walker.go
│   │   └── walker_test.go
│   ├── waiter/
│   │   ├── waiter.go
│   │   └── waiter_test.go
│   ├── v87/
│   │   ├── service.go
│   │   └── service_test.go
│   ├── v88/
│   │   ├── service.go
│   │   └── service_test.go
│   └── v89/
│       ├── service.go
│       └── service_test.go
├── resource/
│   ├── v88/service.go
│   └── v89/service.go
└── cluster/common/
    ├── topology.go
    └── license.go
```

**Structure Decision**: Keep the work inside the existing single-project Go CLI layout. `c8volt/ferrors` remains the only shared classification and exit boundary; message deduplication happens in the command and service wrappers that currently compose repeated text, with focused tests added beside the affected command and helper seams.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/research.md).

- Confirm the lowest-risk boundary for this feature: preserve `ferrors` classification behavior and fix duplication in upstream wrappers rather than inside `Normalize`.
- Inventory the main duplication-pattern families in command, walker, service, and client wrappers that currently stack repeated key-specific or class-specific prose.
- Decide how breadcrumb shortening can stay semantically equivalent without creating ambiguous or drifting stage names.
- Confirm the representative regression anchors for each affected pattern family so coverage proves the cross-repo cleanup without requiring one test per command path.
- Confirm whether any user-facing docs mention exact duplicated failure strings; if not, keep documentation impact explicit but minimal.

### Refreshed Implementation Boundary

| Pattern family | Primary owner layer | Representative audited seams | Out-of-scope for this feature |
|--------|----------------------|------------------------------|-------------------------------|
| Process-instance lookup and traversal | `internal/services/processinstance/walker` plus versioned process-instance services | `walker.Ancestry`, `walker.Descendants`, `walker.Family`, `v88.Service.GetProcessInstance`, `v89.Service.GetProcessInstance`, the equivalent lookup and family wrappers in `v87` | Rewording successful logs or non-user-facing debug output |
| Process-instance mutation and wait follow-up | Versioned process-instance services plus `internal/services/processinstance/waiter` | `CancelProcessInstance`, `DeleteProcessInstance`, `GetProcessInstanceStateByKey`, and waiter-backed `waiting for ... failed` wrappers in `v87`, `v88`, and `v89` | Changing cancellation, delete, or wait semantics |
| Single-resource command fetch wrappers | `cmd/` command handlers | `cmd/get_processdefinition.go`, `cmd/get_resource.go`, `cmd/get_cluster_license.go`, `cmd/get_cluster_topology.go`, and `cmd/get_processinstance.go` fetch wrappers | Help text, flag parsing, or command routing unrelated to failure composition |
| Resource/client orchestration wrappers | `c8volt/resource` client code that feeds CLI commands | `c8volt/resource/client.go` delete/wait wrappers that can restate already-complete lower-layer failures | Internal-only wrappers that never surface through CLI rendering |

The implementation boundary for this feature is now locked to wrapper seams that shape user-facing CLI failures. The feature does not change `ferrors` normalization, class selection, exit-code mapping, or command success-path behavior.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/quickstart.md)
- [contracts/cli-error-rendering.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/112-error-context-dedup/contracts/cli-error-rendering.md)

- Keep `c8volt/ferrors/errors.go` unchanged as the classification and exit-code boundary; the rendered prefix remains the shared class error already selected by normalization.
- Keep the shared helper seams explicit: `ferrors.Normalize`, `ferrors.WrapClass`, `cmd.normalizeCommandError`, `cmd.normalizeBootstrapError`, and the `handle*Error` helpers may classify, select exit behavior, and supply logger context, but they must not become a second message-dedup layer.
- Introduce one repository-native error-composition contract for affected wrappers: wrappers may add stage breadcrumbs, may shorten breadcrumb wording when meaning stays equivalent, but must not restate the same root failure detail or class meaning multiple times.
- Treat duplication by pattern family, not by one command at a time: process-instance lookup/traversal, process-instance mutation/wait flows, single-resource fetch commands, and simple transport fetch wrappers are planned as separate regression families.
- Prefer the smallest local change at each seam: if the lower layer already produces the root failure detail, upper layers should contribute only stage context and let `ferrors` render the class prefix once.
- Keep docs updates conditional. If no user-facing docs assert the old duplicated text, tasking should record that documentation remains unchanged because behavior stayed semantically equivalent.

### Authoritative Rendering Boundary

| Concern | Required design contract |
|--------|---------------------------|
| Shared class prefix | Preserve the existing normalized prefix such as `resource not found:` or `unsupported capability:` |
| Shared helper boundary | Preserve current class-selection and exit-code behavior in `ferrors` and command/bootstrap helpers; dedup remains an upstream wrapper concern |
| Breadcrumb context | Keep concise stage breadcrumbs in order; shortening is allowed only when the same stage remains identifiable |
| Root failure detail | Render the root resource or operation failure once in the final message |
| Cross-class behavior | Apply the same prefix-preserving dedup rule to other shared error classes when the duplication pattern is otherwise the same |
| Test strategy | Add representative regression coverage for each affected duplication-pattern family, not just the original process-instance walk path |

### Representative Regression Anchors

| Pattern family | Required anchor files | What those anchors must continue proving |
|--------|------------------------|------------------------------------------|
| Process-instance lookup/traversal | `cmd/walk_test.go`, `internal/services/processinstance/walker/walker_test.go` | Ordered breadcrumbs remain visible while lookup/traversal root detail is rendered once |
| Process-instance mutation/wait | `cmd/cancel_test.go`, `cmd/delete_test.go`, `internal/services/processinstance/v87/service_test.go`, `internal/services/processinstance/v88/service_test.go` | Cancel/delete/wait failures keep the shared class prefix and avoid repeated key/failure prose |
| Single-resource fetch wrappers | `cmd/get_test.go` plus focused `get_*` command tests as needed | Top-level fetch wrappers stop restating already-complete lower-layer failures |
| Shared class/exit behavior | `c8volt/ferrors/errors_test.go`, `cmd/bootstrap_errors_test.go` | Dedup cleanup does not change normalization, classification, or exit-code behavior |

This boundary is the authoritative design target for task generation. Any implementation that changes classification, exit codes, or the meaning of an existing breadcrumb label must first update this plan and the feature spec.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Inventory duplicated CLI error paths by pattern family and mark the wrapper layer that should retain the root detail versus stage-only breadcrumbs.
2. Refactor the shared process-instance lookup, traversal, wait, and mutation seams so nested wrappers stop repeating keys, repeated `not found` wording, or fully formatted lower-layer failure sentences.
3. Sweep additional affected command and service wrappers outside the original process-instance path when they share the same duplication pattern and feed user-facing CLI output.
4. Add representative regression coverage for each affected duplication-pattern family, including not-found and non-not-found class-prefix preservation cases where relevant.
5. Update any user-facing docs only if the audit finds examples that rely on the old duplicated wording, then run focused Go tests and finish with `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design changes only message composition on failure paths and preserves command truthfulness about failure class and stage.
- **CLI-First, Script-Safe Interfaces**: Still passes. Command surfaces, exit behavior, and normalized prefixes remain stable while duplicated wrapped details are removed.
- **Tests and Validation Are Mandatory**: Still passes with representative regression coverage per duplication-pattern family plus final `make test`.
- **Documentation Matches User Behavior**: Still passes. Documentation changes remain conditional and must be made only if actual user-facing guidance references the duplicated wording.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design keeps the fix in current command/service helpers and `ferrors` usage patterns without adding new abstractions or dependencies.

## Final Verification Notes

- Focused validation for this feature should start with:
  - `go test ./c8volt/ferrors -count=1`
  - `go test ./internal/services/processinstance/... -count=1`
  - `go test ./cmd -count=1`
- Repository validation remains `make test`, which is required before the feature is considered complete.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
