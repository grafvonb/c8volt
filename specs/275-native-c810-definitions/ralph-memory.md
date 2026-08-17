# Ralph Memory

Feature: 275-native-c810-definitions
Started: 2026-08-17T14:08:19Z

## Codebase Patterns
- Protected baseline for #275 is commit `f2658425`; it exists locally and is an ancestor of branch `275-native-c810-definitions`.

## Decisions
- T001 completed as a verification-only setup work unit. No production file changes were needed.

## Gotchas
- `specs/275-native-c810-definitions/progress.md` and `ralph-memory.md` began as untracked Ralph artifacts on iteration 1 and are included with the first coordinated commit.

## Reusable Commands
- `git merge-base --is-ancestor f2658425 HEAD`
- `git diff --exit-code f2658425 -- ':(glob)embedded/processdefinitions/C87_*.bpmn' ':(glob)embedded/processdefinitions/C88_*.bpmn' ':(glob)embedded/processdefinitions/C89_*.bpmn'`
- `git diff --exit-code f2658425 -- integration Makefile`

## Do Not Repeat

## Current Handoff
- Next task: T002. Create the eight `embedded/processdefinitions/C810_*.bpmn` files as C89-derived resources, changing only C89-to-C810 identity references and `8.9.0` to `8.10.0`, preserving exporterVersion and leaving C87/C88/C89 files untouched.
