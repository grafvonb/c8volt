# CLI Contract: v5 Flag Intent

## Scope and compatibility

The [specification rename table](../spec.md#functional-requirements) is the complete mapping of the 16 rename rows. Every listed canonical spelling replaces its old spelling in v5.0.0. There are no deprecated aliases or transition warnings. Command names/aliases, unaffected short options, public-library fields, JSON properties, and result schemas retain existing contracts. Retain existing short options only where their meaning is unchanged; do not allocate new short forms as part of this issue.

Removed names are absent from accepted flag registrations, help, capabilities, and generated completion. A diagnostic lookup may recognize them solely to report the replacement. Renaming applies to supported modes only; it never makes an unsupported combination valid.

## Canonical grammar examples

```text
update pi --bid ORDER-123 --set-vars '{"approved":true}'
update element --key KEY --set-vars-file variables.json
update ut --key KEY --set-assignee bob
update ut --key KEY --complete --set-vars '{"approved":true}'
run pi --bpmn-process-id order --set-bid ORDER-123 --set-vars '{"approved":false}'
update job --key KEY --set-retries 3 --set-timeout 5m --timeout 30s
update job --key KEY --fail --set-retries 0 --set-retry-backoff 5m --set-error-message 'worker unavailable'
update job --key KEY --throw-bpmn-error --set-error-code PAYMENT_DECLINED --set-error-message 'card declined'
update job --type cleanup --retries 0 --complete --set-bid ORDER-123 --set-vars '{"approved":true}'
ops repair incident --key KEY --set-retries 3 --set-job-timeout 5m --set-vars-file variables.json
ops repair pi --key KEY --set-retries 0 --dry-run
get pd --key KEY --with-stats
embed list --with-paths
get incident --no-error-message
```

These are grammar examples, not live validation commands. Existing key formats, confirmation/automation requirements and output support apply. Retry and timeout updates may combine only as permitted before migration; the example preserves that existing mode.

## Job intent and failure matrix

| Invocation / condition | Required meaning / outcome |
| --- | --- |
| `get job --retries 0` | Exact zero predicate, no mutation. |
| Keyless `update job --complete --retries 0` with existing valid selectors | Exact zero predicate, existing completion payload and eligibility. |
| Keyed completion plus `--retries` | Invalid combination; do not broaden existing selection support. |
| Retry update or technical failure plus bare `--retries` | Invalid input; guide to `--set-retries`, no fallback. |
| Non-completion mutation with both retry spellings | Reject before confirmation/mutation, including equal values and zero. |
| Completion plus `--set-retries` | Reject; mutation retries do not become a predicate or supported completion payload. |
| `--set-timeout 5m --timeout 30s` in valid job update | Job mutation five minutes; effective HTTP timeout thirty seconds. |
| Only `--set-timeout` | Job mutation only; HTTP timeout resolves independently from existing configuration/defaults. |
| Only key plus HTTP `--timeout` | No mutation selected: invalid input with `--set-timeout` guidance. |
| Another valid canonical mutation plus HTTP `--timeout` | Valid execution control; never a job mutation alias. |
| `--throw-bpmn-error --set-error-code CODE` | Canonical error action and value; retain existing code validation. |
| Error code without active error action or missing/empty/blank required code | Reject before confirmation/mutation under existing code validation rules. |
| `--throw-bpmn-error CODE` or any `--throw-bpmn-error=VALUE` | Reject legacy value syntax with action-plus-code guidance. Includes empty, `true`, `false`, numeric-looking codes and mixed canonical/legacy forms. |
| `--throw-bpmn-error=false --set-error-code CODE` | Reject attached action value; never dispatch by presence alone or silently discard code. |
| Error code supplied as `--set-error-code true` or `false` | Preserve as an opaque code if otherwise valid; these are values of the code flag, not action booleans. |
| `--fail`, `--complete`, BPMN action in incompatible combinations | Existing mutual exclusion remains. |

The canonical BPMN action accepts only the bare token. An attached value is always migration-invalid. The token validator must distinguish a real action flag from that text inside another flag's value or following `--`. Existing positional validation remains responsible for post-terminator positionals and stdin-dash eligibility.

## Exact values and defaults

- Preserve supplied-versus-omitted state and existing validation at every mapping boundary.
- Do not trim/split opaque BID, identity, message or code values merely because flag names change.
- Preserve existing variable number decoding, including exact task-completion numeric behavior, and variable-file errors.
- Preserve supported payload-source exclusions; old/new conflicts reject even if values match.
- Repair `--set-retries 0` skips retry restoration. It does not write a literal retry count of zero. A separate skip control or changed zero semantics is outside scope.
- Existing `--set-bid` means assignment, `--bid` remains selection except that creation now takes `--set-bid`. Existing assignment eligibility/version rules remain authoritative.

## Rejection and output contract

Removed spelling → classified invalid input → one compact canonical replacement explanation using the configured command-context logger. No deprecation alias executes. Old/new input together rejects before confirmation or mutation. Migration messages describe syntax without reproducing arbitrary payloads, full argv, URLs, or transport chains.

For affected early parse/argument errors, configuration/profile/logging/output controls must work even when supplied after the rejected token. Recover reporting context only, respecting actual flag arity and `--`; do not reinterpret payload values as flags. Local preparation does not authenticate, check engine version, read mutation files, consume stdin, or install remote clients. If local configuration itself cannot load, use the existing bootstrap fallback helpers rather than a new logger or direct stderr write.

| Output class | Destination and behavior |
| --- | --- |
| Human errors/warnings | Configured logger and severity; compact errors survive quiet. No raw HTTP detail at normal or verbose levels. |
| Human mutation/dry-run/verbose details | Existing INFO logger helpers; suppressed at WARN and under quiet. |
| Query rows and selected-format results | Configured stdout result renderers, existing schemas. |
| Shared JSON | Exactly one established envelope through EOF, including invalid input and partial failures; no second error renderer. Preserve existing command-specific empty/null rules. |
| Keys-only | Existing key meaning, one key per line, no diagnostic text; empty result is zero bytes. |
| Interactive control text | Plain configured/inherited stderr with existing prompt eligibility, wording, defaults, abort and EOF behavior. |
| DEBUG diagnostics | Existing redacted path, available via debug flag or configured level independently of verbose; no diagnostic records on result stdout. |

Output mode combinations and precedence remain as currently supported; this issue does not make JSON/verbose or other unsupported combinations valid. Existing invalid-input exit classification and no-err-codes override remain intact. Prepared migration errors do not enter normal mutation execution or emit duplicate errors.

## Acceptance traceability

| Requirements | Acceptance coverage |
| --- | --- |
| FR-001, FR-004, FR-006 | Full rename table, unchanged native payloads/defaults/modes, exact values, BID separation, repair zero. |
| FR-002, FR-003, FR-005 | Job matrix, explicit zero, valueless action/legacy Boolean-looking codes, HTTP configuration and payload independence. |
| FR-007, FR-008 | Removed names, old/new conflicts, early parser/Args failures and replacement guidance. |
| FR-009, FR-014 | Negative scope audit of unchanged selectors, controls, public fields, result schemas and supported combinations. |
| FR-010–FR-012 | Configured/inherited logging, separate streams, severity, JSON EOF, partial evidence, real-terminal prompts/empty scope, independent DEBUG/redaction. |
| FR-013 | Source help/examples, README, capabilities/completion, generated docs and maintained invocation fixtures agree. |
