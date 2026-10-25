# Ralph Progress Log

Feature: 346-suspend-process-instances
Started: 2026-10-25 14:42:48

---


---
## Iteration 1 - 2026-10-25 14:50
**Work Unit**: Phase 2 Foundational — T001 Authoritative Submission Evidence; completed
**Tasks Completed**:
- [x] T001: Added suspension domain evidence models, private v810 exact-key submission, narrow generated-client contract/mock support, and isolated protocol/retry/pacing/diagnostic tests.
**Tasks Remaining in Work Unit**: 0; T002 and T003 remain in subsequent phases.
**Commit**: This work-unit commit
**Files Changed**:
- internal/domain/processinstance_suspension.go
- internal/services/processinstance/v810/contract.go
- internal/services/processinstance/v810/service_test.go
- internal/services/processinstance/v810/suspension_submission.go
- internal/services/processinstance/v810/suspension_submission_test.go
- specs/346-suspend-process-instances/tasks.md
- specs/346-suspend-process-instances/ralph-memory.md
- specs/346-suspend-process-instances/progress.md
**Learnings**:
- Retain evidence outside retry returns and restore attempt-local admission classification after HTTP timeout substitution. Earlier uncertainty survives rejection/unsent attempts; only valid empty 204 establishes acceptance.
- Compiling-stub tests first demonstrated missing behavior. Final targeted submission tests passed; full v810 adapter tests passed with `-race -count=1`, covering the updated mock contract and existing cancellation/deletion/state/date behavior. `GOENV=off GOFLAGS=` was used after inspecting isolated test entry points; no live validation occurred.
- Generated serialization sends POST with {} only to the exact selected key. Tests assert attempt/dispatch counts, typed errors, no observation, rejection/proxy differences, retry exhaustion/backoff cancellation, local unsent evidence, actual Client.Timeout substitution and preserved status.
- Failure routing review: helper returns typed causes plus machine-only error evidence and emits no human diagnostics. Plain/JSON configured logger tests prove normal-level silence, no retry chatter, and retained redacted DEBUG records. CLI aggregate rendering/logger/exit checks remain T003 responsibility because this unit exposes no command.
- Fixed a virtual-time fixture race by waiting for context completion explicitly. No shared runtime helper or generated client changed; docs regeneration and full repository race suite are deferred to their owning integrated work unit.
- Formatting and whitespace checks passed. No tasks were added or scope crossed; next work is T002.
---
