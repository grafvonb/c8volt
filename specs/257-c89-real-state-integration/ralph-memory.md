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

- Phase 1, Phase 2, and User Story 1 are complete and committed through this iteration.
- `integration-cli-real-state-jobs` and `integration-cli-real-state-incidents` are implemented and passed against `kind-camunda-platform-local-c89` on Camunda 8.9.9.
- Jobs use `C89_SimpleServiceTask.bpmn`, which currently yields `CREATED` execution-listener jobs. Retries can be confirmed, fail/no-wait can be submitted, and timeout can be planned in dry-run. Confirmed `update job --timeout` still needs an activated-job setup command or direct Camunda activation fallback and is recorded as a command proposal.
- Incidents are created by failing a suite-owned job with retries `0`; `get incident` observes the related `jobKey`, `get job --key` observes the failed job, and `ops repair incident --dry-run` reports related job counts.
- Next iteration should start User Story 2 at T026/T027 and keep the remaining real-state targets reserved until their matching family tests exist.
