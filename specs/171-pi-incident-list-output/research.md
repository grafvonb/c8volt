# Research: Process Instance Incident List Output

## Decision: Reuse Existing Enrichment For List/Search Results

Use `c8volt/process` incident enrichment for the `ProcessInstances` returned by list/search mode, not a separate command-local incident lookup loop.

**Rationale**: `EnrichProcessInstancesWithIncidents` already preserves item order, filters incidents by process-instance key, passes call options through, and has regression tests. Reusing it keeps keyed and list/search JSON shapes aligned and avoids a parallel association implementation.

**Alternatives considered**:

- Inline incident lookups in `cmd/get_processinstance.go`: rejected because it duplicates facade behavior and risks divergent JSON association.
- Add new versioned service methods for list enrichment: rejected unless implementation proves the existing facade cannot support paged/list results.

## Decision: Enrich Only The Results Selected For Rendering

Apply incident enrichment after the list/search paging flow has selected the process instances that will be rendered.

**Rationale**: The issue requires paging and `--limit` compatibility. Enriching only displayed items keeps incident lookup cost bounded to output scope and preserves existing list selection semantics.

**Alternatives considered**:

- Enrich every backend match before applying limits: rejected because it changes paging cost and can query incidents for rows the user will not see.
- Disable incremental paging when incidents are enabled: rejected unless no repository-native path can render enriched pages while preserving prompts and summaries.

## Decision: Keep Human Truncation In Rendering

Apply `--incident-message-limit` in human incident line formatting and leave process/facade/domain incident messages untouched.

**Rationale**: The flag is explicitly human-output only and JSON must keep full messages. Rendering-level truncation prevents accidental data loss in structured output or downstream models.

**Alternatives considered**:

- Truncate in the facade model: rejected because it would affect JSON and tests that expect full data.
- Truncate the full rendered line: rejected because the spec limits only the error message portion.

## Decision: Share The Compact Prefix Through `incidentHumanLine`

Change the shared human incident line helper from `incident <key>:` to `inc <key>:` so both get and walk human output use the same prefix.

**Rationale**: `cmd/cmd_views_processinstance_incidents.go` owns `incidentHumanLine`, and `cmd/cmd_views_walk_incidents.go` already calls that helper. Updating the helper keeps the prefix consistent without duplicate rendering changes.

**Alternatives considered**:

- Add command-specific prefixes: rejected because the issue requires consistent get and walk behavior.

## Decision: Separate Per-Row Indirect Notes From One List Warning

Render a short indented note below each affected row and print one de-duplicated warning after the list when at least one listed row has an incident marker but no direct incident details.

**Rationale**: The issue asks for local row clarity without repeating the long `walk pi` hint under every row. This also keeps the warning discoverable after list rendering completes.

**Alternatives considered**:

- Existing single warning emitted at the first affected row: rejected because it does not attach a short note to each affected row and interrupts row grouping.
- Full `walk pi` hint under every affected row: rejected because the issue explicitly asks to avoid repeated long hints.

## Decision: Update Docs Through Existing Documentation Flow

Update command help text and README examples, then regenerate generated CLI docs through the repository's documentation tooling.

**Rationale**: The constitution requires user-facing docs to match command behavior, and repository guidance prefers generated docs to be updated from source metadata.

**Alternatives considered**:

- Hand-edit generated docs only: rejected because it risks drift from Cobra metadata.
