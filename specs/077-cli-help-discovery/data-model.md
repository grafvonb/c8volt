# Data Model: Improve CLI Help Discovery

## Overview

This feature updates transient CLI help and documentation metadata rather than persistent business data. The model focuses on the public command tree, the help metadata attached to each visible command, and the generated documentation artifacts that must stay synchronized with those sources.

## Entities

### Covered Command

- **Purpose**: Represents one user-visible command node included in the help refresh scope.
- **Fields**:
  - `path`: canonical Cobra command path
  - `kind`: `root`, `group`, or `leaf`
  - `visible`: whether the command is intentionally user-facing
  - `aliases`: public aliases surfaced for user invocation
- **Validation rules**:
  - Only user-visible commands are included in this feature.
  - Hidden/internal commands are excluded unless intentionally promoted to the public CLI.
  - Every covered command must receive refreshed help metadata and examples.

### Help Metadata

- **Purpose**: Captures the user-facing Cobra help content attached to a covered command.
- **Fields**:
  - `short`: concise command summary
  - `long`: expanded usage guidance
  - `examples`: example block shown in help and generated docs
  - `mutation`: read-only or state-changing behavior as described to the user
  - `automationGuidance`: any `--json`, `--automation`, `--auto-confirm`, or `--no-wait` guidance relevant to the command
- **Validation rules**:
  - `short` must help the caller recognize the command quickly.
  - `long` must explain when the command should be used and any important behavior distinctions relevant to that command.
  - `examples` must be refreshed for every covered command.
  - Mutation and automation guidance must stay truthful to current runtime behavior.

### Example Set

- **Purpose**: Represents the refreshed example block for a covered command.
- **Fields**:
  - `commandPath`: the command the examples belong to
  - `exampleRole`: `navigational`, `read`, `write`, `verification`, or `mixed`
  - `lines`: one or more copy-pasteable examples
  - `includesFollowUp`: whether the example set includes a verification or inspection step
- **Validation rules**:
  - Every covered command has at least one refreshed example.
  - Parent/group commands may use navigational examples to steer users toward child commands.
  - Applicable state-changing commands should include verification or follow-up inspection guidance.

### Command Family Entry

- **Purpose**: Represents a public parent/group command that routes users to the correct child workflow.
- **Fields**:
  - `path`: canonical parent command path
  - `familyPurpose`: what the command family is for
  - `childSelectionGuidance`: how users should choose among its children
  - `sharedFlags`: inherited or family-level flags that matter for discovery
- **Validation rules**:
  - Parent/group commands must do more than say “requires a subcommand.”
  - Routing guidance should remain concise and consistent with the existing command hierarchy.

### Generated CLI Page

- **Purpose**: Represents the generated `docs/cli/` page derived from command metadata.
- **Fields**:
  - `sourceCommand`: the covered command that owns the page content
  - `generatedPath`: Markdown path under `docs/cli/`
  - `syncedFromMetadata`: whether the page was regenerated from Cobra metadata
  - `public`: whether the page belongs to a user-visible command
- **Validation rules**:
  - User-visible commands must remain represented through regenerated pages, not hand-edited docs.
  - Hidden/internal commands should not silently become part of the public generated docs surface.

### Top-Level Guidance Surface

- **Purpose**: Represents broader user-facing guidance outside individual command pages.
- **Fields**:
  - `readmeSections`: top-level README guidance affected by command help changes
  - `docsIndexSync`: whether README changes require `make docs-content`
  - `useCasesSections`: workflow-oriented docs that may need updated examples or phrasing
- **Validation rules**:
  - README guidance must stay aligned with command help when user-visible examples or discovery language change.
  - `docs/index.md` should be refreshed through the established sync path when README changes matter to the docs homepage.

## Relationships

- A `Covered Command` owns one `Help Metadata` record.
- A `Covered Command` owns one `Example Set`.
- A `Command Family Entry` is a specialized `Covered Command` with child-selection guidance.
- A `Covered Command` may generate one `Generated CLI Page`.
- A `Top-Level Guidance Surface` must stay consistent with the aggregate command help model when README-level guidance changes.

## Notes

- No persistent storage, network schema, or business-domain entity is introduced by this feature.
- The main correctness risk is drift between live Cobra help, generated CLI docs, and top-level user guidance.
- Planning and tasks should treat the public command tree as the primary coverage model and generated docs as a derived artifact.
