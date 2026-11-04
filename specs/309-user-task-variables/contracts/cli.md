# CLI Contract: User-Task Variable Display

## Invocation and flags

Existing `get user-task`, `get user-tasks`, `get ut`, and `get uts` receive:

| Flag | Default | Contract |
| --- | --- | --- |
| `--with-vars` | false | Attach effective variables to eligible returned task output |
| `--var-value-limit` | 0 | Maximum human value characters; zero is unlimited |

A negative value limit fails validation. Any explicit value limit, including zero, without `--with-vars` fails validation. Validate before task/variable requests. Neither flag becomes a search filter or conflicts with explicit keys. Existing key/search/count conflicts remain unchanged.

## Output selection

| Execution | Variable retrieval | Output |
| --- | --- | --- |
| No `--with-vars` | None | Existing unchanged task output |
| Human plus `--with-vars` | All pages per selected task | Existing task rows, nested variables, existing summary |
| JSON plus `--with-vars` | All pages per selected task | One shared envelope with enriched payload |
| JSON plus keys-only plus `--with-vars` | Yes: JSON wins | Same enriched JSON contract |
| Effective keys-only plus `--with-vars` | None | One task key per line; empty means zero bytes |
| Valid `--total --with-vars` | None | Exact numeric count and newline, including quiet |
| Quiet human plus `--with-vars` | Requested retrieval still applies | Human results suppressed; failures remain errors |
| Quiet JSON plus `--with-vars` | Yes | Enriched JSON retained |
| Empty selected collection | None | Existing mode-appropriate empty result |

Existing invalid total-mode combinations remain invalid. No new mutation, filtering, watch, dry-run, or no-wait semantics are added.

## Human format

Keep task-row fields and alignment unchanged, with assignee last. Use the existing tree branch helpers for a `vars:` subtree and ascending variable names:

```text
<existing task row>
└─ vars:
   ├─ amount=120
   └─ payload=abc... [cli-truncated]
found: 1
```

Follow the current process-variable tree's exact indentation/glyph conventions in implementation tests. A task with no variables has no `vars:` subtree and still contributes to the result count. Structured object/array strings use existing compaction. Values are shortened after the first N Unicode runes only when longer than N; append `...` and existing labels `[api-truncated]`, `[cli-truncated]`, or `[api-truncated,cli-truncated]` as applicable. Zero/default does not shorten locally. The shared formatter receives the limit explicitly and does not read task flags from a process-command global.

## JSON payload

Use the existing shared envelope without schema changes. Its payload has this structure (illustrative task details are abbreviated):

```json
{
  "total": 1,
  "items": [
    {
      "item": {
        "key": "2251799815391233",
        "state": "CREATED",
        "processInstanceKey": "2251799813711967"
      },
      "variables": [
        {
          "name": "amount",
          "value": "120",
          "variableKey": "2251799815391240",
          "processInstanceKey": "2251799813711967",
          "scopeKey": "2251799815391230",
          "apiTruncated": false
        }
      ]
    }
  ]
}
```

Preserve all existing task fields and their omission rules. Variable tags are defined in [data-model.md](../data-model.md). Human value limits never modify received JSON values. No process age `meta` is added. Successful empty payload is `{"total":0,"items":[]}`; a task without variables has `"variables":[]`. Default execution without enrichment retains ordinary task entries rather than these wrappers.

## Interaction and errors

Only selected tasks are enriched. Task page size, result limits, tenant rules, input-key merging, and paging eligibility remain unchanged. Interactive pages finish enrichment and rendering before the continuation prompt. Prompt wording, default, decline/EOF behavior, configured/inherited stderr, and redirected-stdout eligibility remain as in the base command. JSON, automation, and auto-confirm follow existing unattended rules.

Failures on any task/page propagate through existing errors and exit codes; no empty substitute or final success summary. A collected failure emits no successful result envelope. Earlier streamed human output need not be rolled back. Writer errors propagate. Stdout contains results only; diagnostics and interactive control text retain existing stderr routing.
