# Data Model: Confirmation Prompts on Stderr

No persistent entity, public data schema, backend request, or version-specific domain model changes are required. The following describe existing transient interaction values, not new production structs.

| Concept | Values and relationship | Invariant |
|---|---|---|
| Prompt destination | Command stderr writer, or standard stderr fallback | Never result stdout; custom/inherited stderr is honored. |
| Confirmation prompt | Existing text and default-no `[y/N]` or default-yes `[Y/n]` label | Preserve formatting and trailing spacing; no logger prefix. |
| Input eligibility | Existing auto-confirm/automation policy and terminal guards | Writer selection does not alter eligibility or cause a skipped read. |
| Answer | Existing normalized line, empty line, or failed scan/EOF | Case and surrounding whitespace handled as before. |
| Decision | Accepted or `ErrCmdAborted` through existing conversion | Caller retains its own abort or normal paging-stop interpretation. |
| Result stream | Existing rendered results/keys | No prompt text; keys-only remains one key per line. |

## State transitions

1. Existing caller guards may skip the question or reject the command as before.
2. A reached helper returns immediately for auto-confirm or non-terminal stdin under its existing rules.
3. Otherwise, write the unchanged prompt to the selected stderr destination, then scan stdin.
4. Failed scan/EOF yields the existing abort error for either default.
5. Normalized `y` or `yes` accepts; an empty line accepts only default-yes; other answers abort.
6. The caller continues, ends recovery, stops paging normally, or returns an error according to its existing behavior.

No uniqueness, retention, migration, concurrency, or storage rules are introduced.
