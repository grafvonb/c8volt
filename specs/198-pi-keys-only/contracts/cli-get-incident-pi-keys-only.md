# CLI Contract: `get incident --pi-keys-only`

## Commands

- `c8volt get incident --pi-keys-only [filters]`
- `c8volt get incidents --pi-keys-only [filters]`
- `c8volt get inc --pi-keys-only [filters]`
- `c8volt get incident --key <incident-key> --pi-keys-only`
- `printf '%s\n' <incident-key> | c8volt get inc --pi-keys-only -`

## Output

When enabled, output is line-oriented:

```text
<process-instance-key>
<process-instance-key>
```

Rules:

- Each selected incident item with a non-empty process instance key contributes one output line.
- Duplicate process instance keys are preserved.
- Incident items without process instance keys are skipped.
- No human row fields, JSON envelope, message text, or `found:` footer are emitted.

## Compatibility

The flag is mutually exclusive with:

- `--keys-only`
- `--json`
- `--total`
- `--error-message-limit`
- `--with-no-error-message`

The command must fail locally for those combinations before remote calls.

## Examples

```sh
c8volt get incident --state active --error-type job_no_retries --pi-keys-only
```

```sh
c8volt get incident --state active --error-type job_no_retries --pi-keys-only | c8volt cancel pi --auto-confirm -
```

```sh
c8volt get incident --key 2251799813685249 --pi-keys-only
```

## Non-Goals

- `--pi-keys-only` does not dedupe output.
- `--pi-keys-only` does not fetch process instance details.
- `--pi-keys-only` does not change `--keys-only`, which continues to emit incident keys.
