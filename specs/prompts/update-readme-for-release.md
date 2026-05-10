# README Release Update Prompt

Use this prompt whenever `README.md` should be refreshed for a new c8volt
release. Replace the placeholders before running it.

Before using this prompt, follow `specs/prompts/AGENTS.md`. In particular, do
not treat files under `specs/prompts/` as release-change source material.

```text
Update README.md for c8volt release <VERSION>, using <BASE_COMMIT> as the
starting point for changes included in this release. If a release date is
provided, use it as <RELEASE_DATE>; otherwise use today's date.

Inputs:
- Release version: <VERSION>
- Release date: <RELEASE_DATE>
- Base commit: <BASE_COMMIT>
- Target document: README.md
- Primary change source: specs/

Goal:
Refresh README.md so it accurately describes the current product after all
changes since <BASE_COMMIT>, while preserving the existing README style:
operator-focused, concise but useful, command-first, and readable as a single
front-door document.

Required workflow:
1. Read README.md first and preserve its current voice, structure, heading
   style, command examples, and "done is done" positioning.
2. Identify relevant changes since <BASE_COMMIT>. Use `specs/` as the primary
   source of feature intent and behavior. Cross-check with implementation,
   generated CLI docs, tests, and help output when a spec and code differ.
3. Update the beginning of README.md with a short release highlight section for
   <VERSION>. Place it near the top, after the opening product description and
   before the rest of the evergreen content. State that <VERSION> was released
   on <RELEASE_DATE>, then summarize the most important new capabilities in a
   compact bullet list.
4. Extend existing sections instead of creating parallel duplicate sections
   whenever a feature naturally belongs in an existing workflow, command family,
   configuration note, automation note, command map, or everyday command block.
5. Add or update command examples for new commands, flags, output modes, filters,
   and behavior changes when they are user-facing.
6. Remove or rewrite topics that no longer exist, are no longer accurate, or are
   contradicted by the current specs or command behavior.
7. Keep the README deep enough to be useful, but still readable:
   - prefer short paragraphs and scannable command blocks
   - avoid long implementation explanations
   - avoid repeating the same feature in multiple places
   - keep release highlights brief and move durable detail into the relevant
     evergreen sections
8. Preserve externally important links, governance references, and release links
   unless they are obsolete.
9. If CLI reference docs are generated from source metadata, update the source
   and regenerate those docs instead of hand-editing generated output.
10. After editing, review README.md as a reader:
    - the release highlight appears near the beginning
    - new features are documented where users would look for them
    - changed flags and commands are reflected consistently
    - removed features or stale topics are gone
    - examples are plausible and match current command names
    - the document still feels like one cohesive README, not a changelog dump

Suggested investigation commands:
- `git diff --name-only <BASE_COMMIT>...HEAD`
- `git log --oneline <BASE_COMMIT>..HEAD`
- `find specs -mindepth 2 -maxdepth 3 -type f ! -path 'specs/prompts/*' | sort`
- `rg -n "<feature|command|flag keywords>" specs cmd docs/cli README.md`
- `go test ./cmd -count=1` or the closest relevant test package after edits

Output expectations:
- Update README.md only where needed.
- Mention any sections intentionally removed or left unchanged.
- Mention validation run, or explicitly say if validation was not run.
- Do not commit unless asked.
```
