# Implementation Plan: Process Definition Watch Change Indicators

**Branch**: `312-pd-watch-indicators` | **Date**: 2026-10-16 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/312-pd-watch-indicators/spec.md`, issue #312.

## Summary

Add a fixed watch-only marker column, per-refresh change summary, and selective bold values to `get pd --watch`. Compare owned, unstyled displayed values by definition key against the last successfully displayed complete refresh. Keep session history in the existing watch mode and rendering in the PD view file; retain service-owned collection and canonical ordering. Failed collection or output does not advance history. Ordinary output and public APIs remain unchanged.

## Technical Context

**Language/Version**: Go 1.26; repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra, `golang.org/x/term`, `toolx/watch`, and `c8volt/process` snapshot facade. No new dependencies.

**Storage**: Invocation-local in-memory map of copied display values and an initialized flag; no persistence.

**Testing**: Go tests with existing testify assertions and command watch harnesses; exact output and controlled refresh sequences, writer-error checks, and terminal styling capability coverage.

**Target Platform**: Existing supported CLI platforms and configured stdout writers; bold is a terminal capability enhancement, not a new watch eligibility condition.

**Project Type**: CLI and public library; this change is confined to CLI presentation and documentation.

**Performance Goals**: O(previous rows + current rows) comparison and O(current rows) rendering for a fixed number of displayed fields; bounded memory for previous/current projections. No additional remote requests or sleeps.

**Constraints**: Preserve canonical order, selectors, retry/termination policy, output stream routing, normal output bytes, and existing watch mode exclusions. No new flags or duration controls. Compute alignment before ANSI styling.

**Scale/Scope**: All definitions in the existing complete watch snapshot, across supported backend versions; one command mode and its documentation. No backend/domain/facade schema changes.

## Constitution Check

Pre-research and post-design gates both pass:

| Principle | Design evidence | Result |
| --- | --- | --- |
| Operational proof | Only complete successful results generate comparisons; failed collection/output cannot advance the baseline. Counts describe watched-result changes, not backend mutation claims. | Pass |
| CLI-first, script-safe | Explicit watch-only contract; normal human/JSON/keys/XML output preserved; diagnostics remain on stderr. Compatibility note: watch now adds a summary and marker prefix instead of matching ordinary list output exactly. | Pass |
| Proportional validation | Planning uses artifact checks only. Implementation uses focused command tests and docs generation; full race suite only if scope expands into shared runtime behavior or failures justify it. | Pass |
| Documentation matches behavior | README, command Long help, and output metadata updated in the same implementation delivery; `make docs-content` regenerates references. | Pass |
| Small repository-native changes | Existing watch file, view file, width helpers, command harnesses, and terminal library reused; no generic diff/styling framework. | Pass |

Architecture gate: presentation comparison belongs in `cmd`; no backend traversal, retries, waits, or ordering algorithms are added there. The existing runner and collector remain unchanged. Mode declarations stay in `get_processdefinition_watch.go`; final formatting stays in `cmd_views_processdefinition.go`. No conflicts with `AGENTS.md` or `specs/ralph-implementation-rules.md` were identified.

## Project Structure

### Documentation (this feature)

```text
specs/312-pd-watch-indicators/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/watch-output.md
└── tasks.md                     # Next phase; not created by this plan
```

### Source Code (repository root)

```text
cmd/
├── get_processdefinition.go           # Help/output metadata only
├── get_processdefinition_watch.go     # Invocation-local baseline and callback wiring
├── get_processdefinition_watch_test.go
├── cmd_views_processdefinition.go     # Display projection, summary, marker, selective bold
├── cmd_views_processdefinition_test.go
├── cmd_views_flat.go                  # Reuse plain width helpers; no planned behavior change
└── cmd_views_rendermode.go            # Existing repaint retained unchanged
README.md                             # Watch semantics
docs/cli/                             # Generated references
docs/index.md                         # Generated homepage
```

**Structure Decision**: Extend the existing dedicated watch mode and PD view ownership. Keep any style capability helper local to the PD watch/view concern because there is only one consumer. Preserve ordinary row composition through a common PD projection if extraction is needed, and prove byte compatibility with existing tests. Internal services, facades, generated clients, and shared watch runner are reference dependencies rather than planned edit targets.

## Phase 0: Research Results

See [research.md](research.md). Repository inspection resolved ownership, displayed-field identity, optional statistics, alignment, write-error propagation, baseline commitment, styling capability, and validation scope. There is no existing general bold policy; use actual configured stdout terminal detection and conservative environment fallback as documented in the output contract.

## Phase 1: Design

1. Project each complete snapshot into owned plain display values with individually identifiable statistics values. Preserve snapshot order and index prior values by key. The map is initialized separately from whether it contains rows.
2. Build the candidate markers, changed-value masks, three counts, and next baseline without changing current history. New and updated categories are mutually exclusive; removed keys do not create rows.
3. Render the exact summary, fixed two-character marker prefix, aligned existing fields, and existing footer. Compute widths from plain fields; bold only visible changed values of existing rows. Use error-returning writes within the watch view instead of shared helpers that discard write errors.
4. Commit baseline only after successful repaint and view output. Preserve all retry, slow-warning, verbose, termination, and collector behavior. Do not persist state across invocations.
5. Update watch-specific parity expectations to the new contract while retaining ordinary-output checks. Ship help, metadata, README, generated docs, and behavior validation together.

Design artifacts: [data model](data-model.md), [CLI contract](contracts/watch-output.md), and [validation guide](quickstart.md).

## Implementation Work-Unit Guidance

Use two work units. The core story includes comparison, runner wiring, plain rendering, and all baseline/failure/empty-result acceptance coverage. Failure recovery is an invariant of that same mechanism, so do not postpone its tests to a separate hardening task. The second story adds selective styling with its terminal-capability and alignment tests. Include relevant documentation, formatting, and targeted validation in each unit.

The original recovery story is consolidated into story 1 in the specification without dropping its acceptance scenarios; styling is now story 2. This keeps each Ralph iteration within one story. Keep styling separate because capability detection and ANSI-safe layout have a distinct, independently verifiable result. Do not create standalone context, branch-check, formatting, or routine validation tasks. Future Conventional Commit subjects must end with ` #312`.

## Validation Strategy

During implementation, run `gofmt` on touched Go files and targeted `go test ./cmd -run 'Test(GetProcessDefinition|ProcessDefinitionWatch|ListProcessDefinitions|ProcessDefinitionView)' -count=1`. Include baseline/unchanged/update/add/remove/reorder/recovery, initialized-empty state, unsupported incident changes, selective stats highlighting, exact stdout/stderr, write failure, and unchanged normal modes. Use deterministic harnesses rather than real-time waits or cluster mutations. Exercise actual configured terminal output for capability detection and nonterminal writers for plain fallback; do not change prompts or their eligibility.

Run `make docs-content` after source metadata and README updates, review generated diff, and use `git diff --check`. Run relevant shared helper tests and consumers only if implementation modifies shared helpers; use `make test` for broader runtime impact or unresolved regression risk. Do not run runtime tests for this planning-only change.

## Complexity Tracking

No constitution exceptions or new architectural layers. The only added state is the prior displayed snapshot; per-field display metadata is needed to highlight changed values without false positives from padding or unchanged statistics.
