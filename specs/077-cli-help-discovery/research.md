# Research: Improve CLI Help Discovery

## Decision 1: Treat the full public Cobra tree as the planning boundary

- **Decision**: Cover every user-visible command node in the public `c8volt` Cobra tree, including the root command, parent/group commands, and executable leaf commands.
- **Rationale**: Clarification locked the scope to all user-visible commands, not a representative subset. The public CLI is experienced as a tree, so leaving parent commands under-specified would still create discovery gaps even if leaf commands were improved.
- **Alternatives considered**:
  - Cover only representative command families: rejected because the clarified scope explicitly broadened to all user-visible commands.
  - Cover only executable leaf commands: rejected because callers often arrive at group commands first when discovering the CLI.

## Decision 2: Exclude hidden or internal commands from the refresh scope

- **Decision**: Hidden/internal commands remain out of scope unless they are intentionally surfaced as public commands.
- **Rationale**: The clarified scope is “all user-visible commands,” not shell plumbing or framework helpers. The repository already distinguishes discoverable public commands from hidden or internal ones, and generated docs should continue to reflect the same boundary.
- **Alternatives considered**:
  - Refresh hidden/internal commands too: rejected because it would expand the work without improving public command discovery.
  - Refresh internal commands only if they leak into docs: rejected because the better fix is to preserve the public/private boundary, not document internals by default.

## Decision 3: Use Cobra metadata as the single authoritative source of help and generated docs

- **Decision**: Edit command `Short`, `Long`, and `Example` metadata in `cmd/` and regenerate `docs/cli/` from those sources with `make docs`.
- **Rationale**: The repo’s documentation conventions explicitly say CLI reference pages are generated from Cobra command metadata. Updating the source metadata first keeps live help and generated docs synchronized.
- **Alternatives considered**:
  - Hand-edit generated docs under `docs/cli/`: rejected because the repo already provides a generation path and explicitly forbids hand-editing generated pages.
  - Add a second docs-only help layer outside command metadata: rejected because it would create drift between `--help` output and generated docs.

## Decision 4: Parent/group commands need first-class discovery guidance, not placeholder help

- **Decision**: Parent/group commands should explicitly explain the command-family purpose, mutation posture, and how callers should choose the right child command path.
- **Rationale**: Files such as `cmd/get.go` and `cmd/delete.go` currently use minimal family descriptions. Since the feature now covers the full public tree, parent commands must help users route toward the right leaf command instead of only saying that a subcommand is required.
- **Alternatives considered**:
  - Leave parent/group commands mostly unchanged and focus on leaves: rejected because it weakens top-down command discovery.
  - Add long conceptual prose to parent commands only: rejected because parent commands still need concrete examples and chooser guidance, not just overview text.

## Decision 5: Every covered command should receive refreshed examples, but example roles may differ by command type

- **Decision**: Refresh examples for every covered command; parent/group commands may use navigational or chooser-oriented examples, while executable leaves should use realistic operational examples.
- **Rationale**: Clarification explicitly selected examples for every command. Allowing parent commands to use routing-oriented examples keeps the requirement practical without weakening the full-coverage commitment.
- **Alternatives considered**:
  - Refresh examples only for leaf commands: rejected by clarification.
  - Force every example block to be operationally complex: rejected because some parent/group commands exist to route users rather than perform work directly.

## Decision 6: Reuse the repository’s existing automation and machine-contract language

- **Decision**: Align help refresh work with current repository terms and behavior around `capabilities --json`, `--automation`, `--json`, `--auto-confirm`, and `--no-wait`.
- **Rationale**: The repo already established automation guidance in `root.go`, `capabilities.go`, README, and recent machine-contract features. Reusing that language keeps issue `#77` compatible with `#78` and `#79` instead of inventing a parallel phrasing model.
- **Alternatives considered**:
  - Rewrite automation guidance from scratch for this feature: rejected because it would risk drifting from current runtime behavior and recent contract work.
  - Avoid automation guidance in command help: rejected because the issue specifically calls for command-level automation discoverability.

## Decision 7: Keep the implementation metadata-first and behavior-light

- **Decision**: Treat the feature primarily as a command metadata and docs-generation refactor, allowing only small behavioral clarifications when necessary to keep help truthful.
- **Rationale**: The spec and issue both bound broad behavioral changes out of scope. The repository conventions also prefer incremental refactors that preserve externally observable behavior unless a change is explicitly required.
- **Alternatives considered**:
  - Pair the docs refresh with broad runtime changes for consistency: rejected because it would turn a help-discovery issue into a larger behavior project.
  - Accept inaccurate help text where behavior is awkward: rejected because the issue explicitly says behavior gaps should be recorded, not hidden.

## Decision 8: Use focused command-help regression checks as the primary proof

- **Decision**: Add or update targeted `cmd/` regression coverage for root, parent/group, and leaf help output, then regenerate docs and run `make test`.
- **Rationale**: Help behavior is exposed through Cobra command metadata, so focused command-level checks are the most direct proof that the updated guidance stays intact. The constitution also requires tests plus final repository validation.
- **Alternatives considered**:
  - Rely only on manual review of generated docs: rejected because it would not protect live `--help` output or prevent regressions.
  - Require a full snapshot test for every generated CLI page: rejected as heavier than necessary for planning and likely too brittle for this repo’s current workflow.

## Decision 9: Treat README and docs homepage sync as conditional but explicit

- **Decision**: Regenerate CLI docs with `make docs` whenever command metadata changes, and run `make docs-content` when README changes should flow through to `docs/index.md`.
- **Rationale**: The Makefile and AGENTS guidance separate these concerns: CLI docs come from `docsgen`, while docs homepage content is synced from README. Planning should preserve that exact split so tasks do not hand-edit generated docs.
- **Alternatives considered**:
  - Always run both doc-generation commands regardless of touched sources: rejected because the repo already distinguishes the generation paths.
  - Update README without syncing docs homepage: rejected because the AGENTS guidance explicitly requires regeneration when README changes should appear in docs.

## Decision 10: Use the public command inventory in `cmd/` as the rollout map

- **Decision**: Base task generation on the user-visible command files under `cmd/`, including root, family entry points, and leaf commands such as `get_processinstance`, `delete_processdefinition`, `config_show`, `embed_export`, and related siblings.
- **Rationale**: The command files in `cmd/` already define the public surface and carry the metadata that feeds generated docs. Using them as the rollout map keeps the plan concrete and repository-native.
- **Alternatives considered**:
  - Derive the scope only from generated docs: rejected because the source-of-truth edits belong in `cmd/`.
  - Derive the scope only from README examples: rejected because README does not enumerate the full public tree.

## Repository-Native Anchors

| Area | Current anchors | Why they fit this feature |
|--------|-----------------|--------------------------|
| Root and inherited guidance | `cmd/root.go` | owns the top-level product story, inherited flags, and machine-discovery guidance |
| Parent/group routing help | `cmd/get.go`, `cmd/delete.go`, `cmd/deploy.go`, `cmd/run.go`, `cmd/cancel.go`, `cmd/walk.go`, `cmd/expect.go`, `cmd/config.go`, `cmd/embed.go` | define how users discover command families before choosing a leaf |
| Leaf operational guidance | `cmd/*_processinstance.go`, `cmd/*_processdefinition.go`, `cmd/get_cluster_*.go`, `cmd/config_show.go`, `cmd/embed_*.go`, `cmd/get_resource.go`, `cmd/get_variable.go`, `cmd/version.go` | hold the command-level `Short`, `Long`, and `Example` text users actually execute |
| Public/private command boundary | `cmd/command_contract.go`, Cobra `Hidden` state, discoverable command traversal in capabilities/docs generation | already encode which commands are public versus hidden/internal |
| CLI docs generation | `Makefile`, `docsgen`, `docs/cli/` | existing source-to-generated-doc path that must remain authoritative |
| README-synced docs | `README.md`, `docs/index.md`, `make docs-content` | existing path for top-level user guidance that must stay synchronized |

## Public Command Inventory Snapshot

Snapshot captured on 2026-04-19 from the live Cobra tree via `GOCACHE=/tmp/c8volt-gocache go run . capabilities --json`, which reuses the same `isDiscoverableCommand` filtering used for discovery output.

- Public top-level commands, excluding the root: 11
- Public group commands, including the root: 11
- Public executable leaf commands, including top-level leaves: 19
- Total covered command pages in `docs/cli/`, excluding `index.md`: 30
- Hidden/internal commands excluded from this feature: Cobra `help`, hidden commands, and shell completion plumbing such as `__complete*`

### Audit Notes

- The live `capabilities --json` output currently exposes 29 discoverable public command paths beneath the root command; adding `c8volt` yields 30 covered public command pages.
- The generated docs directory currently contains 30 command pages plus `index.md`, matching the public help surface one-to-one.
- Both `get cluster topology` and `get cluster-topology` remain intentionally public and therefore each retain a generated reference page.
- Hidden and internal commands remain excluded from both discovery output and generated docs; no `help`, `completion`, or `__complete*` page is present in `docs/cli/`.

### Public Command Paths In Scope

| Layer | Command paths |
|--------|---------------|
| Root | `c8volt` |
| Top-level groups | `cancel`, `config`, `delete`, `deploy`, `embed`, `expect`, `get`, `run`, `walk` |
| Nested groups | `get cluster` |
| Top-level leaves | `capabilities`, `version` |
| Nested leaves | `cancel process-instance`, `config show`, `delete process-definition`, `delete process-instance`, `deploy process-definition`, `embed deploy`, `embed export`, `embed list`, `expect process-instance`, `get cluster license`, `get cluster topology`, `get cluster-topology`, `get process-definition`, `get process-instance`, `get resource`, `run process-instance`, `walk process-instance` |

### Batching Notes For Implementation

- Batch 1: Setup and foundational guardrails stay inside feature artifacts plus shared discovery seams in `cmd/root.go`, `cmd/capabilities.go`, `cmd/command_contract_test.go`, and `cmd/root_test.go`.
- Batch 2: User Story 1 should refresh entry-point language before leaf help text: root and family-entry commands first, then discovery/config/cluster reads, then the remaining read-oriented retrieval commands, then embed/version utilities.
- Batch 3: User Story 2 should follow the same pattern for state-changing semantics: regression coverage first, then `run`/`deploy`, then `cancel`/`delete`, then `expect`/`walk` verification guidance.
- Batch 4: User Story 3 should update top-level docs text before regeneration so `make docs` and `make docs-content` remain the only source of generated output changes.
- Validation batching should stay incremental: focused `go test ./cmd -count=1` during command metadata work, `make docs` after help metadata changes, `make docs-content` only when README changes, and `make test` before a commit.

## Validation Baseline

- Run focused command metadata/help tests first with `go test ./cmd -count=1`.
- Regenerate CLI reference pages with `make docs`.
- Regenerate README-synced docs with `make docs-content` if README changed.
- Finish with `make test`.
