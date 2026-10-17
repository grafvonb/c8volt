# Planning Context: Issue #339

Source: [issue #339](https://github.com/grafvonb/c8volt/issues/339), read 2026-10-17. These delivery constraints accompany the user-facing specification and must carry into planning and tasks.

## Ownership and architecture

- Keep CLI validation and rendering under `cmd/`, public facades thin, backend mapping in internal services, and version-specific behavior in the matching adapter.
- Keep generated clients and provenance unchanged. No new public suspension/resumption mutation workflow is required.
- #340 owns suspended-instance inclusion in definition deletion and purge; #341 owns cancellation/deletion orchestration and eligibility. #342 owns combined integration validation. None is a prerequisite for #339.

## Exact generated-client baseline

Read from `internal/clients/camunda/v810/camunda/provenance.json`:

- Upstream repository: `https://github.com/camunda/camunda.git`
- Tag: `8.10.3`
- Commit: `3961b79c3fdaf43c48691ed77672d6db42d0abcc`
- Source specification: `zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml`
- Source SHA-256: `857109dabcb998801f749282836a6d2af1a137284d538161801f0c348ae57837`
- Prepared specification SHA-256: `461a1cc051a021a3cf937dbfd28a92537cf9b8b4381bcfb9ad6ab6e1d1f91d1b`
- Generated client SHA-256: `589351ac972ab595af721f2e9647a273819e56a9601f7af2aaaf98e6def2f319`
- Recorded tools: Redocly `2.29.2`; oapi-codegen `2.5.0`.

This client baseline is distinct from the required **Camunda 8.10.4 server validation target**. Do not update the client or provenance to make their version labels match.

## Required validation and documentation work

- Include targeted domain, public-facade, adapter, and command tests for the specification's acceptance scenarios.
- Verify the actual 8.10 suspended search mapping and absence of unsupported requests for explicit suspension on configured 8.8/8.9, through CLI and public facade entry points.
- Use deterministic tests for creation-observation timing and present-to-absent timestamp refresh, including relevant watch behavior.
- Validate state lookup, filtering, expectations, and metadata against Camunda 8.10.4. Setup may use the existing generated suspension/resumption operations.
- Place reusable integration fixtures in the existing test structure so #342 can reuse them; do not introduce a separate test framework.
- Record observed server version, client provenance, tested revision, and validation results. Explicitly report unavailable live validation; never represent it as passing.
- Update affected command metadata, README guidance, and regenerate CLI documentation with `make docs-content` rather than editing generated documentation manually.
- Keep interface changes, implementations, fixtures, and required test-double updates together in compiling work units. Include formatting and relevant validation in those units.
- This specification-only phase uses document review and lightweight consistency checks; it makes no runtime or live-validation claim.
