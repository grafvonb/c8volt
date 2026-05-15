# Research: Ops Execute Retention Policy

## Decision: Add Retention Policy Under `ops execute`

Implement the command as `c8volt ops execute retention-policy` by extending the existing `ops execute` group rather than adding another top-level or purge workflow.

**Rationale**: Issue #187 explicitly places the workflow under `ops execute`. The repository already has #186 ops purge cleanup for orphan process instances, but retention policy is framed as a high-level operational execution workflow with retained c8volt control over discovery, planning, deletion, waiting, and reporting.

**Alternatives considered**:

- Add retention under `ops purge`: rejected because the source issue explicitly chooses `ops execute retention-policy`.
- Add behavior to `delete pi`: rejected because retention discovery, planning, and audit reporting form a predefined ops playbook, not a generic delete mode.
- Add a shell wrapper around `get pi | delete pi`: rejected by the issue and incompatible with structured audit/error/report behavior.

## Decision: Reuse Process-Instance Search Semantics For Retention Discovery

Model retention discovery as a service-owned process-instance search equivalent to `get pi --end-date-older-days <days> --keys-only` plus compatible filters, returning an immutable seed set.

**Rationale**: Existing process-instance search already owns end-date age semantics, tenant/version compatibility, paging, limit, batch size, and filter validation. Freezing the discovered seed set gives dry-run, JSON, and audit reports a stable scope and satisfies the issue's requirement not to chase newly eligible instances.

**Alternatives considered**:

- Reimplement age filtering in the ops command: rejected because it would fork process-instance search behavior.
- Re-query before each delete batch: rejected because the user might confirm a different target set than the one ultimately deleted.

## Decision: Reuse Existing Delete Planning And Deletion Behavior

Pass discovered retention seed keys into existing process-instance delete planning and deletion services instead of introducing a retention-specific hierarchy mode.

**Rationale**: The issue requires root resolution, descendant traversal, duplicate removal, missing ancestor handling, non-final affected instance blocking, cancellation controls, concurrency, confirmation, and waiting to match `delete pi`. Keeping those mechanics service-owned avoids parallel behavior and preserves regression expectations.

**Alternatives considered**:

- Delete only discovered seed keys directly: rejected because it would bypass existing hierarchy and safety semantics.
- Build a separate retention hierarchy planner: rejected because it creates a second source of truth for delete scope.

## Decision: Use #186 Shared Ops Report And Output Helpers

Reuse the shared ops report helpers for report-file validation, format inference, overwrite safety, file writing, semantic notices, and compact human output rhythm.

**Rationale**: The source issue explicitly carries forward #186 conventions. Reusing those helpers keeps report overwrite safety and human output consistent across ops workflows.

**Alternatives considered**:

- Build a separate report writer for retention: rejected because Markdown and JSON reports should render from a shared structured model and match #186 safety behavior.
- Add an extra overwrite prompt: rejected because the issue states existing command confirmation or non-interactive confirmation is the overwrite boundary.

## Decision: Align Automation With The Later Guardrail In Issue #187

For this supported state-changing ops command, `--automation` should implicitly accept supported prompts through the existing `shouldImplicitlyConfirm(cmd)` path and should not require `--auto-confirm` in addition.

**Rationale**: The issue includes an earlier confirmation section that says automation without auto-confirm fails, but the later implementation guardrail specifically corrects the automation contract: `--automation` is the canonical non-interactive mode, and supported ops commands must not require `--auto-confirm` as a companion. The later guardrail is more specific and aligns with repository automation semantics.

**Alternatives considered**:

- Require both `--automation` and `--auto-confirm`: rejected because it conflicts with the explicit implementation guardrail and would produce a failing acceptance path for `--automation --json`.
- Let every prompt be implicitly accepted in automation regardless of command support: rejected because automation compatibility must still be declared and protected by command contract metadata.

## Decision: Protect Error Classes With Subprocess Tests

Use existing invalid-input helpers for static CLI shape errors and `localPreconditionError` for runtime or local precondition failures discovered after planning or preflight.

**Rationale**: Issue #187 calls out exit-code classification as an implementation guardrail. Subprocess tests are the repository's strongest protection for user-visible messages and exit codes.

**Alternatives considered**:

- Assert only returned Go errors in unit tests: rejected because the CLI contract includes process exit codes and stderr/stdout separation.

## Decision: Refresh Generated CLI Docs From Source Metadata

After command metadata changes, run `make docs-content` rather than hand-editing generated CLI docs.

**Rationale**: Repository constitution and workflow rules require generated docs to match command source metadata.

**Alternatives considered**:

- Manually edit `docs/cli/*`: rejected because generated output must come from the command tree.
