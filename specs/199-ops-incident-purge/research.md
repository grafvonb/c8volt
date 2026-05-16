# Research: Ops Purge Process Instances With Incidents

## Decision: Add The Workflow Under `ops purge`

Implement the command as `c8volt ops purge process-instances-with-incidents` with alias `pi-with-incidents`, extending the existing `ops purge` group created by #186.

**Rationale**: The issue explicitly chooses `ops purge` and frames the feature as a destructive cleanup workflow. The repository already has `ops purge orphan-process-instances`, so incident-based cleanup should follow that command hierarchy and output/report rhythm.

**Alternatives considered**:

- Add behavior to `delete pi`: rejected because the issue asks for a high-level operational playbook with discovery, preflight, confirmation, reporting, and audit output.
- Add a shell wrapper around `get incident | delete pi`: rejected because it would fork structured output, error classification, report writing, and automation semantics.
- Add alias `incident-pis`: rejected because the issue explicitly forbids it.

## Decision: Reuse Incident Search As Discovery, Then Freeze Candidate Process Instances

Use existing incident selection semantics to discover candidate incidents, extract process-instance keys, dedupe them, and freeze that candidate set before delete planning.

**Rationale**: The issue says the biggest practical change is that the workflow should read as candidate discovery followed by deterministic family-scope delete planning, not as a pipe. Freezing candidates makes dry-run, JSON, report files, and destructive confirmation refer to the same scope.

**Alternatives considered**:

- Re-run incident discovery before deletion: rejected because the user may confirm a different target set than the one ultimately deleted.
- Treat every incident as a delete request: rejected because duplicate incidents for one process instance must not produce duplicate delete submissions.
- Delete the incident scope directly: rejected because the delete phase must use process-instance family scope exactly as `delete pi` would.

## Decision: Reuse Existing Process-Instance Delete Planning And Deletion

Pass frozen candidate process-instance keys into the existing delete planning and deletion source of truth used by `delete pi`.

**Rationale**: Existing process-instance delete behavior already owns key dedupe, root resolution, descendant/family expansion, non-final affected checks, force cancel-before-delete, no-wait, worker, fail-fast, and no-worker-limit behavior. Reuse protects compatibility and avoids parallel safety logic.

**Alternatives considered**:

- Build an incident-specific delete planner: rejected because it creates a second source of truth for destructive scope and safety checks.
- Submit direct delete calls for candidate keys: rejected because it bypasses root/family expansion and non-final preflight semantics.

## Decision: Reuse #186/#187 Ops Report, Output, And Notice Conventions

Use shared ops report helpers for report-file validation, format inference, overwrite safety, and file writing; render human output in the discovery/plan/action/outcome/report rhythm; keep expected planning notices semantic.

**Rationale**: The issue carries forward #186/#187 conventions, including report overwrite safety and compact human output. Keeping one report lifecycle makes destructive ops workflows predictable.

**Alternatives considered**:

- Build a separate report writer for incident purge: rejected because Markdown and JSON reports should render from a shared structured model and preserve existing overwrite rules.
- Add a second overwrite confirmation prompt: rejected because existing destructive confirmation and non-interactive confirmation are the overwrite boundary.

## Decision: Align Automation With `shouldImplicitlyConfirm(cmd)`

For this supported state-changing ops command, `--automation` should implicitly accept supported prompts through `shouldImplicitlyConfirm(cmd)` and should not require `--auto-confirm`.

**Rationale**: The issue includes an implementation guardrail that `--automation` is the canonical non-interactive mode and that supported ops commands must not require `--auto-confirm` in addition. This matches the current #187 direction.

**Alternatives considered**:

- Require both `--automation` and `--auto-confirm`: rejected because it conflicts with the issue guardrail and would fail the required automation JSON test.
- Let automation bypass all safeguards regardless of command metadata: rejected because automation support must remain declared and protected by command contract metadata.

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
