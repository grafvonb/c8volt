# PRD: Add Process Definition XML Command

## Traceability

- Feature name: `69-process-definition-xml`
- Source status: Derived from Spec Kit artifacts
- Spec: [specs/69-process-definition-xml/spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/spec.md)
- Plan: [specs/69-process-definition-xml/plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/plan.md)
- Tasks: [specs/69-process-definition-xml/tasks.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/tasks.md)

## Problem Statement

`c8volt get process-definition` currently supports summary-style retrieval of deployed process definitions, but it does not expose a user-facing workflow for retrieving the BPMN XML payload for a single definition. Operators who need the exact XML for inspection, reuse, or redirection into a file do not have a script-safe command path for that outcome, even though the underlying versioned processdefinition services already support XML retrieval.

## Goal

Add a repository-native XML retrieval mode to `c8volt get process-definition` that returns the raw process definition XML for one explicitly selected definition, preserves existing non-XML behavior, remains safe for shell redirection, and ships with command-level validation, documentation updates, and repository-standard automated verification.

## Operator Stories

### Story 1: Retrieve One Definition as XML

As an operator, I want to retrieve the BPMN XML for one deployed process definition so that I can inspect or reuse the exact definition content without manual API work.

### Story 2: Use XML Output in Shell Workflows

As an operator, I want the XML output to be safe for stdout redirection so that I can save it directly into a `.bpmn` file or pipe it into other tools without cleanup.

### Story 3: Discover the Correct XML Workflow

As an operator, I want the command help and generated docs to explain the XML option clearly so that I can use it without reading the source code.

## Scope

### In Scope

- Extend the existing `get process-definition` command with an XML retrieval mode for a single process definition.
- Reuse the existing versioned `GetProcessDefinitionXML` service capability through the public `c8volt/process` facade.
- Require explicit single-definition selection for XML retrieval.
- Emit raw XML directly to stdout on success.
- Reject conflicting flag combinations that do not fit the XML workflow.
- Preserve all current non-XML `get process-definition` list and detail behavior.
- Add or update command-level, facade-level, and adjacent service regression coverage where needed.
- Update help text first and regenerate CLI docs via `make docs`.
- Update README or docs index examples only if the new user-facing workflow needs explicit examples to stay in sync.

### Non-Goals

- Introduce a new top-level export or process-definition command family.
- Redesign the generic rendering system to add a broad XML output mode for unrelated commands.
- Change package layout, add dependencies, or introduce a parallel service abstraction.
- Expand the feature into bulk export, multi-definition XML retrieval, or broader process-definition packaging workflows.
- Change existing non-XML search, latest, stats, or detail semantics beyond what is necessary to integrate the XML mode safely.

## Current Pain Points

- Operators cannot retrieve process definition XML through the existing CLI even though the service layer supports it.
- There is no documented, redirect-safe operator workflow for `> example.bpmn`.
- The CLI currently lacks a clear contract for how XML mode should interact with existing render flags and list-style filters.
- Without explicit validation and docs, a new XML mode would risk confusing precedence rules or script-breaking output behavior.

## Target Outcome

- `c8volt get process-definition` supports an explicit XML retrieval mode for one definition.
- Successful XML retrieval writes only the XML payload to stdout.
- Failure paths keep the existing repository-native error and exit behavior.
- Non-XML command behavior remains unchanged.
- Help text and generated docs make the new workflow discoverable and clear.
- Reviewers can verify completion through focused command tests, adjacent regression checks, regenerated docs, and `make test`.

## Requirements

### Functional Requirements

- **AC-001**: The CLI must provide a user-visible XML retrieval workflow under the existing `c8volt get process-definition` command path.
- **AC-002**: XML retrieval must require explicit selection of a single process definition.
- **AC-003**: Successful XML retrieval must return the raw process definition XML through stdout without summary decorations or JSON wrapping.
- **AC-004**: Retrieval failures must preserve clear user-facing error reporting and a non-success exit result.
- **AC-005**: XML retrieval must remain safe for normal shell redirection into a file.
- **AC-006**: XML mode must reject incompatible list-oriented or render-oriented flag combinations instead of silently choosing precedence.
- **AC-007**: Existing non-XML `get process-definition` behaviors must remain unchanged when XML mode is not requested.
- **AC-008**: The public process facade must expose the XML retrieval path using the existing service-layer structure.
- **AC-009**: Help text and generated CLI docs must describe the XML workflow and its usage constraints.
- **AC-010**: The final implementation must define and execute automated validation for command behavior, public facade wiring, and repository-wide regression checks.

### Invariants

- No new dependency introduction.
- No package renames or package layout changes.
- No new command hierarchy outside the current `get process-definition` area.
- No behavioral regression in existing non-XML process-definition retrieval.
- No mixed output mode that makes stdout unsafe for automation or redirection.

## Implementation Notes

- Reuse the current Go CLI, Cobra, and public facade patterns already established in the repository.
- Route XML retrieval through `c8volt/process` rather than calling internal services directly from the command.
- Keep XML rendering narrowly scoped to this command path instead of generalizing the render-mode system unless a broader repository need emerges.
- Follow the repository convention that user-facing CLI docs under `docs/cli/` are generated from command metadata through `make docs`.
- Keep validation centered on `cmd/get_test.go`, affected process facade coverage, versioned processdefinition service tests, and `make test`.
- Align user-facing command wording with the actual supported CLI flag surface; the issue/spec language uses `--id`, but the current command contract already uses `--key` for single-definition lookup and that constraint should remain explicit unless separately changed.

## Execution Shaping

### Work Package 1: Expose the Shared XML Contract

- Extend the public process facade so the CLI can access XML retrieval through the existing service boundary.
- Confirm the shared XML path remains consistent with the current versioned processdefinition services.
- Add foundational coverage for facade wiring and validation boundaries before refining the command behavior.

### Work Package 2: Deliver the MVP XML Retrieval Path

- Add the XML execution path to `get process-definition`.
- Return only raw XML on success.
- Preserve normal failure handling and current non-XML behavior.

### Work Package 3: Make the Output Script-Safe

- Tighten XML flag validation around single-definition use and conflicting render or list flags.
- Prove that stdout redirection yields a reusable BPMN file without cleanup.
- Preserve list/detail flows for non-XML users.

### Work Package 4: Ship Documentation and Proof

- Update help text to reflect the final XML contract.
- Regenerate CLI docs and update README examples only where needed.
- Run targeted tests, doc generation, and `make test`.

## Acceptance Criteria by Story

### Story 1 Acceptance Criteria

- Operators can retrieve the XML payload for one process definition through the existing command family.
- Success produces only the XML payload on stdout.
- Retrieval failures remain visible through the repository’s normal error and exit behavior.

### Story 2 Acceptance Criteria

- Redirecting XML output to a file yields a usable BPMN file without summary lines or JSON wrappers.
- XML mode rejects incompatible flags with clear validation behavior.
- Existing non-XML command behavior remains unchanged after XML support is added.

### Story 3 Acceptance Criteria

- `c8volt get process-definition --help` describes the XML option and its constraints.
- Generated CLI docs reflect the XML workflow and its user-facing purpose.
- Any updated examples remain consistent with shipped command behavior.

## Validation

- Update `cmd/get_test.go`.
- Update process facade regression coverage in `c8volt/process/` as needed.
- Update `internal/services/processdefinition/v87/service_test.go`.
- Update `internal/services/processdefinition/v88/service_test.go`.
- Run `go test ./cmd -run 'TestGet.*ProcessDefinition.*XML|TestGet.*ProcessDefinition.*Help' -count=1`.
- Run `go test ./c8volt/process ./internal/services/processdefinition/... -count=1`.
- Run `make docs`.
- Run `make test`.

## Assumptions / Open Questions

- The issue and upstream spec use `--id`, while the current command surface uses `--key` for single-definition lookup. This PRD assumes the feature will align with the existing `--key` contract unless a deliberate CLI rename is separately approved.
- README updates may not be necessary if no existing process-definition retrieval examples need revision, but generated CLI docs are mandatory because the feature is user-visible.
- The versioned XML service methods are assumed to remain the canonical implementation path for supported Camunda versions rather than requiring generated-client or API regeneration.

## Source Availability

- Available: `spec.md`, `plan.md`, `tasks.md`
- Also referenced for context: `research.md`, `data-model.md`, `quickstart.md`
- Absent: none of the primary Spec Kit artifacts
