# Research: Experimental Camunda 8.10 Alpha Support

## Decision 1: Pin the annotated release to its resolved commit

- **Decision**: Support only Camunda tag `8.10.0-alpha4`, whose annotated tag object `f2995f6e9dfedf97ffbc28a31ff5aac90edb4eea` resolves to commit `4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6`.
- **Rationale**: A tag alone is weaker provenance for a prerelease. Recording both tag and commit makes later reproduction and drift detection deterministic.
- **Alternatives considered**:
  - Track the latest 8.10 prerelease automatically: rejected because it makes support claims move without review.
  - Record only the tag: rejected because the specification requires a resolved revision.

## Decision 2: Generate only the unified product v2 client

- **Decision**: Generate `internal/clients/camunda/v810alpha/camunda/client.gen.go` from `zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml`. Do not generate alpha Operate, Tasklist, or Administration SM clients.
- **Rationale**: Camunda 8.10 removes the legacy Operate and Tasklist component APIs. The pinned product v2 contract exposes every method required by the current v8.9 unified service contracts except the intentionally removed Tasklist fallback method.
- **Alternatives considered**:
  - Copy or reuse v8.9 legacy clients: rejected because those contracts do not establish 8.10 runtime compatibility.
  - Generate legacy clients from current docs: rejected because no pinned 8.10 component contract exists and the upstream APIs are removed.

## Decision 3: Add a target-isolated generation mode

- **Decision**: Extend the existing `api/` workflow with an explicit `v810alpha` target that fetches into a temporary checkout, generates to a temporary output, and may publish only the alpha client plus provenance. Preserve the existing all-client workflow unchanged.
- **Rationale**: The current generator has a fixed multi-version output list and regenerates v8.9 and v8.8 from the same fetched product source. Passing an alpha tag through that path would overwrite stable clients.
- **Alternatives considered**:
  - Run the existing full generator with the alpha source: rejected because it violates stable artifact isolation.
  - Manually run oapi-codegen once: rejected because it is not reproducible or guarded.
  - Maintain a wholly separate undocumented command: rejected because the main API refresh entrypoint must remain discoverable.

## Decision 4: Retain and verify the existing product-v2 mutation chain

- **Decision**: Bundle the relocated v2 specification and apply, in order: search-query schema repair, search-result schema repair, process-instance filter-field repair, job-result discriminator repair, and operation-ID collision repair.
- **Rationale**: A temporary reproduction against the exact alpha4 commit completed successfully with Redocly and oapi-codegen 2.5.0. It produced a 50,132-line client; the operation-ID pass repaired `takeRuntimeBackup` and `deleteResource`, while all methods used by current unified v8.9 service contracts remained present.
- **Alternatives considered**:
  - Skip all mutations because the upstream spec is newer: rejected because generation correctness still depends on the schema-shape and operation-ID repairs.
  - Add alpha-specific mutations preemptively: rejected because the reproduced generation did not demonstrate a need.

## Decision 5: Store machine-readable provenance beside the generated client

- **Decision**: Commit `provenance.json` next to the alpha client with repository URL, tag, commit, source specification, ordered mutation list, generator name/version, and canonical generation command.
- **Rationale**: The current fetch script removes nested Git metadata, causing later ref discovery to return `unknown`. A checked-in record travels with the generated artifact and is straightforward to validate.
- **Alternatives considered**:
  - Put provenance only in commit messages: rejected because commits can be rebased or the file can be consumed outside Git history.
  - Put provenance only in a Markdown guide: rejected because guard tests need a machine-readable contract.

## Decision 6: Accept explicit alpha aliases but reject stable-looking input

- **Decision**: Canonicalize `8.10-alpha`, `810-alpha`, `v810-alpha`, and `v8.10-alpha` to `V810Alpha`. Reject `8.10`, `810`, `v810`, `v8.10`, and `8.10.0-alpha4`.
- **Rationale**: The accepted forms mirror existing normalization conventions while every accepted value retains the word `alpha`. This prevents a stable-looking selector from silently opting into prerelease behavior.
- **Alternatives considered**:
  - Accept only the canonical spelling: viable but inconsistent with existing shorthand conventions.
  - Accept full upstream tags: rejected because runtime identity is intentionally decoupled from a replaceable implementation baseline.

## Decision 7: Separate supported, stable, and experimental metadata

- **Decision**: Include alpha in supported and implemented discovery, add stable and experimental groupings, and expose experimental baseline metadata additively in human and JSON version output.
- **Rationale**: The feature must be discoverable without allowing `Supported Camunda versions` prose to imply that alpha is stable. Additive fields preserve the existing JSON envelope and field types.
- **Alternatives considered**:
  - Omit alpha from supported discovery: rejected by the specification.
  - Append `(experimental)` inside one comma-separated field only: rejected because machine consumers would have to parse presentation text.

## Decision 8: Reuse C89 fixtures explicitly

- **Decision**: Map `V810Alpha.FilePrefix()` to `C89_` and teach integration fixture selection that `8.10-alpha` deliberately uses C89 fixtures.
- **Rationale**: Existing BPMN fixtures exercise c8volt workflows rather than version-exclusive syntax. Explicit reuse avoids duplicated fixtures and prevents silent empty selection.
- **Alternatives considered**:
  - Add `C810A_` copies immediately: rejected because no content difference has been demonstrated.
  - Let unknown-prefix behavior fall through: rejected because embed commands could silently show no files.

## Decision 9: Build native alpha adapters for all eleven versioned families

- **Decision**: Add `v810alpha` implementations for batch operations, cluster, elements, incidents, jobs, process definitions, process instances, resources, tenants, user tasks, and variables. Each imports only the alpha generated client.
- **Rationale**: All eleven factories currently select v8.7-v8.9 implementations, and all unified methods required by their v8.9 contracts exist in the generated alpha client. Explicit adapters keep type differences local and avoid silently routing alpha through a v8.9 generated contract.
- **Alternatives considered**:
  - Return unsupported for all but a minimal smoke subset: rejected because the available unified contract supports the existing command surface and would make the advertised runtime of little use.
  - Route alpha factory cases to v8.9 services: rejected by the strict client boundary.

## Decision 10: Remove Tasklist fallback only from the alpha user-task path

- **Decision**: The alpha user-task adapter uses `GET /user-tasks/{userTaskKey}` from the unified API, validates the returned tenant where configured, and returns unified not-found/unsupported behavior without constructing a Tasklist client.
- **Rationale**: The direct endpoint exists in the generated alpha contract, while Camunda 8.10 removes the Tasklist v1 API and job-based user-task querying. Stable v8.8/v8.9 fallback behavior remains untouched.
- **Alternatives considered**:
  - Keep search-first plus Tasklist fallback: rejected because the fallback endpoint is removed.
  - Reuse the v8.9 Tasklist client against 8.10: rejected because generated compatibility does not prove endpoint availability.

## Decision 11: Model capabilities by name, not version ordering

- **Decision**: Add narrow capability predicates keyed by explicit runtime identities, starting with history-safe process-definition deletion, and use them in command and ops validation.
- **Rationale**: `V89` equality currently stands for “8.9 or newer.” Lexical or semantic ordering is brittle with `8.10-alpha`; named predicates are clear and make future stable 8.10 support additive.
- **Alternatives considered**:
  - Replace equality with `V89 || V810Alpha` at each call site: rejected because it repeats policy.
  - Add generic semantic-version comparison: rejected because runtime support is capability-specific and prerelease identities are not a total compatibility order.

## Decision 12: Account for 8.10 resource eventual consistency

- **Decision**: Preserve deployment visibility polling and use existing bounded read-retry helpers in the alpha resource path when a newly deployed resource is temporarily absent.
- **Rationale**: Camunda's 8.10 migration guide states that resource retrieval now uses secondary storage and is eventually consistent. Immediate one-shot retrieval can regress existing deploy/get workflows.
- **Alternatives considered**:
  - Add sleeps in commands: rejected because retry mechanics belong in services and fixed delays are unreliable.
  - Change all stable resource adapters: rejected because the behavior change is specific to the alpha target.

## Decision 13: Use a focused alpha smoke instead of claiming full release-suite certification

- **Decision**: Add a dedicated alpha smoke wrapper/report for authentication, topology, one read, one confirmed mutation, and deterministic unsupported behavior. Keep broader command-family proof in automated fake-server and service tests.
- **Rationale**: The feature is experimental and pinned; a focused live gate proves connectivity and core behavior without presenting alpha as equivalent to a stable release certification suite.
- **Alternatives considered**:
  - Run only unit tests: rejected because the specification requires a live pinned-baseline smoke.
  - Rebrand the full v8.9 release suite as alpha: rejected because removed legacy APIs and prerelease instability need explicit treatment.

## Sources Consulted

- Camunda release `8.10.0-alpha4` and annotated tag metadata in `camunda/camunda`.
- Pinned source tree under `zeebe/gateway-protocol/src/main/proto/v2/`.
- Camunda 8.10 APIs & Tools migration guide, including generated-client regeneration, legacy API removal, search-filter changes, and resource eventual consistency.
- c8volt generation scripts, version helpers, eleven service factories and v8.9 adapters, fixture selection, documentation generator, and integration harness.
