# Implementation Plan: Process-Definition Lifecycle State

**Branch**: `348-definition-lifecycle-state` · **Updated**: 2026-10-28 · [Specification](spec.md)

## Scope and structure

Keep one optional backend string throughout existing reads and output. The post-implementation simplification removes duplicate test combinations and shortens documentation; runtime behavior, minimal watch integration and logger/error fixes remain.

Go 1.26.0 / toolchain 1.26.2, existing Cobra/Viper, generated Camunda clients, flat renderer and slog support. No new dependency, persistence, public method, request, polling loop or state machine.

| Owner | Change retained |
| --- | --- |
| `internal/domain/processdefinition.go`, `c8volt/process/model.go` | `State string` with `json:"state,omitempty"`. |
| `internal/services/processdefinition/v810/convert.go`, `c8volt/process/convert.go` | Mechanical state copying. Older adapters leave it empty. |
| `cmd/cmd_views_processdefinition.go` | State in the shared row projection, fixed column before statistics, existing watch comparison/highlighting. |
| `cmd/cmd_views_processdefinition.go`, `cmd/get_processdefinition_watch.go` | Configured watch summary/warning/timing logging; propagate enabled summary-write failure before baseline commit. |
| `cmd/get_processdefinition.go` | Brief help and bounded ordinary-read errors, preserving original JSON/exit evidence. |
| `integration/cli/real_state_destructive_test.go` | Existing draining observer uses CLI JSON; independent cleanup evidence retained. |
| `README.md`, generated `docs/cli/` and `docs/index.md` | Brief lifecycle explanation and one statistics example; regenerate from source. |

The fixed state cell must remain even when empty so mixed rows cannot shift statistics. Watch uses the existing candidate/commit flow: removing its small state comparison would leave a displayed value outside change reporting. Logger fixes must not be reverted to private handlers or direct stderr writes.

## Focused validation ownership

| Concern | Owner; avoid duplication |
| --- | --- |
| Known/unknown/omitted/null backend values | `TestService_ProcessDefinitionLifecycle`; include serialized search response bodies. |
| Public result shapes | `TestConvertProcessDefinitionLifecycle` and existing facade delegation assertions; compare mapped values without re-testing JSON in every shape. |
| JSON field/omission | `TestProcessDefinitionLifecycleJSON`; command tests separately verify real envelopes and streams. |
| Statistics conversion | Existing `TestConvertProcessDefinitionSuspendedStatistics` also asserts lifecycle preservation. |
| Finite commands | Focused state/omission lookup/list × human/JSON checks, existing mode/older-version/mixed-row/error tests. |
| Draining with counts | Four command cases for active=0/suspended=2; other populations remain in adapter tests and unknown/omitted values in mixed-row tests. |
| Layout | One `TestProcessDefinitionLifecycleColumns` table for ordinary/watch rows and missing/unsupported statistics. |
| Watch and diagnostics | Retain deterministic transitions, real-PTY coverage, configured logger matrices, write failures, bounded errors, DEBUG/redaction and original exit/envelope checks. These are distinct boundary checks, not duplicates. |

Run targeted command/facade tests after test consolidation. No adapter/runtime logic changes require new service test runs for the simplification. Regenerate docs with `make docs-content`. Run only lightweight document checks after prose-only edits; no full suite is warranted without wider runtime impact or unresolved failures. See [quickstart.md](quickstart.md).

## Constitution check

All gates remain satisfied: display only supplied evidence; preserve script/output contracts; use proportional isolated validation; keep source/generated docs aligned; reuse existing owners. No exceptions or new complexity. Real Camunda behavior is not an implementation validation gate.

## Supporting design and history

[Research](research.md), [data model](data-model.md), [CLI contract](contracts/cli.md), and [library contract](contracts/library.md) retain detailed representation and compatibility decisions. [Tasks](tasks.md) preserve completed IDs; [progress](progress.md) preserves original execution evidence and the simplification follow-up. The earlier [consolidation review](consolidation-review.md) records the initial task grouping, not a requirement for repeated test matrices.
