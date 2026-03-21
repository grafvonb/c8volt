# CLI Command Contract: Resource Lookup By Id

## Scope

This contract defines the expected public CLI behavior for retrieving one resource by id.

## Preferred Command

- **Command**: `c8volt get resource --id <resource-id>`
- **Purpose**: Retrieve one deployed resource as the normal single-resource object/details view.
- **Expected Behavior**:
  - Uses the existing `get` command tree.
  - Requires an explicit `--id` value before any lookup is attempted.
  - Returns the normal single-resource details/object output on success.
  - Preserves the repository's normal failure output and non-success exit behavior when validation or lookup fails.

## Flag Rules

- `--id` is required for this command.
- Missing, empty, or invalid `--id` input must fail validation before any backend request.
- No raw resource-content flag is introduced in this feature.
- Existing root and `get` command flags continue to propagate normally.

## Output Rules

- Successful lookup renders the normal single-resource object/details output rather than raw resource content.
- Not-found, transport, and malformed-response failures must not be presented as successful empty output.
- A `200 OK` response without the expected resource payload is treated as a malformed-response error.

## Help and Documentation Rules

- `c8volt get --help` must expose `resource` as a subcommand if it is part of the shipped command tree.
- `c8volt get resource --help` must describe the required `--id` flag and the command’s single-resource purpose.
- Generated CLI docs must include the resource command and required identifier semantics.
- `README.md` updates are required only if existing usage guidance would otherwise omit or contradict the new workflow.

## Testable Acceptance Signals

- Running the command with a valid resource id returns one resource in the normal object/details form.
- Running the command without `--id` fails validation before any lookup is attempted.
- Running the command for a missing resource id preserves clear non-success behavior.
- A malformed success response does not render as an empty or successful lookup.
