# Quickstart: Process-Instance Dry Run Scope Preview

## Targeted Development Loop

1. Add dry-run command tests around the current preflight helpers:

   ```bash
   go test ./cmd -run 'Test(Cancel|Delete).*DryRun' -count=1
   ```

2. Add or update facade tests for structured expansion behavior:

   ```bash
   go test ./c8volt/process -run DryRunCancelOrDelete -count=1
   ```

3. Implement `--dry-run` for keyed cancel/delete paths and verify mutation stubs are not called.

4. Extend search-mode dry run and verify paged selection behavior:

   ```bash
   go test ./cmd -run 'Test.*ProcessInstance.*DryRun.*Search|Test.*ProcessInstance.*DryRun.*Paged' -count=1
   ```

5. Add partial orphan-parent regression coverage:

   ```bash
   go test ./cmd ./c8volt/process -run 'DryRun.*Orphan|DryRun.*Partial' -count=1
   ```

6. Refresh docs after command metadata changes:

   ```bash
   make docs-content
   ```

7. Run final validation:

   ```bash
   make test
   ```

## Manual Smoke Examples

Preview cancellation scope for one selected process instance:

```bash
./c8volt cancel pi --key <process-instance-key> --dry-run
```

Preview deletion scope for completed search results:

```bash
./c8volt delete pi --state completed --batch-size 250 --limit 25 --dry-run
```

Inspect structured output:

```bash
./c8volt --json cancel pi --key <process-instance-key> --dry-run
```

Expected result: output reports requested keys, resolved roots, affected family keys, traversal outcome, warning/missing ancestor data when present, and indicates no mutation was submitted.
