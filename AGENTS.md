# AGENTS.md

## Ralph PRD generation
- When generating or updating `scripts/ralph/prd.json`, split work into multiple small user stories.
- Each story must be feasible in one Ralph iteration.
- Prioritize stories by dependency and execution order.
- Keep each story narrowly scoped and independently verifiable.
- Write precise, minimal, testable acceptance criteria.
- Reuse existing project patterns and avoid introducing parallel structures.
- If `scripts/ralph/prd.json` already exists and the task is an update, prefer adding follow-up stories instead of rewriting completed stories, unless they are no longer valid.

## Git and GitHub branch rules
- For issue-based work, first check whether a matching local or remote branch already exists and reuse it when appropriate.
- If the issue already has a linked or existing branch, use that exact branch name.
- Do not invent a different branch name when an issue branch already exists.
- If no matching branch exists, create one using GitHub-style issue naming:
  - format: `<issue-padded>-<description>`
  - example: for issue #58 with the description "Review and refactor internal service cluster api implementation", use `058-review-and-refactor-internal-service-cluster-api-implementation`
- Use a three-digit zero-padded GitHub issue number prefix for new issue branches:
  - use `058-...`, not `58-...`
- If the existing matching branch does not use the required three-digit zero-padded prefix, stop and report that the branch format is incompatible with subsequent Spec Kit skills, which expect a `NNN-description` branch name.
- Keep `<description>` concise, lowercase, and hyphen-separated.
- Do not add extra prefixes such as `codex/` unless the user explicitly asks or the repository explicitly requires them.
- Do not create or switch to a different feature branch unless the user explicitly asks.

## Commit rules
- Commit messages must follow Conventional Commits format.
- Add a scope in parentheses after the type when a clear scope exists.
- Reference the GitHub issue in the subject when applicable, for example:
  - `feat(cli): add command #<issue>`
  - `fix(api): handle empty response #<issue>`
  - `refactor(service): simplify implementation #<issue>`
  - `test(module): add coverage for edge cases #<issue>`
  - `docs(readme): update usage examples #<issue>`
- Use lowercase by default, except where capitalization is required for correct names such as `README.md` or product/library names.
- Prefer small commits grouped by purpose.
- Do not use vague commit messages such as `update`, `fix stuff`, or `changes`.

## Validation
- Before committing, run:
  - `make test`

## Project conventions
- Prefer existing project patterns over introducing new structural styles.
- For refactoring work, preserve externally observable behavior unless the issue explicitly asks for behavioral change.
- Favor incremental refactors with verification over broad rewrites.
- When changing generated or generated-adjacent artifacts, update the source and regenerate rather than editing derived output by hand when the repository already provides a generation path.

## Testing conventions
- Add or update tests alongside refactoring and bug fixes.
- Prefer targeted tests near the changed package, then run the broader repository test suite.
- For refactors, ensure tests verify preserved behavior, not just new internal structure.

## Documentation conventions
- User-facing documentation and examples should stay in sync with behavior changes.
- When changing user-facing commands, APIs, or workflows, update the relevant documentation in the same change.

## Technology baseline
- Follow the repository's current toolchain, dependency, and framework conventions.
- Prefer the libraries and frameworks already established in the project unless the user explicitly asks for a change.

## Issue-specific guidance
- Issue-specific requirements belong in the GitHub issue, the Spec Kit feature artifacts, and the PRD.
- Do not add changing issue-specific details to this file unless they become stable repository rules.