# Prompt Catalog

This folder contains reusable manual prompts for repository maintenance,
release preparation, validation, and focused refactor work. Treat these files as
prompt templates, not product source material or Speckit feature artifacts.

Before running any prompt from this folder, read `AGENTS.md` in this directory.
Most prompts contain a fenced `text` block; run them by copying that block into
Codex or by asking Codex to run the prompt from the file path.

## Prompt Families

- Runtime stability: real-cluster validation of supported customer targets.
- Release documentation: README, docs, help examples, and release notes.
- Codebase maintenance: architecture and comment-quality cleanup prompts.
- Folder guidance: local rules and this catalog.

## Files

| File | Purpose | How To Run |
| --- | --- | --- |
| `AGENTS.md` | Defines shared rules for this prompt folder: scope, safety, prompt style, and how to avoid treating prompt templates as release or feature source material. | Read it before using or editing any prompt in this folder. It is guidance, not a runnable prompt. |
| `camunda-88-mainstream-stability-validation.md` | Runs a real Camunda 8.8 stability validation using `config.yaml` profile `kind-camunda-platform-local-c88`, with private C88 substitutions and no docs/help/example edits. | Ask Codex: `Run the prompt from specs/prompts/camunda-88-mainstream-stability-validation.md`. Make sure the local C88 cluster is running and reachable through the required profile first. |
| `release-docs-help-example-validation.md` | Performs the canonical release real-cluster validation for README, docs, generated docs, CLI help examples, and VHS syntax review. Includes a README-only/report-only mode for narrower checks. | Ask Codex: `Run the prompt from specs/prompts/release-docs-help-example-validation.md`. Use before releases or when docs/help examples need real validation. |
| `release-readme-refresh.md` | Refreshes `README.md` for a release by summarizing changes since a base commit and updating evergreen documentation where needed. | Replace `<VERSION>`, `<RELEASE_DATE>`, and `<BASE_COMMIT>`, then ask Codex to run it. |
| `release-github-changelog.md` | Produces a short paste-ready GitHub Release changelog for a release range, using specs and implementation as source material. | Replace `<VERSION>`, `<RELEASE_DATE>`, and `<BASE_COMMIT>`, then ask Codex to run it. |
| `architecture-facade-layering-refactor.md` | Drives the full facade-layering refactor across `c8volt/...` and `internal/...`, with branch safety checks, full scan, staged refactor slices, and validation. | Ask Codex: `Run the prompt from specs/prompts/architecture-facade-layering-refactor.md`. Start from a suitable branch or allow the prompt workflow to create one if on `develop`. |
| `code-comment-quality-audit.md` | Improves meaningful function and test comments across `c8volt/...` and `internal/...` without changing runtime behavior. | Ask Codex: `Run the prompt from specs/prompts/code-comment-quality-audit.md`. Run from `develop` so the prompt can create a feature branch as instructed. |

## Regular Execution Priority

1. `camunda-88-mainstream-stability-validation.md`: run before releases, after
   risky runtime changes, and after ops/process-instance/process-definition
   changes because Camunda 8.8 is the mainstream customer target.
2. `release-docs-help-example-validation.md`: run before every release,
   especially after command, help, docs, or example changes.
3. `release-readme-refresh.md`: run once per release after feature scope has
   stabilized and before final example validation.
4. `release-github-changelog.md`: run once per release after the README/release
   scope is clear.
5. `architecture-facade-layering-refactor.md`: run only during planned
   architecture cleanup.
6. `code-comment-quality-audit.md`: run opportunistically after broad refactors
   or before handoff when comment quality needs attention.

## Usage Pattern

For a runnable prompt, use one of these forms:

```text
Run the prompt from specs/prompts/<prompt-file>.md
```

or:

```text
Use specs/prompts/<prompt-file>.md with:
- <PLACEHOLDER>: <value>
- <PLACEHOLDER>: <value>
```

Some prompts perform real cluster mutations, create branches, or update files.
Read the prompt's safety and default-mode sections before running it, especially
for release validation and local-cluster stability prompts.
