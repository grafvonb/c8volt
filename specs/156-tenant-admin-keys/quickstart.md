# Quickstart: Tenant Scope For Discovery And Explicit Admin Keys

## Planning And Ralph Context

Every implementation session must start with:

```bash
sed -n '1,240p' specs/ralph-implementation-rules.md
```

Ralph launch instructions must include:

```text
--implementation-context specs/ralph-implementation-rules.md
```

## Targeted Validation

Run targeted tests while implementing each slice:

```bash
go test ./cmd -run 'Test(Get|Cancel|Delete|Expect|Walk).*Tenant|Test.*Direct.*Tenant|Test.*Stdin.*Tenant' -count=1
go test ./c8volt/process ./c8volt/resource -count=1
go test ./internal/services/processinstance ./internal/services/processdefinition ./internal/services/resource -count=1
```

After command help or docs metadata changes:

```bash
make docs-content
go test ./docsgen ./cmd -count=1
```

Before commit:

```bash
make test
```

## Manual Smoke Commands

Use a configured Camunda 8.8 or 8.9 test environment where the backend can return an explicit target whose returned tenant metadata differs from the selected tenant:

```bash
./c8volt --tenant tenant-a get pi --key <tenant-b-process-instance-key> --json
./c8volt --tenant tenant-a walk pi --key <tenant-b-process-instance-key>
./c8volt --tenant tenant-a expect pi --key <tenant-b-process-instance-key> --state active --json
./c8volt --tenant tenant-a cancel pi --key <tenant-b-process-instance-key> --dry-run
./c8volt --tenant tenant-a delete pi --key <tenant-b-process-instance-key> --dry-run
./c8volt --tenant tenant-a get pd --key <tenant-b-process-definition-key> --json
./c8volt --tenant tenant-a get pd --key <tenant-b-process-definition-key> --xml
./c8volt --tenant tenant-a get resource --id <tenant-b-resource-id> --json
```

Then verify:

- Explicit key and ID commands do not fail solely because selected tenant and returned tenant metadata differ.
- Backend authorization and not-found errors still surface normally.
- Search/list commands with `--tenant tenant-a` still discover only tenant-scoped results where supported.
- Search-derived `cancel pi` and `delete pi` previews use the tenant-scoped discovered candidate set.
- Create, deploy, and run flows still preserve `<default>` tenant behavior.
