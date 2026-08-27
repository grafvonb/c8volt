# Contract: Camunda 8.10 Version Selection and Disclosure

## Accepted Inputs

Normalization is trim- and case-insensitive.

| Input | Result |
|-------|--------|
| `8.10` | `toolx.V810` |
| `810` | `toolx.V810` |
| `v810` | `toolx.V810` |
| `v8.10` | `toolx.V810` |

## Rejected Inputs and Default

Source/release identifiers such as `8.10-alpha4`, `8.10.0-alpha4`, release-candidate/patch variants, and unknown aliases return the existing unknown-version classification. Missing configuration selects V89; issue #277 supersedes the V88 default used when V810 was originally delivered.

## Supported and Implemented Discovery

- Supported versions: `8.7, 8.8, 8.9, 8.10`.
- Implemented versions use the same set only after all eleven V810 factories exist.
- V810 display/report identity is always `8.10`.
- Baseline tag/status is separate metadata and never changes configuration identity.

## Gateway Compatibility

| Configured | Observed | Result |
|------------|----------|--------|
| `8.10` | `8.10` | match |
| `8.10` | `8.10.0` | match |
| `8.10` | `8.10.0-alpha4` | match |
| `8.10` | another major/minor | mismatch diagnostic |
| `8.10` | empty/unparseable | unrecognizable diagnostic |

Mismatch/unrecognizable values are never accepted as compatibility matches. They emit the established diagnostics and do not create a mandatory command failure. Stable-version diagnostic behavior remains unchanged.

## Human Version Output

Human output stays compact and adds one disclosure line:

```text
Supported Camunda versions: 8.7, 8.8, 8.9, 8.10
Camunda 8.10 baseline: 8.10.0-alpha4 (prerelease)
```

## JSON Version Output

The shared command envelope and existing string fields remain unchanged. Payload additions are string fields:

```json
{
  "version": "<c8volt version>",
  "commit": "<commit>",
  "date": "<date>",
  "supportedCamundaVersions": "8.7, 8.8, 8.9, 8.10",
  "camunda810Baseline": "8.10.0-alpha4",
  "camunda810BaselineStatus": "prerelease"
}
```

Baseline updates change only the two baseline values.

## Compatibility Guarantees

- Existing JSON fields retain types/meaning.
- Root help and `--camunda-version` list V810 without alpha-specific identity.
- Command text uses capability-accurate “8.8 or newer” or “8.9 or newer” once V810 is verified.
- Operational human, JSON, keys-only, quiet, automation, prompt, activity, and exit behavior remains unchanged.
- `capabilities --json` schema remains `v1`; it is not runtime negotiation.
