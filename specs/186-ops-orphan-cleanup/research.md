# Research: Ops Purge Orphan Process Instances

## Decision: Extend Existing Ops Foundation With Purge

Use existing `cmd/ops.go` and shared ops workflow contracts from issue #197 as the foundation, keep `cmd/ops_execute.go` for execute workflows such as smoke tests, and add `cmd/ops_purge.go` plus `cmd/ops_purge_orphan_processinstances.go` for this destructive cleanup workflow.

**Rationale**: The repository already has discovery-only `ops`, `ops execute`, and `ops repair` grouping commands plus shared status/report contracts from issue #197. Orphan process-instance cleanup is primarily destructive removal, so it belongs under a new `ops purge` grouping command rather than the more general `ops execute` group.

**Alternatives considered**:

- Add cleanup under `delete pi`: rejected because the issue calls for a high-level predefined operational playbook under `ops`.
- Keep cleanup under `ops execute orphan-cleanup`: rejected because `execute` should remain available for playbooks such as `ops execute smoke-test`, while destructive cleanup workflows are clearer under `ops purge`.
- Create a separate top-level `cleanup` command: rejected because it introduces a parallel command hierarchy.

## Decision: Keep Resource-Specific Logic In Process-Instance Services

Discovery, orphan filtering, deletion planning, and deletion execution should be service-owned process-instance behavior. The ops workflow may orchestrate those primitives but must not duplicate generated-client or filter mechanics.

**Rationale**: `cmd/get_processinstance_filtering.go`, `cmd/get_processinstance_paging.go`, `cmd/delete_processinstance.go`, `c8volt/process/dryrun.go`, and `internal/services/processinstance` already own the behavior this workflow needs. The Ralph rules explicitly prohibit facades from implementing backend pagination loops, worker pools, wait loops, and resource-specific service mechanics.

**Alternatives considered**:

- Shell out to `c8volt get pi` and `c8volt delete pi`: rejected by the issue and would break structured error/report behavior.
- Copy `get pi` filtering and `delete pi` planning into the ops command: rejected because it would fork operational semantics and tests.

## Decision: Treat Orphan Discovery Result As Immutable Cleanup Scope

The workflow should discover orphan child process instances once, store that key set in the cleanup result model, and delete only those keys after validation and confirmation.

**Rationale**: The issue explicitly says the command must not keep discovering newly orphaned process instances forever. Freezing the initial key set also makes dry-run, JSON, and audit reports deterministic.

**Alternatives considered**:

- Re-run discovery before each delete batch: rejected because it can expand the cleanup scope after the user previewed or confirmed a different set.

## Decision: Use Existing Root Confirmation And Automation Semantics

Reuse `--auto-confirm`, `--automation`, `requireAutomationSupport`, `shouldImplicitlyConfirm`, and existing destructive confirmation patterns.

**Rationale**: Existing delete commands already separate interactive and unattended execution. The new command should fit the same script-safe contract and fail before mutation when automation lacks explicit confirmation.

**Alternatives considered**:

- Add a workflow-specific confirmation flag: rejected as duplicated command semantics.
- Let automation imply confirmation: rejected because destructive automation must require explicit `--auto-confirm`.

## Decision: Build Audit Reports From A Stable Structured Model

Represent cleanup request, discovery, deletion plan, deletion results, errors, timestamps, and outcome as a structured model before rendering Markdown or JSON.

**Rationale**: The issue requires Markdown and JSON reports with the same information. `cmd/ops_contract.go` already has shared status and report-format primitives that can be extended while preserving deterministic JSON.

**Alternatives considered**:

- Render Markdown directly during command execution: rejected because JSON would drift from Markdown and failures after discovery need a stable report snapshot.

## Decision: Refresh Generated CLI Docs From Source Metadata

After command metadata changes, run `make docs-content` rather than hand-editing generated CLI docs.

**Rationale**: Repository rules and constitution both require generated docs to stay in sync through the generator.

**Alternatives considered**:

- Manually edit `docs/cli/*`: rejected because generated output must come from command source.
