# Research: Ops Paged Discovery Scope

## Decision: Use service-owned paged discovery helpers for each candidate type

**Rationale**: Architecture memory and `specs/ralph-implementation-rules.md` require ops playbooks to compose service capabilities rather than infer completeness in commands. Existing `DiscoverRetentionProcessInstances` and `DiscoverOrphanProcessInstances` already show the local pattern: fetch pages, apply any local candidate filtering, append matches, stop on end-of-results or explicit limit, and preserve deterministic order.

**Alternatives considered**:
- Command-layer pagination: rejected because commands should pass flags and render service results, not own backend discovery mechanics.
- Increasing the first-page size: rejected because it still silently truncates large populations and keeps `--batch-size` acting like a scope cap.

## Decision: Represent discovery completeness explicitly in domain and facade results

**Rationale**: The issue requires JSON and Markdown reports to expose whether discovery completed fully or was user-limited. Notices alone are too weak because current bounded notices are warning-like hints rather than a stable machine contract. A small status shape attached to frozen discovery result models lets all renderers and reports use the same source.

**Alternatives considered**:
- Continue emitting only `bounded_search_scope` notices: rejected because success paths need positive full-completion evidence, not only a hint when counts reach page size.
- Infer completeness from counts in renderers: rejected because renderer inference can drift from service discovery rules.

## Decision: Keep `--batch-size` as page-size only and `--limit` as total cap

**Rationale**: This is the explicit operator contract in the issue and matches existing process-instance search conventions in README. The discovery loop should request pages of `--batch-size`, append candidate matches, and stop early only when cumulative matching candidates reaches `--limit`.

**Alternatives considered**:
- Treat `--batch-size` as both page size and default cap: rejected because it is the defect.
- Add a new cap flag: rejected because `--limit` already expresses total scope.

## Decision: Add process-definition page support instead of reusing one-shot search

**Rationale**: `ops purge all-process-definitions` currently depends on `SearchProcessDefinitions` and `SearchProcessDefinitionsLatest`, which return one bounded list. Generated v8.9 responses expose page metadata and request builders already include page objects, so a page-capable internal service API is the smallest route to full discovery without leaking generated clients into ops code.

**Alternatives considered**:
- Repeatedly call existing search with larger sizes: rejected because it cannot advance through pages.
- Command-specific generated client use: rejected by layering rules.

## Decision: Preserve related smoke-test cleanup as follow-up unless shared discovery makes it trivial

**Rationale**: The issue names smoke-test process-definition cleanup eligibility as a related safety path to review in the same fix or a follow-up, while the definite affected workflows are purge/repair commands. Planning keeps the primary stories narrow and independently verifiable; if a shared process-definition/process-instance paging helper directly removes the smoke-test risk, it may be included, otherwise it should be captured as follow-up.

**Alternatives considered**:
- Make smoke-test cleanup eligibility mandatory in this feature: rejected because it risks widening a defect fix beyond the issue's definite affected workflows.

## Decision: Validate with targeted service tests before broader command/docs checks

**Rationale**: The defect lives in service discovery and frozen-scope semantics. Service tests can prove multi-page collection, limit stops, and frozen reuse cheaply. Command tests then verify prompts, automation, reports, and help text use the returned service state.

**Alternatives considered**:
- Only subprocess tests: rejected because they are slower and less precise for discovery loop edge cases.
