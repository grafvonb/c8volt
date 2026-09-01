# Quickstart: Implement and Verify Semantic Progress

## 1. Lock the feature context

Work on branch `285-semantic-progress-milestones` and read, in order:

1. `specs/285-semantic-progress-milestones/spec.md`
2. `specs/285-semantic-progress-milestones/plan.md`
3. `specs/285-semantic-progress-milestones/research.md`
4. `specs/285-semantic-progress-milestones/data-model.md`
5. `specs/285-semantic-progress-milestones/contracts/semantic-progress-contract.md`
6. `specs/ralph-implementation-rules.md`

Ralph runs must include:

```text
--implementation-context specs/ralph-implementation-rules.md
```

## 2. Build the shared vertical slice first

Implement and test this path before adding command families:

```text
service worker completion
  -> internal/domain completion fact
  -> facade conversion
  -> command semantic reporter
  -> workflow activity + diagnostic stderr
```

Use an injected clock. Do not wait 10 real seconds in tests and do not add a ticker.

## 3. Prove the reporter contract

Cover these deterministic cases before command integration:

- clean completion before 10 seconds: transient update only, no durable line;
- first completion at or after 10 seconds: one compact aggregate line;
- rapid later completions: no flood;
- immediate failure: warning on the same callback;
- activated scope with later progress: one idempotent final flush;
- verbose: one identity/outcome line per item and no paced aggregate;
- quiet: failure warnings only;
- automation, JSON, keys-only: no human progress;
- affected pointer-to-zero versus unavailable;
- concurrent out-of-order facts under `-race`.

Suggested focused run:

```bash
go test ./internal/domain ./c8volt/foptions ./c8volt/ops ./toolx/logging -race -count=1
go test ./cmd -run 'Progress|Activity' -race -count=1
```

## 4. Add command families in pragmatic slices

1. Process-instance cancel/delete across direct, stdin, and search; include force and no-wait.
2. Basic and all-process-definition deletion, then deployment visibility.
3. Retention, orphan, incident-selected purge, incident repair, and process-instance repair.
4. Smoke-test deploy/start/walk/cleanup stages.
5. Bulk start, slow analysis, and multi-key expect; retain transient-only behavior for the assessed exclusions in `research.md`.

For destructive commands, stop planning activity before prompting and create the reporter only for the confirmed mutation scope.

## 5. Validate each slice

Run the closest service and command tests first. Typical commands are:

```bash
go test ./internal/services/processinstance/... -run 'Progress|Cancel|Delete' -race -count=1
go test ./internal/services/processdefinition/... -run 'Progress|Delete' -race -count=1
go test ./internal/services/ops/... -run 'Progress|Purge|Repair|Smoke' -race -count=1
go test ./cmd -run 'Progress|CancelProcessInstance|DeleteProcessInstance|Purge|Repair|Smoke' -race -count=1
```

Confirm that existing result assertions remain unchanged. New assertions should target diagnostic stderr/activity capture, not result stdout.

## 6. Finish documentation and full validation

Update command help and README guidance for the visible progress behavior, then regenerate CLI documentation:

```bash
make docs-content
git diff --check
make test
```

Do not hand-edit generated files under `docs/cli/`. Conventional commit subjects for this feature end with `#285`.
