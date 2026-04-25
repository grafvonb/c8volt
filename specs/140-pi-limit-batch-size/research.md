# Research: Process-Instance Limit and Batch Size Flags

## Decision: Rename the page-size flag at Cobra registration points

**Rationale**: The current user-facing ambiguity comes from the command flag name. The existing command helpers already treat the value as a per-page size, so the repository-native change is to register `--batch-size` with short `-n` and update all help/examples that mention the affected process-instance commands.

**Alternatives considered**:

- Keep `--count` as an alias: rejected because the issue explicitly says not to keep it.
- Add `--batch-size` while hiding `--count`: rejected because hidden aliases still preserve the ambiguous interface.

## Decision: Implement `--limit` at the command paging layer

**Rationale**: The limit applies to matched process instances after search results and existing local filters are known, and before rendering or destructive actions. The command paging layer already owns local filtering, continuation prompts, incremental rendering, and the keys passed to cancel/delete planning.

**Alternatives considered**:

- Push limit into versioned services: rejected because service behavior differs by backend version and would not cover local filters consistently.
- Apply limit only after aggregating all pages: rejected because it would still fetch unnecessary pages and violate the stop-cleanly requirement.

## Decision: Reject `--total` combined with `--limit`

**Rationale**: `--total` is count-only output, while `--limit` controls limited detail output or destructive processing. Treating them as mutually exclusive keeps script behavior explicit and avoids surprising users with ignored limits or redefined totals.

**Alternatives considered**:

- Ignore `--limit` when `--total` is present: rejected because silently ignoring a user-provided limit is not script-safe.
- Count only up to the limit: rejected because it would redefine the established `--total` meaning and make totals incomparable.

## Decision: Add an explicit limit-reached continuation state

**Rationale**: Existing progress output distinguishes complete, prompt, auto-continue, partial completion, and warning stop. A limit-reached state lets output truthfully report why execution stopped and prevents reusing "complete" for a different stop reason.

**Alternatives considered**:

- Treat limit reached as no-more-matches: rejected because it would be misleading when additional backend matches remain.
- Suppress progress details: rejected because the issue requires truthful stop reasons.

## Decision: Let removed `--count` fail through standard invalid-argument handling

**Rationale**: Cobra unknown flag behavior already provides repository-standard invalid argument handling for removed command flags. Keeping custom compatibility code would risk preserving the alias by accident.

**Alternatives considered**:

- Add a custom `--count` validation error: rejected unless tests show Cobra's standard error is inconsistent with repository expectations.

## Decision: Regenerate CLI docs from command metadata

**Rationale**: This repository keeps generated CLI docs under `docs/cli/` and existing guidance says generated artifacts should be regenerated from source metadata. The command help and examples are the source of truth for affected generated docs.

**Alternatives considered**:

- Hand-edit generated docs: rejected because it risks drift from Cobra metadata.
