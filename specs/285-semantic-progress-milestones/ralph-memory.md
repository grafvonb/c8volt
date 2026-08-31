# Ralph Memory

Feature: 285-semantic-progress-milestones
Started: 2026-08-31T17:14:26Z

## Codebase Patterns
- Active feature artifacts live under `specs/285-semantic-progress-milestones/`; Ralph iterations for this feature must include `--implementation-context specs/ralph-implementation-rules.md`.
- Branch is `285-semantic-progress-milestones`; issue-backed commit subjects for this feature use Conventional Commits with scope `ralph` and end with `#285`.

## Decisions
- Phase 1 setup was completed as a tracking-only work unit before any source implementation.

## Gotchas
- `progress.md` and `ralph-memory.md` started untracked in this worktree; include them with the coordinated task commit.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./internal/domain ./c8volt/foptions ./c8volt/ops ./toolx/logging -race -count=1`
- `go test ./cmd -run 'Progress|Activity' -race -count=1`
- `git diff --check`

## Do Not Repeat
- Do not begin User Story 1 until foundational tasks T002-T006 are completed and validated.

## Current Handoff
- Next iteration should start Phase 2 with T002, adding the foundational failing completion-fact mapping tests before implementation.
