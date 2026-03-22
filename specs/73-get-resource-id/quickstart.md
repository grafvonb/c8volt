# Quickstart: Add Resource Get Command By Id

## Goal

Implement a repository-native `c8volt get resource --id <id>` workflow that reuses the existing resource service lookup capability and returns the normal single-resource object/details output.

## Implementation Steps

1. Extend `c8volt/resource.API`, `c8volt/resource/client.go`, and `c8volt/resource/model.go` with a single-resource retrieval method and public model mapping.
2. Add a `cmd/get_resource.go` command that:
   - registers under `get`
   - requires `--id`
   - calls `NewCli(cmd)` and routes through `cli.GetResource(...)`
   - reuses existing `ferrors.HandleAndExit` behavior for failures
3. Add a resource rendering helper alongside existing `cmd_views_*` helpers so success output matches the normal single-item object/details pattern used by other `get` commands.
4. Keep underlying malformed-payload behavior unchanged by relying on the current versioned service implementations and shared payload validation.
5. Update command help text, review whether `README.md` needs changes, and regenerate CLI docs with `make docs`.

## Validation Steps

1. Run targeted tests for the command package, resource facade, and versioned resource services.
2. Confirm the command fails fast when `--id` is missing or invalid.
3. Confirm successful lookup renders the expected single-resource object/details output.
4. Confirm not-found and malformed-response cases preserve non-success behavior.
5. Run `make test`.

## Expected Artifacts

- New or updated `cmd/get_resource.go`
- Updated `cmd/get_test.go` and any resource-specific command tests
- Updated `c8volt/resource/api.go`
- Updated `c8volt/resource/client.go`
- Updated `c8volt/resource/model.go`
- Updated resource service tests where needed
- Regenerated `docs/cli/` output
