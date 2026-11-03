# Specification: Global refactoring preparation

**Status:** Planned only; implementation is not authorized by this planning request.
**Date:** 2026-11-03. **Audit baseline:** `v4.4.0`. **Design baseline:** `33eb3dce`.
**Location:** `specs/371-global-refactoring-preparation/`; 371 is a local sequence, not a GitHub issue association.

## Objective and authorization

Address all twelve findings from the conversation's whole-codebase static audit through
compatible, independently verifiable refactoring outcomes. Preserve CLI behavior and public
APIs. This request authorizes planning artifacts only: no production/test/tooling edits,
application tests, implementation launch, staging, commit or push. Preserve the untracked
`specs/prompts/global-refactoring-scan.md` and completed 370 artifacts. Keep the existing
branch and active-plan pointer unchanged. This directory does not activate implementation.

## Finding inventory

| ID | Finding | Origin relative to v4.4.0 |
| --- | --- | --- |
| F01 | JSON normalization, comparison and snapshots duplicated with divergent typed-value ownership | New |
| F02 | Unused bulk alternatives and `toolx/pool` dependency on facade options | Older |
| F03 | Divergent command failure/exit handling, including raw-error fallback | Mixed |
| F04 | Repeated task/scope association scans during validation and result assembly | New |
| F05 | Repeated suspend/resume traversal, bulk, submission and observation mechanics | New |
| F06 | Repeated bounded human failure aggregation | Mostly new |
| F07 | Version-independent variable-response decoding repeated across adapters | New/extended |
| F08 | Selector backend query and fallback strategy still owned by commands | Older |
| F09 | Deployment-to-start preparation and fallback duplicated in command workflows | Older |
| F10 | Repeated cross-version contract tests and fixtures | Mixed |
| F11 | Planning/execution and models/converters grouped into oversized concern-spanning files | Recent growth/older base |
| F12 | Dead scaffolding, duplicate environment binding and repeated documentation tooling | Mostly older |

The audit is evidence of candidates, not a runtime failure report. All proposed fixes must
be checked against current code; do not invent a fixed defect count or repeat the completed
facade-layering work. No generated Camunda client edits or dependency upgrades are required.

## User stories and acceptance

### US1 — Safe, correctly layered shared primitives (P1)

Library and service consumers retain exact payloads and stable APIs while sharing JSON and
worker mechanics. Isolated ownership probes cover typed collections, custom JSON values,
large numbers and copied plans. Existing pool exports remain callable with the same types,
and internal production packages no longer acquire facade dependencies through worker code.

### US2 — Stable command results and diagnostics (P1)

Operators and scripts retain command grammar, modes, results, exit classes and terminal
controls. Six failure reducers share aggregation without changing established wording.
Command failure routes retain partial results and exactly one supported JSON envelope;
configuration/logger destinations, levels, activity and DEBUG redaction remain effective.

### US3 — Efficient task/scope fan-out (P2)

Equivalent frozen plans yield identical ordered outcomes and mutation counts with indexed
local association lookup. Test-owned fixtures establish linear indexing work; isolated
benchmarks compare sparse and shared-scope shapes without claiming live-engine speedups.

### US4 — Shared process-instance lifecycle mechanics (P2)

Suspend/resume retain separate public models, capability gates, eligibility, success and
failure rules while sharing repeated mechanics. Isolated tests prove every phase preserves
request counts, cancellation, timeout, fail-fast, uncertainty and skipped-slot evidence.

### US5 — Consistent variable-page decoding (P2)

Equivalent native response bodies yield identical domain pages across supported versions.
Required/optional field presence, root/local/effective scope semantics, sparse pages and
cursor/offset behavior remain unchanged. Version-specific request construction stays explicit.

### US6 — Service-owned command preparation (P2)

Selector existence/near-match queries and deployment run-identifier preparation have internal
owners. CLI prompts, rendering stages, option forwarding and fake-backend request transcripts
are unchanged; external implementations of existing facade interfaces still compile.

### US7 — Maintainable contract coverage (P2)

Shared cross-version test cases replace duplicated mechanics while retaining every meaningful
assertion, supported version and wire-specific case. Test helpers cannot reach real services.

### US8 — Cohesive files and maintained tooling (P3)

Concern-specific files, removal of unused internal scaffolding and shared docs tooling reduce
maintenance duplication without removing public symbols, registering a dormant command,
changing configuration precedence or causing publication during validation.

## Binding compatibility requirements

- C01: Preserve existing exported names, signatures, type identity, interface method sets,
  field/tag shapes, error sentinels and constructor behavior. Adding required methods to an
  existing interface breaks external implementers and is forbidden. Additive free functions
  or separate narrow interfaces are allowed only where existing methods cannot express the
  capability; existing interfaces and downstream fakes must continue compiling unchanged.
- C02: Preserve command/flag/alias registration, defaults, validation precedence, payloads,
  tenant/version rules, prompt eligibility/wording/defaults, abort/EOF handling and output modes.
- C03: Preserve partial results, original error class/exit code and existing machine schemas,
  omission/null/empty-array conventions and zero-byte empty keys output. Shared JSON is one
  envelope followed by EOF. Render failures must not trigger a second result write.
- C04: Preserve selected-format stdout, plain configured-stderr prompts and configured logger
  diagnostics. Successful mutation/dry-run/verbose summaries remain INFO and obey WARN/quiet.
  Retain established bounded wording and redacted DEBUG behavior. Never preserve raw HTTP
  disclosure by weakening mandatory repository logging rules.
- C05: The preservation request does not silently authorize a new public behavior. If a raw
  legacy diagnostic conflicts with mandatory logging rules, implement only compatible routing
  and plumbing; record the exact bounded replacement as a separate remediation requiring
  explicit acceptance before changing observable wording. Do not call the affected slice
  complete while an applicable required diagnostic assertion remains failing. Other slices
  may proceed. No aliases/flags/schemas/exit rules may be changed to resolve that conflict.
- C06: Preserve accepted input and serialized JSON, exact numeric values, nil/empty conventions
  and documented concrete value representation. Shared snapshots must isolate accepted mutable
  JSON storage; accidental aliasing is not the frozen-plan ownership contract. Characterize
  caller-visible representation before migration; retain wrappers where policies differ.
- C07: Preserve backend call counts/order where established, stable result ordering, frozen
  discovery, sparse-page continuation, mutation acceptance versus confirmation, and existing
  retry/no-wait/fail-fast/context semantics. No cross-invocation cache or speculative parallelism.
- C08: Keep generated mappings and endpoint-specific rules explicit. Use existing helpers first;
  no universal workflow/pagination engine or new dependency is justified by this plan.
- C09: Every implementation unit includes its necessary tests/fakes, interface/consumer updates,
  comments, formatting and relevant isolated validation; no standalone setup/review/commit tasks.
- C10: Never run real Camunda, cluster, smoke or integration-tagged tests as implementation
  validation. Preserve historical evidence and state any unverified engine behavior accurately.

## Completion criteria

All findings map to tasks or explicit compatibility dispositions; all tasks preserve C01–C10.
Executed units compile and pass their targeted isolated checks. A dependency/architecture scan
shows neutral internal worker paths and thin facade/command adapters. Comparable API/wire/CLI
contracts pass, and any independently accepted diagnostic remediation is reported separately.
Planning-only delivery is complete when the artifact set, links, task graph and coverage mapping
are validated; it does not establish any implementation or runtime pass.
