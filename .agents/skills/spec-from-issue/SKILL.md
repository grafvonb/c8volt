---
name: spec-from-issue
description: Create or update a feature specification from a GitHub issue URL or issue-backed feature request. Use when you want the Speckit specify workflow, but the branch number must come from the GitHub issue number instead of auto-detection.
compatibility: Requires spec-kit project structure with .specify/ directory
metadata:
  author: local-wrapper
  wraps: speckit-specify
---

# Spec From Issue

Use this skill when the feature is tied to a GitHub issue and the branch number must match the issue number.

## Inputs

- A GitHub issue URL such as `https://github.com/grafvonb/c8volt/issues/45`
- Optionally, extra feature description text in the same request

## Required Behavior

1. Parse the GitHub issue URL from the user request.
2. Extract the numeric issue id from the URL path segment `/issues/<number>`.
3. Generate a concise 2-4 word short name using the same conventions as `speckit-specify`.
4. Run `.specify/scripts/bash/create-new-feature.sh` exactly once with:
   - `--json`
   - `--short-name "<generated-short-name>"`
   - `--number "<issue-number>"`
   - the feature description text
5. Treat the raw issue number as authoritative for the final branch and feature directory names.
6. If the script returns a zero-padded feature number such as `061`, normalize it back to the raw issue number form such as `61` for:
   - the branch name
   - the spec directory under `specs/`
   - any reported paths or follow-on workflow state
7. Do not use the script's auto-number detection when an issue URL is present.
8. If the issue URL is missing or malformed, stop and tell the user what was missing instead of guessing the issue number.

## Branch Number Override

This skill intentionally overrides the default `speckit-specify` instruction that says not to pass `--number`.

When a valid GitHub issue URL is provided, the issue number is the branch number.
Use the exact issue number text for the final branch and folder prefix. Do not keep zero-padded formatting from the wrapped script.

Examples:

- `https://github.com/grafvonb/c8volt/issues/45` -> branch must start with `45-`, not `045-`
- `https://github.com/grafvonb/c8volt/issues/61` -> branch must start with `61-`, not `061-`
- `https://github.com/NTTDATA-DACH/viewnode/issues/47` -> branch must start with `47-`, not `047-`

## Short Name Rules

- Keep the short name concise and descriptive.
- Use hyphen-separated words.
- Prefer action-noun phrasing when it reads naturally.
- Preserve important technical terms and acronyms.

Examples:

- `Add user authentication` -> `user-auth`
- `Implement OAuth2 integration for the API` -> `oauth2-api-integration`
- `Add ns command list subcommand for listing namespaces in a context` -> `add-ns-list`

## Execution Flow

1. Read the user request and find the issue URL.
2. Extract the issue number.
3. Determine the feature description from:
   - explicit user text, or
   - the issue title/summary if it is already available in the conversation
4. Generate the short name.
5. Run the script with `--number <issue-number>`.
6. Read the JSON output and inspect `BRANCH_NAME`, `SPEC_FILE`, and `FEATURE_NUM`.
7. If the script output uses zero-padded numbering, rename the branch and feature directory so they use the exact raw issue number instead.
8. After normalization, ensure downstream Spec Kit scripts resolve the normalized branch and spec directory names correctly before handing off to follow-on commands such as `speckit-clarify` or `speckit-plan`.
9. Continue with the normal `speckit-specify` workflow to write and validate the spec using the normalized names.

## Notes

- Only override branch numbering. Keep the rest of the `speckit-specify` workflow the same.
- Do not run the feature creation script more than once per feature.
- Do not treat zero-padded output from the wrapped script as authoritative when an issue URL is present.
- Prefer fixing shared Spec Kit branch-prefix validation to accept raw issue-number branches instead of relying on a one-off clarify workaround.
- If both a GitHub issue URL and an explicit conflicting `--number` are supplied by the user, prefer the issue number and mention that choice in the final report.
