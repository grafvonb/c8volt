# Contract: Operations, Provenance and Output

**Requirements**: FR-017–FR-024 | **Model**: [data-model.md](../data-model.md)

## Operation coverage

| Operation | BID path and eligibility | Limit unit |
| --- | --- | --- |
| Process-instance repair | Original PI selection plus existing incident eligibility | Distinct matching instances frozen for repair |
| Retention-policy execution | Original PI selection plus fixed derived retention end-date boundary | Distinct matching instances frozen for cleanup |
| Orphan-process-instance purge | Original PI selection plus child selection and existing orphan-parent checks | Distinct matching child candidates inspected, before orphan classification |
| Slow-process analysis | BID alone, or BID AND one existing definition selector | Distinct matching instances frozen for analysis |

No BID selector is added to incident-selected repair/purge, process-definition deletion or global definition purge. Do not change existing task/job/element selector families.

Slow analysis preserves explicit-key/search exclusivity and definition-selector exclusivity. Its existing `process_definition_search` mode remains for definition-based input; additive `process_instance_search` identifies BID-only discovery. Both modes preserve preflight wording/routing, progress and read-only behavior. For any BID discovery, defer confirmation until an actual matching record has been selected, before enrichment or expensive follow-on work; sparse empty pages and unknown totals must not cause a prompt for ultimately empty scope. Preserve automation and abort behavior. Do not add a report format that the operation does not already support.

## Discovery and confirmation

Carry BID through shared registration/population and every operation-specific selector detector, request converter, validation path and native search request. Keep both process and ops facade filter conversions synchronized, including reverse mappings. Capability/value checks run before frozen-key and empty shortcuts.

Use shared trustworthy selection continuation and deduplication before limits. Sparse pages cannot end discovery merely because they contain no selected records. Retention and repair count distinct seeds; orphan classification counts each distinct inspected child once even when it is not an orphan. This corrects the current implementation/test that counts orphans instead of the documented candidates; document the compatibility correction for all affected callers.

Confirmation consumes a copied typed operation plan. It does not replace search provenance with plain keys and then rediscover/re-expand work. Repair retains reviewed PI, incident, job and variable actions; retention and orphan retain reviewed roots and expanded delete scope. Any necessary action-time safety checks may fail work but cannot silently broaden the approved client-selected targets.

Retention and orphan consume frozen deletion members and observed parent/child order through the same plan-aware executor as BID delete. Existing recursive deletion discovers and submits live children, so it must not be used unchanged for these plans. A newly appearing child after approval is not submitted for client deletion; backend conflicts remain explicit failures, and existing force cancellation retains native cascade semantics.

Bare `DiscoveredKeys` with BID is insufficient proof of a completed selection. Reject it without a validated completed plan. Existing non-BID caller behavior remains unchanged except the explicit documented discovery/traversal corrections.

## Reports and selection provenance

Keep original BID and other selection criteria in request, frozen-set, discovery and audit selection fields, including after confirmation switches execution to frozen keys. The criterion is distinct from each resource's observed `BusinessId`; neither a report nor a preview invents a related resource's value.

Preserve supported JSON/Markdown report shapes and schema identifiers; add optional BID to existing filter objects. JSON round trips preserve exact opaque text, nil absence, selected-versus-expanded scope and frozen criteria. Human descriptions remain one line and quote opaque/control characters; Markdown escapes delimiters, markup and line breaks safely. Rendering performs no discovery requests.

## Output and failure contract

| Output category | Destination and behavior |
| --- | --- |
| Ordinary query results | stdout in requested format |
| JSON results/errors | Exactly one existing shared envelope; EOF after it; quiet does not suppress it |
| Keys-only | One key per line according to existing command meaning; zero bytes when empty |
| Human mutation/dry-run summary and verbose outcomes | Configured INFO logger, including inherited destination and activity writer; suppressed at WARN and quiet |
| Warnings/errors | Configured logger at established severity; bounded aggregate human error, no duplicate emission |
| Interactive control text | Plain configured stderr, unchanged eligibility/default/input/EOF behavior |
| HTTP diagnostics | Existing redacted DEBUG path through flag or configured level; never enabled by verbose alone |

Completed empty selections never prompt or mutate and never report acceptance/pending status under no-wait. Preserve each command's established empty payload and null/omission rules; do not invent rows to express no work.

Discovery failures preserve original classification and cannot trigger mutations. Post-submission partial failures retain partial results and machine error evidence in one envelope. Normal, quiet and verbose-only human failures group repeated causes and report supported counts with bounded reasons; raw methods, URLs, bodies and transport chains must be absent from both streams. Do not alter machine error schema merely to sanitize human output.

After rendering an envelope, exit using the original normalized error without a second error renderer. Rendering failure must not cause duplicate envelope attempts or turn an existing operation failure into success. Existing no-error-code override precedence remains intact.

## Isolated acceptance matrix

For all four operations, test fresh discovery and confirmed/frozen execution; combined predicates; tenant isolation; duplicated conversions; empty and sparse pages; limits; aborted/failed discovery; changes appearing during confirmation; and report round trips. Test no BID on 8.8/8.9 as regression, and supplied BID on older versions before both fresh and frozen-key paths.

Through command execution, assert configured/inherited destinations, logger severity, plain and structured formats, empty human mutation stdout, quiet/WARN suppression, and preserved explicit machine results. Include repeated failures containing a distinctive URL, method, body and resource key; verify normal/verbose/quiet redaction and independent flag/config DEBUG visibility. Use real terminal stdin with separately captured streams for eligible nonempty prompts and prompt-free empty scopes. Tests must detect direct-print substitutions, raw error disclosure and duplicate envelopes.
