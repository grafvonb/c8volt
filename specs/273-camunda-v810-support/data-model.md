# Data Model: Camunda 8.10 Support

This feature adds no persistent business database. The model describes configuration, generated-source evidence, compatibility selection, and service-boundary records represented in Go values and checked-in files.

## CamundaVersion

| Field | Type | Rules |
|-------|------|-------|
| canonical | string | `8.10` for V810; unique |
| aliases | set | `8.10`, `810`, `v810`, `v8.10`; trim/case-insensitive |
| supported | boolean | true after complete support |
| implemented | boolean | true after all eleven factories exist |
| default | boolean | false for V810; V89 is the current default after issue #277 superseded the original V88 default |

Alpha/RC/patch tags are never aliases. Unknown input returns the existing error. Display/report identity stays `8.10` for every baseline.

## CamundaBaseline

| Field | Type | Initial value/rules |
|-------|------|---------------------|
| versionIdentity | CamundaVersion | V810 |
| status | enum | `prerelease`; later RC/final |
| tag | string | `8.10.0-alpha4` |
| commit | SHA | `4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6` |
| repository | URL | canonical Camunda product repository |
| sourceSpec | path | product v2 OpenAPI source |

Exactly one active baseline belongs to V810 and produces one client/provenance pair. State advances in place:

```text
alpha -> newer alpha -> release candidate -> final
```

Concurrent active baselines are forbidden; replacing one does not create another version or adapter family.

## GenerationProvenance

| Field | Type | Rules |
|-------|------|-------|
| schemaVersion | scalar | required/recognized |
| repository, tag, commit | strings | equal active baseline |
| sourceSpec, sourceSpecSha256 | path/digest | allowlisted path and SHA-256 |
| transformations | ordered list | script path and SHA-256 |
| tools | object | exact Redocly/oapi-codegen versions |
| preparedSpecSha256 | digest | post-mutation SHA-256 |
| generatedClientSha256 | digest | published client SHA-256 |
| command | string | canonical reproduction command |

Paths cannot escape the target; all fields are deterministic; timestamps, host paths, and temp paths are excluded. Client digest must agree before publication.

## GeneratedClientArtifact

| Field | Type | Rules |
|-------|------|-------|
| package | string | `camunda` |
| path | path | `internal/clients/camunda/v810/camunda/client.gen.go` |
| sourceKind | enum | Orchestration Cluster v2 only |
| syntaxValid | boolean | true before publication |
| requiredSymbolsPresent | boolean | true for adapters |

No V810 Operate, Tasklist, or Administration SM sibling exists. Target generation cannot alter v87-v89 and is diff-free on repeat.

## ServiceCompatibility

| Field | Type | Rules |
|-------|------|-------|
| family | enum | one of eleven families |
| version | CamundaVersion | V810 |
| adapter | path | `internal/services/<family>/v810` |
| generatedClient | path | V810 unified client only |
| interfaceSatisfied | boolean | compile-time true |
| factorySelected | boolean | test-verified true |
| outcome | enum | supported or operation-specific unsupported |
| representativeProof | list | success/error/malformed/mutation tests |

Exactly eleven records exist. Each implements the existing version-neutral API. Process instances construct the V810 variable service internally.

## CapabilitySet

| Field | Type | Rules |
|-------|------|-------|
| name | identifier | behavior-oriented |
| supportedVersions | explicit set | `{V89,V810}` for full PD-history deletion |
| consumers | paths | direct deletion and bulk purge |

Membership never uses ordinal/string comparison. Unsupported mutations fail before remote mutation as required.

## GatewayCompatibility

| Field | Type | Rules |
|-------|------|-------|
| configured, observed | strings | raw identities |
| configuredLine, observedLine | major.minor | parsed release lines |
| result | enum | match, mismatch, unrecognizable |
| diagnostic | string | empty on match; stable detail otherwise |

`8.10` matches `8.10`, `8.10.0`, and `8.10.0-alpha4`. Different major/minor is mismatch; empty/unparseable is never a match.

## FixtureCompatibilityMapping

| Field | Type | Rules |
|-------|------|-------|
| version | CamundaVersion | V810 |
| sourcePrefix | string | `C810_` |
| scope | enum | production embedded/smoke fixtures only |
| reportedVersion | string | `8.10` |

The mapping creates no live integration profile or fixture. Unknown versions do not inherit the latest prefix.
