# Contract: CLI Automation Mode

## Entry Contract

Automation mode is exposed through one dedicated root flag.

| Concern | Required behavior |
|--------|-------------------|
| Entry point | Expose automation mode through one dedicated root flag, planned as `--automation` |
| Scope | The flag is a root persistent flag and applies to the invoked command path |
| Intent | The flag is the canonical non-interactive opt-in for AI agents, scripts, and CI |
| Compatibility | The flag layers onto the current CLI and does not replace human-facing command usage |

Automation mode is an explicit opt-in. Commands must behave exactly as they do today when the flag is absent.

## Support Contract

Every discoverable command should report whether it supports automation mode.

| Status | Required meaning |
|--------|-------------------|
| `full` | The command defines supported automation-mode behavior |
| `unsupported` | The command must reject automation mode explicitly |

Support for automation mode is distinct from support for the shared JSON result envelope.

## Unsupported Command Contract

If a command has not opted into automation support, the CLI must reject the invocation explicitly.

| Rule | Required behavior |
|------|-------------------|
| No fallback | Do not fall back to interactive behavior |
| No guessing | Do not infer a best-effort non-interactive default |
| Error shape | Return an actionable failure message; in JSON mode, return a `failed` result envelope |
| Exit behavior | Preserve existing process-level exit semantics through `ferrors` |

## Prompt Contract

Supported commands in automation mode treat supported prompts as accepted.

| Prompt type | Required behavior |
|------------|-------------------|
| Confirmation prompts | Implicitly accepted |
| Paging continuation prompts | Implicitly accepted |
| Unsupported prompt flow | Reject automation mode explicitly rather than prompting |

Automation mode must not leave a supported command waiting for terminal input.

## Output Contract

Automation mode must keep the machine-readable path deterministic.

| Concern | Required behavior |
|--------|-------------------|
| JSON mode | Continue using the shared result envelope from `#78` |
| Stdout | When `--json` is requested with automation mode, stdout is reserved for the machine-readable result |
| Stderr | Human-oriented logs, progress, and diagnostics go to stderr or are suppressed |
| Plain mode | Human-oriented output may remain available when JSON is not requested |

Automation mode does not create a new payload schema. It reuses the current machine-readable contract where supported.

## Async Contract

Automation mode does not make commands asynchronous by default.

| Concern | Required behavior |
|--------|-------------------|
| Default behavior | Supported commands still wait for confirmed completion unless the caller opts out |
| Explicit async | `--no-wait` remains the explicit selector for accepted/not-yet-complete execution |
| Accepted outcome | In JSON mode, automation mode plus `--no-wait` returns `accepted` |
| Confirmation semantics | `accepted` must not imply the work is already complete |

## Discovery Contract

The machine-readable discovery surface must describe automation mode truthfully.

| Concern | Required behavior |
|--------|-------------------|
| Discovery entry point | `c8volt capabilities --json` remains the canonical discovery surface |
| Automation metadata | Discovery reports whether each command supports automation mode |
| Invocation guidance | Discovery and command help must describe `--automation` as the canonical non-interactive opt-in |
| Truthfulness | Discovery must not mark commands as automation-capable unless runtime behavior matches the contract |

## Representative Coverage Contract

The initial rollout must cover representative read and write flows built on current command seams.

- `capabilities`
- `get process-instance`
- `run process-instance`
- `deploy process-definition`
- `delete process-instance`
- `cancel process-instance`
- explicit supported or unsupported handling for `expect process-instance`
- explicit supported or unsupported handling for `walk process-instance`

## Documentation Contract

User-facing documentation for automation mode must be updated in:

- command help text for the root flag and affected commands
- `README.md`
- `docs/use-cases.md`
- generated CLI docs under `docs/cli/`

Generated CLI docs must be refreshed from Cobra metadata rather than edited by hand.
