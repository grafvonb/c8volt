# Contract: Camunda Version Selection and Disclosure

## Accepted inputs

| Input | Result |
|-------|--------|
| `8.10-alpha` | `V810Alpha` |
| `810-alpha` | `V810Alpha` |
| `v810-alpha` | `V810Alpha` |
| `v8.10-alpha` | `V810Alpha` |

Normalization remains trim- and case-insensitive, matching existing version behavior.

## Rejected inputs

The following values must return the existing unknown-version error and must not select alpha behavior:

- `8.10`
- `810`
- `v810`
- `v8.10`
- `8.10.0-alpha4`
- any later or earlier prerelease tag
- unknown or misspelled alpha aliases

## Defaults and fixtures

- Missing version configuration continues to select `V88`.
- `V810Alpha.String()` returns `8.10-alpha`.
- `V810Alpha.FilePrefix()` returns `C89_` by explicit compatibility decision.
- Gateway versions with major/minor `8.10` match configured `8.10-alpha`; other major/minor values produce the existing mismatch warning.

## Discovery and output

- Supported and implemented version collections include `V810Alpha` after all factories are wired.
- Stable grouping contains `8.7`, `8.8`, and `8.9`.
- Experimental grouping contains `8.10-alpha` with baseline `8.10.0-alpha4`.
- Human root help, `version`, README, and generated docs show stable and experimental support separately.
- Existing JSON envelope and existing field types remain unchanged.
- Version JSON adds explicit experimental metadata fields rather than encoding stability only in display text.

Recommended additive payload shape:

```json
{
  "supportedCamundaVersions": "8.7, 8.8, 8.9, 8.10-alpha",
  "stableCamundaVersions": "8.7, 8.8, 8.9",
  "experimentalCamundaVersions": "8.10-alpha",
  "experimentalCamundaBaseline": "8.10.0-alpha4"
}
```

## Compatibility guarantees

- Selecting 8.7, 8.8, or 8.9 produces the same factory paths and command behavior as before this feature.
- Adding alpha does not alter the default.
- No output calls `8.10-alpha` stable, generally available, or equivalent to final 8.10.
