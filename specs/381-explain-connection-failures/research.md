# Research: Explain Known Connection and Authentication Failures

**Date**: 2026-11-18 | **Branch**: `381-explain-connection-failures` | **Baseline**: `10bd90ae`

Research used repository source and focused read-only research agents. No runtime tests, external technology selection, live services, or implementation edits were required. All design questions below are resolved.

## 1. Where the original message originates

**Decision**: Change `boundedCommandFailureReason`, preserving existing callers and error handling.

**Rationale**: `cmd/get_processdefinition.go` already supplies `search process definitions: ` plus the bounded reason to `handleCommandError`. `cmd/cmd_views_contract.go` currently reduces that reason to the class except for request-rate waits. The existing handler logs the human override through the configured logger and ignores it for JSON. The defect is thus missing explanation in one shared pure formatter, not missing routing infrastructure.

**Alternatives considered**: Rewriting every caller duplicates policy. Changing `ferrors.HandleAndExit` affects unrelated output. Importing #379 violates the requested baseline and scope.

## 2. Recognition and wording ownership

**Decision**: Extract a private recognized-reason function from `configConnectionFailureMessage` in `cmd/cmd_views_config_test_connection.go`; return a reason and a recognition boolean. Keep `commandConnectionFailureDetails` as the sole existing adapter to transport evidence.

**Rationale**: The connection view already owns bounded advice for typed timeout, DNS, TLS, invalid response, refused/reset/EOF connection errors, HTTP rejections, and authentication failures. `cmd/cmd_errors.go` delegates fact inspection to `httpc.FailureDetails`. Keeping the wording together avoids drift and permits a tiny call in the shared formatter.

**Alternatives considered**: Duplicating a switch in `cmd_views_contract.go`, adding a new diagnostic framework, and inspecting error strings were rejected. Moving wording into `httpc` mixes CLI guidance with transport mechanics.

## 3. Caller-specific behavior and precedence

**Decision**: Preserve request-rate handling first; generic connection recognition declines cancellation and unknown causes. Keep connection-test prefix, cancellation text, topology-404 wording, and class-only fallback local. Use only a small explicit context to choose topology versus generic endpoint wording inside the one status mapping.

**Rationale**: Request-rate output explains unsent work without denying earlier submissions. The connection test currently says `connection test canceled`, which is inappropriate elsewhere. Its gateway 404 identifies the topology endpoint; ordinary operations cannot make that claim. Authentication 400/404 advice is already service-specific and reusable. Generic unknowns must return the existing class rather than connection-test setup advice.

**Alternatives considered**: Reusing the entire prefixed config message, stripping prefixes from its string, or globally applying config fallbacks would change unrelated contracts. New status/context type hierarchies are unnecessary for one endpoint-label distinction.

## 4. Typed evidence and normalization limits

**Decision**: Consume only evidence currently available to each caller. Preserve the existing `FailureDetails` ordering and joined-error semantics; leave facade conversion unchanged.

**Rationale**: `internal/services/httpc/failure.go` obtains status/auth annotations and malformed-response evidence through `errors.As`, then derives transport facts from types. `diagnostics_redaction.go` handles cancellation, deadline/timeout, DNS, certificates, and supported connection causes. `c8volt/ferrors/errors.go` preserves original cause trees for HTTP/auth annotations, request-rate errors, and mutation failures, but bare transport causes can be reduced to class/text wrapping. Generic rendering must not infer lost evidence from text. Joined-error annotations can originate in different branches; this issue must not introduce a new attribution model.

**Alternatives considered**: Expanding `ferrors.wrap` would change compatibility across packages. Parsing retained strings would violate typed-evidence requirements. Reporting a precise cause for class-only failures would invent evidence.

## 5. Isolated reproduction and test reuse

**Decision**: Add formatter cause/precedence tests and extend existing command-execution fixtures. For the original failure, use an OAuth token request through a child-process-only fake default transport returning typed `ECONNREFUSED`; disable version advisory only in that fixture and set zero retries.

**Rationale**: This reproduces the authentication annotation, facade conversion, command prefix, configured logger, JSON handling, and exit behavior without a real refused service or nondeterministic port. `LogTransport` falls back to `http.DefaultTransport`; existing subprocess scoping confines the replacement to one child. Unknown requests must fail locally rather than delegating to the network.

Reuse:

- `cmd/get_processdefinition_error_envelope_test.go`: `TestGetProcessDefinitionLifecycleSearchDiagnostics`, subprocess helper, both logger destinations/formats, cause-safe output, machine evidence, and request counts.
- `cmd/config_test_connection_diagnostics_test.go`: `TestConfigConnectionFailureReasons` and `TestConfigConnectionDiagnosticLogging` retain exact specialized advice and actual fake authentication flows.
- `cmd/delete_processdefinition_unfinished_test.go`: `TestDeleteProcessDefinitionUnfinishedDiagnosticLogging` covers repeated failures, exact aggregate counts, partial JSON with EOF, DEBUG, quiet, and configured/inherited writers.
- `cmd/request_rate_mutation_test.go`: local unsent, interruption, confirmation, and partial-reporting cases; `isolateRequestRateEnvironment` clears inherited target/helper state.
- `cmd/update_job_batch_logging_test.go`: `bulkCompletionDiagnosticRecords` parses whole `plain-time` and `json` records; it does not accept `plain`. Retain the search fixture's dedicated `plain` assertions instead of misusing this parser.

**Alternatives considered**: Closed-port probing is unnecessary; test-owned fake transports are deterministic. A new batch/subprocess framework duplicates existing coverage. Renderer-only assertions cannot prove logger routing or exit behavior.

## 6. Documentation and check selection

**Decision**: Update README, configuration diagnostics guidance, and a concise root help sentence, then use `make docs-content` during implementation. Planning runs document checks only; implementation starts with the targeted command checks in the quickstart.

**Rationale**: README currently distinguishes verbose and DEBUG, while configuration docs describe actionable causes primarily for connection tests. The corrected shared behavior needs a compact clarification. `Makefile` runs `go run ./docsgen` for docs content; `docsgen/main.go` constructs `cmd.Root()` metadata without executing API commands and syncs README to `docs/index.md`. The constitution requires validation proportional to the diff; this feature introduces no service/concurrency changes requiring a mandatory full suite.

**Alternatives considered**: Editing generated CLI Markdown is prohibited. Building the entire documentation site adds unrelated dependencies. Runtime tests for planning documents or live Camunda validation provide no relevant evidence and are not completion gates.
