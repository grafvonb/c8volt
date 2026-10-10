# Ralph Memory

Feature: 333-fix-smoke-test-cleanup
Started: 2026-10-10T09:21:28Z

## Codebase Patterns

- Smoke-test deployment returns all three fixture definitions; preserve response order and select the runnable main fixture only by its exact BPMN ID.
- `processdefinition.FindUnrelatedProcessInstancesForDefinition` and single-key `DeleteProcessDefinitions` remain the safety and mutation authorities; collection orchestration belongs in `internal/services/ops/smoke_test_cleanup.go`.
- Existing singular deployment and cleanup fields are compatibility projections of the main fixture; `DefinitionResults` and `DefinitionSummary` are authoritative for new executions.
- Command rendering consumes deployment and cleanup collections directly; a nil `DefinitionSummary` is the only legacy singleton fallback and renderers never perform discovery.
- Default human output stays aggregate-first, while verbose output and Markdown audit tables enumerate every returned definition identity and disposition.

## Decisions

- Definition keys are stably deduplicated. Conflicting nonempty evidence and missing keys remain reportable scope errors and never authorize name-based deletion.
- Collection cleanup is sequential for deterministic per-key attribution. Default execution continues after independent errors; fail-fast and cancellation leave later entries explicitly unattempted.
- Aggregate status precedence is failed/unattempted, retained/skipped, submitted, then confirmed. Accepted no-wait responses are submitted, never confirmed.
- CLI cleanup wording derives definition counts and mixed-state evidence from `DefinitionSummary`; JSON remains the shared envelope over the public model.

## Gotchas

- Command fake-server request sequences now include eligibility and deletion calls for the two dependency definitions before the main definition.
- Deployment tenant fallback remains on the legacy main projection; collection entries keep unknown tenant evidence unknown.

## Reusable Commands

- `go test ./internal/services/ops -run 'Test.*SmokeTest' -count=1`
- `go test ./c8volt/ops -run 'Test.*SmokeTest' -count=1`
- `go test ./cmd -run 'Test.*SmokeTest' -count=1`
- `go test ./internal/services/ops ./c8volt/ops ./cmd -count=1`

## Do Not Repeat

- Do not restore BPMN-name fallback when the main deployment key is missing; exact deployment identity is required before starting instances.
- Do not batch the three definition deletions through one aggregate call because accepted/wait-error evidence must remain attributable per key.

## Current Handoff
- Start US2 with T003: complete fault-path evidence and actionable mixed-result reporting across service, facade, and command tests without rebuilding the collection loop.
