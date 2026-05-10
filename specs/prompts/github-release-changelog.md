# GitHub Release Changelog Prompt

Use this prompt whenever a short changelog summary is needed for the GitHub
Release "Changelog" field. Replace the placeholders before running it.

Before using this prompt, follow `specs/prompts/AGENTS.md`. In particular, do
not treat files under `specs/prompts/` as release-change source material.

```text
Create a short GitHub Release changelog for c8volt release <VERSION>, using
<BASE_COMMIT> as the starting point for changes included in this release. If a
release date is provided, use it as <RELEASE_DATE>; otherwise use today's date.

Inputs:
- Release version: <VERSION>
- Release date: <RELEASE_DATE>
- Base commit: <BASE_COMMIT>
- Target output: GitHub Release changelog field
- Primary change source: specs/
- Style reference: https://github.com/grafvonb/c8volt/releases/tag/v3.6.0

Goal:
Produce a concise, paste-ready Markdown list of the most important user-facing
changes since <BASE_COMMIT>. The result should read like the c8volt v3.6.0
release changelog: short, practical, release-facing, and focused on core
operator value rather than implementation history.

Required output style:

## Changelog
- feat: Add incident presence expectations for `expect process-instance`, including stdin key piping and combined state checks. #170
- feat: Show incident details in `get pi --with-incidents` list/search output with compact human rows, JSON preservation, limits, and paging support. #171
- feat: Add `get pi --with-vars` process-variable enrichment with sorted output, JSON metadata, value limits, and incident combination support. #173
- feat: Validate BPMN process-definition selectors before process-instance get/cancel/delete/run operations to separate missing definitions from empty instance results. #175
- feat: Add follow-up config source logging for `config show` diagnostics. #166

Use the same format for the new release:
- Start with `## Changelog`.
- Use one Markdown bullet per core change.
- Start each bullet with a conventional-change prefix such as `feat:`, `fix:`,
  or `docs:` when that matches the user-visible change.
- Keep each bullet to one sentence.
- Mention the relevant command or flag in backticks when useful.
- End each bullet with the primary issue number when a clear issue drove the
  change.

Required workflow:
1. Inspect the existing GitHub Release style before drafting:
   - Prefer `gh release view v3.6.0 --repo grafvonb/c8volt --json body` when
     credentials are available.
   - Otherwise inspect the public release page at the style-reference URL.
   - If the release cannot be fetched, use the embedded v3.6.0 style example
     above as the source of truth.
   - Preserve the compact release-note shape, but do not copy old release text.
2. Identify relevant changes since <BASE_COMMIT>:
   - `git log --oneline <BASE_COMMIT>..HEAD`
   - `git diff --name-only <BASE_COMMIT>...HEAD`
   - `find specs -mindepth 2 -maxdepth 3 -type f ! -path 'specs/prompts/*' | sort`
   - targeted `rg` searches for command, flag, API, README, and docs keywords
3. Use `specs/` as the primary source of release intent. Cross-check notable
   items against implementation, tests, README updates, generated CLI docs, and
   help output when the source material disagrees.
4. Group changes by user-visible capability, not by commit order. Merge several
   small commits into one changelog bullet when they describe one operator
   outcome.
5. Include only notable release-facing changes, for example:
   - new or expanded CLI commands
   - important flags, selectors, filters, or output modes
   - safer mutation behavior or better operator guardrails
   - Camunda version support changes
   - automation, JSON, or pipelining improvements
   - significant README/help/docs changes that affect real user workflows
6. Exclude internal noise unless it materially changes user experience:
   - prompt-only changes under `specs/prompts/`
   - test-only changes
   - generated docs churn without behavioral or documentation-value change
   - refactors that preserve observable behavior
   - dependency or repository maintenance that is not release-relevant
7. Do not overclaim. If a change appears in a spec but cannot be confirmed in
   code, tests, CLI docs, README, or help output, omit it or mark it as a
   candidate outside the paste-ready changelog.
8. Keep the changelog short. Prefer 5-10 bullets. If there are more candidate
   changes, keep the ones that a user upgrading from the previous release most
   needs to notice.
9. Write for users of the CLI, not for repository maintainers. Prefer concrete
   command-family language over issue numbers, Speckit terms, or commit hashes.

Suggested investigation commands:
- `git log --oneline <BASE_COMMIT>..HEAD`
- `git diff --name-only <BASE_COMMIT>...HEAD`
- `git log --oneline --no-merges <BASE_COMMIT>..HEAD`
- `find specs -mindepth 2 -maxdepth 3 -type f ! -path 'specs/prompts/*' | sort`
- `rg -n "<feature|command|flag keywords>" specs cmd docs/cli README.md`
- `rg -n "^## |^### |^\\* |^- " specs/*/{spec,plan,tasks,quickstart}.md README.md docs/cli`
- `gh release view v3.6.0 --repo grafvonb/c8volt --json body`

Output expectations:
- Produce the paste-ready GitHub Release changelog first.
- Use the embedded v3.6.0 `## Changelog` bullet format unless the caller
  explicitly asks for a different release-note shape.
- Keep each bullet short and outcome-oriented.
- Do not include validation logs, raw commit lists, or long explanatory
  sections in the paste-ready changelog.
- After the paste-ready changelog, include a short "Notes" section only when
  needed for uncertainty, omitted candidates, or style-reference access issues.
- Do not update files and do not commit unless asked.
```
