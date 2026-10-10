# Contract: Smoke-Test Complete Cleanup

## Command and options

Command remains `c8volt ops execute smoke-test`. Preserve flag names, defaults, confirmation wording and stream routing, automation/auto-confirm eligibility, count/workers controls, tenant rules, and exit handling. Update descriptive help and preview wording from a singular definition to all deployed fixture definitions. No new output mode is introduced.

Default cleanup considers only returned exact definition keys after successful instance cleanup. Blocked dependencies are retained independently. No-cleanup makes no cleanup calls; dry-run makes no mutations. No-wait uses normal safety checks and reports acceptance without confirmation. Fail-fast stops scheduling after errors, not ordinary safety skips; cancellation similarly preserves unattempted evidence.

## Human output

Preserve existing log formatting and stage ordering. Replace the singular cleanup resource label with accurate counts, for example `cleanup: removed 1 process instance and 3 process definitions`. Accepted no-wait work uses `submitted ... (--no-wait)`. Mixed results report counts by disposition and never prefix all resources with removed. Default output stays compact. Verbose output and audit reports expose each retained/failed identity and available reason. Quiet suppresses informational human summaries without suppressing explicitly requested JSON.

Progress uses the distinct deployment scope as one fixed definition total. Nested calls do not emit repeated 1/1 stages or per-key INFO noise. Skipped/failed items are not reported as confirmed completions. HTTP diagnostics remain DEBUG-only.

## JSON and public Go facade

Keep `ops.smoke-test.v1`, existing envelope/error handling, and existing singular field types and semantics. Add fields defined in [data-model.md](../data-model.md): deployment `processDefinitions` and `definitionScopeComplete`; cleanup `definitionResults` and `definitionSummary`. New fields use optional omission conventions; existing null/empty/omission rules remain unchanged. New records never label an empty response as a confirmed deletion.

Singular fields describe the main fixture only; consumers seeking deployment-wide state use `definitionSummary` and `definitionResults`. Existing outcome vocabulary and exit conventions remain:

| Situation after successful proof steps | Outcome | Cleanup evidence |
| --- | --- | --- |
| All required deletions confirmed | passed | confirmed; full counts |
| All required requests accepted with no-wait | passed | submitted, no-wait; no removal claim |
| Explicit no-cleanup or safety retention without errors | passed_cleanup_skipped | skipped/retained counts and reasons |
| Eligibility, deletion, confirmation, or identity failure | partially_failed | failed and unattempted entries; existing error exit policy |

For mixed results, errors take precedence over intentional skips; intentional skips take precedence over submission/confirmation for the aggregate status. Earlier-stage failure behavior remains unchanged, but known deployed identities remain accounted for as unattempted. A completed main-definition deletion cannot override a dependency error.

## Audit files

JSON reports contain the same definition identities and outcomes as the command result. Markdown adds complete deployed-definition and cleanup tables with key, BPMN ID, tenant when known, disposition, and reason. Existing main-fixture fields remain available. Report-file timing and error handling remain unchanged.

## Compatibility and regression scope

No dependency discovery by renderer, name-based deletion expansion, or retroactive cleanup of earlier runs. Test deterministic collection order, singular projection compatibility, accepted-response-plus-wait-error handling, one JSON envelope followed by EOF, separate stdout/stderr, quiet+JSON, supported automation combinations, and any existing keys-only behavior without adding unsupported modes. Interactive routing tests use real terminal stdin if the affected execution path includes a prompt; preserve configured/inherited stderr destinations.
