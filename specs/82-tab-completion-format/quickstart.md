# Quickstart: Fix Terminal Command Completion Suggestion Formatting

## Goal

Verify that `c8volt` interactive completion produces readable user-facing suggestions without internal helper leakage or usage-style output.

## Prerequisites

- A local checkout on branch `82-tab-completion-format`
- Go toolchain matching the repository baseline
- A terminal shell where generated completion can be exercised manually if needed

## Command Checks

1. Run the focused command tests for completion behavior:

```bash
go test ./cmd -run 'Test.*Completion' -count=1
```

2. If the implementation adds or renames specific completion regression tests, run the targeted set covering:

```bash
go test ./cmd -run 'TestRoot.*Completion|Test.*Nested.*Completion|Test.*Flag.*Completion' -count=1
```

3. Regenerate CLI docs if public help text changed:

```bash
make docs
```

4. Run the repository-wide validation required before commit:

```bash
make test
```

## Manual Smoke Check

1. Generate and install the shell completion for a local terminal session, for example with zsh:

```bash
c8volt completion zsh > "${fpath[1]}/_c8volt"
```

2. Open a fresh shell session and trigger completion for representative paths:

```bash
c8volt <TAB>
c8volt get <TAB>
c8volt walk process-instance --mode <TAB>
```

Expected result:

- Suggestions remain readable and aligned with the prompt
- Internal completion helpers such as `__complete` are not shown as normal user-facing suggestions
- Candidate descriptions remain concise where available
- Full help or usage text does not replace the normal suggestion list
