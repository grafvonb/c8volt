# Implementation Plan: Advisory Camunda Version Check

**Branch**: `222-camunda-version-check` | **Date**: 2026-10-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/222-camunda-version-check/spec.md`, originating in GitHub issue #222.

## Summary

Add a default-on, invocation-scoped gateway version check immediately before the first operational request. Compare major/minor versions, warn once on mismatch or check failure, and retain configured adapter selection and operational outcomes. `--no-version-check` disables the automatic advisory.

A pre-Do wrapper on generated service clients consults a context-scoped checker before entering the ordinary HTTP client's timeout. The checker belongs to the version-neutral cluster service, reuses existing topology adapters, and applies a two-second cap, no retries, and a recursion bypass. Existing authentication, pacing, activity, and redacted diagnostics remain shared. CLI code installs the service and renders typed outcomes with its configured logger: WARN advisories and one DEBUG summary of the same topology response.

## Technical Context

**Language/Version**: Go 1.26.0; repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, Viper 1.21.0, standard `context`/`net/http`/`sync`/`log/slog`, testify 1.11.1, generated Camunda v88/v89/v810 clients. No new dependencies.

**Storage**: Invocation-memory state only; no persistent cache or new configuration field.

**Testing**: Go unit tests, existing generated-client doubles, test-owned HTTP servers, separate-stream command/subprocess tests, and existing real-terminal fixtures where prompt eligibility matters. No live environments.

**Target Platform**: Existing CLI-supported Go targets; retain cross-platform behavior and existing platform-specific terminal test handling.

**Project Type**: Go CLI with public facade and version-neutral/versioned internal services.

**Performance Goals**: At most one logical advisory attempt per eligible invocation, no retry, at most two seconds of check waiting or a shorter applicable timeout; no extra request on local/disabled paths. Concurrent callers share the attempt and can cancel their wait.

**Constraints**: Preserve operational HTTP timeout and parent cancellation, output/envelope/exit contracts, original adapter selection, configured logger routing, shared pacing, authentication, and mutation evidence. Do not modify generated clients, probe live endpoints, add polling, or introduce public facade signature changes.

**Scale/Scope**: All CLI operational requests using 11 service areas across three versions. Mechanical wiring changes span 33 service constructors and 36 generated-client arguments, including legacy Operate and Tasklist. Standalone library callers without the invocation context retain current behavior.

## Constitution Check

Both the pre-research and post-design checks pass. No exception is requested.

| Principle or repository gate | Before research | After design |
| --- | --- | --- |
| Operational proof | Advisory outcome cannot substitute for operation success. | Checker reports match/mismatch/failure facts only; actual response, verification, and failure exits remain authoritative. |
| CLI-first, script-safe | Add one discoverable opt-out without changing result schemas. | WARN uses configured logger; JSON/keys/prompt behavior remains separate; explicit connection-test JSON evidence is retained. |
| Proportional validation | Planning requires document checks only. | Implementation starts with focused tests, then full race suite due to shared request plumbing and concurrency. All dependencies are isolated. |
| Documentation accuracy | User-visible default and latency require documentation. | Root help, metadata, README and generated docs change with activation; no hand-editing generated docs. |
| Repository-native scope | Reuse cluster topology, shared HTTP, logging, and test helpers. | No new dependency or service package; three cohesive implementation outcomes below. Wider mechanical adapter wiring prevents timeout corruption. |
| Layering and file cohesion | Mechanics belong in internal services. | `httpc` owns request gate plumbing, `cluster` owns policy, cmd owns flag/composition/rendering; no direct generated/versioned access from cmd and no new lifecycle inside root.go. |
| Diagnostic and empty-result contract | No added result payload or raw diagnostic detail. | Advisory warnings are bounded facts; local empty previews perform no check; discovery-derived emptiness checks once before discovery. |
| Implementation testing boundary | No live-target designation or check. | Quickstart uses isolated test commands only; real-engine behavior remains unverified by this workflow. |

## Project Structure

### Documentation (this feature)

```text
specs/222-camunda-version-check/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/cli-version-check.md
```

`tasks.md` is produced by the subsequent tasks workflow, not this planning command.

### Source Code (repository root)

```text
cmd/
├── root.go                              # Flag and help
├── root_services.go                     # Install invocation support after auth
├── root_version_check.go                # New focused wiring/outcome access
├── cmd_views_version_check.go           # New bounded WARN presentation
├── config_test_connection.go            # Deduplicate human warnings, preserve JSON
├── config_diagnostics.go                # Bound human fallback; retain machine warning logic
└── root_version_check*_test.go          # New execution and logging coverage
internal/domain/version_check.go        # New version-neutral outcome facts
internal/services/
├── httpc/before_request.go              # New pre-Do wrapper/context gate/bypass
├── common/deps.go                       # Add APIClient; retain HTTPClient
├── cluster/version_check.go             # New single-attempt policy and comparison
└── <area>/{v88,v89,v810}/service.go      # Mechanical generated-client wiring
    # Areas: batchoperation, cluster, element, incident, job,
    # processdefinition, processinstance, resource, tenant, usertask, variable
toolx/camunda_release.go                 # New bounded parser for advisory and safe display
README.md                               # Default/opt-out/latency/compatibility guidance
docs/cli/*, docs/index.md                # Regenerated from command source and README
```

New behavior has tests beside its owner. Existing HTTP, logging, request-rate, connection-test, and strict fake-server fixtures receive only required compatibility updates. Names above are proposed files, not claims that implementation exists.

**Structure Decision**: Keep the existing layers and public API. The HTTP package depends only on a small internal gate contract, not the cluster package; the cluster checker can reuse `httpc` controls without a cycle. Command composition may construct this neutral support service, just as it constructs auth/HTTP services, but does not execute generated or version-specific operations. No generic workflow framework is introduced.

## Phase 0: Research Results

[research.md](research.md) resolves request timing, timeout isolation, version source, concurrency, auth/pacing interactions, parsing, duplicate warnings, and fixture impact. A RoundTripper-only solution was rejected because it consumes the first operation's timeout during preflight. No unresolved research question remains.

## Phase 1: Design

### Invocation and request flow

1. Root configuration and logger resolution remain unchanged. Parse `--no-version-check` as an invocation flag; explicit false leaves checking enabled. Reset it in existing command-test cleanup alongside other flags.
2. `installRemoteCommandServices` constructs shared HTTP/auth and completes existing auth initialization. If enabled, attach a newly allocated cluster checker to the resulting invocation context through the focused command support helper. Construction performs no version request. Any lazy probe-construction failure becomes an advisory failure, not a new bootstrap rejection.
3. `PrepareServiceDeps` supplies an `APIClient` Do wrapper in addition to the original concrete `HTTPClient`. All generated operational clients use `APIClient`; auth and nested service construction retain the concrete client as appropriate.
4. At the first generated request's Do boundary, the wrapper checks context cancellation and the bypass marker, then joins/starts the checker. The winner performs one probe; other callers wait for completion or their own cancellation. All terminal outcomes consume the single attempt.
5. The probe uses the effective configured cluster factory and a shallow client copy with shared transport/jar and advisory-only redirect refusal. Its derived context bypasses the gate, disables read retries, owns a fresh pacing observation, and is capped at `min(2s, positive HTTP timeout, remaining caller deadline)`. No verbose facade options are passed.
6. Checker classification produces a bounded typed outcome. A callback renders mismatch/failure once through the configured WARN logger; matches remain silent outside DEBUG. The callback also emits a single DEBUG summary with safe gateway/broker versions, available cluster counts, and the fixed failure category. Publish the outcome and completion only after callback handling so concurrent operations cannot precede its warning.
7. If the original caller remains active, delegate the original request to the original HTTP client. Its ordinary timeout starts now; no probe context, error, or pacing observation becomes an operational result. If canceled, return the original cancellation class without sending the operation and clean up an unconsumed request body.
8. Subsequent operations skip the completed check. Context inheritance spans pagination, batches, polling, nested services, and repeated facade construction. A new invocation installs fresh state; authentication calls never consult the gate.

### Output and compatibility

Use [contracts/cli-version-check.md](contracts/cli-version-check.md) for exact messages and flag behavior. `cmd/cmd_views_version_check.go` formats only typed, bounded facts and uses the configured logger; no error string or raw observed version crosses this boundary. Existing DEBUG transports retain responsibility for detailed redacted evidence. Normal version warnings never pass through `ferrors` or command result renderers.

`config test-connection` retains its explicit topology request and actual-result JSON warnings. A checker warning outcome suppresses its additional legacy human warning, even if WARN was filtered. A silent match leaves a subsequently changed actual result eligible for its existing compatibility diagnostic. With the automatic check disabled, explicit connection-test diagnostics still apply. Unsafe fallback diagnostic values are bounded using the strict parser without changing the legacy parser/classification or machine warning strings, even for historically permissive inputs. Explicit `get cluster version` and `get cluster topology` perform their ordinary request in addition to the separate advisory attempt.

Compatibility changes are intentional: one default extra lookup, one pacing slot when dispatched, bounded preflight latency, and one compact advisory instead of duplicate connection-test diagnostics. Existing operation timeout, selected adapter, result structure, exit behavior, prompts, and library consumers without a gate remain unchanged. The check's two-second limit is not a total-command-duration guarantee under rate limits or existing workflow deadlines.

### Validation design

| Coverage group | Required evidence |
| --- | --- |
| Comparison and errors | Numeric major/minor, patch and prerelease matches, major/minor mismatch, denied access, malformed/absent/unsafe versions, timeout, unavailable response, redirect refusal; one attempt and no raw reason text. |
| Invocation gate | Zero local/disabled requests; first operational ordering; concurrent starts; cancellation while waiting; one failed attempt retained; fresh invocation state; recursion bypass; standalone pass-through. |
| Timeout/auth/pacing | Short HTTP timeout expires probe but allows a fast operation with its own budget; parent deadline prevents dispatch; auth bootstrap/refresh excluded; one shared limiter and DEBUG counter; fresh probe evidence cannot mark mutation unsent. |
| Actual command execution | Read, discovery dry-run/empty result, pagination, batch, polling, explicit version/topology, local preview/validation/abort, and remote `config test-connection`; cover all three adapters and relevant legacy clients. |
| Diagnostics | Complete WARN message/severity and configured/inherited destinations in plain and structured formats; quiet/ERROR suppression; DEBUG by flag and by config independently of verbose; redaction on both streams. |
| Partial failures and results | Failed check then repeated batch backend failures plus one success: one advisory WARN, one compact aggregate ERROR, original exit, retained evidence, exactly one JSON envelope plus EOF, exact keys/empty keys; quiet suppresses WARN but retains ERROR. |
| Existing contracts | Matching topology fixture updates and explicit independent-test opt-outs; preserve command help/capabilities, real-terminal prompt routing, no-op modes, service retries, request-rate behavior, and configured output precedence. |

Checks must fail if WARN is replaced with a direct stderr print, if raw errors appear at normal/verbose levels, if a duplicate envelope is emitted, or if a probe pacing failure contaminates operational submission evidence. Use real terminal stdin with separate streams for prompt-eligible local abort/empty completion cases; introduce no new prompt.

See [quickstart.md](quickstart.md) for safe runnable checks after implementation. For this planning-only change, validate headings, references, feature identity, requirement coverage, placeholders, and whitespace; do not run Go tests or docs generation.

### Cohesive implementation outcomes for task generation

1. **Checker and release comparison**: bounded parser, typed outcome, required context gate/bypass contract and pre-Do wrapper, concurrency/cancellation policy, probe deadline/no-retry/pacing isolation, and isolated unit tests together. Keep required parser/checker tests in that unit so all dependencies compile together; do not replace legacy connection-test machine-warning classification.
2. **Shared operational request wiring**: connect the completed pre-Do wrapper through additive `ServiceDeps.APIClient`, all 36 generated-client call-site changes, necessary doubles, and timeout/pass-through/adapter coverage together. Keep the public concrete-client signatures and nested construction working throughout.
3. **CLI activation and complete operator contract**: flag, invocation install, configured WARN rendering, connection-test deduplication, fixture adjustments, execution/diagnostic/partial-output acceptance tests, help/README metadata, docs regeneration, formatting, and targeted plus final race validation together.

These are design outcomes, not a task checklist. Adjacent work was reviewed for consolidation: parser/checker policy can be tested independently; shared request wiring is a mechanical compiling unit; activation owns all observable behavior and its tests/docs. Do not split each adapter or user story into duplicate implementations, separate mandatory tests from their behavior, or add context-reading/branch checks as standalone tasks. If activation exceeds a practical iteration, subdivide only at independently verifiable behavior boundaries while retaining complete tests per unit.

## Complexity Tracking

No constitution violations. The only substantial breadth is mechanical generated-client wiring in 33 constructors. It is justified by preserving the HTTP timeout outside the advisory; a smaller transport interceptor would violate that requirement. Additive internal Do capability avoids public API changes and preserves the existing concrete HTTP client for nested construction.
