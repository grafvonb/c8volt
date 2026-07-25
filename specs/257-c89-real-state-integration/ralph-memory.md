# Ralph Memory: C89 Real-State Semantic Integration Coverage

## Durable Decisions

- Real-state targets are a third integration layer, separate from 255 baseline targets and 256 volume targets.
- The current foundation targets Camunda 8.9 only; proposal records and evidence should keep version fields explicit for later minor-release extension.
- Real-state Make targets are reserved as stable entry points before family scenarios exist. Reserved targets must fail with a clear not-implemented message rather than accidentally passing because no Go test matched.
- Real-state setup must prefer c8volt commands and existing embedded BPMN fixtures. Direct Camunda API setup and missing fixture behavior must become proposal evidence.
- Evidence files stay under the integration workdir and outside `docs/`.

## Codebase Patterns

- `integration/cli/volume_harness_test.go` is the closest target catalog pattern.
- `integration/cli/volume_evidence_test.go` is the closest family report and evidence writer pattern.
- `integration/cli/volume_assertions_test.go` already owns reusable JSON, keys-only, and machine stdout cleanliness checks; real-state helpers should wrap these rather than duplicate logic.
- `integration/cli/deploy_embed_run_test.go` and `integration/cli/volume_seed_test.go` already provide embedded fixture deployment and process-instance start helpers.
- `integration/cli/harness_test.go` owns default-local config selection, explicit `--config` rejection, proposal registration, JSON evidence writing, and run summary content.

## Current Handoff

- Phase 1 and Phase 2 scaffolding are complete and committed in this iteration.
- Next iteration should start at T014/T015 under User Story 1 and replace the reserved `integration-cli-real-state-jobs` and `integration-cli-real-state-incidents` Make targets only when their matching family tests exist.
- Do not run reserved real-state Make targets as validation until their family tests are implemented; use the helper validation command in `quickstart.md` instead.
