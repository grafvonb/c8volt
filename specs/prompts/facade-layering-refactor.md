# Facade Layering Refactor Prompt

Use this prompt to run the complete facade-layering refactor against either a
clean repository state or an existing dirty refactor branch. It requires branch
and working-tree safety checks, a full scan of `c8volt/...` and `internal/...`,
implementation of every planned slice, and no commits.

Before using this prompt, follow `specs/prompts/AGENTS.md`.

```text
Apply specs/ralph-implementation-rules.md to the current repository state and perform the complete facade-layering refactor.

Repository and working-tree safety gate:
Before any implementation work, inspect:
- current git branch
- git status
- current diff, if any

If the current branch is develop:
- create a new feature branch before editing files
- use the repository's existing branch naming and numbering conventions where possible

If the current branch is not develop and the working tree is dirty:
- do not reset, revert, discard, overwrite, or stage existing changes
- inspect the diff first
- treat coherent existing changes as already completed refactor slices
- summarize what they appear to complete before making more edits
- continue from the current state

If the current branch is not develop and the branch appears unrelated to this refactor:
- stop and report the branch name and dirty files before editing

Do not implement directly on develop.
Do not commit.
Do not push.
Leave changes unstaged unless explicitly asked.

Mandatory full scan:
After the branch is ready, deeply scan every Go package under:
- c8volt/...
- internal/...

Do not limit the scan to known packages. Include every package and subpackage under both trees.

Identify all facade-layering violations and misplaced backend execution mechanics, especially logic that belongs in internal services but currently lives in public facade packages.

Focus on moving these mechanics out of public facade packages and into internal services:
- worker pools
- polling and waiters
- mutation confirmation
- pagination loops
- bulk execution strategy
- impact analysis
- dependency expansion
- retry or fail-fast behavior
- cross-resource backend workflows
- reusable domain/service logic currently duplicated in facade code

Before editing files, produce a complete refactoring plan.

For each required change, include:
- current file/function
- exact layering problem
- target internal package/service where the logic should move
- whether a new service method, domain type, or test is needed
- migration order
- risk level
- targeted tests to run after that slice

Execution requirement:
After producing or updating the plan, implement every remaining required refactor slice.
Do not stop after the first successful slice.
Do not stop merely because one coherent slice was completed.
After each slice:
- update the visible progress checklist
- run targeted validation
- continue to the next planned slice

Only stop before completing all slices if:
- a test failure or compiler error blocks further progress
- required behavior is ambiguous and cannot be safely inferred
- the refactor would require changing public behavior
- the repository state changed in a conflicting way

If blocked, report the exact blocker, files involved, and what remains.

Completion gate:
Do not provide the final answer until all planned slices are either completed or explicitly marked blocked with reasons.

Before final response, re-scan c8volt/... and internal/... for remaining facade-layering smells, including:
- toolx/pool usage in c8volt/...
- toolx/poller usage in c8volt/...
- waiter package usage in c8volt/...
- logging.StartActivity used for backend workflow orchestration in c8volt/...
- worker count selection in c8volt/...
- custom goroutine/channel worker code in c8volt/...
- backend pagination loops in c8volt/...
- mutation confirmation or wait loops in c8volt/...
- cross-resource backend workflows in c8volt/...

For every remaining smell, either fix it or list why it is intentionally left.

Implementation rules:
Complete one coherent slice at a time, but complete all slices.
Do not mix unrelated slices in one edit.
Preserve externally observable CLI behavior.
Keep public facade methods thin: map inputs, delegate to internal services, map outputs, normalize errors.
Do not add worker pools, waiters, polling, confirmation logic, pagination loops, retry logic, impact analysis, dependency expansion, or backend workflows to public facade packages.
Do not contaminate cobra command files with helper methods.
Do not create files with generic names like helper.go, helpers.go, util.go, utils.go, or similar.
Prefer existing packages, services, clients, domain types, and toolx helpers before adding new abstractions.
Use current Camunda 8.9 and v2 client/service paths where applicable.
Add or update focused tests for preserved behavior.
Add comments for every created or modified exported and non-exported function.

Validation:
Run targeted validation after each slice.
After all slices, run broader validation covering:
- c8volt/...
- internal/...

If validation cannot be run, state why.

Commit policy:
Do not commit.
Do not push.
Leave all changes unstaged unless explicitly asked otherwise.

Final response must include:
- existing dirty changes that were preserved, if any
- every completed slice
- every remaining violation, if any
- validation commands run
- whether anything is blocked
- confirmation that no commit, push, reset, revert, discard, or staging was performed
```
