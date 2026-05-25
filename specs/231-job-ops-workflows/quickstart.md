# Quickstart: Job Ops Workflow Primitives

## Preconditions

- c8volt is configured for Camunda 8.8 or 8.9.
- Use a test or development cluster with jobs safe to inspect or mutate.
- For Ralph implementation, launch with `--implementation-context specs/ralph-implementation-rules.md`.

## Discover Jobs

```bash
c8volt get job --state FAILED --limit 50
c8volt get job --type payment-worker --limit 20
c8volt get job --pi-key <process-instance-key>
c8volt get job --element-instance-key <element-instance-key>
c8volt get job --element-id <bpmn-element-id>
c8volt get job --worker worker-a
c8volt get job --retries 0
c8volt get job --kind TASK_LISTENER --listener-event-type COMPLETING
```

Expected result: matching jobs are listed using existing output modes. Invalid filters fail locally before remote calls.

## Preserve Existing Keyed Lookup

```bash
c8volt get job --key <job-key>
c8volt --json get job --key <job-key>
```

Expected result: exactly one job is returned, preserving existing fields and JSON behavior.

## Preserve Retry And Timeout Updates

```bash
c8volt update job --key <job-key> --retries 3 --dry-run
c8volt update job --key <job-key> --retries 3 --auto-confirm
c8volt update job --key <job-key> --timeout 5m --auto-confirm
```

Expected result: retry and timeout behavior remains compatible with the current command contract.

## Preview Worker Outcomes

```bash
c8volt update job --key <job-key> --fail --retries 0 --message "upstream unavailable" --dry-run
c8volt update job --key <job-key> --throw-bpmn-error PAYMENT_DECLINED --message "card declined" --dry-run
c8volt update job --key <job-key> --complete --vars '{"approved":true}' --dry-run
```

Expected result: each command renders the planned outcome and submits no mutation.

## Submit Worker Outcomes

```bash
c8volt update job --key <job-key> --fail --retries 2 --retry-backoff 5m --message "upstream unavailable" --auto-confirm
c8volt update job --key <job-key> --throw-bpmn-error PAYMENT_DECLINED --message "card declined" --auto-confirm
c8volt update job --key <job-key> --complete --vars '{"approved":true}' --auto-confirm
```

Expected result: c8volt submits the selected worker outcome through supported Camunda 8.8 or 8.9 APIs and reports the stable command result.

## Validate Unsupported Version

```bash
c8volt --camunda-version 8.7 get job --state FAILED
c8volt --camunda-version 8.7 update job --key <job-key> --complete --auto-confirm
```

Expected result: unsupported capability errors are returned before unsupported mutation paths are used.

## Validation Commands

```bash
go test ./cmd ./c8volt/job ./internal/domain ./internal/services/job ./internal/services/job/v87 ./internal/services/job/v88 ./internal/services/job/v89 -count=1
make docs-content
make test
```

Run targeted tests first during implementation. Run the broader validation before commit readiness.
