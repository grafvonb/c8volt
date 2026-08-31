# Ralph Memory

Feature: 285-semantic-progress-milestones
Started: 2026-08-31T17:14:26Z

## Codebase Patterns
- Active feature artifacts live under `specs/285-semantic-progress-milestones/`; Ralph iterations for this feature must include `--implementation-context specs/ralph-implementation-rules.md`.
- Branch is `285-semantic-progress-milestones`; issue-backed commit subjects for this feature use Conventional Commits with scope `ralph` and end with `#285`.

## Decisions
- Phase 1 setup was completed as a tracking-only work unit before any source implementation.
- The canonical completion fact is `OpsCompletionProgress` / `CompletionProgress` with `submitted`, `confirmed`, and `failed` dispositions; `AffectedCount *int` distinguishes unavailable from trustworthy zero.
- Completion facts are mirrored mechanically through `c8volt/foptions` and `c8volt/ops`; command wording remains out of domain, services, and facades.

## Gotchas
- `progress.md` and `ralph-memory.md` started untracked in this worktree; include them with the coordinated task commit.
- `c8volt/foptions` previously had no test file; `c8volt/foptions/options_test.go` now covers service-to-facade progress callback mapping.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./internal/domain ./c8volt/foptions ./c8volt/ops ./toolx/logging -race -count=1`
- `go test ./internal/domain ./c8volt/foptions ./c8volt/ops ./internal/services -race -count=1`
- `go test ./cmd -run 'Progress|Activity' -race -count=1`
- `git diff --check`

## Do Not Repeat
- Do not begin User Story 1 until foundational tasks T004-T006 are completed and validated.

## Current Handoff
- Next iteration should continue Phase 2 at T004 by adding failing reporter-construction, output-policy, aggregate-invariant, and idempotent-close tests in `cmd/ops_semantic_progress_test.go`.
