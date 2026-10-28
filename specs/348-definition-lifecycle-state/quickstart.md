# Validate Definition Lifecycle Output

[Specification](spec.md) · [Plan and coverage ownership](plan.md) · [CLI contract](contracts/cli.md)

Run from the repository root with the declared Go toolchain. Inspect selected test helpers first: use test-owned HTTP servers, generated-client mocks and isolated CLI/PTY fixtures only. The clean environment below removes inherited profiles, credentials and build tags; test commands supply their own fake-server configuration. No real Camunda installation is needed or permitted for implementation validation.

## Focused checks

For command or facade changes:

```sh
env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS= \
  go test ./cmd -run 'ProcessDefinition' -count=1

env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS= \
  go test ./c8volt/process -run 'ProcessDefinition' -count=1
```

For adapter changes, additionally run the matching service tests:

```sh
env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS= \
  go test ./internal/services/processdefinition/... -run 'ProcessDefinition' -count=1
```

Expected evidence: state propagation/omission, human and single-envelope JSON output, legacy modes, mixed columns, independent statistics, unchanged requests, watch comparisons/baselines, real-terminal styling, and configured diagnostic routing/redaction/exit behavior. Preserve the logger matrices required by `AGENTS.md`; renderer-only checks cannot replace execution-path assertions. Report terminal skips. Avoid duplicating each value across every result shape and count population; the plan assigns coverage to the owning boundary.

## Documentation and completion

After source help or README changes, regenerate CLI references and the docs homepage:

```sh
env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS= make docs-content
```

Help changes also need the cross-command prose and metadata checks, which the `ProcessDefinition` pattern alone does not select completely:

```sh
env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS= \
  go test ./cmd -run 'TestAllCommandHelpFocusesOnActions|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestProcessDefinitionSelectorValidationHelpContract' -count=1
```

Run `gofmt` on touched Go files and `git diff --check`; validate changed local links. Record actual results in [progress.md](progress.md). Do not rerun runtime tests for subsequent prose-only edits or commits. Broaden to `make test` only for wider runtime/concurrency impact or unresolved focused failures.

The existing draining integration observer is unchanged by the simplification. If its source changes later, compile it with `go test -tags=integration -c ./integration/cli -o <temporary-path>/cli.test`; do not execute the binary, TestMain or integration Make targets. No new no-wait scenarios or live fixture matrix are needed. Original compile-only evidence is retained in progress; real-engine transition timing remains unverified.
