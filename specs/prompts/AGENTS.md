# AGENTS.md

## Purpose
- This folder stores reusable manual prompts for repository maintenance tasks.
- Files in this folder are prompt templates, not Speckit feature artifacts.
- Do not treat this folder as source material for product behavior, feature
  requirements, release notes, README changes, implementation planning, or Ralph
  task generation unless the user explicitly asks to edit or use a prompt here.

## Scope Rules
- When mining `specs/` for implemented features, release highlights, changelog
  input, README updates, planning context, or Ralph context, ignore
  `specs/prompts/**`.
- Treat Speckit feature sources as directories that follow the repository's
  feature naming pattern, such as `specs/<number>-<feature>/`.
- If a prompt in this folder needs common behavior, put the shared rule here
  instead of duplicating it across prompt files.

## Prompt Style
- Keep prompts specific enough to run without extra interpretation.
- Prefer placeholders for caller-provided values, for example `<VERSION>` or
  `<BASE_COMMIT>`.
- State required inputs, source material, output expectations, and validation
  expectations.
- Keep reusable prompts free of issue-specific or release-specific facts unless
  they are placeholders.

## Safety
- Prompts in this folder may instruct an agent how to inspect the repository, but
  they should not ask for commits, pushes, releases, or destructive changes
  unless the user explicitly requests that workflow.
