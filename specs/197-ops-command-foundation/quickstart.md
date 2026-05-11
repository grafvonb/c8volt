# Quickstart: Ops Command Foundation

## Manual Verification

1. Verify top-level help includes the ops command family:

   ```sh
   ./c8volt ops --help
   ```

   Expected: Help describes high-level operational workflows and exits successfully without requiring Camunda configuration.

2. Verify execute grouping help:

   ```sh
   ./c8volt ops execute --help
   ```

   Expected: Help describes predefined operational playbooks and does not execute any workflow.

3. Verify repair grouping help:

   ```sh
   ./c8volt ops repair --help
   ```

   Expected: Help describes repair/remediation workflows, does not execute remediation, and does not advertise an ambiguous top-level `--key`.

4. Verify discovery metadata:

   ```sh
   ./c8volt capabilities --json
   ```

   Expected: JSON remains valid and includes discoverable ops command metadata without breaking existing command entries.

5. Regenerate generated CLI documentation after source command changes:

   ```sh
   make docs-content
   ```

   Expected: Generated docs include the ops command family and no generated docs were hand-edited.

## Automated Validation Targets

Run targeted tests first:

```sh
go test ./cmd -run 'Test.*Ops|TestCapability.*Ops|TestRootHelp' -count=1
```

Before completing the feature, run broader validation appropriate to the touched files. The repository's full test target is:

```sh
make test
```

## Ralph Launch Requirement

Any Ralph run for this feature must include:

```sh
--implementation-context specs/ralph-implementation-rules.md
```
