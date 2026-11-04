# Release GitHub Changelog Prompt

Use this prompt for a compact, paste-ready GitHub Release changelog. Replace
the inputs before running it. Follow `specs/prompts/AGENTS.md`; prompt templates
are not evidence of product changes.

```text
Create the GitHub Release changelog for c8volt <VERSION>, comparing the released
baseline <BASE_REF> with the exact target <TARGET_REF>.

Inputs:
- Release version: <VERSION>
- Previous release tag: <BASE_REF>
- Target commit or release tag: <TARGET_REF>
- Primary Camunda version: <CAMUNDA_FOCUS>
- Style release: <STYLE_RELEASE> (optional; default to the baseline release)
- GitHub milestone: <MILESTONE_URL> (optional)
- Release date: <RELEASE_DATE> (optional; omit if unspecified)

Goal:
Explain what changes for an operator upgrading from the baseline release.
Prioritize the primary Camunda version, breaking changes, and important new
capabilities. Keep the result compact and professional for Camunda DevOps users.

Required workflow:
1. Resolve both refs to commits and record the comparison range. For a major
   release, use the latest stable tag of the preceding major unless the caller
   specifies another baseline. Do not infer the baseline from git describe.
   Verify that the baseline is an ancestor of the target; surface a mismatch
   instead of silently changing the comparison. Resolve HEAD once if supplied.
2. Inspect the style release with gh release view <STYLE_RELEASE> --repo
   grafvonb/c8volt --json body, or its public release page. If unavailable, use
   the generic output shape below and mention the limitation outside the draft.
   Reuse the compact format, not the historical release text or feature list.
3. Inspect changes in the exact range:
   - git log --oneline --no-merges <BASE_REF>..<TARGET_REF>
   - git diff --name-only <BASE_REF> <TARGET_REF>
   - git diff <BASE_REF> <TARGET_REF> -- cmd c8volt internal/services toolx
   - rg --files specs -g spec.md -g '!specs/prompts/**'
   Inspect relevant feature artifacts, source metadata, tests, README, and CLI
   references at the selected refs. Exclude uncommitted work and prompt files.
4. Build a complete candidate inventory before shortening it: version support,
   command families, selectors, mutation actions, variable scopes, confirmation,
   output/automation contracts, and operational controls. Group related commits
   by capability. Do not let a large feature disappear behind a recent small fix.
5. Verify each candidate against implementation at both refs. Specs explain
   intent; changed implementation establishes shipped behavior. A closed issue,
   completed task, current README claim, or generated page alone is insufficient.
   Use milestone issues as additional candidates, not proof of implementation.
6. Separate new capabilities from existing foundations. Do not present an
   existing workflow as introduced in this release merely because its docs were
   rewritten. Describe an improvement only where the comparison proves it.
7. For a major release, identify breaking CLI grammar, removed flags/aliases,
   removed version support, and necessary script changes explicitly. Verify
   retained compatibility separately from features requiring the primary version.
   Do not describe earlier prerelease support as the first appearance of support
   when the actual change is stable API alignment or expanded functionality.
8. Exclude internal refactors, test-harness fixes, prompt maintenance, generated
   churn, and dependency updates without a material user-facing effect. Keep
   exceptional flags and implementation mechanics out of the highlights.
9. Review every draft bullet for novelty, implementation evidence, correct flag
   spelling, scope, version availability, and issue attribution. Omit unverified
   claims and list any material uncertainty separately.

Required output:
- Start with ## Changelog.
- Prefer 5-10 one-sentence bullets, grouped by operator outcome.
- Lead with support for the primary Camunda version when it is the release
  headline. Follow with breaking changes and their necessary upgrade actions,
  then other capabilities and fixes in operator-priority order.
- Use feat!: or fix!: for breaking changes; otherwise use feat:, fix:, or docs:
  only when that prefix matches the user-visible change.
- Put commands and flags in backticks; keep version hints short and unobtrusive.
- End a bullet with its primary issue reference when verified; use a few related
  references for a grouped capability when necessary. Never invent attribution.
- Preserve important feature coverage by grouping related work before dropping it.
- Do not include raw commit lists, validation logs, architecture narration, or
  claims that unexecuted tests passed.

Output shape (placeholders illustrate format, not release-change evidence):

## Changelog
- feat: <Primary Camunda version support and retained compatibility>. #<ISSUE>
- feat!: <Breaking change and required operator migration>. #<ISSUE>
- feat: <New capability and its practical operator benefit>. #<ISSUE>
- fix: <Corrected behavior and affected workflow>. #<ISSUE>

Produce the paste-ready changelog first. Add a short Notes section afterward
only for material uncertainty, omitted candidates, or unavailable source/style
references. Do not invent a release date.

Do not run tests, build the CLI, contact Camunda, or change cluster data for this
task. Do not modify files, commit, push, or publish unless explicitly requested.
```
