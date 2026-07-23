# Quickstart: Walk PI Elements

## Prerequisites

- A c8volt configuration for a Camunda 8.8 or 8.9 environment with process instances that have runtime element instances.
- A known process-instance key suitable for family, children, parent, and flat walk validation.
- For unsupported-version validation, a Camunda 8.7 test configuration.

## Targeted Validation

Run focused command tests while implementing:

```sh
go test ./cmd -run 'TestWalkProcessInstanceCommand' -count=1
```

Add or update targeted cases covering:

- Help and command capability metadata include `--with-elements`.
- `walk pi --key <key> --with-elements` renders `elements:` under walked process-instance rows.
- `walk pi --key <key> --children --with-elements` enriches selected descendants.
- `walk pi --key <key> --parent --with-elements` enriches ancestry rows in ancestry order.
- `walk pi --key <key> --flat --with-elements` preserves flat path rendering.
- `walk pi --key <key> --with-vars --with-incidents --with-elements` renders sections as `vars:`, `incidents:`, `elements:`.
- `--json walk pi --key <key> --with-elements` preserves traversal metadata and includes per-item `elements`.
- `--keys-only --with-elements` fails before element enrichment.
- Camunda 8.7 returns the existing unsupported element-search capability error.
- Default walk output without `--with-elements` remains unchanged.

## Manual Smoke Scenarios

Inspect a process-instance family with runtime elements:

```sh
./c8volt walk pi --key <process-instance-key> --with-elements
```

Expected outcome:

- The same process-instance rows appear as a normal family walk.
- Rows with runtime elements include an `elements:` section.
- Child process instances are not visually nested under `elements:`.

Inspect descendants:

```sh
./c8volt walk pi --key <process-instance-key> --children --with-elements
```

Expected outcome:

- The selected process instance and descendants are shown in descendant order.
- Elements are attached only to their owning process-instance rows.

Inspect ancestry:

```sh
./c8volt walk pi --key <process-instance-key> --parent --with-elements
```

Expected outcome:

- Ancestry order is preserved.
- Element details appear under matching rows.

Inspect flat family output with all enrichments:

```sh
./c8volt walk pi --key <process-instance-key> --flat --with-vars --with-incidents --with-elements
```

Expected outcome:

- Flat path separators remain readable.
- Detail sections appear in the order `vars:`, `incidents:`, `elements:`.

Inspect JSON output:

```sh
./c8volt --json walk pi --key <process-instance-key> --with-elements
```

Expected outcome:

- The JSON payload includes traversal fields such as `mode`, `outcome`, `rootKey`, `keys`, and `edges`.
- Each walked item includes an `elements` array when element enrichment is requested.

Validate incompatible output mode:

```sh
./c8volt --keys-only walk pi --key <process-instance-key> --with-elements
```

Expected outcome:

- The command fails with a clear validation error before element enrichment.

## Documentation Validation

After implementation updates command behavior or flags:

```sh
make docs-content
```

Confirm README and generated CLI docs describe `walk pi --with-elements` consistently.

## Full Validation

Before commit or merge:

```sh
make test
```
