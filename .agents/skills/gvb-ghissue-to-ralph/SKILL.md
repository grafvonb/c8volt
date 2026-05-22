---
name: gvb-ghissue-to-ralph
description: Create an issue-backed Speckit feature and drive it through clarify, plan, tasks, and Ralph launch using the repository's standard flow.
compatibility: Requires spec-kit project structure with .specify/ directory
metadata:
  author: local-wrapper
  wraps:
    - gvb-ghissue-to-speckit
    - speckit-clarify
    - speckit-plan
    - speckit-tasks
    - speckit-ralph-run
---

# GVB Issue To Ralph Flow

Use this skill when the user wants the repository's standard issue-driven delivery flow executed from a GitHub issue through ready-to-run implementation.

This wrapper standardizes the following sequence:

1. `gvb-ghissue-to-speckit`
2. `speckit-clarify`
3. `speckit-plan`
4. `speckit-tasks`
5. `speckit-ralph-run` or the equivalent `after_tasks` hook

The intended outcome is an issue-backed feature with traceable spec artifacts, completed planning artifacts, generated tasks, and a Ralph loop launch path that respects the repository's hook configuration.

## Inputs

- A GitHub issue URL such as `https://github.com/grafvonb/c8volt/issues/45`
- Optionally, extra instructions in the same request
- Optionally, Ralph launcher arguments such as `--max-iterations 5` or `--model gpt-5.5`

A bare issue URL is sufficient to begin.

## Required Behavior

1. Parse the GitHub issue URL from the user request.
2. Treat the issue number as the canonical workflow identifier for the entire run.
3. Run the `gvb-ghissue-to-speckit` workflow first and preserve all of its guarantees:
   - fetch issue title and body
   - enforce issue-number-based feature identity
   - verify shell-layer compatibility before feature creation
   - persist issue traceability into `spec.md`
   - ensure both `spec.md` and `checklists/requirements.md` exist before moving on
4. After specification, always continue immediately into `speckit-clarify` unless the user explicitly asked to skip clarification in the same `gvb-ghissue-to-ralph` request.
5. Treat clarification as a required gate before planning:
   - ask at most the allowed clarification questions
   - wait for the user to answer each clarification question
   - integrate accepted answers into `spec.md`
   - do not run `speckit-plan` or `speckit-tasks` until clarification has completed, all accepted answers have been written to `spec.md`, or the user explicitly instructs you to proceed without resolving remaining clarification questions
6. After clarification completes, run `speckit-plan`.
7. After planning completes, run `speckit-tasks`.
8. After tasks are generated, decide how Ralph should start:
   - always stop and ask the user for explicit confirmation before any Ralph launch path is allowed to continue
   - treat this confirmation as a budget check, not as a mere informational prompt
   - on macOS, always phrase the eventual launch as `speckit-ralph-run in Terminal.app`
   - if `.specify/extensions.yml` defines an enabled `after_tasks` hook targeting `speckit.ralph.run`, prefer that hook-driven path after the user confirms and do not manually invoke `speckit-ralph-run` a second time
   - otherwise invoke `speckit-ralph-run` explicitly only after the user confirms
9. When a hook-driven Ralph launch is available, surface that to the user clearly and ask whether they want to spend budget on starting Ralph now.
10. Carry the GitHub issue number through all downstream naming and traceability where relevant, including commit messages.

## Commit Rule

For any commit created as part of this workflow or by downstream skills acting on the generated feature, preserve Conventional Commits formatting and append `#<issue-number>` as the final token in the subject.

Examples:

- `feat(spec): add issue traceability #45`
- `chore(plan): record architecture decisions #45`
- `test(view): complete final polish #59`

## Ralph Launch Rule

This repository may already wire Ralph through `.specify/extensions.yml`.

When the `after_tasks` hook already points at `speckit.ralph.run`:

- always ask the user whether Ralph should run now before allowing the hook to proceed
- on macOS, require the launch to use Terminal.app visibly
- prefer the hook-managed flow
- do not manually duplicate the Ralph launch
- report that Ralph is available through the existing hook and that budget confirmation is required

When that hook is absent or disabled:

- always ask the user whether Ralph should run now before launching it
- on macOS, launch it as `speckit-ralph-run in Terminal.app`
- invoke `speckit-ralph-run` directly
- pass through any user-provided Ralph arguments unchanged when they are compatible with the launcher

## Clarification Gate

When this skill is invoked, `speckit-clarify` is mandatory immediately after `gvb-ghissue-to-speckit` creates or updates the spec.

Only skip clarification when the user explicitly says to skip clarification while invoking `gvb-ghissue-to-ralph`. General urgency, a bare issue URL, Ralph options, or a request to run the full flow are not skip signals.

Clarification is complete only after one of these conditions is true:

- `speckit-clarify` finds no critical ambiguities worth formal clarification
- the user has answered all clarification questions selected by `speckit-clarify`, and those answers have been integrated into `spec.md`
- the user explicitly stops the clarification loop or instructs the agent to proceed despite outstanding clarification items

Do not run `speckit-plan`, `speckit-tasks`, an `after_tasks` hook, or `speckit-ralph-run` before the clarification gate is complete.

## Execution Flow

1. Read the user request and locate the GitHub issue URL.
2. Extract any extra feature instructions and any Ralph runtime options.
3. Run the complete `gvb-ghissue-to-speckit` flow.
4. Confirm the created feature artifacts exist and that issue traceability is present in the spec.
5. Unless the current `gvb-ghissue-to-ralph` request explicitly says to skip clarification, run `speckit-clarify` against the active feature immediately after spec creation.
6. Complete the clarification loop before planning:
   - ask one clarification question at a time
   - wait for the user's answer
   - integrate each accepted answer into `spec.md`
   - stop only when `speckit-clarify` reports no critical ambiguities, all selected questions are answered, the question limit is reached, or the user explicitly stops or proceeds with remaining ambiguity
7. Run `speckit-plan` only after the clarification gate is complete.
8. Run `speckit-tasks` after planning.
9. Inspect `.specify/extensions.yml` for an enabled `after_tasks` hook targeting `speckit.ralph.run`.
10. Stop and ask the user whether they want to launch Ralph now, explicitly framing the question as a budget check.
11. If the user declines, stop after confirming tasks generation is complete and report the next command to run later.
12. If the user approves and the hook exists, present the hook-driven Ralph handoff and stop after confirming the launch path.
13. If the user approves and that hook does not exist, invoke `speckit-ralph-run`.
14. In the final handoff, restate:
   - the feature directory
   - the spec, plan, and tasks artifact paths
   - whether Ralph was hook-driven or manually launched
   - the issue-suffixed commit-message rule

## Notes

- This skill is intentionally opinionated for repositories that use GitHub issues as the primary feature entry point.
- `speckit-clarify` remains interactive by design. That is a feature, not a flaw. Do not try to bypass it unless the user explicitly chooses to skip it in the current `gvb-ghissue-to-ralph` request.
- Ralph launch is also an explicit pause point in this repository because the user wants to check budget before implementation starts.
- Prefer the repository's existing hook configuration over inventing a second orchestration path.
- Do not run `speckit-ralph-run` twice.
- If the issue-backed specification step fails, stop there and surface the exact blocker instead of attempting later stages.
- If planning fails, do not generate tasks.
- If task generation fails, do not attempt Ralph launch.
- If the user asks for a dry run or to stop before implementation, still run clarification unless they explicitly ask to skip it; then finish at the latest completed stage and report the next command to run.
