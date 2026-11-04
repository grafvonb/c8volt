# Compatibility and isolated acceptance contract

All cases are future implementation requirements, not tests executed during planning.
C01–C10 in spec.md are binding. Every task owns its relevant subset; this document is not a
standalone test task or permission to run a live environment.

## Public API and wire compatibility

Capture the current exported surface before moving declarations: root c8volt and public
subpackages, foptions/ferrors, config and affected toolx/pool exports. Preserve names,
signatures, package/type identity, interface method sets, struct fields/tags, constructor
behavior and errors.Is/As behavior. Additive preparation functions are permitted; adding a
required method to an existing API is not. Use explicit external-style implementations and
compile assertions for affected existing interfaces; embedding the interface would defeat
this check. Test legacy pool signatures with existing FacadeOption values and generic report
types. Keep c8volt/variable's current package name and constructors even though it is unused
in-repo. Generated clients, configuration keys and release pins remain unchanged.

Compare successful/partial/failed machine payloads structurally and through JSON EOF. Cover
omission/null/empty-array differences and optional pointer ownership. Do not rename fields or
change schemas to accommodate shared internals. API byte/schema assertions must distinguish
intentional new additive declarations from forbidden modifications.

## JSON snapshot and plan ownership

Use accepted ordinary and typed maps/slices/arrays, large signed/unsigned integers, json.Number,
json.RawMessage, nested empty/nil containers and supported custom JSON marshalers. Retain
current equality semantics for numeric spellings. Probe caller mutation after planning and
adapter/result mutation after conversion: frozen payloads and other observations must remain
independent. Include malformed raw JSON, unsupported values and cycles, preserving existing
rejection timing/class/text. Reject no new accepted input as incidental cleanup. Service/facade
coverage must prove mutations use the approved exact payload with test-owned dependencies.

## Worker and lifecycle contracts

Exercise empty/single/multiple/duplicate selections, automatic/explicit/unlimited workers,
fail-fast on/off, already-canceled context, in-flight cancellation and timing preflight. Assert
stable output slots, joined errors, scheduling limits, staggering and no goroutine outliving
the call. No-wait skips only its existing confirmation phase. A local admission failure cannot
become submitted work; a later rejection cannot erase earlier unknown/accepted evidence.

Run suspend/resume separately through capability validation, traversal, native submission and
observation with fakes. Preserve descendant/independent-child rules, sparse and invalid chain
evidence, terminal conditions, 404/authorization behavior, wrong identity, read-after-deadline,
attempt exhaustion, backoff and debug suppression. Native request bytes and retry counts stay
endpoint-specific; use test-owned HTTP for 204, unexpected 2xx, 4xx, 429, 503 problem and proxy
responses, missing metadata and call-error/response combinations.

## CLI result, diagnostic and prompt contract

Exercise commands, not just formatters, using configured/inherited logger writers and separate
stdout/stderr. Include mutation success, dry-run, verbose, quiet, WARN, plain and structured
formats. Human mutation summaries are INFO logger records and human stdout is empty; query
rows, keys and machine payloads stay stdout. Plain prompts use configured stderr. Verify
nonempty requests, authoritative empty scopes, explicit keys, sparse pages, abort/EOF and
supported automation/auto-confirm/no-wait combinations. Use real terminal stdin for eligible
prompts and prompt-free empty scopes; piped stdin/mocked terminal predicates are insufficient.

Use a repeated multi-resource fake failure containing a distinctive URL, HTTP method, body,
resource key and secret marker. At normal/verbose levels no raw HTTP evidence appears on
either stream; errors are bounded/grouped and quiet retains compact ERROR. DEBUG via flag and
configured level independently retains redacted diagnostic evidence; stdout stays diagnostic-
free. Decode exactly one supported JSON envelope then EOF, retaining partial evidence and the
original exit status/no-err-codes override. Assert destination, severity, exact record count and
no default-logger fallback; substring-only stderr checks are insufficient. The checks must
fail if routing is reverted to direct prints, raw aggregates or a duplicate envelope.

Preserve established bounded phrases, punctuation, ordering, prompt controls and schema.
Changing legacy raw human wording is subject to C05: isolate it in a reproduction, record the
proposed compact replacement and obtain explicit correction acceptance before implementation.
Do not weaken a required diagnostic assertion to label the refactor complete. Bootstrap/local
config failures and already-attempted output paths keep their distinct fallback/exit handling.

## Indexes and decoding

Compare indexed usertask validation/result assembly against the current output for independent
and shared scopes, duplicate/missing associations, contradictory tenants and shuffled input.
Count actual mutations, discovery and verification calls. Bench only pure local fan-out with
increasing task/edge counts and ReportAllocs; no absolute timing threshold is an acceptance
condition. Validate that helper/index construction occurs once rather than rescanning targets.

Run one raw JSON corpus through all three effective decoders and a separate root/local corpus
through all three variable decoders. Cover missing/null required fields, malformed scalar
fields, truncated values, wrong owner/scope/tenant, totals absent/present/exact/lower bound,
raw-count contradictions, sparse pages, repeated/nonadvancing cursors and overflow. Preserve
version-specific request body/query assertions and existing traversal behavior.

## Selector/deployment contract

Compare fake request transcripts before/after selector migration: ID normalization/order,
version/tag/latest, multiple IDs, missing matches, near-match tenant override, paging and
failure timing. Keep TTY listings and no-prompt machine/automation behavior. Commands may not
reimplement search or count/fallback strategy after extraction.

For deployment run preparation, preserve definition-key/ID/BPMN precedence, resource order,
multiple process IDs, duplicates, malformed XML and no-ID errors. Prove preparation issues
zero HTTP requests. Command tests preserve tenant/version/options and deployment-result
rendering before creation; later creation failure retains the established deployment evidence,
exit class and result count. Do not redesign the existing staged output contract.

## Tooling and cleanup contract

Reference scans include production, tests, scripts, docs and generation hooks before deleting
internal/private symbols. Preserve public compatibility shims, dormant command registration
state and config binder semantics. Architecture checks follow transitive production imports;
internal packages use the neutral worker core rather than the facade-aware legacy pool path.

Parse workflow YAML; compare triggers, permissions, concurrency, setup versions, version
argument precedence, secret forwarding and publish destination. Execute shell only in a
fixture-owned PATH with make, sudo/apt, lftp, ruby/bundle/server commands replaced as applicable;
use dummy credentials and forbid network. Never run CI publication, downloads, installation,
Jekyll serving, real SFTP, cluster scripts or integration suites for this refactor.
