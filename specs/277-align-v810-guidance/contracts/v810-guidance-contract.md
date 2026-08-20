# Contract: Camunda 8.10 Guidance Consistency

## Maintainer Version Contract

Every current maintainer inventory that claims to enumerate supported Camunda runtime adapters or generated contracts must include:

- V87 / `v87`
- V88 / `v88`
- V89 / `v89`
- V810 / `v810`

V810 is the newest supported runtime line. V89 is the default when configuration does not select a version. "Newest supported" and "default" must never be used as synonyms.

Version-neutral interfaces remain above version-specific adapters. Adding V810 to guidance does not permit commands or public facades to depend directly on generated Camunda contracts.

## Operator Identity Contract

| Property | Required value |
|----------|----------------|
| Canonical identity | `8.10` |
| Accepted aliases | `8.10`, `810`, `v810`, `v8.10` |
| Default | `8.9` |
| Active baseline | `8.10.0-alpha4` |
| Baseline status | prerelease |
| Update model | replace the V810 baseline in place |

Alpha, release-candidate, final-release, and patch source identifiers are provenance, not additional configuration aliases.

## Gateway Diagnostic Contract

| Observed value for configured `8.10` | Compatibility classification | Diagnostic behavior |
|--------------------------------------|------------------------------|---------------------|
| `8.10` | match | none |
| `8.10.x` | match | none |
| 8.10 prerelease | match | none |
| another major/minor | non-match | established mismatch diagnostic |
| empty | unverifiable | established empty-version diagnostic |
| unparseable | unverifiable | established unrecognizable-version diagnostic |

A non-match is not accepted as compatible. Issue #277 does not convert these diagnostics into command failures.

## Embedded Definition Contract

- V810 listing, export, deployment, and smoke selection use native C810 definitions.
- No active guidance may permit V810-to-C89 fallback.
- C89 may be named as the behavioral source from which C810 definitions were derived.
- Historical records of the former C89 selection remain valid history only when the #275 supersession context is preserved.

## Guidance Ownership Contract

| Guidance kind | Source of truth | Update rule |
|---------------|-----------------|-------------|
| Repository implementation guidance | Current repository instructions and Ralph rules | Correct in place |
| Architecture facts and synthesis | Current architecture memory | Correct observable support facts in place |
| Gateway requirement | #273 specification plus its version-selection contract | Align normative wording with the detailed contract |
| Fixture selection | #275-finalized active #273 design | C810 only |
| Operator template and help | Authored template and command metadata | Correct source, test rendered output |
| Operator overview | README and version metadata | Change only when a concrete mismatch exists |
| Generated documentation | Existing documentation workflow | Never hand-edit; regenerate from source |
| Live integration guidance | Existing stable integration matrix | Preserve V87–V89 scope |
| Completed implementation history | #273 tasks, progress, and Ralph memory | Preserve with supersession context |

## Protected Boundaries

Except for the intentional omitted-version promotion from V88 to V89, the refinement must leave unchanged:

- CLI commands, flags, exit behavior, human output, and structured output;
- explicit runtime service selection and version-specific behavior;
- configuration normalization rules other than the V89 fallback;
- generated Camunda clients and provenance;
- C87, C88, C89, and C810 BPMN definitions;
- live integration profiles, scripts, targets, and scenarios;
- the active Camunda 8.10 source baseline.
