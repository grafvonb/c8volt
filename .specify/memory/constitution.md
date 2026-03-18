<!--
Sync Impact Report
Version change: template -> 1.0.0
Modified principles:
- template principle 1 -> I. Operational Proof Over Intent
- template principle 2 -> II. CLI-First, Script-Safe Interfaces
- template principle 3 -> III. Tests and Validation Are Mandatory
- template principle 4 -> IV. Documentation Matches User Behavior
- template principle 5 -> V. Small, Compatible, Repository-Native Changes
Added sections:
- Project Constraints
- Delivery Workflow
Removed sections:
- None
Templates requiring updates:
- ✅ updated /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/.specify/templates/plan-template.md
- ✅ updated /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/.specify/templates/spec-template.md
- ✅ updated /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/.specify/templates/tasks-template.md
- ✅ reviewed /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/.specify/templates/agent-file-template.md
- ⚠ pending directory not present: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/.specify/templates/commands/
Follow-up TODOs:
- None
-->
# c8volt Constitution

## Core Principles

### I. Operational Proof Over Intent
Every user-facing command MUST verify the operational outcome it claims unless an
explicit opt-out flag already exists for that workflow. Features MUST preserve the
project's "done is done" standard: report success only after the observable target
state is reached or clearly state that confirmation was skipped. This keeps c8volt
trustworthy for administrators and automation.

### II. CLI-First, Script-Safe Interfaces
Behavior MUST be exposed through stable CLI commands, flags, exit codes, and text or
structured output that work in interactive and scripted environments. New commands
MUST follow existing Cobra layouts, naming, and flag propagation rules already used in
the repository. Changes that break documented command behavior require an explicit
compatibility note in the plan and accompanying documentation updates.

### III. Tests and Validation Are Mandatory
Every code change MUST add or update automated tests at the closest useful level, with
preference for realistic command execution paths when inherited flags or wiring matter.
Before merge or commit, contributors MUST run `make test`; a change is incomplete if
validation is skipped or knowingly failing. When a defect cannot be reproduced in an
automated test, the implementation plan MUST explain the gap and define the manual
verification path.

### IV. Documentation Matches User Behavior
User-visible command changes MUST update `README.md` and any relevant generated CLI
documentation in the same unit of work. Examples, flags, defaults, and caveats MUST
match shipped behavior so operators do not need to infer hidden rules from code. If a
change is internal-only, documentation may stay unchanged, but that decision MUST be
explicit in the plan or task list.

### V. Small, Compatible, Repository-Native Changes
Work MUST reuse existing repository patterns, dependencies, and command structures
before introducing new abstractions. Feature slices, PRDs, and tasks MUST stay small,
dependency-ordered, and independently verifiable so work can be implemented in short
iterations. New complexity requires a written justification in the implementation plan,
including why simpler repository-native alternatives were rejected.

## Project Constraints

- The canonical implementation stack is Go with Cobra-based CLI commands and the
  existing internal service layout.
- Command trees and subcommands MUST mirror established repository structure rather
  than inventing parallel hierarchies.
- Configuration-sensitive commands MUST honor root-level flag resolution and explicit
  kubeconfig or config-file persistence rules documented in repository guidance.
- Changes that affect operational state transitions, process polling, or bulk actions
  MUST preserve deterministic exit behavior and user-facing status reporting.

## Delivery Workflow

- Specifications MUST describe independently testable user stories ordered by
  delivery priority.
- Implementation plans MUST include a Constitution Check that confirms operational
  verification, CLI compatibility, required tests, documentation impact, and any
  justified complexity.
- Task lists MUST include concrete validation tasks and documentation tasks whenever a
  story changes user-visible behavior.
- Reviews MUST block merges when constitution requirements are unmet, even if code is
  otherwise functional.

## Governance

This constitution supersedes informal local practice for planning, implementation,
review, and documentation in this repository. Amendments require: (1) updating this
file, (2) recording sync impact at the top of the document, and (3) updating affected
templates or runtime guidance in the same change. Versioning follows semantic rules for
governance: MAJOR for incompatible principle changes or removals, MINOR for new
principles or materially stronger obligations, PATCH for clarifications that do not
change expected behavior. Compliance review is required in every feature plan and code
review, with unresolved exceptions documented under the plan's complexity or risk
tracking section.

**Version**: 1.0.0 | **Ratified**: 2026-03-15 | **Last Amended**: 2026-03-15
