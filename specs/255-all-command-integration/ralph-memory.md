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

## Decisions

- Use `c8volt capabilities --json` from the built binary as the live command inventory oracle.
- Keep the initial MVP non-destructive: it only builds the binary, runs `capabilities --json`, validates the 55-command manifest, and writes `inventory.json`/`coverage.json`.
- Keep the coverage manifest explicit for command paths, aliases, flags, output modes, owners, and destructive classification so future command or flag drift fails loudly.
- Use `C8VOLT_IT_PROFILES` as an explicit comma-separated profile selection; when it is empty, fall back to the active profile from the default local config for real profile gates.
- Read only default local config metadata from `$XDG_CONFIG_HOME/c8volt/config.yaml`, `$HOME/.config/c8volt/config.yaml`, or `$HOME/.c8volt/config.yaml`; never pass that path as `--config`.

## Gotchas

- The `capabilities` command currently exposes root persistent flags in its own flag list; the manifest intentionally includes them.
- The first compile gate should be run with `-tags=integration`; without the tag, the package has no active Go files.
- `integration/README.md` still documents existing shell suites that use `C8VOLT_IT_CONFIG`; the new Go suite section explicitly says it uses default `$HOME/.config/c8volt`.
- `version` and `config validate` can report successful human output through stderr logging rather than stdout in subprocess evidence; JSON smoke checks should still require valid stdout JSON.

## Reusable Commands

- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run TestCommandInventory -count=1 -timeout=10m`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestProfiles|TestReadOnlySmoke' -count=1 -timeout=10m`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -count=1 -timeout=10m`

## Do Not Repeat

- Do not place all-command suite evidence or rules under `docs/`.
- Do not use direct facade calls for the command inventory MVP; this suite exists to validate the built CLI path.
- Do not launch `speckit-ralph-run` for this feature without explicit user instruction.

## Current Handoff

- Next iteration should start with User Story 3 tasks T034-T042: embedded fixture discovery, seed process deployment/run data through the CLI, dirty-cluster-safe assertions, and data evidence under `integration/cli/`. Preserve the default-local config behavior and do not pass `--config`.
