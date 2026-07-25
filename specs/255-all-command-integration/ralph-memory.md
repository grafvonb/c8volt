# Ralph Memory

Feature: 255-all-command-integration
Started: 2026-07-25T06:27:38Z

## Codebase Patterns

- `integration/` is harness material, not product implementation or generated public docs. Keep all all-command suite code under `integration/cli/` and reusable suite context under `integration/assets/`.
- The first Go integration package uses `//go:build integration` and package `cli_test` so normal `go test ./...` does not run real-cluster tests.
- The suite builds the root CLI binary once in `TestMain`, then validates behavior through subprocess execution.
- Evidence defaults to a temp workdir and can be stabilized with `C8VOLT_IT_WORKDIR`; command logs live under `logs/` and JSON evidence under top-level files in that workdir.
- Subprocesses run from the evidence workdir so Cobra's normal config search cannot accidentally pick up a repo-local `config.yaml` before the operator's default local config.
- `config_test.go` owns the real profile readiness and read-only smoke gates; selected profile evidence is written to `profiles.json`, and smoke command evidence is written to `readonly-smoke.json`.
- `deploy_embed_run_test.go` owns seeded data setup through `embed list`, `embed deploy`, `run process-instance`, and direct `get process-instance` verification.
- `get_test.go` owns dirty-cluster-safe assertion helpers; seeded checks assert required keys are present among unrelated results, never exact global counts.
- `suite.marker` is the in-memory run marker used by tests and persisted to `run.json`, so later seeded-data commands use the same value recorded in evidence.
- US4 family slices use `runFamilyCoverageScenarios` to execute canonical and alias `--help` subprocesses for every command path in the family, assert manifest-declared flags appear in help, and write `coverage-<family>.json` evidence with manifest entries, subprocess records, destructive paths, and output modes.
- US5 proposal reports are aggregate harness evidence: `TestProposalReports` writes `proposals-command.json` and `proposals-embedded-bpmn.json` from per-family append functions in `walk_test.go`, `update_test.go`, `ops_analyse_test.go`, and `ops_execute_test.go`.
- US6 example validation lives in `integration/cli/examples_test.go`: it extracts live `--help` examples and generated `docs/cli/c8volt*.md` examples, substitutes suite-owned seeded placeholders, executes safe/read-only and dry-run disposable examples, and writes `examples.json`.
- `TestMain` now writes `summary.md` after `m.Run()` so stable `C8VOLT_IT_WORKDIR` runs include a human-readable evidence index alongside `run.json`.
- `integration/cli/package_test.go` is intentionally untagged and empty so `go test ./integration/cli -count=1` is harmless without the `integration` tag.

## Decisions

- Use `c8volt capabilities --json` from the built binary as the live command inventory oracle.
- Keep the initial MVP non-destructive: it only builds the binary, runs `capabilities --json`, validates the 55-command manifest, and writes `inventory.json`/`coverage.json`.
- Keep the coverage manifest explicit for command paths, aliases, flags, output modes, owners, and destructive classification so future command or flag drift fails loudly.
- Use `C8VOLT_IT_PROFILES` as an explicit comma-separated profile selection; when it is empty, fall back to the active profile from the default local config for real profile gates.
- Read only default local config metadata from `$XDG_CONFIG_HOME/c8volt/config.yaml`, `$HOME/.config/c8volt/config.yaml`, or `$HOME/.c8volt/config.yaml`; never pass that path as `--config`.
- Seeded data uses version-matched embedded `SimpleUserTask` BPMN fixtures because they leave process instances observable after start.
- Retain seeded process instances after US3 and record cleanup tracking as `retained`; later command-family slices may mutate or clean up their own targets.
- The final all-command matrix and Go suite README describe `integration/cli/` as the all-command suite source of truth while keeping existing shell suites separate.

## Gotchas

- The `capabilities` command currently exposes root persistent flags in its own flag list; the manifest intentionally includes them.
- The first compile gate should be run with `-tags=integration`; without the tag, the package has no active Go files.
- `integration/README.md` still documents existing shell suites that use `C8VOLT_IT_CONFIG`; the new Go suite section explicitly says it uses default `$HOME/.config/c8volt`.
- `version` and `config validate` can report successful human output through stderr logging rather than stdout in subprocess evidence; JSON smoke checks should still require valid stdout JSON.
- `embed list --json` returns a shared envelope, while `embed deploy --json` can return a direct JSON array; use `decodeCommandPayload` for both shapes.
- Deployment evidence may not include stable Camunda resource IDs on every version; `data/seeded-data.json` records returned resource names and falls back to the embedded fixture path.
- `go test ./integration/cli -count=1` without `-tags=integration` should report `[no tests to run]`; if it regresses to "build constraints exclude all Go files", restore the untagged package shim.
- US4 family coverage currently proves CLI path, alias, help flag, destructive-classification, and output-mode manifest satisfaction without needing selected profiles; later proposal/example stories can add deeper real-state scenario evidence without changing this baseline.
- Proposal writers normalize nil slices to empty JSON arrays so no-gap reports persist as `[]`, not `null`.
- Example validation records documentation `--profile prod`, `--config`, unresolved placeholders, pipelines, hard-coded demo selectors, and unsupported resource IDs as blocked/actionable evidence rather than executing them.

## Reusable Commands

- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run TestCommandInventory -count=1 -timeout=10m`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestProfiles|TestReadOnlySmoke' -count=1 -timeout=10m`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run TestSeededData -count=1 -timeout=20m`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -count=1 -timeout=60m`
- `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1`
- `make test`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run TestProposalReports -count=1 -timeout=10m`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'Test(CommandProposal|EmbeddedBPMNProposal|Proposal)' -count=1 -timeout=10m`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'Test(HelpExampleExtraction|GeneratedDocsExampleExtraction|PlaceholderSubstitution|DestructiveWarningDetection|Examples)$' -count=1 -timeout=20m`

## Do Not Repeat

- Do not place all-command suite evidence or rules under `docs/`.
- Do not use direct facade calls for the command inventory MVP; this suite exists to validate the built CLI path.
- Do not launch `speckit-ralph-run` for this feature without explicit user instruction.

## Current Handoff

- Feature complete; no handoff required.
