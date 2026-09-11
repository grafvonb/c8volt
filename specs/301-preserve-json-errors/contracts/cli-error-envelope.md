# CLI Contract: Command Execution Error Envelopes

## Eligibility and scope

Applies to the audited validation/runtime paths of commands already declaring full machine-contract support, listed in [research.md](../research.md). It does not extend contract support. Bootstrap errors, argument/flag parser failures before execution, and valid raw XML retrieval/write errors remain outside the correction.

## Output and termination

| Mode | Stdout | Error diagnostic | Exit behavior |
| --- | --- | --- | --- |
| JSON, full contract | Exactly one established error envelope | No repeated human error on stderr | Existing classified code |
| JSON, full contract, `--no-err-codes` | Same error envelope | No repeated human error on stderr | Zero, with immediate termination |
| Ordinary human | Zero result bytes on these failures | Existing stderr error | Existing classified code |
| Ordinary human, `--no-err-codes` | Zero result bytes | Same stderr error | Zero, with immediate termination |
| Non-full contract | Existing behavior | Existing behavior | Existing behavior |

JSON has existing precedence over keys-only. Quiet must not suppress explicitly selected JSON. Supported automation does not change failure classification or introduce human result text. Preserve existing guards where automation is unsupported. Existing unrelated stderr context is not a duplicate failure diagnostic; do not refactor tenant or activity reporting.

The consumer must be able to decode one JSON value and then reach EOF. No success or accepted envelope may precede/follow the error. Do not add payload, suggestion, or tenant-context fields where the existing error renderer omits them. See [data-model.md](../data-model.md).

## Validation semantics

- Search-flag, stdin-key, and XML-option validation failures retain `outcome: invalid` and `class: invalid_input`.
- For stdin beginning with `filter: `, retain the existing instruction to use `--keys-only`; do not append the generic malformed-key diagnostic.
- For other invalid stdin keys, retain quoted key text, processed zero-based index, and existing guidance.
- `get process-definition --xml --json` remains rejected. With no key, preserve the existing missing-key error priority; with a key, preserve the incompatible-options error. Both are command-execution validation failures eligible for a JSON error envelope.
- Runtime classification is not forced to `invalid`: unavailable, malformed response, not found, and other existing classes retain their established mappings.
- Empty embedded-file scope retains `outcome: failed` and `class: local_precondition`.

Illustrative exact stdin filter error, assuming no attached tenant context:

```json
{
  "outcome": "invalid",
  "class": "invalid_input",
  "command": "get process-instance",
  "detail": {
    "message": "validating keys from stdin failed: use --keys-only flag to get only keys as input",
    "class": "invalid_input"
  }
}
```

Whitespace layout is not contractual. Existing classification and normalization remain authoritative; tests assert the actual error's precise message rather than a new template.

## Validation obligations

Each reachable corrected path must have command-path coverage for JSON/human output and ordinary/suppressed exit status. Capture streams separately, require EOF after one decoded value, assert precise class/outcome/detail/command, and verify no subsequent work. Use both invalid stdin forms across every current helper caller. Exercise the additional runtime/validation paths, existing successes, mode precedence, and non-full fallback described in [plan.md](../plan.md).

Embedded failure tests invoke the same production runner with a deterministic listing dependency; real CLI success tests prove its wiring. Do not claim renderer-write failure coverage when the existing renderer cannot return such failures. No schema version or capability metadata changes are authorized by this contract.
