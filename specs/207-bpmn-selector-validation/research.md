# Research: BPMN Selector Validation for Operational Commands

## Decision: Extend the existing shared selector validator instead of adding command-specific diagnostics

**Rationale**: Issue #175 already introduced a command-level visible process-definition validation helper with shared missing-selector formatting, near-match listing, prompt eligibility, and process-definition list rendering. Extending that helper keeps diagnostics consistent across `get pi`, `run pi`, `cancel pi`, `delete pi`, `get incident`, `get pd`, and `delete pd`.

**Alternatives considered**:

- Add separate missing-selector logic in every command. Rejected because error wording, prompt behavior, and machine-mode suppression would drift again.
- Move validation below command code into resource services. Rejected because services do not own CLI prompt/listing policy and would affect non-BPMN or keyed flows unnecessarily.

## Decision: Validate direct `cancel pi` and `delete pi` selectors before process-instance search paging

**Rationale**: These commands accept `--bpmn-process-id` directly and can otherwise turn a typo into `found: 0` or a harmless-looking dry run. Validation must occur after normal flag validation and before paging, dry-run aggregation, confirmation, or mutation submission.

**Alternatives considered**:

- Depend on users to pipe from `get pi`. Rejected because the commands themselves expose `-b` and must be safe when used directly.
- Validate after the first process-instance page. Rejected because the bug is that process-instance paging is not authoritative for selector visibility.

## Decision: Treat `get incident -b` as a BPMN process-definition scoped incident search

**Rationale**: The flag asks for incidents associated with a BPMN process definition. A missing or invisible BPMN selector should not be indistinguishable from a legitimate empty incident set, especially when incident keys can feed repair, purge, or cancel pipelines.

**Alternatives considered**:

- Preserve empty incident results for missing BPMN selectors. Rejected because it leaves the same typo/tenant/permission ambiguity in the incident workflow.
- Require `--pd-key` for strict validation. Rejected because `--bpmn-process-id` is already a user-facing selector and should be validated when directly supplied.

## Decision: Audit direct process-definition commands as selector commands with explicit missing behavior

**Rationale**: `get pd -b` and `delete pd -b` search process definitions directly, so they do not have the same "runtime resource search hiding selector failure" shape. They still need explicit missing-selector behavior so operators see a clear outcome and tests document whether the shared diagnostic applies.

**Alternatives considered**:

- Leave direct process-definition commands as implicit empty searches. Rejected because the issue requires an explicit audit decision and documentation.
- Force every direct process-definition search to fail on zero results. Rejected unless scoped to direct BPMN selectors, because broad process-definition listing may legitimately be empty.

## Decision: Preserve pipeline validation boundaries

**Rationale**: When a downstream command receives keys through stdin, it no longer has a BPMN selector to validate. The upstream selector command, such as `get pi -b <id> --keys-only`, remains responsible for validation, and downstream keyed commands should preserve existing key semantics.

**Alternatives considered**:

- Try to infer upstream selector context from stdin. Rejected because key streams do not carry reliable BPMN selector metadata.

## Decision: Keep architecture memory unchanged for this feature

**Rationale**: The work stays inside existing command contract, process/incident facade, mutation safety, and docs-generation boundaries. It does not introduce a new runtime owner, external system, deployment assumption, or module boundary.

**Alternatives considered**:

- Refresh the architecture memory. Rejected because the issue is a narrow CLI behavior consistency fix and the existing architecture already covers command contract, mutation safety, and docs-derived behavior constraints.
