# Function Comment Quality Prompt

Use this prompt to improve meaningful function and test comments across
`c8volt/...` and `internal/...`. It is intentionally not a mechanical doc-comment
generation task; comments must explain purpose, contract, behavior, or test
intent.

Before using this prompt, follow `specs/prompts/AGENTS.md`.

```text
Apply specs/ralph-implementation-rules.md to the current repository state and improve function comments across:
- c8volt/...
- internal/...

Repository safety gate:
Before any implementation work, check the current git branch.
If the current branch is not develop, stop and report the branch name.
If the current branch is develop, create a new feature branch before editing files.
Use the repository's existing branch naming and numbering conventions where possible.
Do not implement directly on develop.

Mandatory preparation:
After the branch is ready, deeply scan all Go packages under:
- c8volt/...
- internal/...

Do not limit the scan to selected files. Include every package and subpackage under both trees.

Goal:
Add or improve comments for every related created, modified, exported, and non-exported function touched by this work.
Pay special attention to test functions and test helpers.

Comment quality rules:
- Comments must explain purpose, behavior, contract, or test intent.
- Do not add mechanical comments that merely repeat the function name.
- Do not add comments like `foo does foo`.
- Use existing high-quality comments in nearby packages as style references.
- Align existing comments when they are stale, vague, misleading, or inconsistent with the implementation.
- Keep comments concise but meaningful.
- Prefer explaining why the function exists, what behavior it guarantees, or what scenario a test protects.
- For test functions, describe the behavior, regression, or user-facing contract being verified.
- For test helpers, describe the role of the helper in the test setup or assertion flow.
- For internal service functions, describe service responsibility and important boundary behavior.
- For facade functions, describe public behavior and delegation boundary, not implementation mechanics.
- For command functions, describe command construction or CLI contract, not low-level helper logic.
- Do not use comments to justify bad layering; fix the layering if this task includes implementation work.
- Do not generate comments mechanically from function names.
- If a function is trivial and genuinely self-explanatory but must be commented because it is touched, write the smallest useful contract comment.

Implementation rules:
After the scan, update comments in small coherent slices.
Do not change runtime behavior.
Do not rename functions, move code, or refactor logic unless strictly required to make a stale comment truthful.
Do not contaminate cobra command files with helper methods.
Do not create files with generic names like helper.go, helpers.go, util.go, utils.go, or similar.
Preserve existing package style and terminology.
Use Go doc style where applicable:
- exported identifiers should have comments beginning with the identifier name when practical and natural
- non-exported function comments do not need to begin with the function name if a clearer purpose comment is better
- test comments should prioritize scenario and expected behavior over naming convention

Validation:
Run targeted validation after each slice.
If validation cannot be run, state why.

Commit policy:
Do not commit.
Do not push.
Leave all changes unstaged unless explicitly asked otherwise.
```
