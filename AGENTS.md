# AGENTS.md

## Purpose
- This file defines repository-specific guidance for agents working in `c8volt`.
- Keep instructions focused on durable project conventions, architecture boundaries, validation, and documentation.
- Do not add temporary issue details here. Put feature-specific requirements in the matching `specs/<feature>/` artifacts.

## Required Project Context
- Before implementation work, read the active feature artifacts under `specs/<feature>/` when they exist: `spec.md`, `plan.md`, `tasks.md`, and supporting files such as `research.md`, `data-model.md`, `quickstart.md`, and `contracts/`.
- Ralph-driven implementation work must also read and follow `specs/ralph-implementation-rules.md`.
- The Ralph rules are mandatory for Ralph iterations and contain the detailed project map, layering rules, CLI UX rules, validation guidance, and iteration discipline.
- If feature artifacts conflict with `specs/ralph-implementation-rules.md`, stop and surface the conflict before implementing.

## Project Architecture
- CLI commands, flags, validation, command metadata, and rendering live under `cmd/`.
- Public facade APIs live under `c8volt/` and `c8volt/<area>/`.
- Public facade options live in `c8volt/foptions`; public facade error conversion lives in `c8volt/ferrors`.
- Version-neutral domain types live in `internal/domain`.
- Version-neutral service contracts and factories live in `internal/services/<area>`.
- Camunda version-specific adapters live in `internal/services/<area>/v87`, `v88`, and `v89`.
- Generated Camunda clients live in `internal/clients/camunda`; avoid hand-editing generated code unless explicitly required.
- Shared production helpers live in `toolx`; shared test helpers live in `testx`.
- Feature planning artifacts live under `specs/<feature>/`.

## Layering Rules
- Prefer existing package ownership and local patterns before introducing new structures.
- `cmd` may call public facades and command support helpers, but must not call generated Camunda clients or versioned service implementations directly.
- Public facades should be thin: map public inputs to internal service inputs, delegate to internal services, map outputs back, and convert errors through `c8volt/ferrors`.
- Backend mechanics such as pagination loops, worker pools, polling, retries, wait loops, dependency expansion, and mutation workflows belong in internal services, not in CLI commands or public facades.
- Keep version-specific API differences explicit in the matching `v87`, `v88`, or `v89` service package.
- Reuse `toolx`, `toolx/pool`, `internal/services/common`, and `testx` helpers before adding new helper code.

## CLI And Operator UX
- Preserve established command output patterns and wording from nearby command renderers and tests.
- Human output should be compact, stable, and scan-friendly.
- JSON and other machine-readable output must use stable structs and the shared command envelope when the command supports the shared contract.
- Keys-only output must print one key per line and nothing else.
- Do not add noisy endpoint, request, cursor, or per-key lifecycle detail to default human output; keep diagnostics behind `--verbose`.
- When command output changes, update tests for the affected human, JSON, keys-only, error, prompt, and activity behavior where relevant.

## Testing And Validation
- Add or update tests close to the changed package.
- Prefer targeted validation first, then broader validation when the change has wider blast radius.
- For command changes, start with targeted `go test ./cmd -run '<TestNameOrPattern>' -count=1`.
- For facade changes, start with targeted `go test ./c8volt/<area> -run '<TestNameOrPattern>' -count=1`.
- For internal service changes, start with targeted `go test ./internal/services/<area>/... -run '<TestNameOrPattern>' -count=1`.
- The repository full test target is `make test`, which runs `go test ./... -race -count=1`.
- Run `gofmt` on touched Go files.
- If validation cannot be run, call that out explicitly.

## Documentation And Generated Artifacts
- Keep user-facing documentation and examples aligned with behavior changes.
- When command behavior, flags, aliases, examples, or output contracts change, update command source metadata and regenerate CLI docs with `make docs-content`.
- Do not hand-edit generated CLI docs under `docs/cli/*` when the command source can regenerate them.
- For generated Camunda clients, prefer updating the OpenAPI mutation or refresh workflow under `api/` and regenerating clients.
- Speckit memory under `.speckit/memory` and feature artifacts under `specs/` are project context and should be preserved.

## Git And Commit Rules
- Reuse existing issue or feature branches when they already exist.
- Do not create or switch branches unless the user explicitly asks.
- For GitHub issue-backed Spec Kit work, the GitHub issue number is authoritative for the `specs/<number>-<slug>/` prefix and feature branch label. This overrides `.specify/extensions/git/git-config.yml` `branch_numbering: sequential`; pass the issue number explicitly with `--number <issue>` or correct the generated folder and references before planning or implementation continues.
- Commit messages must follow Conventional Commits format.
- Add a scope in parentheses when a clear scope exists.
- Reference the issue in the subject when applicable.
- Prefer small commits grouped by purpose.
- Run the closest relevant checks before committing when the repository provides them.

## Active Speckit Plan
<!-- SPECKIT START -->
- Active Speckit implementation plan: `specs/139-pi-variable-search/plan.md`
<!-- SPECKIT END -->
