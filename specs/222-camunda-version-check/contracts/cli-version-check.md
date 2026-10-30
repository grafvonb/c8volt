# CLI Contract: Advisory Camunda Version Check

This contract implements [spec.md](../spec.md). It adds no command result schema or public library method.

## Flag and scope

| Input | Automatic advisory |
| --- | --- |
| Flag omitted | Enabled for operational requests. |
| `--no-version-check` / `--no-version-check=true` | Disabled; zero advisory requests and no automatic version warning. |
| `--no-version-check=false` | Enabled. |
| Quiet, verbose, JSON, keys-only, automation, auto-confirm, or no-wait | Existing precedence applies; none independently disables the check. |

The flag is persistent and can appear before or after the selected subcommand under existing flag parsing. It does not persist configuration or change the selected adapter. Help/capabilities expose the flag. It is safe but has no network effect on local commands.

Authentication bootstrap is not an operational request. A local validation failure, local preview, declined local-only operation, or local empty scope must not trigger the check. Remote discovery is operational even when it leads to an empty scope, dry-run, or later abort. No extra confirmation is added.

Each eligible invocation has one advisory attempt, including failures, shared across facades, pages, worker batches, polling, and operational retries. A new invocation starts fresh. Explicit topology/version commands still perform their requested operation separately. Disabling the automatic check never suppresses requested version results or explicit connection-test compatibility evaluation.

## Timing and continuation

The check completes immediately before the first operational HTTP call begins. A match is silent. Mismatch and failure warn and continue with the configured version if the original invocation remains active.

The check's complete waiting budget is at most two seconds, bounded further by a shorter positive HTTP timeout and inherited request/workflow deadline. It does not retry or follow redirects. A lookup blocked by shared request pacing still consumes the one check attempt and finishes within its budget, possibly without a dispatched request. Thus exactly one attempt does not guarantee one network start under cancellation or admission failure.

The operation retains its ordinary full per-request timeout after the check, subject to the unchanged parent deadline. Parent cancellation stops new operational dispatch. No detached continuation, deadline extension, adapter switch, config repair, or extra state verification is allowed.

## Warning records

Exactly one automatic WARN record is eligible for emission for a mismatch or failed attempt. Match yields none. Configured logger filtering may suppress it; quiet suppresses WARN while retaining operational ERROR behavior.

| Outcome | Message text for configured 8.9 |
| --- | --- |
| Observed 8.10.3 | `version mismatch: configured 8.9, gateway 8.10.3; using 8.9` |
| Unauthorized or forbidden | `version check: access denied; using configured 8.9` |
| Check timeout or bounded admission timeout | `version check: timeout; using configured 8.9` |
| Missing/malformed/unsafe version | `version check: invalid gateway version; using configured 8.9` |
| Other lookup/backend/redirect failure | `version check: unavailable; using configured 8.9` |
| Check canceled by its caller | `version check: canceled; using configured 8.9` |

The canceled message names the retained configuration; it is not a promise that a canceled operation will continue. Cancellation exit semantics remain those of the operation's original context.

The configured command-context logger provides severity, timestamp, format, destination, and activity integration. The message must not embed these prefixes or use direct printing. A validated version display is at most 128 bytes; failure reasons come from the fixed table, never raw error text. Normal and verbose-only output must not expose methods, URLs, bodies, credentials, resource keys, or transport chains. Existing DEBUG diagnostics remain available through `--debug` and configured DEBUG independently of verbose, honoring quiet/flag precedence and existing redaction.

## Result and diagnostic separation

- Existing human results remain unchanged. The check adds no stdout summary, even for match or disabled state.
- Existing JSON commands emit exactly one command result/envelope with EOF afterward, including operational failure or partial success. The advisory never becomes a JSON error or a second envelope.
- Keys-only output contains exactly the qualifying operational keys, one per line; no keys means zero bytes. Advisory failure cannot add a key or blank line.
- A failed advisory followed by an operational failure may emit one version WARN and the existing compact operational ERROR. These are distinct facts; preserve original operational error class, exit status, partial evidence, and quiet handling.
- Prompts remain plain configured stderr text with existing eligibility and behavior. Warnings are logger records, never prompt text.

## Explicit connection test

`config test-connection` remains remote. It performs its own actual topology request and derives its established JSON warnings from that response. Those warning strings, legacy classification (including historically permissive version inputs), and schema remain unchanged. The stricter new advisory parser must not replace this machine-evidence path.

If the automatic checker already produced a mismatch/failure outcome, suppress duplicate human compatibility-warning emission from the explicit result. This suppression uses the outcome, not whether WARN was visible. If the automatic check matched but the actual topology later differs, the explicit result may issue its existing compatibility warning. If automatic checking is disabled, explicit diagnostics continue as before. Unsafe raw values in any touched human fallback must use a bounded safe message while machine evidence stays unchanged.

## Requirement traceability

| Requirements | Contract section | Planned proof |
| --- | --- | --- |
| FR-001, FR-007, FR-009 | Flag and scope; Timing | Ordered requests; concurrent/page/poll checks; disabled/local/empty paths; independent invocations. |
| FR-002, FR-013, FR-014 | Warning records | All supported adapter versions, patch/prerelease, numeric comparison, malformed/unsafe input. |
| FR-003, FR-004, FR-010, FR-012 | Warning records; Separation | Exact WARN records, configured/inherited destinations, plain/structured logs, redaction and both DEBUG paths. |
| FR-005, FR-006 | Timing and continuation | Short probe timeout then successful operation; no retries/redirects; caller cancellation and unchanged selection. |
| FR-008, FR-015 | Flag and scope | Flag forms/placement, help/capability metadata, README and generated docs. |
| FR-011 | Result and diagnostic separation | One JSON value plus EOF, partial results and original exit, exact keys, quiet combinations. |
