# Contract: Public CLI Help Discovery

## Scope

This contract defines the expected public help and generated documentation behavior for issue `#77`.

The contract covers all user-visible `c8volt` commands in the public Cobra tree:

- the root command
- parent/group commands
- executable leaf commands

Hidden or internal commands remain out of scope unless they are intentionally surfaced as public commands.

## Source-of-Truth Contract

| Concern | Required behavior |
|--------|-------------------|
| Authoritative source | Edit Cobra `Short`, `Long`, and `Example` metadata in `cmd/` |
| Generated docs | Regenerate `docs/cli/` from command metadata using the repository generation path |
| Hand edits | Do not hand-edit generated CLI reference pages |
| README sync | If README guidance changes, refresh README-synced docs through the repository generation path |

## Public Command Coverage Contract

| Command type | Required behavior |
|-------------|-------------------|
| Root command | Explain the CLI’s overall purpose, discovery surface, and shared guidance |
| Parent/group command | Explain the command-family purpose and help callers choose the right child path |
| Leaf command | Explain what the command does, whether it reads or changes state, and any important automation or verification semantics |
| Hidden/internal command | Remain excluded from the public help-refresh scope unless intentionally made public |

## Help Content Contract

Every covered command must provide refreshed help metadata appropriate to its role.

| Help field | Required behavior |
|-----------|-------------------|
| `Short` | Concise summary that helps the caller identify the command quickly |
| `Long` | Expanded guidance explaining when the command should be used and any important behavioral distinctions |
| `Example` | Refreshed examples for every covered command |

## Example Contract

| Command type | Required example behavior |
|-------------|---------------------------|
| Root command | Show top-level discovery or starting-point examples |
| Parent/group command | Show navigational or chooser-oriented examples that help users select child workflows |
| Leaf command | Show realistic, copy-pasteable command examples |
| State-changing leaf command | Include a realistic follow-up inspection or verification command where relevant |

Examples must stay truthful to current command behavior and must not imply unsupported automation or completion guarantees.

## Automation Guidance Contract

When a command already participates in the machine-readable or automation-aware surface, help text must stay aligned with current repository behavior.

| Concern | Required behavior |
|--------|-------------------|
| Discovery | Preserve `c8volt capabilities --json` as the canonical machine-readable discovery surface |
| Structured output | Recommend `--json` where that is the preferred automation path |
| Non-interactive guidance | Describe `--automation` only where the command already supports that contract |
| Prompt behavior | Describe `--auto-confirm` and `--no-wait` truthfully where relevant |

## Documentation Parity Contract

User-facing documentation for this feature must stay aligned across:

- live Cobra help output
- generated CLI docs under `docs/cli/`
- `README.md` when top-level workflow examples or discovery guidance change
- README-synced docs such as `docs/index.md` when README content is regenerated

## Validation Contract

The implementation is not complete until:

1. Focused command-level validation has been run against updated help metadata.
2. `make docs` has been run after public command metadata changes.
3. `make docs-content` has been run when README changes should flow into docs.
4. `make test` passes.
