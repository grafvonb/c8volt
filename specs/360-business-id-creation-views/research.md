# Research: Business-ID Creation and Resource Views

**Date**: 2026-10-31
**Feature**: [spec.md](spec.md)

Research used repository code and read-only upstream source. No live service, integration test, or runtime test was used. The planning research agents independently reviewed creation, model propagation, and output/validation paths; their findings are consolidated here.

## 1. Supported-version boundary

**Decision**: Implement native business-ID support in v810. Reject a non-nil creation business ID at the v88/v89 adapter entry with `domain.ErrUnsupported`, before submission. Leave older read mappings and the default version unchanged.

**Rationale**: `toolx` and the service factories select explicit versions. Generated v88/v89 types happen to contain some business-ID fields, but issue #360 explicitly limits this feature to 8.10+. Public calls reach the same adapters, so CLI-only checks would be insufficient. Generated v810 clients already expose the needed request and result fields.

**Alternatives considered**: Inferring support from generated field presence would violate scope. A generic capability framework or client regeneration adds unnecessary work. No new version is introduced.

**Evidence**: `internal/services/processinstance/{factory.go,v88/service.go,v89/service.go,v810/service.go}`, `internal/clients/camunda/v810/camunda/client.gen.go`, `api/README.md`.

## 2. Exact input and Unicode contract

**Decision**: Use optional `*string` input. Nil means omitted; non-nil empty is invalid. Preserve whitespace, commas, case, and Unicode without normalization. Validate at most 256 UTF-16 code units, counting BMP characters once and supplementary characters twice. Reject malformed UTF-8 public Go strings rather than allowing JSON encoding to replace bytes and silently alter an opaque ID. Keep the validator in a narrowly named domain file such as `internal/domain/business_id.go`, used by service entry points; CLI parsing preserves presence while authoritative capability and value validation remain below the facade. v88/v89 reject any non-nil ID before value validation; v810 validates its value before submission. No new dependency is required.

**Rationale**: `api/camunda/zeebe/gateway-protocol/src/main/proto/v2/identifiers.yaml` specifies nonempty strings and maximum length 256. The repository's v810 source baseline is Camunda 8.10.3, commit `3961b79c3fdaf43c48691ed77672d6db42d0abcc`. At that revision, both the REST validator and engine use Java `String.length()`, which counts UTF-16 units. The custom Java maximum check does not itself reject empty strings; explicit-empty rejection follows the issue and checked-in schema. Do not borrow blank-string rules from late assignment.

**Primary evidence**:

- [RequestValidator: maximum 256 and Java length check](https://github.com/camunda/camunda/blob/3961b79c3fdaf43c48691ed77672d6db42d0abcc/gateways/gateway-mapping-http/src/main/java/io/camunda/gateway/mapping/http/validator/RequestValidator.java#L214-L219).
- [ProcessInstanceRequestValidator: both creation variants](https://github.com/camunda/camunda/blob/3961b79c3fdaf43c48691ed77672d6db42d0abcc/gateways/gateway-mapping-http/src/main/java/io/camunda/gateway/mapping/http/validator/ProcessInstanceRequestValidator.java#L64-L98).
- [Engine BusinessIdValidator](https://github.com/camunda/camunda/blob/3961b79c3fdaf43c48691ed77672d6db42d0abcc/zeebe/engine/src/main/java/io/camunda/zeebe/engine/processing/processinstance/BusinessIdValidator.java).

| Input | Expected local result |
| --- | --- |
| Nil | Omit businessId |
| Empty string | Invalid input, no mutation |
| Space or `  order,AbC  ` | Accepted unchanged |
| 256 ASCII or BMP characters such as é | Accepted |
| 257 BMP characters | Rejected |
| 128 supplementary characters such as 😀 | Accepted: 256 units |
| 129 supplementary characters | Rejected: 258 units |
| 254 BMP characters plus one supplementary character | Accepted |
| 255 BMP characters plus one supplementary character | Rejected |

**Alternatives considered**: Byte length rejects valid multibyte characters; rune count accepts IDs rejected by the backend; trimming or comma splitting changes identity. A custom string type or new facade option is unnecessary for an optional creation field.

## 3. Nullable resource observations and copied plans

**Decision**: Add `BusinessId *string` with public/output tag `json:"businessId,omitempty"`. Copy pointers across mutable model boundaries with `toolx.CopyPtr`. Preserve nil, backend-provided empty strings, and differing related-resource values. Do not validate observed output values against creation rules.

**Rationale**: Optional pointers distinguish absence from an explicit input and preserve backend facts. No business-ID field exists in the current domain/facade models. Existing enrichment wrappers embed resource models; their shape need not change. Runtime listener collection pointers distinguish unrequested from requested-empty and must remain untouched.

**Alternatives considered**: Plain strings collapse absent and empty. Parent fallback or creation-input fallback manufactures facts after late assignment. Adding a new enrichment section duplicates identity and changes established output contracts.

**Evidence**: `internal/domain/{processinstance.go,job.go,usertask.go,job_completion.go,ops_slow_process_analysis.go}`, `c8volt/{process,job,task,element,ops}/model.go`.

## 4. Creation request, waiting, and partial results

**Decision**: Extend `ProcessInstanceData` and the existing creation methods without changing their signatures. Populate both branches of `v810.toProcessInstanceCreationInstruction`, map native creation responses, and replace the returned business ID with the actual observed ID after successful ordinary waiting, including an observed nil. Preserve the native response value under no-wait.

Preserve accumulated successful results and the original error together through sequential service creation, counted creation, facade conversion, and CLI output. Filter failed/unattempted zero-value pool slots in the process-instance creation workflow using successful completion evidence; do not alter general pool behavior. Preserve existing order, worker bounds, fail-fast scheduling, retry policy, and wait semantics.

**Rationale**: `internal/services/processinstance/bulk.go:CreateProcessInstances` currently discards accumulated results on a later failure. `c8volt/process/client.go:CreateProcessInstances` and `bulk.go:CreateNProcessInstances` also discard results on error. The pool returns fixed-length results, including zero values. `cmd/run_processinstance.go` exits through `handleCommandError` before rendering partial results. Fixing only the command would therefore be insufficient.

**Alternatives considered**: New bulk result schemas or pool redesign exceed the feature. Retrying with generated alternate IDs violates opaque input and backend uniqueness. A waiting failure must not become verified success; this feature does not redesign single-attempt uncertain-submission evidence.

## 5. Complete propagation requires explicit copy-site coverage

**Decision**: Cover generated-to-domain, domain-to-public, public-to-domain, and custom reconstruction paths; use the [resource-view contract](contracts/resource-views.md) as the acceptance inventory.

**Rationale**: Updating only the ordinary process converter misses these paths:

- `c8volt/ops/convert.go` has an independent `fromDomainProcessInstance`.
- `RuntimeListenerJobFromJob` and three public listener converters copy observations.
- `internal/services/job/completion/prepare.go` reconstructs candidates; domain/public candidate shapes are directly cast today. Add the field to both together, copy the pointer in both directions, and snapshot it in `execution_plan.go`.
- `slowProcessAnalysisRootFromProcessInstance` and the slow root renderer rebuild process data.
- Process-instance dry-run JSON uses bespoke structs, and `aggregateDeleteSearchPlan` reconstructs instances from preview entries.
- Operations Markdown writers enumerate scalar fields rather than serializing the public model.

**Alternatives considered**: An extra lookup on every output would add requests and could replace historical snapshots with current facts. Generic reflection-based mapping would hide omissions rather than make version and ownership boundaries explicit.

## 6. Output and diagnostic ownership

**Decision**: Move the touched dedicated creation renderer and row builder into `cmd/cmd_views_processinstance_run.go`; keep flags and dispatch in `run_processinstance.go`. Add a result/error renderer that places the existing `ProcessInstances` payload in one failure envelope, then exits with the original class. Use configured INFO logging for human creation results/summaries and existing compact ERROR patterns for failures. Keep ordinary query rows on stdout, keys-only as keys, and prompts plain stderr.

**Rationale**: The existing creation renderer uses `renderOutputLine` for mutation rows and `found:`. That is an existing violation, not a pattern to extend. `jobCompletionBatchView`, `resultEnvelopeForError`, `exitAfterRenderedResult`, and the job/process logging view files provide applicable local patterns. Raw joined errors must not appear at normal or verbose levels.

**Alternatives considered**: Calling `handleCommandError` before partial rendering loses data; calling it after the envelope duplicates JSON. Adding logger prefixes to message text breaks configured formatting. Making verbose enable HTTP detail violates the independent DEBUG contract.

## 7. Validation, documentation, and work-unit boundaries

**Decision**: Use targeted unit, fake-client, test-owned HTTP, command subprocess, and terminal tests; generate CLI docs from source. After targeted tests pass during implementation, run the race-enabled repository suite once because model propagation and creation failure paths span multiple packages. During planning, run document checks only.

**Rationale**: `Makefile`'s `test` target is the isolated default suite; live suites use the `integration` build tag and separate targets. Clear inherited flags and live configuration before testing. Command tests must inspect configured/inherited logger destinations, levels, plain and structured formatting, exact envelope EOF, request counts, and original exit status.

**Alternatives considered**: Renderer-only or substring-only assertions miss logger bypasses. Live Camunda validation is prohibited during implementation. A standalone cleanup/testing task or separate single-item then bulk implementation would fragment cohesive work.

All research questions are resolved. Future actual-engine behavior remains unverified; the pinned source evidence and fake-based acceptance tests define the implementation contract without a live completion gate.
