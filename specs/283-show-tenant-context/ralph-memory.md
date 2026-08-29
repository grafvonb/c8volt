# Ralph Memory

Feature: 283-show-tenant-context
Started: 2026-08-29T12:20:03Z

## Codebase Patterns
- Progress tracking for this feature lives in `specs/283-show-tenant-context/progress.md` and now contains artifact links, work-unit status, validation results, and codebase-pattern sections before iteration entries.

## Decisions
- Phase 1 setup was treated as the first work unit because T001 was the first incomplete task and Phase 2 depends on it.

## Gotchas
- Follow `specs/ralph-implementation-rules.md` in addition to this feature's artifacts; it is binding for Ralph iterations.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `git diff --check -- specs/283-show-tenant-context/progress.md`

## Do Not Repeat
- Do not start Phase 2 tasks until T001 is committed with `tasks.md`, `ralph-memory.md`, and `progress.md`.

## Current Handoff
- Next iteration should start at Phase 2 foundational test task T002 in `tasks.md`, after reading this memory file first.
