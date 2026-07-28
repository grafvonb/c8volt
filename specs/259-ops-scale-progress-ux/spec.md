# Feature Specification: Ops-Scale Preflight And Progress UX

**Feature Branch**: `259-ops-scale-progress-ux`

**Created**: 2026-07-27

**Status**: Draft

**GitHub Issue**: [#259](https://github.com/grafvonb/c8volt/issues/259) - feat(ops): add ops-scale preflight and progress UX

**Input**: GitHub issue #259 requests a consistent, fast, operator-facing preflight and progress experience for c8volt commands that can process thousands of Camunda resources.

## Issue Traceability

- **GitHub Issue**: #259
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/259
- **Issue Title**: feat(ops): add ops-scale preflight and progress UX

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See Scope Before Expensive Work Starts (Priority: P1)

As a Camunda operator, I want c8volt to show the apparent size and cost of a broad batch command before expensive discovery, enrichment, planning, or mutation begins so I can understand the blast radius instead of wondering whether the tool is stuck.

**Why this priority**: c8volt is most valuable when Camunda contains thousands of resources; broad selectors without early scope feedback undermine operator trust before any command-specific result appears.

**Independent Test**: Run a high-volume command with a broad selector such as one BPMN process ID, verify that c8volt reports matching scope information and consequence text before the expensive phase begins, and verify that the first discovered page is reused as work continues.

**Acceptance Scenarios**:

1. **Given** a broad process-instance selector matches more resources than one page, **When** the operator starts a long-running human command, **Then** c8volt displays a preflight message with the best available count kind, page-size context, and consequence summary before deeper processing begins.
2. **Given** exact count information is available from the initial discovery response, **When** preflight completes, **Then** c8volt reports the exact total and continues from the already discovered resource page without restarting discovery.
3. **Given** exact count information is unavailable or would require expensive extra checks, **When** preflight completes, **Then** c8volt reports a lower bound, estimate, or unknown total and explains why exact scope is unavailable.

---

### User Story 2 - Track Long Work By Phase And Exact Counters (Priority: P2)

As a Camunda operator, I want long-running commands to show meaningful phase progress and exact done/total counters once the command has frozen its working resource set so I can tell that c8volt is still moving and how far it has to go.

**Why this priority**: A spinner alone does not answer the operator's main question at scale: which kind of work is happening and how much remains.

**Independent Test**: Run `ops analyse slow-process-instances` against a multi-page selection and verify that discovery progress, frozen process-instance total, element-enrichment progress, elapsed time, and completion counts are visible without `--debug`.

**Acceptance Scenarios**:

1. **Given** discovery uses paging and the page count is known or safely estimable, **When** c8volt is discovering resources, **Then** it displays current page progress and seen-resource progress using the best available total semantics.
2. **Given** the command freezes 800 selected process instances, **When** c8volt enriches or analyzes them, **Then** progress uses exact counters such as `48/800 process instance(s)` until that phase completes.
3. **Given** a workflow moves through multiple long phases, **When** the phase changes, **Then** c8volt names the new phase using operator language such as preflight, discovering process instances, freezing candidate scope, loading runtime elements, planning delete scope, repairing incidents, or deleting roots.

---

### User Story 3 - Preserve Script-Safe Output Contracts (Priority: P3)

As an automation author, I want progress and preflight reporting to never corrupt JSON, keys-only, quiet, or deterministic automation output so I can adopt the improved UX without breaking scripts.

**Why this priority**: c8volt is both an interactive ops tool and a scripting surface; progress improvements must not make machine output unsafe.

**Independent Test**: Run affected commands in human, verbose, quiet, JSON, keys-only, and automation modes over the same large fake population and verify that progress appears only in allowed human/activity channels while machine stdout remains contract-compliant.

**Acceptance Scenarios**:

1. **Given** JSON output is requested, **When** a long-running command executes, **Then** stdout remains one valid JSON document and progress does not appear inside or around it.
2. **Given** keys-only output is requested, **When** a long-running command executes, **Then** stdout contains only one key per line and no progress, preflight, prompt, cursor, or diagnostic text.
3. **Given** quiet or deterministic automation mode is active, **When** preflight or progress events occur, **Then** they are suppressed or represented only through the mode's established structured reporting rules.

---

### User Story 4 - Estimate Time Remaining Responsibly (Priority: P4)

As a Camunda operator, I want c8volt to show approximate throughput and ETA only after enough work has completed to make the estimate useful so I can decide whether to continue, narrow the selector, or run the command elsewhere.

**Why this priority**: ETA is helpful for thousands of resources, but premature or unstable estimates damage trust.

**Independent Test**: Run a long-running fake-volume command with controlled per-resource timing and verify that no ETA appears before the minimum sample threshold, then verify that elapsed time, rate, and approximate remaining time appear and update as more samples complete.

**Acceptance Scenarios**:

1. **Given** too few samples exist for a phase, **When** c8volt reports progress, **Then** it omits ETA and may show only current counters and elapsed time.
2. **Given** enough samples exist and a known or frozen total is available, **When** c8volt reports progress, **Then** it displays approximate rate and remaining time with wording that makes the estimate non-authoritative.
3. **Given** processing speed changes materially, **When** c8volt updates ETA, **Then** the displayed estimate adjusts without claiming precision.

### Edge Cases

- A selector matches zero resources; preflight and completion must still make the empty scope visible.
- A selector matches thousands of resources but the backend can provide only a lower bound; c8volt must label the total as a lower bound.
- A command cannot know the final affected resource count until after dependency, parent, incident, or eligibility checks; c8volt must distinguish candidate discovery from frozen-scope progress.
- Preflight itself can be expensive for some workflows; c8volt must warn before starting an expensive preflight when cheap scope metadata is unavailable.
- A first discovery page may already contain all matching resources; c8volt must avoid overstating cost or page count.
- The resource population may change while a command is running; c8volt must report the scope used for the command rather than implying a live global count.
- Interactive confirmation may be unavailable; c8volt must still report scope and consequence information according to non-interactive mode rules.
- Output may be redirected or run in a terminal that does not support transient activity; c8volt must choose durable, compact reporting that does not corrupt command results.
- Commands with explicit small inputs, such as a handful of known keys, must not add noisy preflight that reduces usability.
- Failed discovery, authorization errors, or unsupported Camunda versions must still end with clear status and must not leave the operator guessing whether processing continued.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Commands that can process high-volume Camunda resources MUST provide preflight scope information before expensive batch processing begins in default interactive human mode.
- **FR-002**: Preflight scope information MUST identify the core resource type being selected, the best available total, and whether that total is exact, a lower bound, estimated, or unknown.
- **FR-003**: Preflight scope information MUST include page-size and page-count context when paging is involved and page count is known or safely estimable.
- **FR-004**: Preflight MUST be fast by default and MUST reuse already discovered resources when initial discovery also supplies usable scope metadata.
- **FR-005**: If precise preflight requires expensive extra work, c8volt MUST avoid that work by default unless the command's safety model requires it, and MUST tell the operator what level of scope certainty is available instead.
- **FR-006**: If even preflight discovery is expected to be expensive, c8volt MUST warn the operator before starting that preflight in interactive human mode.
- **FR-007**: For broad selectors, preflight MUST include consequence text that explains the major follow-on work, such as discovering all matches, loading timelines, checking dependencies, planning mutations, or applying actions.
- **FR-008**: Interactive human mode MUST require confirmation before continuing from broad high-volume preflight into expensive or potentially destructive work unless an existing explicit confirmation or automation rule already covers that decision.
- **FR-009**: Non-interactive, automation, or explicit confirmation modes MAY skip prompts but MUST still expose preflight scope and consequence information according to their established output contracts.
- **FR-010**: Long-running workflows MUST report operator-meaningful phases rather than only low-level request activity.
- **FR-011**: Once a command has a final or frozen set of core resources, progress MUST use exact done/total counters for processing those resources.
- **FR-012**: Discovery progress MUST show seen-resource progress and current page progress when paging is involved.
- **FR-013**: Progress MUST distinguish candidate discovery counts from final affected-resource or frozen-scope counts when those populations differ.
- **FR-014**: Progress MUST include elapsed time for long-running phases once the phase has been running long enough to be noticeable.
- **FR-015**: Progress MAY include throughput and approximate ETA only after enough completed work exists to avoid misleading estimates.
- **FR-016**: ETA wording MUST make clear that remaining time is approximate and MUST be omitted when totals are unknown or sample quality is insufficient.
- **FR-017**: Progress and preflight reporting MUST NOT corrupt machine-readable stdout.
- **FR-018**: JSON output MUST remain one valid JSON document for commands that emit JSON.
- **FR-019**: Keys-only output MUST remain one key per line and nothing else.
- **FR-020**: Quiet mode MUST suppress human progress and preflight chatter except for required prompts or errors already allowed by quiet-mode behavior.
- **FR-021**: Automation output MUST remain deterministic and non-interactive while still making large-scope information auditable through the mode's supported structured fields or reports.
- **FR-022**: Default human progress MUST avoid noisy endpoint names, request details, cursors, or per-resource debug information.
- **FR-023**: Verbose or debug modes MAY expose additional durable detail, but default human output MUST stay compact and scan-friendly.
- **FR-024**: `ops analyse slow-process-instances` MUST serve as the first proof case for the shared UX because broad BPMN process selection currently demonstrates repeated discovery and per-process-instance enrichment without visible progress.
- **FR-025**: For `ops analyse slow-process-instances`, broad process-definition selection MUST report discovery preflight, discovery/page progress, frozen process-instance scope, and runtime element loading progress without requiring `--debug`.
- **FR-026**: The feature MUST assess and cover high-volume basic inspection commands for process instances, incidents, jobs, and elements.
- **FR-027**: The feature MUST assess and cover high-volume process-instance actions, including cancel, delete, walk, and bulk run or smoke-test flows where count-driven work can be large.
- **FR-028**: The feature MUST assess and cover high-volume ops workflows, including slow-process analysis, retention policy execution, orphan process-instance purge, incident-bearing process-instance purge, all-process-definition purge, incident repair, and process-instance repair.
- **FR-029**: Commands with a known small explicit input set MUST keep output concise and MUST NOT force broad preflight prompts that do not add operator value.
- **FR-030**: User-facing command help and documentation MUST explain preflight scope, total certainty levels, progress behavior, quiet/automation behavior, and how batch size and limits affect operator-visible scope.
- **FR-031**: Automated tests MUST cover human, verbose, quiet, JSON, keys-only, automation, and large fake-volume scenarios for the shared UX and the first proof command.
- **FR-032**: Tests MUST verify that progress events do not change command result ordering, final counts, exit behavior, or mutation safety semantics.
- **FR-033**: Planning and implementation for this feature MUST follow `specs/ralph-implementation-rules.md` when Ralph is used, and Ralph runs MUST include that file as implementation context.

### Key Entities *(include if feature involves data)*

- **Preflight Scope**: The operator-facing summary of the resource population a command appears likely to process before expensive work begins.
- **Total Certainty**: The classification attached to a count: exact, lower bound, estimated, or unknown.
- **Core Resource**: The primary resource population being selected or processed by a command, such as process instances, incidents, jobs, elements, or process definitions.
- **Candidate Scope**: The resources discovered before final eligibility, dependency, or safety checks.
- **Frozen Scope**: The final resource set a command will analyze, preview, confirm, mutate, or report against.
- **Phase Progress**: The visible progress for one named stage of a long-running workflow.
- **Page Progress**: The visible traversal state for paged discovery, including current page and known or estimated page count when available.
- **Consequence Summary**: The compact explanation of what proceeding will do after preflight, especially for expensive enrichment or destructive workflows.
- **Progress Channel**: The allowed destination for human progress that keeps stdout safe for JSON, keys-only, quiet, and automation modes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For high-volume interactive human commands with broad selectors, operators see preflight scope information before expensive batch processing starts in 100% of covered command-family tests.
- **SC-002**: `ops analyse slow-process-instances -b <bpmnProcessId>` visibly reports discovery and runtime-element enrichment progress without `--debug` in large fake-volume tests.
- **SC-003**: Once a frozen scope is known, covered commands report exact done/total counters for the core processing phase in 100% of large fake-volume human-mode tests.
- **SC-004**: JSON-mode tests for covered commands produce exactly one valid JSON document and no progress text on stdout.
- **SC-005**: Keys-only tests for covered commands produce only one key per line and no preflight, progress, prompt, or diagnostic text on stdout.
- **SC-006**: Quiet and automation-mode tests preserve existing non-interactive and deterministic output guarantees while retaining auditable scope information where those modes support it.
- **SC-007**: ETA is omitted before the minimum sample threshold and appears only with approximate wording after sufficient progress samples in controlled timing tests.
- **SC-008**: Preflight for commands with cheap first-page scope metadata avoids duplicate full discovery in the first proof workflow.
- **SC-009**: Documentation and generated command help describe preflight, progress, total certainty, quiet/automation behavior, and batch-size/limit effects for every covered command family.
- **SC-010**: Operators can distinguish candidate counts, frozen-scope counts, and final affected-resource counts in human output for workflows where those counts differ.

## Assumptions

- Operators use c8volt primarily when Camunda's own UI is insufficient for broad or repeated operational work.
- Existing command output modes, exit codes, prompts, and mutation safety rules remain authoritative unless this specification explicitly adds progress or preflight behavior.
- The first implementation slice will use `ops analyse slow-process-instances` as the proof command, then expand the shared UX to destructive ops workflows and basic high-volume inspection commands.
- Progress is user-facing operational feedback, not debug tracing; endpoint calls, cursors, and low-level request details remain behind debug or verbose behavior.
- "High-volume" means commands that can naturally traverse more than one page, enrich many resources, or mutate/analyze a broad selected population, not commands that operate on a few explicit keys.
- The live Camunda population may change during a run, so c8volt reports the scope it observed or froze for the command rather than promising an immutable cluster-wide truth.

## Out of Scope

- Replacing existing command results with a dashboard or report-only experience.
- Adding progress to every trivial single-resource command.
- Displaying low-level HTTP request logs in default human output.
- Changing JSON result schemas solely to carry transient progress events.
- Implementing new Camunda filtering semantics unrelated to preflight, progress, or batch scope.
- Running Ralph implementation without `specs/ralph-implementation-rules.md` as explicit implementation context.
