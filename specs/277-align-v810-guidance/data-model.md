# Data Model: Camunda 8.10 Guidance Alignment

This feature changes documentation state rather than runtime data. The model defines the guidance records and relationships that must remain consistent.

## V810 Compatibility Guidance

| Field | Required value | Validation rule |
|-------|----------------|-----------------|
| operator identity | `8.10` | Exactly one current V810 identity exists |
| accepted aliases | `8.10`, `810`, `v810`, `v8.10` | Source tags, prerelease tags, and patch identifiers are not aliases |
| newest supported runtime | V810 | Must be distinguished from default selection |
| default runtime | V89 | Intentionally supersedes the V88 default recorded by #273 |
| active source baseline | `8.10.0-alpha4` prerelease | Disclosed separately from operator identity |
| baseline lifecycle | replace in place | Later 8.10 sources do not create parallel identities |
| service-adapter family | native V810 | Listed alongside V87, V88, and V89 |
| generated-contract family | native V810 | Listed alongside V87, V88, and V89 |
| embedded family | C810 | No active C89 fallback permitted |
| live integration coverage | stable versions through 8.9 | No V810 integration profile is implied |

## Guidance Record

| Field | Meaning | Allowed values |
|-------|---------|----------------|
| audience | Who relies on the statement | maintainer, operator, reviewer |
| topic | Contract being described | version inventory, architecture boundary, gateway compatibility, baseline disclosure, embedded selection, integration scope |
| authority | Source decision governing the statement | #273, #275, current repository behavior |
| lifecycle | How the record should be treated | active normative, generated derivative, integration-only, historical record |
| expected statement | Current value or outcome | One value from the V810 guidance or gateway matrices |
| edit policy | Permitted refinement action | correct in place, regenerate from source, preserve scoped matrix, preserve with supersession context |

### Lifecycle Rules

- **Active normative** records define current behavior and must contain no contradiction.
- **Generated derivative** records must be changed only through their owning source and generator.
- **Integration-only** records may intentionally end at V89 because live V810 integration remains out of scope.
- **Historical record** entries retain the wording of completed work and must not be treated as current alternatives.
- A historical record does not transition back to active. A later decision is represented by active guidance and an explicit supersession relationship.

## Gateway Release-Line Result

| Configured identity | Observed gateway value | Classification | Required operator-visible result |
|---------------------|------------------------|----------------|----------------------------------|
| `8.10` | `8.10` | match | no compatibility warning |
| `8.10` | `8.10.x` | match | no compatibility warning |
| `8.10` | 8.10 prerelease | match | no compatibility warning |
| `8.10` | another major/minor | diagnostic non-match | mismatch diagnostic; no new mandatory failure |
| `8.10` | empty | unverifiable | empty-version diagnostic; no new mandatory failure |
| `8.10` | unparseable | unverifiable | unrecognizable-version diagnostic; no new mandatory failure |

### Validation Rules

- Matching is based on the observed major/minor release line, not exact source-tag equality.
- A diagnostic non-match is never described as compatible.
- "Not a compatibility match" does not mean that issue #277 adds a hard-failure path.

## Embedded Definition Guidance

| Concept | Current state | Historical relationship |
|---------|---------------|-------------------------|
| V810 runtime selection | C810 only | Replaces the temporary C89 mapping |
| C810 workflow behavior | Derived from the corresponding C89 workflow | Derivation does not authorize C89 selection |
| #273 active design artifacts | Describe native C810 selection | Corrected by #275 |
| #273 checked C89 tasks/progress/memory | Preserved historical delivery record | Superseded by #275 and not active guidance |

## Artifact Classification

| Artifact category | Lifecycle | Planned handling |
|-------------------|-----------|------------------|
| Durable repository instructions and current architecture facts | active normative | Correct all supported-version inventories |
| #273 specification gateway requirements | active normative | Clarify diagnostic semantics |
| #273 fixture specification, plan, research, data model, quickstart, and service contract | active normative | Verify C810 consistency; change only on concrete drift |
| #273 checked tasks, progress, and Ralph memory | historical record | Preserve; retain existing #275 supersession context |
| Integration profile and real-state matrices | integration-only | Preserve stable V87–V89 scope |
| Configuration example template | active operator guidance | Add V810 and default disclosure |
| Authored command help | active operator guidance | Complete the gateway diagnostic matrix |
| README, API guide, CLI landing page, and command metadata | active operator source | Verify against contract; edit only on concrete drift |
| Generated docs homepage and CLI pages | generated derivative | Regenerate only through the established workflow |
