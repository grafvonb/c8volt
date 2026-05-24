# Progress: Tenant Scope For Discovery And Explicit Admin Keys

## Implementation Context

- Every Ralph iteration MUST read and apply `specs/ralph-implementation-rules.md` before selecting or implementing a work unit.
- Ralph launch instructions MUST include `--implementation-context specs/ralph-implementation-rules.md`.
- Commit subjects for this feature MUST use Conventional Commits format and append `#156` as the final token.

## Codebase Patterns

- Commands own Cobra flags, local validation, explicit search-vs-keyed mode decisions, and output dispatch.
- Direct keys and stdin keys are parsed through existing command helpers such as `readKeysIfDash`, `mergeAndValidateKeys`, and `validateKeys`.
- Public facade options flow through `c8volt/foptions` into `internal/services.CallOption`; do not pass internal options directly from commands.
- Version-specific Camunda behavior belongs in internal service packages under v87, v88, and v89, with unsupported versions returning explicit errors.
- Generated CLI docs must be refreshed with `make docs-content` after command help or metadata changes.

## Work Log

- 2026-05-24: Created issue-backed Speckit feature for GitHub issue #156.
- 2026-05-24: Clarification gate completed with no critical formal questions.
- 2026-05-24: Architecture grounding reused existing architecture memory; no architecture refresh needed.
