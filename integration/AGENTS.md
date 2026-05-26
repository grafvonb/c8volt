# AGENTS.md

## Purpose
- This folder contains real-cluster release integration validation assets.
- These assets are test harness material, not product design source material.
- Do not mine this folder for feature requirements, implementation plans,
  Speckit scope, Ralph stories, README claims, or user-facing examples unless
  the user explicitly asks to update the integration harness itself.

## Scope Rules
- Keep harness concepts out of product implementation unless a report finding is
  explicitly accepted as product work.
- Do not copy temporary fixture names, generated run IDs, report wording, or
  release-suite shortcuts into command help, documentation, or production code.
- Treat `integration/prompts/**` as operational prompts for release validation,
  not as repository requirements.
- Treat `integration/scripts/**` as executable validation tooling, not as a
  supported public API.

## Safety
- The suites run against real local Camunda clusters and intentionally exercise
  mutating commands in dirty, disposable environments.
- Existing process definitions, process instances, incidents, jobs, resources,
  and other local-cluster state may be modified, cancelled, deleted, resolved,
  repaired, or purged without extra confirmation when a suite is running.
- Preserve the default behavior that gates the run by target Camunda minor
  version before mutation.
- If a suite needs new coverage, add narrowly scoped cases and evidence capture
  instead of broad flag permutations. Broad destructive workflows are allowed
  when they reflect release validation of daily ops behavior.
