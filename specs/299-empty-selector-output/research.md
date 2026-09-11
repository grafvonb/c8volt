# Research: Empty Selector Result Output

## 1. Render only authoritative zero-match outcomes

**Decision**: Replace the three direct `found: 0` writes guarded by `planned.RequestedCount == 0` with one focused process-instance empty-result view helper. Keep the guards after successful planning and preserve surrounding returns and side effects.

**Rationale**: `cmd/delete_processinstance_selector.go` has separate dry-run planning and mutation-planning branches; `cmd/cancel_processinstance_selector.go` has the third branch. Each already knows discovery completed without matches. Their top-level runners return when reports or previews are absent, so rendering here produces exactly one result without changing result structs or service contracts.

**Alternatives considered**: Inferring no-op from empty reports in the runners risks classifying aborted/nonempty workflows as success. Adding a new result-state field is unnecessary because the authoritative count is already available. Rendering each empty page would break sparse-page traversal. Repeating discovery would violate FR-008.

## 2. Reuse existing JSON payloads and success rendering

**Decision**: For normal execution use `process.DeleteReports{}` or `process.CancelReports{}` with `renderSucceededResult`. For dry-run use `newProcessInstanceDryRunSummary(operation, nil)` through the existing successful dry-run result path. Restrict invocation to JSON mode.

**Rationale**: `c8volt/process/model.go` marks report `items` as `omitempty`, so an empty report payload is `{}`. `cmd/cmd_views_processinstance_dryrun.go` already constructs a complete zero-count preview summary with `mutationSubmitted: false`; its nil collections encode as `null`. Both are existing schema semantics. `cmd/cmd_views_contract.go` attaches existing tenant context. Its general `renderCommandResult` chooses `accepted` with `--no-wait`, which is inappropriate when nothing was submitted.

**Alternatives considered**: A new no-op outcome/schema would change the shared contract. Forcing arrays to `[]` globally would change unrelated payloads. Synthetic report or preview entries would imply work that did not occur. Calling the full human dry-run summary renderer would replace the required compact `found: 0` message.

## 3. Preserve output precedence and quiet behavior

**Decision**: Use `pickMode()` first: JSON emits a result; keys-only returns without writing; human mode checks `flagQuiet` and otherwise uses `renderOutputLine` for exactly `found: 0` with a newline. Keep this final formatting in `cmd/cmd_views_processinstance.go`.

**Rationale**: `cmd/cmd_views_rendermode.go` resolves JSON before keys-only. Its raw output helper preserves the existing unprefixed human stdout. `renderHumanLine` depends on logger context and can add prefixes or fall back to unsuppressed output, so it does not provide this exact contract by itself. Quiet must not discard requested JSON.

**Alternatives considered**: A global renderer refactor or logger change exceeds issue scope. Moving the summary to stderr changes the required human contract. Returning early for quiet before inspecting the output mode suppresses required JSON.

## 4. Prove behavior at command and terminal boundaries

**Decision**: Extend existing selector HTTP tests with mode/execution matrices and request assertions; add focused view coverage and real-terminal empty-scope command coverage using `testx.NewCmdTerminalRunner`.

**Rationale**: Both selector test files contain `Test{Delete,Cancel}ProcessInstanceBpmnSelectorVisiblePreservesSearchNoOp`, which assert human output and exactly two discovery requests (definition validation, instance search). Date-filtered scaffold cases and `newProcessInstanceSearchCaptureServer` support one-request empty discovery. Existing dry-run tests provide `stubProcessAPI`, reset helpers, confirmation spies, and mutation guards. The terminal runner independently captures stdout/stderr with real terminal stdin, satisfying repository prompt validation requirements. Sparse-page continuation tests prevent premature no-op rendering.

**Alternatives considered**: Renderer-only tests miss command wiring and duplicate output. Pipe-only stdin cannot prove that an eligible interactive execution avoids prompting. Live destructive validation is unnecessary because existing local HTTP fixtures exercise the real command path deterministically.

## 5. Documentation and verification

**Decision**: Update README and delete/cancel command `Long`/`Example` metadata, regenerate through `make docs-content`, and run targeted command tests followed by `make test` during implementation.

**Rationale**: The constitution requires user-facing documentation and full race-enabled validation. `cmd/cmd_processinstance_test.go` already covers destructive command help. Generated CLI pages come from command metadata.

**Alternatives considered**: Hand-editing generated docs violates repository guidance. Omitting documentation leaves scripts dependent on an unexplained behavior change. Broad service or client changes add no value to this output-only fix.

## Resolution

All design unknowns are resolved from repository evidence. No external dependency upgrade, backend integration change, or user clarification is needed. Research was read-only; implementation tests have not been run.
