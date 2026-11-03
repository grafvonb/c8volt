# Research: Standardize Flag Intent

## Scope and evidence

Research used the checked-out repository and the accepted specification. The speckit-plan research agent independently inspected parsing, configuration, and early error handling. No runtime tests, external service calls, or dependency upgrades were required. All design questions are resolved below.

## 1. Keep migration in command ownership

**Decision**: Change command registration, presence detection, validation, input mapping, metadata, examples, and tests. Preserve public facade/domain fields and backend service behavior.

**Rationale**: `cmd/update_variables_payload.go` shares variable-source selection across updates; `cmd/ops_repair_variables.go` has separate optional payload logic. `cmd/update_usertask_assignment_input.go` and `cmd/update_usertask_variables.go` inspect assignment presence independently of registration. `cmd/update_processinstance_assignment.go` preserves BID-versus-variable exclusions. Updating registrations alone would silently drop inputs or route to the wrong mode. Existing mode files remain owners of their respective lifecycles.

**Alternatives considered**: Global string replacement would damage valid query filters, JSON properties, public fields, historical specifications, and execution controls. A new facade abstraction is unnecessary because mutation semantics do not change.

## 2. Split retry predicates and submitted values

**Decision**: Register distinct job retry storage/presence for `retries` (predicate) and `set-retries` (mutation). Update `cmd/update_job_request.go`, `cmd/update_job_outcome.go`, and `cmd/update_job_batch.go` together. Preserve explicit zero through pointer/presence semantics. Non-completion mutation modes reject bare retries with replacement guidance; keyed completion continues to reject retry predicates.

**Rationale**: The current `cmd/update_job.go` registration stores all three meanings in one flag. Batch completion already has a distinct selection request. Separating command input before request construction lets the existing facade/service requests remain unchanged.

**Alternatives considered**: An alias or mode-dependent mutation fallback contradicts the chosen clean break. Treating numeric zero as absence breaks both selection and failure submission.

## 3. Restore HTTP timeout binding

**Decision**: Remove the local job `timeout` registration and use `set-timeout` for the mutation. Keep `http.timeout` bound exclusively to inherited `timeout` in `cmd/root_config.go`; verify the resolved flag belongs to the inherited execution control. Replace every job outcome exclusion/presence test referring to mutation timeout. Do not add a job mutation configuration key.

**Rationale**: `initViper` binds `http.timeout` via `cmd.Flags().Lookup("timeout")`. A local job flag currently shadows the persistent flag from `cmd/root.go`, so a parsing-only test misses incorrect configuration. A key with only HTTP timeout provides no mutation and must fail with job-timeout guidance. A valid canonical mutation may use HTTP timeout independently, including alongside job `set-timeout`.

**Alternatives considered**: Retaining a local alias would preserve shadowing and ambiguity. Renaming the HTTP configuration key would unnecessarily break unrelated callers.

## 4. Make the BPMN-error action strictly valueless

**Decision**: Canonical syntax is a bare `--throw-bpmn-error` plus `--set-error-code CODE`; the optional message is `--set-error-message`. Reject every attached value (`=CODE`, including `=true`, `=false`, and empty) and separated legacy positional codes. Do not trim or normalize accepted nonblank codes beyond existing validation. Explicit false with a code is invalid; attached Boolean forms do not become an alternate canonical spelling.

**Rationale**: Ordinary Boolean parsing would silently accept old codes named `true` or `false`. `cmd/update_job_request.go` permits only the existing optional stdin dash; scalar error mode has no positional code contract. Existing outcome construction already maps a code into a worker outcome request and can retain that structure.

**Alternatives considered**: A string flag with an optional value keeps the old ambiguity. A plain Boolean conversion without token-aware validation silently reinterprets Boolean-looking old codes.

## 5. Reject removed syntax without aliases

**Decision**: Maintain a small command-scoped old-to-new diagnostic map in proposed `cmd/flag_migration.go`. Do not register legacy names as active, deprecated, or hidden aliases. Use the map only to explain a rejected invocation and detect old/new conflicts. Retained bare retries and HTTP timeout are handled by semantic validation, not this map. Canonical help/capability/completion surfaces therefore expose only current names.

**Rationale**: The user selected immediate v5 break with no transition. The table is diagnostic metadata, not a compatibility execution layer. Diagnostics must not quote arbitrary supplied payloads or raw argv.

**Alternatives considered**: Hidden reject-only legacy registrations simplify parsing but can leak obsolete names into introspection and completion. Generic unknown-flag text lacks actionable replacement guidance.

## 6. Prepare configured error context before remote bootstrap

**Decision**: Extract/reuse local configuration and logger preparation from `cmd/root.go`. Preserve the executing leaf command and original invocation arguments at the execution boundary. For affected parse/argument failures, use a bounded token-aware diagnostic preparation helper, proposed `cmd/flag_migration_input.go`, to recover only known configuration/logging/output controls, including those after the rejected token. Build an isolated diagnostic flag context using registered flag arity and inherited metadata, then route the classified error through the existing shared command error renderer. Never replay normal execution or install remote services to report migration failures.

**Rationale**: `useInvalidInputFlagErrors` in `cmd/get_processinstance_validation.go` classifies errors but does not create a logger or envelope. `Execute` currently calls `handleBootstrapError(rootCmd, err)`; `cmd/bootstrap_errors.go` can use the default logger and calls `ferrors.HandleAndExit` directly. Parse errors occur before `PersistentPreRunE`; argument errors can do so as well. The renderer needs leaf identity, output selection, no-err-codes behavior, and effective configured destination even when parsing stopped early. Local preparation should reuse `initViper`, normalization, logger/activity setup, and existing fallback helpers. It must avoid `installRemoteCommandServices`, configuration credential validation, authentication, and version requests when reporting migration input errors. Invalid/unreadable local configuration uses the established bootstrap fallback, preserving both the migration cause and useful configuration failure without duplicate emission.

The token walk must respect known value arity, inherited flags, short forms/clusters supported by pflag, attached values, opaque values beginning with `--`, and the `--` terminator. Unknown tokens remain errors. A flag-looking payload or post-terminator positional argument must not become an output/configuration override. Temporary recovery state is diagnostic-only; do not mutate original command input, consume stdin, read mutation variable files, or rerun Cobra. Use an explicit context carrier/return path so normal command errors do not pass through a second renderer.

**Alternatives considered**: Direct stderr printing violates logger configuration; global logging violates context ownership. Merely returning a typed Cobra error misses JSON and destination routing. Ignoring unknown flags during normal parsing risks executing a changed mutation. Broad changes to backend error schemas are unnecessary.

## 7. Preserve operational and output contracts

**Decision**: Reuse existing renderers, logger helpers, prompt writers, service-owned workflows, and output precedence. Mutation summaries stay INFO logger output; machine results stay stdout. Migrate existing isolated command, terminal, payload, partial-error, empty-scope, and logging assertions to canonical syntax, retaining selected legacy cases as negative coverage.

**Rationale**: Existing test families already cover these semantics, including assignment logging and atomic completion. `cmd/update_usertask_complete.go` preserves exact numeric values separately from generic variable updates; renaming must not change decoders.

**Alternatives considered**: Renderer-only tests do not prove configured logger routing or single-envelope exits. Live Camunda testing is outside the implementation boundary.

## 8. Documentation and validation scope

**Decision**: Update source help/examples and README in each behavior work unit, then regenerate with `make docs-content`. Audit `demos/vhs/fast-start.tape`, maintained scripts, integration invocation fixtures, and command-matrix documentation without running live suites. Preserve historical `specs/` evidence. Use targeted isolated tests, then cmd/config/docsgen checks for the shared parsing boundary.

**Rationale**: The Makefile docs target runs local docsgen only. Full `make test` is race-enabled and may be justified if implementation alters shared bootstrap behavior outside affected commands; it is not required for these planning documents. Integration build tags/targets and live helpers are excluded.

**Alternatives considered**: Hand-editing generated CLI references would drift. Executing recording or integration scripts would cross the no-live-validation boundary. Blanket runtime tests during planning add no useful coverage.
