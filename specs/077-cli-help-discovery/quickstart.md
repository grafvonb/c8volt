# Quickstart: Improve CLI Help Discovery

## Goal

Refresh the full public `c8volt` help surface so every user-visible command has clearer purpose, better examples, and docs that regenerate cleanly from the updated Cobra metadata.

## Prerequisites

- A local checkout on branch `077-cli-help-discovery`
- Go toolchain matching the repository baseline
- The ability to regenerate CLI docs locally with the repository make targets

## Suggested Working Order

1. Inventory the public command tree under [`cmd/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd) and separate:
   - root command
   - parent/group commands
   - executable leaf commands
   - hidden/internal commands that should remain out of scope
2. Refresh root and parent/group command metadata first so command-family routing language is consistent.
3. Refresh leaf command help metadata and example blocks, keeping behavior descriptions aligned with current runtime semantics.
4. Regenerate CLI docs after metadata changes.
5. Sync README-derived docs if README changed.
6. Finish with repository validation.

## Implementation Slices

1. Setup slice:
   - Update [`research.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/research.md) with the live public command inventory and batching notes.
   - Keep this slice documentation-only.
2. Foundational slice:
   - Refresh shared discovery wording in [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go) and [`cmd/capabilities.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go).
   - Strengthen shared public-command regression seams in [`cmd/command_contract_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract_test.go) and [`cmd/root_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root_test.go).
3. User Story 1 slices:
   - Refresh root and family-entry command guidance first.
   - Refresh discovery/config/cluster-read commands next.
   - Refresh the remaining read-oriented retrieval commands after shared language is stable.
   - Finish the story with embed/version examples and matching help regressions.
4. User Story 2 slices:
   - Add help regressions for state-changing and verification-oriented commands before editing help text.
   - Refresh `run`/`deploy`, then `cancel`/`delete`, then `expect`/`walk`.
5. User Story 3 slices:
   - Update top-level guidance in `README.md` and `docs/index.md`.
   - Regenerate `docs/cli/` through `make docs`.
   - Regenerate `docs/index.md` through `make docs-content` only if README changed.

## Key Files To Start With

- Root and shared guidance:
  - [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go)
  - [`cmd/capabilities.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go)
  - [`cmd/command_contract.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract.go)
- Parent/group commands:
  - [`cmd/get.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get.go)
  - [`cmd/delete.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete.go)
  - [`cmd/deploy.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy.go)
  - [`cmd/run.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run.go)
  - [`cmd/cancel.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel.go)
  - [`cmd/walk.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk.go)
  - [`cmd/expect.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect.go)
  - [`cmd/config.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config.go)
  - [`cmd/embed.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/embed.go)
- Representative leaf commands:
  - [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go)
  - [`cmd/delete_processdefinition.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processdefinition.go)
  - [`cmd/config_show.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_show.go)
  - [`cmd/embed_export.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/embed_export.go)
  - [`cmd/version.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/version.go)

## Verification Commands

1. Run focused command tests first:

```bash
go test ./cmd -count=1
```

2. Regenerate the CLI reference from command metadata:

```bash
make docs
```

3. If README examples or guidance changed, sync README-derived docs:

```bash
make docs-content
```

4. Run the repository-wide validation gate:

```bash
make test
```

## Validation Checklist

1. During command metadata work, use focused coverage first:
   - `go test ./cmd -count=1`
2. When command help text changes, regenerate generated CLI docs:
   - `make docs`
3. When README examples or discovery guidance change, sync README-derived docs:
   - `make docs-content`
4. Before committing any completed work unit, run the repository gate required by the repo instructions:
   - `make test`
5. Confirm the public/private boundary still matches discovery output:
   - the live `capabilities --json` tree should exclude hidden commands, `help`, and `__complete*`

## Verification Record

Validation rerun completed on 2026-04-19 for the final polish pass.

1. Focused command-help regression suite:
   - `go test ./cmd -count=1`
   - Result: pass
2. Public command-surface audit:
   - `GOCACHE=/tmp/c8volt-gocache go run . capabilities --json`
   - Result: 29 discoverable public command paths beneath the root command; hidden/internal commands stayed excluded.
3. Generated docs audit:
   - `ls docs/cli`
   - Result: 30 command reference pages plus `index.md`, including separate pages for `get cluster topology` and `get cluster-topology`.
4. Repository-wide validation gate:
   - `make test`
   - Result: pass

## Manual Review Checklist

1. Root help still explains the product story and points automation callers at `capabilities --json`.
2. Parent/group commands do more than say “requires a subcommand”; they help users pick the right child workflow.
3. Every public leaf command exposes refreshed examples appropriate to its role.
4. State-changing commands describe waiting, confirmation, and follow-up verification truthfully.
5. Generated docs under [`docs/cli/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli) match the updated metadata.
6. Hidden/internal commands remain out of the public docs and help-refresh scope.
7. The audited public count is 30 covered command pages including the root command.
