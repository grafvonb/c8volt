# PRD: Add Resource Get Command By Id

## Overview

Add a repository-native `c8volt get resource --id <id>` command that exposes the existing internal resource lookup capability through the public `c8volt` facade, returns the normal single-resource details/object view, preserves current `get` command conventions, and ships with targeted regression coverage plus updated generated CLI documentation.

## Goals

- Expose single-resource lookup through the existing `c8volt get` command family.
- Reuse the current internal resource service capability introduced by issue `#71` instead of adding a parallel operator workflow.
- Keep successful output in the normal single-resource details/object form rather than raw resource content.
- Preserve repository-native validation, error handling, and exit behavior used by other `get` commands.
- Update help text and generated CLI documentation in the same change.
- Finish with targeted tests and repository-required validation, including `make test`.

## User Stories

### US-001: Expose resource lookup through the public facade
**Description:** Add the missing public `c8volt/resource` capability so CLI code can retrieve one resource by id through the existing facade boundary.

**Acceptance Criteria:**
- `c8volt/resource/api.go` defines a single-resource retrieval method for one resource id.
- `c8volt/resource/client.go` maps the internal domain `Resource` to a public `c8volt/resource` model without introducing raw-content fields.
- `c8volt/contract.go` includes the new resource capability so `NewCli(cmd)` can expose it through the existing aggregate API.

### US-002: Deliver the MVP `get resource --id` command
**Description:** Add a working `c8volt get resource --id <id>` command that retrieves one resource and renders the normal single-resource details/object view.

**Acceptance Criteria:**
- `cmd/get_resource.go` registers a `resource` subcommand under `get`.
- Running `c8volt get resource --id <id>` with a valid resource id returns one resource in the expected details/object output form.
- Running the command with an unknown resource id preserves clear non-success behavior instead of rendering a successful empty result.

### US-003: Enforce repository-native validation and error semantics
**Description:** Make the new command match existing `get` command conventions for required-flag validation, malformed-success handling, and failure mapping.

**Acceptance Criteria:**
- The command fails validation before any backend lookup when `--id` is missing, empty, or whitespace-only.
- A backend `200 OK` response without a resource payload is treated as a malformed-response error rather than empty success.
- Existing error and exit handling continue to flow through the repository’s standard command-path behavior.

### US-004: Keep rendering and CLI integration consistent
**Description:** Integrate the new resource lookup path into the existing CLI rendering and command-tree behavior without changing unrelated `get` workflows.

**Acceptance Criteria:**
- Resource output uses a dedicated single-resource renderer aligned with existing one-item `get` views.
- JSON and normal detail output behavior remain consistent with the current rendering patterns used by other `get` commands.
- Adding the resource command does not alter existing `get` subcommand behavior outside the new resource lookup path.

### US-005: Make the workflow discoverable through help and docs
**Description:** Update help text and generated documentation so operators can find and use the new command without reading source code.

**Acceptance Criteria:**
- `c8volt get --help` exposes the new `resource` subcommand.
- `c8volt get resource --help` describes the required `--id` flag and the command’s single-resource purpose.
- Generated CLI docs under `docs/cli/` include the new command and match the shipped help text.

### US-006: Prove completion with focused and repository-wide validation
**Description:** Add the regression coverage and validation steps needed to ship the feature confidently and preserve current service semantics.

**Acceptance Criteria:**
- Command tests cover success, missing-id validation, and at least one failing lookup path.
- Versioned resource service tests cover single-resource retrieval and malformed-success payload handling for supported versions.
- Targeted Go tests pass for the command, facade, and resource services.
- `make test` passes.

## Functional Requirements

- FR-001: The CLI MUST provide a `c8volt get resource --id <id>` workflow for retrieving a single resource by id.
- FR-002: The command MUST require an explicit `--id` value before any lookup is attempted.
- FR-003: Successful lookup MUST return the normal single-resource details/object output rather than raw resource content.
- FR-004: The implementation MUST reuse the existing internal resource retrieval capability through the public `c8volt/resource` facade.
- FR-005: Not-found and lookup-failure outcomes MUST preserve clear user-facing error behavior and a non-success exit result.
- FR-006: A success HTTP status without the expected resource payload MUST be treated as a malformed-response error.
- FR-007: The new command MUST preserve established `c8volt get` command conventions for structure, validation, output, and exit behavior.
- FR-008: The implementation MUST avoid package renames, layout changes, new dependencies, and raw-content export behavior.
- FR-009: Help text and generated CLI docs MUST be updated in the same change because the feature is user-visible.
- FR-010: Automated validation MUST cover command behavior, facade wiring, versioned service response handling, and final repository regression checks.

## Non-Goals

- Adding raw resource content retrieval or export behavior.
- Introducing a new top-level resource command hierarchy outside `c8volt get`.
- Redesigning the shared render-mode system for unrelated commands.
- Changing package layout, adding dependencies, or bypassing the existing `c8volt` facade layer.
- Broadening the feature into resource list, search, batch retrieval, or other unrelated resource workflows.

## Implementation Notes

- Use the existing layering `cmd -> c8volt facade -> internal/services/resource` as the canonical implementation path.
- Extend `c8volt/resource/api.go`, `c8volt/resource/client.go`, `c8volt/resource/model.go`, and `c8volt/contract.go` rather than calling internal services directly from command code.
- Add the new command in `cmd/get_resource.go` and keep it aligned with current Cobra patterns used by other `get` subcommands.
- Add resource-specific rendering in `cmd/cmd_views_get.go` instead of inventing a parallel view system.
- Preserve malformed-success behavior by keeping the current `internal/services/common.RequirePayload` semantics already used in `internal/services/resource/v87/service.go` and `internal/services/resource/v88/service.go`.
- Keep the feature scoped to metadata/details lookup only; raw resource content remains a follow-up concern if ever needed.
- Update command help first, then regenerate `docs/cli/` with `make docs` rather than hand-editing generated pages.
- Follow repository guidance from `AGENTS.md`, including adjacent test updates and final `make test`.

## Validation

- Update `cmd/get_test.go` with resource-command success, validation, and help-output coverage.
- Add or update regression coverage in `c8volt/resource/client_test.go`.
- Add or update single-resource retrieval and malformed-success tests in `internal/services/resource/v87/service_test.go`.
- Add or update single-resource retrieval and malformed-success tests in `internal/services/resource/v88/service_test.go`.
- Run `go test ./cmd -run 'TestGet.*Resource' -count=1`.
- Run `go test ./c8volt/resource ./internal/services/resource/... -count=1`.
- Run `make docs`.
- Run `make test`.

## Traceability

- GitHub Issue: #73
- GitHub URL: https://github.com/grafvonb/c8volt/issues/73
- GitHub Title: Add `c8volt get resource --id` command
- Feature Name: 73-get-resource-id
- Feature Directory: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id
- Spec Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/spec.md
- Plan Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/plan.md
- Tasks Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/tasks.md
- PRD Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/tasks/prd-73-get-resource-id.md
- Source Status: derived from Speckit artifacts

## Assumptions / Open Questions

- The final single-resource output should follow the repository’s existing detail/object rendering conventions rather than introducing a new custom presentation format.
- `README.md` may or may not need changes depending on whether current user-facing usage guidance already covers the `get` workflow sufficiently, but generated CLI docs are mandatory because the feature is user-visible.
- The supported versioned resource services from issue `#71` remain the canonical implementation path and do not require generated-client regeneration for this feature.
