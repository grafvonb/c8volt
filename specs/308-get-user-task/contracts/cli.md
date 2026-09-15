# CLI Contract: get user-task

## Grammar and inputs

```text
c8volt [inherited flags] get user-task [flags] [-]
Aliases: user-tasks, ut, uts
```

| Flag | Alias | Contract |
| --- | --- | --- |
| --key | -k | Repeated/comma-separated 16-digit keys, using existing key validation |
| --pi-key | | Exact owning process-instance key |
| --pd-key | | Exact process-definition key |
| --bpmn-process-id | -b | Exact BPMN process ID |
| --element-id | | BPMN task element ID |
| --state | -s | Case-insensitive supported state; default `all` |
| --assignee | | Exact assignee |
| --candidate-user | | Exact candidate-user membership |
| --candidate-group | | Exact candidate-group membership |
| --batch-size | -n | Positive page size up to 1000; default 1000 |
| --limit | -l | Positive maximum returned tasks; omitted means unlimited |
| --total | | Exact matching count only |

Reuse inherited tenant, output, quiet, verbose/debug, worker, auto-confirm, and automation options and validation. Do not add task mutation, form, variable, date, sort, or watch flags.

Read nonterminal stdin with or without `-`. Merge flags before stdin and deduplicate stably. Trim lines and skip blank lines using current parsing rules; retain existing scanner bounds and malformed-input guidance. Explicit `-` with terminal stdin or an empty stream errors as it does for `get pi`. Empty implicit stdin contributes no keys. Never block on terminal stdin to infer a key selection. Other positional arguments error.

Validate all flag and stdin keys, including process-selector keys. Default state `all` is not an explicit selector; supplying `--state all` is a search flag for conflict checking. Repeated scalar search flags retain ordinary pflag semantics; only `--key` is a multi-value flag.

## Modes and conflicts

- Nonempty key input selects strict native GET lookup. Any missing key fails; no successful partial collection is rendered. Tenant settings do not post-filter explicit keys, and actual tenant metadata is retained.
- No keys selects backend search with AND-combined filters and effective discovery tenant scope. No filters means all visible tasks in that scope.
- Keys conflict with every explicitly supplied search filter, `--limit`, and `--total`. Inherited `--tenant` is not a key conflict. Batch size is validated but does not change direct lookup behavior.
- `--total` conflicts with `--limit`, `--json`, and `--keys-only`, even when another output mode would otherwise take precedence. It never prompts and emits no list summary.
- State names are listed in [data-model.md](../data-model.md). `assigned` is invalid. Unknown versions follow existing configuration errors; selected 8.7 yields unsupported operation for new native reads.
- Command validation failures produce established invalid-input errors before task retrieval. Version validation uses the resolved configuration and existing error machinery.

## Results

### JSON

One collection payload for any keyed cardinality and for search, passed to the existing full-contract renderer. Minimal successful empty result (optional existing tenantContext may also be present):

```json
{
  "outcome": "succeeded",
  "command": "get user-task",
  "payload": {"total": 0, "items": []}
}
```

Nonempty payload example:

```json
{
  "total": 1,
  "items": [{
    "key": "2251799815391233",
    "state": "CREATED",
    "name": "Approve invoice",
    "elementId": "approve_invoice",
    "assignee": "alice",
    "processInstanceKey": "2251799813711967",
    "processDefinitionId": "invoice",
    "tenantId": "accounting"
  }]
}
```

Task fields and omission rules are defined in the data model. `total` is the returned item count. Do not expose generated request bodies, cursor data, or count-cap metadata in this CLI payload. Use shared alias canonicalization and tenant-context generation. Consumers can decode one envelope and then require EOF. On errors use the shared error envelope and existing exit classification; no successful incomplete JSON is emitted.

### Human, keys, and quiet

Human rows use existing flat-row alignment without headings. Column order is task key, tenant, element ID, state, optional `name:<name>`, BPMN process ID (`processDefinitionId`), `pi:<process-instance-key>`, `ei:<element-instance-key>`, `pd:<process-definition-key>`, and always `assignee:<assignee>` last (`assignee:<unassigned>` when empty). The display name does not replace the technical element ID. Other empty optional fields are omitted using existing alignment behavior. The list summary is `found: N` with a trailing newline. Completed empty search emits exactly `found: 0\n`.

Example:

```text
2251799813900041 tenant-a SimpleUserTask_UserTask CREATED name:Simple User Task C89_SimpleUserTask pi:2251799813900036 ei:2251799813900040 pd:2251799813873873 assignee:<unassigned>
found: 1
```

Keys-only emits each task key followed by a newline, and zero bytes for no matches. Select JSON first, then keys-only, then human. Quiet suppresses human rows and summaries after that selection, preserving JSON and keys output. Quiet does not suppress an explicitly requested numeric `--total` result. A total is a base-10 integer plus one newline, with no prompt, header, footer, or diagnostic content on stdout.

### Paging and streams

Services own continuation, limits, and total counting. The command page visitor owns rendering and asking whether to continue. Empty pages with continuation proceed without a question. JSON collects and renders once. Automation and auto-confirm continue without prompts. Reaching the overall limit stops without a prompt; supplying a limit does not otherwise disable eligible paging before that boundary. Otherwise use existing terminal-stdin eligibility; redirected stdout does not disable an otherwise eligible prompt.

Use the existing prompt helper with `cmd.ErrOrStderr()` (including inherited destinations). Adapt its resource wording:

```text
Fetched N user task(s) on this page (M loaded). More matching user tasks remain.
Continue? [y/N]:
```

The existing helper owns final spacing, default answer, and input parsing. `y`/`yes` continues; decline, other input, or EOF stops under existing get behavior. Incremental rows already written on a later failure remain visible, but the command must exit/report failure. Numeric totals and JSON must not claim completion after a failed traversal. No mutation or mutation-confirmation requests occur.

## Metadata and documentation

Register invalid-input flag handling, read-only mutation metadata, full shared-contract support, and full automation support. Add canonical name, aliases, examples, limitations, and count conflicts to command help and relevant parent help; update README and regenerate CLI docs via `make docs-content`. Retain existing HTTP/debug and activity routing.
