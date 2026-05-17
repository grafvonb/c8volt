# Contract: Ops Command Foundation

## Command Surface

| Command | Type | Required behavior |
| --- | --- | --- |
| `c8volt ops` | Grouping command | Shows help for high-level operational workflows and performs no workflow behavior |
| `c8volt ops execute` | Grouping command | Shows help for predefined operational playbooks and performs no workflow behavior |
| `c8volt ops repair` | Grouping command | Shows help for repair/remediation workflows, performs no workflow behavior, and exposes no ambiguous top-level `--key` |

## Help Contract

- Each grouping command must return help or normal Cobra grouping behavior.
- Help must not require Camunda configuration or runtime clients.
- Help text must make clear that concrete playbooks are added by target-specific subcommands later.
- Existing top-level command help must remain unchanged except for adding `ops` to discovery.

## Discovery Contract

- `ops` command metadata must classify the command family as operational/state-changing.
- Grouping commands must not claim full automation support until concrete leaf commands define full machine contracts.
- Existing `capabilities --json` discovery behavior must remain valid JSON and include ops metadata when discoverable.

## Shared Future Workflow Contract

Future concrete ops workflows should reuse these conventions:

- State-changing metadata for mutating commands.
- Explicit automation support metadata and guardrails.
- `--dry-run` previews that do not mutate state.
- Structured report model before Markdown or JSON rendering.
- Report file output with format inference from extension where supported.
- Deterministic JSON stdout for `--automation --json`.
- Progress, logging, and activity output kept off stdout for automation JSON paths.
- Step statuses from the shared set: `planned`, `skipped`, `submitted`, `confirmed`, `confirmation_failed`, `blocked`, `failed`.

## Out-of-Scope Contract

The following commands must not be introduced by this feature:

- `c8volt ops execute orphan-cleanup`
- `c8volt ops execute retention-policy`
- `c8volt ops execute smoke-test`
- `c8volt ops repair incident`
- `c8volt ops repair process-instance`
