# Feature Specification: Standardize Flag Intent

**Feature Branch**: `369-standardize-flag-intent`

**Created**: 2026-11-03

**Status**: Ready for planning

**Input**: [GitHub issue #369: standardize write flags as set-* and enrichment flags as with-*](https://github.com/grafvonb/c8volt/issues/369)

## Clarifications

### Session 2026-11-03

- Q: Which compatibility policy applies? → A: Clean v5 breaking change. Remove renamed legacy spellings without a deprecated-alias transition.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Distinguish Selection from Submitted Values (Priority: P1)

As an operator, I want flag names to distinguish selecting resources from supplying values to write, so I can review a command's intent before changing operational state.

**Why this priority**: Overloaded property names make selection and mutation easy to confuse across related commands.

**Independent Test**: Exercise each affected update, run, and repair operation with canonical spelling against isolated fixtures, comparing selection and submitted values with existing behavior.

**Acceptance Scenarios**:

1. **Given** processes selected by BID, **When** I use `update pi --bid ORDER-123 --set-vars '{"approved":true}'`, **Then** BID selects existing instances and variables are written without assigning a new BID.
2. **Given** a job eligible for retry or timeout update, **When** I use `--set-retries 3` or `--set-timeout 5m` in its supported mode, **Then** the respective property receives the requested value with existing confirmation and verification behavior.
3. **Given** selector-based completion, **When** I use `update job --type cleanup --retries 0 --complete --set-bid ORDER-123 --set-vars '{"approved":true}'`, **Then** zero retries is an exact filter, and BID and variables accompany each completion under existing eligibility rules.
4. **Given** process creation, **When** I use `run pi --bpmn-process-id order --set-bid ORDER-123 --set-vars '{"approved":false}'`, **Then** creation initializes those exact values under existing defaults and restrictions.
5. **Given** user-task assignment, **When** I use `update ut --key <task-key> --set-assignee bob`, **Then** the assignee changes to bob; `get ut --assignee alice` continues to select tasks assigned to alice.
6. **Given** either repair command, **When** I supply renamed retry, job-timeout, or variable inputs, **Then** repair preserves its existing behavior, including zero retries meaning skip retry restoration.
7. **Given** job mutation and HTTP timeouts supplied together, **When** I use `--set-timeout 5m` and inherited `--timeout 30s`, **Then** the job receives five minutes and HTTP execution uses thirty seconds, with existing configuration precedence and valid inherited-flag positioning preserved.

---

### User Story 2 - Separate Worker Actions from Error Values (Priority: P1)

As a worker operator, I want an action selector separate from the values it submits, so technical failures and business errors have predictable spelling without expanding supported operations.

**Why this priority**: A value-bearing flag becoming an action selector can otherwise silently change existing automation.

**Independent Test**: Exercise canonical failure/error commands, missing values, incompatible actions, and legacy invocations through isolated command execution.

**Acceptance Scenarios**:

1. **Given** an eligible job, **When** I use `--throw-bpmn-error --set-error-code PAYMENT_DECLINED --set-error-message 'card declined'`, **Then** the action submits that code and message with existing error semantics.
2. **Given** technical failure, **When** I use `--fail --set-retries 0 --set-retry-backoff 5m --set-error-message 'worker unavailable'`, **Then** zero remains a submitted retry value and other values retain their existing meaning.
3. **Given** a missing or invalid required error code, a code without its action, or conflicting actions, **When** the command executes, **Then** it fails as invalid input before confirmation or mutation.
4. **Given** a previously unsupported mode combination, **When** I spell its inputs canonically, **Then** it remains unsupported; `--set-retries` neither selects completion targets nor permits retry mutation during completion.

---

### User Story 3 - Identify Returned Detail and Suppression (Priority: P2)

As a CLI user, I want `with-*` to consistently describe additional returned information, so help distinguishes enrichment from mutation and suppression.

**Why this priority**: Predictable naming reduces lookup effort and misleading terminology.

**Independent Test**: Exercise each enrichment/suppression rename and compare returned information with established behavior.

**Acceptance Scenarios**:

1. **Given** process-definition retrieval, **When** I request `get pd --with-stats`, **Then** I receive the additional statistics previously requested by `--stat`.
2. **Given** embedded-artifact listing, **When** I request `embed list --with-paths`, **Then** I receive the path details previously requested by `--details`.
3. **Given** incident retrieval, **When** I request `get incident --no-error-message`, **Then** suppression matches the previous `--with-no-error-message` behavior.
4. **Given** unaffected enrichment or presentation controls, **When** I use their established names, **Then** their spelling, defaults, and behavior remain unchanged.

---

### User Story 4 - Migrate Automation with Clear Guidance (Priority: P1)

As an automation maintainer, I want an explicit compatibility policy and consistent guidance so removed mutation syntax fails before changing resources and retained filter/execution flags have one documented meaning.

**Why this priority**: Silent reinterpretation is unacceptable for state-changing automation.

**Independent Test**: Execute canonical, legacy, conflicting, and invalid invocations in supported output modes using isolated command/terminal fixtures; compare guidance across published CLI surfaces.

**Acceptance Scenarios**:

1. **Given** a legacy invocation, **When** it executes, **Then** removed spellings and old mutation forms fail as invalid input with canonical replacement guidance before confirmation or mutation; retained bare retry filters and HTTP timeout controls follow their explicitly documented roles.
2. **Given** old and new names for the same property, even with equal values, **When** the command executes, **Then** it fails clearly before confirmation or mutation rather than choosing a value.
3. **Given** legacy `--throw-bpmn-error CODE` or `--throw-bpmn-error=CODE`, **When** it executes, **Then** it fails as invalid input before confirmation or mutation with guidance to use `--throw-bpmn-error --set-error-code CODE`; the code is never discarded or treated as Boolean truth.
4. **Given** human, quiet, verbose, or supported machine output, **When** migration failures occur, **Then** diagnostics honor configured logging, prompts remain plain configured stderr, and result stdout stays uncontaminated.
5. **Given** partial failure or completed empty selection, **When** renamed syntax executes, **Then** existing evidence, exit classification, no-op behavior, and output precedence remain intact; shared JSON contains exactly one envelope.
6. **Given** help, capabilities, completion, README, generated references, and maintained invocation examples, **When** I follow current guidance, **Then** they agree on canonical syntax and explain the v5 breaking change, absence of legacy aliases, retained retry-filter meaning, and HTTP timeout meaning.

### Edge Cases

- Omission remains distinct from explicit zero, false, empty, and whitespace-only values under existing validation. Accepted business IDs, identities, messages, and variables retain exact values.
- `get job --retries 0` and keyless `update job --complete --retries 0` remain exact filters. Keyed completion still rejects retry predicates; other mutation modes gain no selectors.
- Legacy retry mutation cannot be a global alias because the same spelling remains a filter in other contexts.
- Inherited HTTP `--timeout` remains meaningful on `update job`. It is exclusively an execution control, never a legacy job-mutation alias. A key plus HTTP timeout alone supplies no mutation and must fail before confirmation or mutation with `--set-timeout` guidance. With another valid canonical mutation it controls HTTP execution only; documentation must explicitly explain this breaking meaning change.
- An error code with `--throw-bpmn-error=false`, an empty code, or an incompatible action fails before mutation.
- Variable/file exclusions include mixed old/new spellings. Renaming adds no variable-file support to commands that lack it.
- Conflicting aliases fail even when values match or one is explicit zero.
- Repair `--set-retries 0` still skips retry restoration; its name does not promise literal zero-setting.
- Dry-run remains a preview. No-wait does not manufacture confirmation. Completed empty scopes do not prompt, mutate, or claim accepted/pending work. Incomplete discovery is not empty success; sparse pages continue discovery.
- Migration diagnostics do not expose raw HTTP details at normal/verbose levels or cause duplicate result envelopes.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Canonical names MUST distinguish submitted write/initialization values (`set-*`), additional returned detail (`with-*`), bare resource/property filters, and action/execution controls. Scope MUST be bounded by this table and the BPMN-error split below.

| Commands / modes | Previous spelling | Canonical spelling |
| --- | --- | --- |
| `update element`, `update pi`, `update ut`, supported `update job` modes | `--vars` | `--set-vars` |
| `update element`, `update pi`, `update ut` | `--vars-file` | `--set-vars-file` |
| `update job` retry update and technical failure | mutation `--retries` | `--set-retries` |
| `update job` timeout update | mutation `--timeout` | `--set-timeout` |
| `update job --fail` | `--retry-backoff` | `--set-retry-backoff` |
| `update job` technical failure and BPMN error | `--message` | `--set-error-message` |
| `update ut` assignment | `--assign <identity>` | `--set-assignee <identity>` |
| `run pi` | `--bid` | `--set-bid` |
| `run pi` | `--vars` | `--set-vars` |
| `ops repair incident`, `ops repair pi` | `--retries` | `--set-retries` |
| Both repair commands | `--job-timeout` | `--set-job-timeout` |
| Both repair commands | `--vars` | `--set-vars` |
| Both repair commands | `--vars-file` | `--set-vars-file` |
| `get pd` | `--stat` | `--with-stats` |
| `embed list` | `--details` | `--with-paths` |
| `get incident` | `--with-no-error-message` | `--no-error-message` |

- **FR-002**: `--retries` MUST remain an exact filter on `get job` and selector-based completion, including explicit zero. `--set-retries` MUST supply only a mutation value in existing supported modes. Neither spelling MUST broaden mode combinations or keyed/collection/selector eligibility.
- **FR-003**: Canonical `--throw-bpmn-error` MUST select an action without carrying its code. `--set-error-code` MUST supply the required nonempty code; `--set-error-message` MUST supply the optional message under existing value rules. Missing code, code without its action, and incompatible actions MUST fail before mutation.
- **FR-004**: Existing `--set-bid` MUST remain unchanged. BID selection and assignment MUST remain distinct, preserving issue #362 eligibility, exact-value, atomic-completion, and version rules.
- **FR-005**: Job `--set-timeout` and inherited HTTP `--timeout` MUST be independently usable and resolve to their intended settings, including configured defaults and existing explicit-flag precedence. Job mutation MUST no longer shadow HTTP execution timeout.
- **FR-006**: Renaming MUST preserve defaults, value validation, omission/presence semantics, exclusions, submitted values, authorization, tenant scope, discovery, confirmation, dry-run, no-wait, and exit classifications. Invalid combinations MUST fail before mutation. Repair zero retries MUST still skip restoration and be documented as an exception.
- **FR-007**: v5.0.0 MUST introduce this as a breaking change with no deprecated aliases or transition period for the renamed spellings in FR-001. Removed names MUST fail as invalid input with canonical replacement guidance before confirmation or mutation. Canonical help, capabilities, and completion MUST advertise only current syntax. Bare `--retries` MUST be accepted only as the existing exact search predicate; legacy retry-update/failure usage MUST fail with `--set-retries` guidance, including when both spellings are supplied in a mutation mode. Legacy value-bearing BPMN-error syntax in either separated or equals form MUST fail with action-plus-code guidance. Inherited `--timeout` MUST mean HTTP execution only, never job mutation; without an independently selected valid mutation, a key plus timeout MUST fail with `--set-timeout` guidance. With a valid canonical mutation, both `--timeout` and `--set-timeout` are independent inputs, not conflicting aliases. Documentation MUST call out that changed role explicitly.
- **FR-008**: Legacy invocations MUST NOT be silently reinterpreted. Simultaneous old/new inputs for one property MUST fail before confirmation or mutation, even when equal. Retained retry-filter and HTTP-timeout spellings MUST follow the contextual rules in FR-007; removed mutation meanings MUST NOT survive as aliases. Legacy BPMN-error values MUST be rejected rather than converted to the Boolean action.
- **FR-009**: Resource/property filters, date and variable predicates, expectation inputs, actions (`--complete`, `--fail`, `--unassign`, deploy `--run`), scope, execution/backoff, output, artifact, and tenant-context controls MUST retain their roles. Existing `--with-vars`, `--with-incidents`, `--with-elements`, `--with-listeners`, `--with-brokers`, and `--with-full-timeline` MUST remain unchanged. Hidden `--log-with-source` MAY remain unchanged.
- **FR-010**: Human warnings, failures, mutation/dry-run summaries, and verbose outcomes MUST use configured command-context logging with existing severity, level, format, destination, quiet, and activity behavior. Prompts MUST remain plain configured stderr; selected-format results MUST remain stdout. Migration errors MUST follow the same logging contract; no deprecation-warning transition is introduced.
- **FR-011**: Failures MUST retain original exit classification and partial evidence. Shared JSON MUST emit exactly one envelope with existing payload/empty/null conventions and no duplicate error rendering. Quiet MUST preserve requested machine results. Empty keys output MUST contain zero bytes.
- **FR-012**: Normal/verbose human errors MUST remain compact, using bounded aggregate reasons for repeated failures without raw HTTP details on either stream. Existing redacted DEBUG diagnostics MUST remain available via explicit debug and configured debug level independently of verbose.
- **FR-013**: Affected flag recognition, presence detection, validation/exclusivity, configuration resolution, help, metadata/capabilities, completion, README, generated references, and maintained repository invocation scripts/fixtures MUST agree on canonical names. Historical feature evidence MUST be preserved. Guidance MUST state the migration policy and repair-zero exception.
- **FR-014**: This feature MUST NOT change backend mutation semantics, add modes/unsupported combinations, rename public-library fields solely for CLI spelling, change result schemas solely for naming, redesign command trees/output modes, or change repair zero behavior.

### Key Entities *(include if feature involves data)*

- **Flag intent**: Selection predicate, submitted value, enrichment, action, scope, execution, presentation, or artifact control.
- **Canonical flag**: Documented spelling for a command and its supported modes, retaining value and presence semantics.
- **Legacy invocation**: Removed spelling or mutation syntax rejected under the v5 breaking policy, distinct from retained retry filters and HTTP execution controls.
- **Migration diagnostic**: Invalid-input explanation with canonical replacement guidance, delivered through configured logging without contaminating results.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every rename-table row is usable canonically in every previously supported mode; each acceptance fixture selects the same resources and submits the same values as before migration.
- **SC-002**: All retry-selection/mutation acceptance cases distinguish intent, including explicit zero, with zero newly permitted incompatible combinations.
- **SC-003**: Operators can supply job and HTTP timeouts independently in one invocation; every configured-default/explicit-override acceptance case uses the intended value for each.
- **SC-004**: Every removed legacy/mixed mutation-syntax acceptance case fails with migration guidance before mutation, while retained filter/execution controls keep their documented roles; zero cases silently discard values or perform a different mutation.
- **SC-005**: Every supported output-mode acceptance case preserves result shape, evidence, exit classification, and stream roles; shared JSON contains one envelope and empty keys output zero bytes.
- **SC-006**: Every affected help, capability, completion, maintained example/script, and current command-reference entry agrees on canonical syntax and the v5 breaking policy, allowing operators to distinguish selection, writing, enrichment, and suppression without reading source.

## Assumptions

- Issue #362 is a merged prerequisite in this checkout (merge commit `390e5950`, PR #370). Its behavior is the baseline; this issue does not reopen assignment policy.
- Issue #369 targets v5.0.0. The user explicitly selected a clean breaking change on 2026-11-03: no deprecated legacy aliases or transition period. Retained bare retries and inherited HTTP timeout have only the roles specified above.
- Existing operation-specific validation and supported output modes remain authoritative. Renaming adds no `--set-vars-file` support to `run pi` or `update job` and does not normalize unrelated controls.
- Acceptance verification uses unit tests, generated-client mocks, test-owned HTTP servers, and isolated command/subprocess/terminal fixtures. No live Camunda service, cluster discovery, provisioning, or smoke test is an implementation gate.
- Output verification captures stdout/stderr separately and checks configured/inherited logger destinations, plain/structured formats, quiet/WARN suppression, repeated backend failures with distinctive HTTP details, both independent debug paths/redaction, and exact JSON decoding through EOF. Real terminal stdin covers confirmation routing and eligible prompt-free empty scopes; retain nonempty, explicit-key, abort, and error regressions.
- Planning must include command metadata/documentation alignment and regeneration using `make docs-content`; generated references are not hand-edited. This specification-only change requires lightweight document validation, not runtime tests.
