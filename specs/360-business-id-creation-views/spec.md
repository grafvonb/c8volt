# Feature Specification: Business-ID Creation and Complete Resource Views

**Feature Branch**: `360-business-id-creation-views`

**Created**: 2026-10-31

**Status**: Draft

**Input**: [GitHub issue #360](https://github.com/grafvonb/c8volt/issues/360) — Support setting a business ID when creating process instances and expose observed business IDs throughout existing resource views on Camunda 8.10 and later supported versions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create Instances with a Business Reference (Priority: P1)

As an operator or library consumer, I want to supply my business reference when starting a process instance so that its creation result and subsequent resource views retain that reference.

**Why this priority**: Setting the reference at creation is the primary new action and connects business work to process execution.

**Independent Test**: Exercise creation against isolated successful, rejected, and delayed responses, using both process selection methods and both waiting modes; compare supplied values with submitted values and returned observations.

**Acceptance Scenarios**:

1. **Given** Camunda 8.10 or a later supported version, **When** I create an instance with `run pi --bid <value>` by BPMN process ID or process-definition key, **Then** the selected creation request carries the exact supplied business ID and the creation result preserves the backend-returned ID.
2. **Given** creation through the public library interface, **When** I provide or omit the optional business ID, **Then** both selection methods obey the same validation, submission, and result rules as the command.
3. **Given** an omitted business ID, **When** I create an instance, **Then** no business ID is submitted and existing creation behavior is retained; explicitly supplying an empty string is rejected before mutation.
4. **Given** valid IDs containing commas, mixed case, whitespace, or Unicode characters, **When** I create instances, **Then** each ID remains one opaque value without trimming, splitting, case normalization, or conversion into a process variable. Values of 1 through 256 UTF-16 code units are accepted locally; longer values are rejected before mutation. A supplementary Unicode character counts as two units, matching the supported backend.
5. **Given** ordinary waiting, **When** creation becomes observable under the existing creation-visibility rules, **Then** the observed business ID survives into the result; with `--no-wait`, the accepted creation response retains its business ID without claiming that visibility or process completion has been verified.
6. **Given** `--count` greater than one, **When** I supply one business ID, **Then** every requested creation uses that identical value, with no generated suffixes or alternate IDs.
7. **Given** one or more successful creations followed by a uniqueness conflict or another failure, **When** the overall command returns, **Then** successful creation results remain available with their business IDs, the command reports partial failure with the original failure classification and exit behavior, and no rollback or all-success outcome is implied.

---

### User Story 2 - Inspect Accurate Business References Everywhere (Priority: P1)

As an operator investigating a business transaction, I want existing resource views, enriched views, and reports to show the business IDs actually observed on each resource so that I can correlate work without being misled by reconstructed data.

**Why this priority**: A reference is useful only if it survives the views operators use during investigation and review.

**Independent Test**: Supply distinct and absent IDs for related resources in isolated observations, then inspect every surface in the coverage table in both human and machine representations where supported.

**Acceptance Scenarios**:

1. **Given** an observed process instance, job, user task, or runtime listener job with a business ID, **When** it appears in any applicable surface below, **Then** its own ID survives ordinary conversion, enrichment, copying into previews, and report generation.
2. **Given** an instance whose currently observed ID is `order-new` and related jobs, tasks, or listeners with `order-old` or no ID, **When** I inspect them together, **Then** each resource retains its own value or absence; the process instance's newer ID is never substituted into related snapshots.
3. **Given** a resource with no business ID, **When** it is converted or rendered, **Then** absence remains absence without a synthesized value, and established omission and null conventions are preserved.
4. **Given** family, activity, variable, incident, element, age, or timeline enrichment, **When** business IDs are displayed, **Then** the existing envelope and enrichment-section shapes remain intact and IDs are attached to the resource they describe.
5. **Given** an operations preview or a generated report, **When** observed process-instance or job data is included, **Then** its business ID is retained in supported human, JSON, and saved Markdown forms.
6. **Given** a key/status-only result or a local exact-key preview with no observed resource data, **When** it is rendered, **Then** no additional read is performed solely to populate a business ID and no ID is invented.

#### Required View and Report Coverage

Each row covers present and absent values and all existing human and JSON forms unless the surface has a different explicitly named format.

| Existing surface | Required coverage |
| --- | --- |
| `run pi` | Dedicated creation results, normal waiting, no-wait results, and successful results retained after partial failure |
| `get pi` | Single and list results; age metadata; variables, incidents, elements, and combined activity views |
| `walk pi` | Family and parent/child views; flat and tree forms; enriched views |
| `get job` | Single and list results |
| Job mutation selection | Update previews and bulk completion candidates that copy or reconstruct observed job data |
| `get user-task` | Ordinary and variable-enriched views |
| Runtime listener jobs | Listeners beneath process activity, element results, and slow-process analysis |
| Slow-process analysis | Represented process instances, summaries, and full-timeline output |
| Process-instance mutations | Previews, plans, and reports containing observed process-instance data |
| Operations | Previews, JSON reports, and saved Markdown reports containing observed process-instance or job data |
| Process-definition deletion | Impact data containing observed process instances |

---

### User Story 3 - Keep Existing Versions and Automation Reliable (Priority: P2)

As an existing user or script author, I want the new reference support to preserve version compatibility, machine output, and diagnostic behavior so that existing workflows remain dependable.

**Why this priority**: The feature extends established workflows used for both interactive operations and automation.

**Independent Test**: Execute isolated command and public-interface scenarios for 8.8, 8.9, and 8.10, capture result and diagnostic destinations separately, and compare existing output contracts with and without business-ID input.

**Acceptance Scenarios**:

1. **Given** Camunda 8.8 or 8.9, **When** I supply a business-ID creation input through the command or public interface, **Then** it is rejected before any mutation with a compact explanation of the required version.
2. **Given** no business-ID input on 8.8 or 8.9, **When** I use existing creation, query, enrichment, or report workflows, **Then** their prior behavior remains available and the default configured Camunda version does not change.
3. **Given** JSON output on a command using the shared contract, **When** creation succeeds, partially fails, fails validation, or an affected query returns empty results, **Then** exactly one envelope is emitted, preserving established payload shapes, partial evidence, and failure classification, with no trailing diagnostic output on stdout. Quiet plus JSON retains that requested result.
4. **Given** keys-only or totals output, **When** affected commands run, **Then** keys-only remains exactly one key per line and zero bytes for no keys, and totals remain counts without business-ID text.
5. **Given** human mutation output, **When** a summary, dry-run summary, or verbose outcome is produced, **Then** it uses the configured informational logger, honors inherited destinations and plain or structured formats, leaves result stdout empty, and is suppressed under quiet or a warning-level threshold. Ordinary query results remain selected-format stdout output.
6. **Given** repeated backend failures, **When** normal, quiet, or verbose-only output is selected, **Then** one compact aggregate diagnostic retains useful counts and bounded reasons without exposing raw methods, URLs, bodies, or transport chains on either stream. Existing redacted DEBUG diagnostics remain independently available through either the debug flag or configured debug level.
7. **Given** an eligible interactive prompt, **When** it accompanies results, **Then** it remains plain text on configured stderr with existing wording, eligibility, and input behavior. Business-ID display introduces no extra prompts or discovery requests.
8. **Given** command help, examples, README guidance, and generated CLI documentation, **When** I consult business-ID creation guidance, **Then** it explains version support, both selection methods, opaque-value handling, waiting behavior, and the potential effect of backend uniqueness enforcement on repeated creation.

### Edge Cases

- Omission differs from an explicitly supplied empty string; whitespace and commas are not silently removed or interpreted as separators.
- Length validation covers empty, 1-unit, 256-unit, and over-limit values, counting UTF-16 code units; 256 BMP characters and 128 supplementary characters are within the limit, while 129 supplementary characters exceed it.
- An accepted creation response and a later observation may differ; each result preserves its own source value, without fabricating an observed ID from the input.
- A process instance may have an ID while older related resources have another value or none; family and enriched views must preserve these differences.
- With repeated or concurrent creation, backend uniqueness enforcement may reject some attempts; already successful or accepted outcomes remain available without invented replacement IDs or implicit retries with different IDs.
- Empty lists, missing enrichment sections, absent listener data, and successful no-ops preserve established empty representations, quiet behavior, and zero-byte keys output.
- Local exact-key previews and key/status-only outcomes remain usable without additional lookups, even when no business ID can be shown.
- Timeouts, cancellation, and backend rejection retain their existing failure classification and partial-result evidence; no-wait acceptance is not reported as verified execution completion.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Business-ID functionality MUST be supported only for Camunda 8.10 and later versions already supported by c8volt. Existing non-business-ID workflows on 8.8 and 8.9 and the default configured version MUST remain unchanged.
- **FR-002**: `run pi` MUST accept optional `--bid <value>`, and the public creation interface MUST expose an equivalent optional input, for both BPMN process ID and process-definition key selection.
- **FR-003**: Both native creation request variants MUST carry the exact supplied business ID when present and omit it when absent. Creation responses, ordinary waiting results, and no-wait results MUST retain backend-provided business IDs.
- **FR-004**: Creation inputs MUST distinguish omission from explicit invalid input and enforce nonempty input and the supported backend's maximum of 256 UTF-16 code units, before mutation. IDs MUST remain opaque strings without trimming, comma splitting, case normalization, or process-variable interpretation.
- **FR-005**: Unsupported-version business-ID creation input MUST be rejected before mutation through both CLI and public-interface entry points.
- **FR-006**: Count-based creation MUST apply the same supplied ID to every requested instance. Uniqueness enforcement MUST remain the backend's responsibility, without client-generated alternatives or additional uniqueness discovery.
- **FR-007**: Partial creation failure MUST preserve successful creation evidence, including business IDs, and the original failure classification and exit behavior. Existing waiting and acceptance semantics MUST remain intact.
- **FR-008**: Optional business IDs MUST survive every applicable inward and outward conversion, including resource and operations representations, enrichment, and output representations, for process instances, creation results, jobs, user tasks, runtime listener jobs, and process instances in slow-process analysis.
- **FR-009**: Existing previews, plans, reports, and candidates containing observed process-instance or job data MUST retain its business ID, including copied or reconstructed observations.
- **FR-010**: Every resource MUST retain its own backend-provided value or absence. Neither rendering nor enrichment may synthesize a related resource's business ID from a process instance's current ID.
- **FR-011**: Every applicable surface in the required coverage table MUST expose observed business IDs in its existing human and machine representations, including dedicated creation rendering and saved Markdown operations reports.
- **FR-012**: Business-ID additions MUST preserve existing envelopes, enrichment-section shapes, and absent-value conventions. Keys-only output MUST remain one key per line, empty keys output MUST contain zero bytes, and totals MUST remain counts.
- **FR-013**: Key/status-only results and local exact-key previews MUST NOT perform reads solely to populate business IDs. Rendering MUST NOT add discovery or mutation requests.
- **FR-014**: Human output MUST remain compact. Mutation summaries, dry-run summaries, verbose outcomes, warnings, and errors MUST use the configured command logger at their established levels and destinations; selected-format results MUST remain on stdout and prompts MUST remain plain configured stderr text.
- **FR-015**: Shared-contract JSON execution MUST emit exactly one envelope, including validation errors, partial failures, and empty results, without duplicate error rendering or loss of original failure evidence. Quiet and output-mode precedence MUST retain existing semantics.
- **FR-016**: Normal and verbose-only human failures MUST use compact bounded aggregate diagnostics rather than raw joined errors. Existing redacted DEBUG diagnostics MUST remain available independently of verbose, without contaminating stdout.
- **FR-017**: User-facing command metadata, examples, README guidance, and generated CLI documentation MUST describe the version boundary, input rules, repeated-creation behavior, and result semantics.

### Key Entities *(include if feature involves data)*

- **Business ID**: An optional opaque business reference associated with an observed resource; absence is distinct from a supplied invalid creation value. Backend uniqueness rules concern process definitions within tenant scope.
- **Creation intent**: The selected process definition, optional business ID, requested count, and existing wait preference. One supplied ID applies unchanged to all requested creations.
- **Resource observation**: A process instance, job, user task, or runtime listener job with its own key and optional backend-provided business ID at the time represented.
- **Creation outcome**: Successful or accepted creation evidence and any failure, retaining observed IDs and the established distinction between acceptance and verified visibility.
- **Resource view or report**: An existing presentation of observations, including enrichment and mutation impact, that preserves resource identity and absence without acquiring extra data solely for business-ID display.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators and library consumers can supply one business reference through each of the two creation selection methods; 100% of valid boundary and opaque-string acceptance cases retain the exact supplied value in the submitted creation intent.
- **SC-002**: Every applicable surface in the coverage table preserves the resource's observed business ID or absence in all supported representations; zero cases replace a related resource's value with a newer parent value.
- **SC-003**: For repeated creation with mixed outcomes, 100% of successful creations remain available to the caller, and the overall result retains the original failure classification without claiming complete success.
- **SC-004**: Every business-ID creation attempt against 8.8 or 8.9 is rejected before mutation, while representative workflows without business-ID input retain their existing behavior and default version.
- **SC-005**: All affected machine-output acceptance cases preserve their established envelope, key, or count contract, with zero diagnostic records on result stdout and no extra reads solely to populate IDs in key/status-only results or local previews.
- **SC-006**: Published help and examples explain both creation selectors, version support, opaque IDs, waiting, and repeated-creation uniqueness; operators can determine these behaviors without consulting source code.

## Assumptions

- This feature is independent of the other business-ID features described in issue #360. Issues #203 and #239 are historical context, not implementation dependencies.
- Filtering or resolving existing instances by business ID, late assignment, and atomic assignment during job completion are out of scope. No new message or decision command families, generic capability framework, or unrelated refactoring are included.
- “Later supported versions” does not require adding a new Camunda version or changing the default version as part of this feature.
- The checked-in contract requires nonempty IDs and a maximum length of 256. Planning verified that the supported backend counts UTF-16 code units: BMP characters count once and supplementary characters count twice. See [research.md](research.md) for pinned source evidence and boundary cases. Byte count and Unicode code-point count are not substitutes.
- The backend controls uniqueness. Its documented scope is an active root process instance for the same process definition within a tenant when uniqueness enforcement is enabled; active child instances are not included in that uniqueness check. Documentation must explain possible repeated-creation conflicts without promising uniqueness when enforcement is disabled.
- Existing authentication, authorization, tenant handling, concurrency, ordering, confirmation, waiting, cancellation, and retry behavior remain applicable unless an explicit requirement above changes the business-ID behavior.
- Acceptance validation uses unit tests, generated-client mocks, test-owned HTTP servers, and isolated command/subprocess/terminal tests. Coverage includes mapping round trips, exact supplied/omitted creation payloads for both variants, length and Unicode boundaries, partial creation, waiting modes, output routing, and single-envelope results. No live Camunda environment is an implementation completion requirement.
