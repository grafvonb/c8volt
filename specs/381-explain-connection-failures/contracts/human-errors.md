# CLI Contract: Known Connection and Authentication Failures

**Spec**: [../spec.md](../spec.md) | **Plan**: [../plan.md](../plan.md)

## Scope

This contract applies to callers of the existing shared bounded human formatter, including process-definition search, and to compatibility of the existing connection-test wrapper. It introduces no command, flag, public Go interface, or JSON schema. Existing command-specific messages remain authoritative.

## Selection order

1. Preserve the existing class-derived reason and request-rate-wait suffix when local admission evidence exists. Connection evidence cannot override it.
2. Otherwise return a recognized bounded connection/authentication reason from the connection helper.
3. On cancellation or no recognized evidence, return the existing class label with underscores replaced by spaces. Do not manufacture a next check.

The connection-test wrapper retains its prefix, cancellation text, topology-specific endpoint advice, and existing class-only fallback. Both callers share recognized-cause wording rather than maintaining parallel mappings.

## Human reason contract

`<target>` means `authentication service` when the existing annotation identifies that stage, otherwise `gateway`. Recognition and attribution use typed evidence only.

| Existing evidence | Ordinary shared human reason |
| --- | --- |
| Refused connection | `<target> connection refused; check the configured host and port and that the service is running` |
| Reset / unexpected EOF | `<target> connection interrupted; check the service and proxy` |
| Timeout | `<target> timed out; check service availability, network access and configured timeout` |
| DNS failure | `cannot resolve <target> host; check the configured hostname and DNS` |
| Certificate validation failure | `<target> TLS certificate validation failed; check the certificate trust, hostname and expiry` |
| Invalid response | `invalid <target> response; check the configured URL and proxy routing` |
| HTTP 401 | `<target> HTTP 401: authentication rejected; check the selected profile and credentials` |
| HTTP 403 | `<target> HTTP 403: access denied; check credentials, roles and API permissions` |
| Gateway HTTP 404 | `gateway HTTP 404: endpoint not found; check the Camunda API base URL and proxy routing` |
| Authentication HTTP 404 | `authentication service HTTP 404: authentication endpoint not found; check the authentication URL and proxy routing` |
| Authentication HTTP 400 | `authentication service HTTP 400: authentication request rejected; check client credentials and scope configuration` |
| HTTP 429 | `<target> HTTP 429: rate limit reached; retry later or reduce request rate` |
| HTTP 500 / 502 / 503 / 504 | `<target> HTTP <status>: service error; check service health and proxy logs` |
| Other existing nonzero typed status | `<target> HTTP <status>: request rejected; check the selected profile and service configuration` |
| Authentication annotation without a more specific cause | `authentication failed; check the authentication configuration; use --debug for diagnostic details` |

Use the existing cause-selection precedence rather than table row order. Bare classes alone do not establish a connection cause. A normalized error whose original type was discarded receives its established fallback.

Original regression example, including its unchanged caller prefix:

```text
search process definitions: authentication service connection refused; check the configured host and port and that the service is running
```

For `config test-connection`, gateway 404 remains `topology endpoint not found`, cancellation remains `connection test canceled`, and existing setup/class-only/unknown advice remains unchanged.

## Streams, logging, and confidentiality

| Mode / condition | Required behavior |
| --- | --- |
| Normal human failure | One final ERROR through the configured command-context logger; stdout has zero human diagnostic bytes |
| Verbose | Same bounded final cause; existing safe operational details only; no extra low-level HTTP diagnostics |
| Quiet | Keep the compact ERROR, suppress informational output; preserve existing quiet/DEBUG precedence |
| Plain and structured logger formats | Respect configured format, severity, and writer; do not insert prefixes into message text or bypass the logger |
| Configured or inherited diagnostic writer | Route all diagnostic records there; no duplicate on process stderr when a separate writer is selected |
| `--debug` or configured DEBUG | Preserve existing redacted DEBUG evidence independently of verbose; stdout remains diagnostic-free |
| Repeated batch failures | Preserve aggregate prefix and counts, one final bounded ERROR, and no per-resource transport-error dump |

Ordinary and verbose human diagnostics expose no credentials, full URLs, request methods/bodies, response bodies, cursors, wrapped transport chains, or raw per-resource failures. DEBUG retains existing redaction rules. Fixed status labels such as `HTTP 401` are safe bounded facts, not raw HTTP detail.

## Machine and execution compatibility

- Original machine schemas, error detail/evidence, class, outcome, and partial results remain unchanged.
- Shared-contract JSON produces one envelope followed by EOF, without a second human error renderer. Human safety assertions do not remove established machine evidence.
- Exit classification and `--no-err-codes` behavior use the original error in all modes.
- Requests, retry counts, confirmations, mutation submission, cancellation, successful results, and output-mode precedence remain unchanged. Formatting adds zero requests.
- Existing command support boundaries remain unchanged; this feature does not expand JSON support to connection-test failures or migrate unrelated callers.

## Acceptance evidence

Formatter cause/precedence tests prove recognition; real command execution with isolated service dependencies proves routing, redaction, envelopes, and exits. Both are required. Multi-resource tests retain partial results and exact request counts. Checks must reject a direct stderr print or raw joined error even when it contains the expected operation prefix. See [../quickstart.md](../quickstart.md) for execution steps.
