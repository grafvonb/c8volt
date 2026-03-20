# CLI Command Contract: Process Definition XML Retrieval

## Scope

This contract defines the expected public CLI behavior for retrieving one process definition as raw XML.

## Preferred Command

- **Command**: `c8volt get process-definition --key <process-definition-key> --xml`
- **Purpose**: Retrieve the BPMN XML for one deployed process definition and emit it directly to standard output.
- **Expected Behavior**:
  - Uses the existing `get process-definition` command path.
  - Requires an explicit process definition key.
  - Returns only the XML payload on successful standard output.
  - Preserves the repository's normal error output and non-success exit behavior when retrieval fails.

## Flag Rules

- `--xml` is valid only for single-definition retrieval.
- `--xml` requires `--key`.
- `--xml` must reject conflicting list-oriented or render-oriented flag combinations instead of guessing precedence.
- Existing non-XML behaviors for `get process-definition` remain unchanged when `--xml` is not supplied.

## Output Rules

- Successful XML retrieval writes raw XML to stdout without one-line summaries, item counts, or JSON wrapping.
- The output must remain safe for shell redirection such as `c8volt get process-definition --key <key> --xml > example.bpmn`.
- Failure output must not be presented as a successful XML export.

## Help and Documentation Rules

- `c8volt get process-definition --help` must describe the XML option and its single-key intent.
- Generated CLI docs must include the XML flag and its user-facing purpose.
- README updates are required only if existing process-definition retrieval examples would otherwise become incomplete or misleading.

## Testable Acceptance Signals

- Running the command with a valid key returns XML content on stdout.
- Redirecting stdout to a file produces a reusable BPMN file without extra formatting.
- Running the command without `--key` or with conflicting flags fails with clear validation semantics.
- Retrieval failures preserve non-success exit behavior and do not silently create a successful export experience.
