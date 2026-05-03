# Quickstart: Preserve Concise CLI Error Breadcrumbs

## Intended Outcome

- CLI failures keep the existing normalized class prefix, such as `resource not found:`.
- Breadcrumb context remains visible and ordered, but duplicated identifiers and repeated failure meaning are removed.
- The same prefix-preserving dedup rule applies to matching non-not-found error classes too.
- Representative regression coverage proves the contract for each affected duplication-pattern family.

## Implementation Starting Points

1. Start at the reported process-instance walk path:
   - [`internal/services/processinstance/walker/walker.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go)
   - [`internal/services/processinstance/v88/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go)
   - [`internal/services/processinstance/v87/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go)
   - [`internal/services/processinstance/v89/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service.go)
2. Identify the deepest layer that should own the root failure detail, then convert outer wrappers to stage-only breadcrumbs where needed.
3. Sweep the other confirmed duplication-pattern families:
   - process-instance mutation and wait flows
   - single-resource fetch commands
   - resource/client orchestration wrappers that bubble into CLI output
   - cluster license/topology failures where service and command layers previously repeated the same fetch-stage wording
4. Keep [`c8volt/ferrors/errors.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go) behavior stable except for any tests needed to prove unchanged classification behavior.

## Verification Focus

1. Confirm the final rendered error keeps the same normalized class prefix as before.
2. Confirm the root failure detail appears once.
3. Confirm breadcrumb labels remain recognizable even when shortened.
4. Confirm representative paths in each affected pattern family no longer repeat the same identifier or failure meaning.
5. Confirm exit-code and classification behavior stay unchanged.

## Suggested Test Order

```bash
go test ./c8volt/ferrors -count=1
go test ./internal/services/processinstance/... -count=1
go test ./cmd -count=1
make test
```

Run the focused suites first so any regression is isolated to shared classification, helper composition, or command rendering before the repository-wide gate.

## Manual Smoke Idea

Reproduce the reported family-style failure after implementation with a wrong or missing process-instance key:

```bash
./c8volt --config /tmp/c8volt.yaml walk pi --key 2251799813720823 --tenant tenant-a
```

Check that the result:

- still starts with the same normalized class prefix
- still shows recognizable breadcrumb stages
- does not repeat `not found` or the same process-instance key unnecessarily

If the audit fixes another non-not-found pattern family, smoke one representative command from that family too and verify the same prefix-preserving dedup rule.

One shipped example is `get cluster topology` against a failing endpoint: the final CLI error should keep `service unavailable:` while mentioning `get cluster topology` only once and never repeating `fetch cluster topology`.
