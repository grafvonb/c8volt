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

2. Run the targeted set covering the representative top-level, nested, and flag-completion paths:

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

1. Generate the shell completion from the repository checkout, for example with zsh:

```bash
go run . completion zsh > /tmp/_c8volt
```

2. In a fresh shell session, make the generated completion file available to the shell and trigger representative completion paths:

```bash
fpath=(/tmp $fpath)
autoload -Uz compinit && compinit
go run . <TAB>
go run . get <TAB>
go run . walk process-instance --<TAB>
```

Expected result:

- Suggestions remain readable and aligned with the prompt
- Internal completion helpers such as `__complete` are not shown as normal user-facing suggestions
- Candidate descriptions remain concise where available
- Full help or usage text does not replace the normal suggestion list
