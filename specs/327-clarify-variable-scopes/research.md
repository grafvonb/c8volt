# Research: Clarify Variable Destination Scopes

All design questions are resolved from repository evidence. No dependency, protocol, or technology change is needed.

## 1. Preserve already collected process identity

**Decision**: Add `ProcessInstanceKey string` to `internal/domain.UserTaskVariablePlan` and `c8volt/task.UserTaskVariablePlan`; exclude the public field from JSON with `json:"-"`. Populate it from the selected task in `internal/services/usertask/update.go` and preserve it through both conversions in `c8volt/task/convert.go`.

**Rationale**: `internal/domain/usertask.go` already records this identity; all three task adapters (`v88`, `v89`, `v810`) populate it. `userTaskVariablePlanBuilder.addTask` already receives the fetched task. The plan currently retains its element key but drops its process key. Existing plan copies copy the containing struct, so the new string needs no new copying mechanism. One field per task supplies the view with a backend fact without duplicating it on every variable.

**Alternatives considered**: New lookups violate the zero-extra-request requirement. New adapter APIs are unnecessary. Serialized scope-kind fields change the machine contract. Rebuilding scope ownership in the command would move planning responsibility across the architecture boundary.

## 2. Keep scope classification in human views

**Decision**: Annotate only planned additions and changes. PI entries use `(pi:<key>)`. UT entries use `(local, element:<key>)`, `(inherited, pi:<key>)`, or `(inherited, element:<key>)`. Use existing `Inherited` and `ScopeKey` facts; compare inherited scope keys with the known owning process key. Reuse a small pure UT destination formatter between detailed and bulk views rather than adding a new production package.

**Rationale**: `cmd/cmd_views_usertask_vars.go` already uses `pi:` and `element:` for variable scope identities. `cmd/cmd_views_processinstance_update.go` already has the selected process key. `cmd/cmd_views_usertask_update.go` owns both per-variable annotations and the compact inherited-target summary. Formatting needs no domain enum or new service behavior.

**Alternatives considered**: Labeling all inherited variables as roots is incorrect for intermediate scopes. Inferring BPMN element types requires information the plan does not need. Moving formatting into services or facades violates established ownership. A general shared package is excessive for this narrow view behavior.

## 3. Preserve incomplete-plan compatibility without guessing

**Decision**: If a legacy or incomplete UT plan has no process-instance identity, retain the existing `(inherited scope <key>)` annotation and bare inherited-summary key for unknown inherited destinations. Known task-local destinations still use `element:<key>`. Do not add validation, requests, or execution preconditions to recover presentation metadata.

**Rationale**: `validateSelectedUserTask` currently requires task and element keys, not the process key. Public plans can also be manually constructed or round-tripped through JSON, which intentionally excludes the new field. An unknown inherited destination cannot truthfully be called an element or a root. Normal plans from complete task responses carry the required identity and satisfy the fully typed preview contract.

**Alternatives considered**: Guessing `element:` for unknown inheritance can mislabel a root. Failing previously accepted plans changes execution behavior. Adding the field to JSON violates compatibility. A new warning or unknown-scope vocabulary adds unnecessary output changes.

## 4. Preserve compact output and machine results

**Decision**: Keep existing summary text, display eligibility, counts, separators, variable values, result formatting, unchanged-requested entries, and untouched entries. Continue deduplicating bulk inherited destinations by raw scope key in existing traversal order; label the retained entries using their task metadata. Do not expose per-variable bulk details unless verbose is selected.

**Rationale**: `inheritedUserTaskVariableScopes` already deduplicates in stable order. The existing scope helper is also used by unchanged-requested entries, so blindly changing all its callers would expand the feature. `renderUpdateUserTaskVariablePreview` and the PI equivalent already separate machine results from human rendering.

**Alternatives considered**: Globally replacing scope strings could change final results or unchanged-variable output. Replacing compact summaries with detailed rows breaks bulk behavior. Adding a view field to a public JSON payload breaks the explicit compatibility requirement.

## 5. Prove behavior through existing fixtures

**Decision**: Extend nearby service/facade, exact view, subprocess output, and real-terminal tests. Count task reads, variable reads, and all relevant backend requests rather than only mutations. Reuse `testx.NewCmdTerminalRunner`, existing fake-server fixtures, and single-envelope decoders.

**Rationale**: `TestPlanUserTaskVariableUpdatesBuildsCompleteScopeSafePlan` already covers root/local/intermediate scopes and sparse pages. `TestVariableUpdateFacadeDelegatesAndCopiesNestedValues` covers facade plan round trips. `TestUpdateUserTaskOutputModes` captures streams and backend effects; `TestUpdateUserTaskTerminal` covers actual terminal stdin. Auto-confirm skips the normal confirmation preview, so dry-run plus auto-confirm alone cannot prove preview parity.

**Alternatives considered**: View-only tests miss lost metadata and accidental lookups. Pipe-only input cannot prove terminal prompt behavior. Full runtime tests during this planning-only change would conflict with proportionate validation.
