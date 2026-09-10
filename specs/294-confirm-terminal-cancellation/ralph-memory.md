# Ralph Memory

Feature: 294-confirm-terminal-cancellation
Started: 2026-09-10T06:42:27Z

## Codebase Patterns

- Baseline validation uses the five targeted commands in `quickstart.md`; passing output alone is insufficient when a package reports `[no tests to run]`.

## Decisions

## Gotchas

- The current versioned test filter selects no tests in v89 and v810; T002 must record the actual runnable prefixes before story tests are added.

## Reusable Commands

- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- Targeted baseline commands are recorded in `quickstart.md`.
- `make test` is the required pre-commit gate and passed for iteration 1.

## Do Not Repeat

## Current Handoff
- Complete T002 by mapping contract rows A–I to existing versioned and cleanup test seams, including actual runnable prefixes and bounded polling configuration.
