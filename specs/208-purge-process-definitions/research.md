# Research: Ops Purge All Process Definitions

## Decision: Add The Workflow Under `ops purge`

Implement the command as `c8volt ops purge all-process-definitions` with alias `all-pds`, extending the existing `ops purge` group used by the orphan and incident purge workflows.

**Rationale**: The issue explicitly chooses `ops purge` and frames the feature as a destructive cleanup workflow. The repository already has purge workflows with dry-run, confirmation, report, and automation behavior, so all process-definition cleanup should follow that command hierarchy and output/report rhythm.

**Alternatives considered**:

- Add behavior to `delete pd`: rejected because the issue asks for a high-level operational playbook with discovery, frozen scope, preflight, confirmation, reporting, and audit output.
- Add a shell wrapper around `get pd | delete pd`: rejected because it would fork structured output, error classification, report writing, confirmation, and automation semantics.
- Add aliases such as `purge-definitions` or `delete-all`: rejected because the issue explicitly forbids broad ambiguous aliases.

## Decision: Reuse `get pd` Selection Semantics, Then Freeze Candidate Process Definitions

Use existing process-definition selection semantics to discover candidate process-definition versions, dedupe keys, and freeze that candidate set before delete preflight, confirmation, or mutation.

**Rationale**: The issue says the workflow should discover candidates, freeze the exact versions, and then run the same deterministic delete plan as `delete pd`. Freezing candidates makes dry-run, JSON, report files, and destructive confirmation refer to the same scope.

**Alternatives considered**:

- Re-run process-definition discovery before deletion: rejected because the user may confirm a different target set than the one ultimately deleted.
- Treat every search result as a delete request without dedupe: rejected because duplicate process-definition keys must not produce duplicate delete submissions.
- Make `--latest` the default: rejected because the issue says default discovery covers every visible process-definition version, with `--latest` only as explicit narrowing.

## Decision: Reuse Existing Process-Definition Delete Preflight And Deletion

Pass frozen candidate process-definition keys into the existing delete preflight and deletion source of truth used by `delete pd`.

**Rationale**: Existing process-definition delete behavior already owns key dedupe, active process-instance impact analysis, force cancel-before-delete behavior, process-instance history cleanup, process-definition deletion, no-wait, worker, fail-fast, and no-worker-limit behavior. Reuse protects compatibility and avoids parallel safety logic.

**Alternatives considered**:

- Build an ops-specific process-definition delete planner: rejected because it creates a second source of truth for destructive scope and safety checks.
- Submit direct delete calls for candidate keys: rejected because it bypasses active-instance safety and process-instance cleanup semantics.

## Decision: Reuse Shared Ops Report, Output, And Notice Conventions

Use shared ops report helpers for report-file validation, format inference, overwrite safety, and file writing; render human output in the discovery/plan/action/outcome/report rhythm; keep expected planning notices semantic.

**Rationale**: The issue carries forward shared ops workflow conventions proven by orphan cleanup, retention policy, and incident purge workflows. Keeping one report lifecycle makes destructive ops workflows predictable.

**Alternatives considered**:

- Build a separate report writer for all-PD purge: rejected because Markdown and JSON reports should render from a shared structured model and preserve existing overwrite rules.
- Add a second overwrite confirmation prompt: rejected because existing destructive confirmation and non-interactive confirmation are the overwrite boundary.

## Decision: Align Automation With `shouldImplicitlyConfirm(cmd)`

For this supported state-changing ops command, `--automation` should implicitly accept supported prompts through `shouldImplicitlyConfirm(cmd)` and should not require `--auto-confirm`.

**Rationale**: The issue says `--automation` is the canonical non-interactive mode and that supported ops commands must not require `--auto-confirm` in addition. This keeps the command script-safe for deterministic JSON automation.

**Alternatives considered**:

- Require both `--automation` and `--auto-confirm`: rejected because it conflicts with the issue guardrail and would fail the required automation JSON test.
- Let automation bypass safeguards regardless of command metadata: rejected because automation support must remain declared and protected by command contract metadata.

## Decision: Protect Error Classes With Subprocess Tests

Use existing invalid-input helpers for static CLI shape errors and `localPreconditionError` for runtime or local precondition failures discovered after discovery, preflight, or planning.

**Rationale**: The issue explicitly calls out exit-code classification. Subprocess tests protect user-visible messages, stdout/stderr separation, and exit codes better than unit-only error assertions.

**Alternatives considered**:

- Assert only returned Go errors in package tests: rejected because CLI users and automation depend on process exit behavior.

## Decision: Refresh Generated CLI Docs From Source Metadata

After command metadata changes, run `make docs-content` rather than hand-editing generated CLI docs.

**Rationale**: Repository constitution and workflow rules require generated docs to match command source metadata.

**Alternatives considered**:

- Manually edit `docs/cli/*`: rejected because generated output must come from the Cobra command tree.
