# Ralph Memory

Feature: 275-native-c810-definitions
Started: 2026-08-17T14:08:19Z

## Codebase Patterns
- Protected baseline for #275 is commit `f2658425`; it exists locally and is an ancestor of branch `275-native-c810-definitions`.

## Decisions
- T001 completed as a verification-only setup work unit. No production file changes were needed.
- T002 created the eight `C810_*.bpmn` resources by mechanically copying the C89 production family and replacing `C89` with `C810` plus `8.9.0` with `8.10.0`; source `exporterVersion` values were preserved.

## Gotchas
- `specs/275-native-c810-definitions/progress.md` and `ralph-memory.md` began as untracked Ralph artifacts on iteration 1 and are included with the first coordinated commit.
- `go test ./embedded -count=1` currently reports `[no test files]`; the C810 structural test is planned for T009.

## Reusable Commands
- `git merge-base --is-ancestor f2658425 HEAD`
- `git diff --exit-code f2658425 -- ':(glob)embedded/processdefinitions/C87_*.bpmn' ':(glob)embedded/processdefinitions/C88_*.bpmn' ':(glob)embedded/processdefinitions/C89_*.bpmn'`
- `git diff --exit-code f2658425 -- integration Makefile`
- `for src in embedded/processdefinitions/C89_*.bpmn; do dest=${src/C89_/C810_}; diff -u <(perl -0pe 's/C89/C810/g; s/8\\.9\\.0/8.10.0/g' "$src") "$dest" >/dev/null; done`

## Do Not Repeat

## Current Handoff
- Next task: T003 in US1. Update the V810 expectation to `C810_` while retaining V87/V88/V89 mappings and unknown-version rejection in `toolx/fixture_compatibility_test.go`; continue within US1 only.
