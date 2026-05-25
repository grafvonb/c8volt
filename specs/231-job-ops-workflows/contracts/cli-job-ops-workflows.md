# CLI Contract: Job Ops Workflow Primitives

## `c8volt get job`

### Keyed Lookup

```bash
c8volt get job --key <job-key>
```

**Behavior**:

- Returns exactly one job matching `<job-key>`.
- Preserves existing human, JSON, and error-message-limit behavior.
- Rejects list/search-only filters when `--key` is present.

### List/Search Mode

```bash
c8volt get job --state failed
c8volt get job --type payment-worker
c8volt get job --pi-key <process-instance-key>
c8volt get job --element-instance-key <element-instance-key>
c8volt get job --element-id <bpmn-element-id>
c8volt get job --worker worker-a
c8volt get job --retries 0
c8volt get job --kind bpmn_element
c8volt get job --kind task_listener --listener-event-type completing
c8volt get job --limit 50
```

**Behavior**:

- Enters list/search mode when `--key` is omitted.
- Applies supported filters through the job service.
- Accepts enum-style filters (`--state`, `--kind`, and `--listener-event-type`) case-insensitively while sending Camunda's canonical uppercase values to the API.
- Uses `element-id` and `element-instance-key` flag names.
- Does not introduce `flow-node`, `flow-node-id`, `fni`, or `fni-key` job aliases.
- Supports default human rows, JSON rows, keys-only output, and documented limit behavior.

**Validation**:

- invalid enum values, numeric keys, retry counts, or limits fail before remote calls.
- keyed lookup and list/search filters are mutually exclusive.

## `c8volt update job` Existing Update Mode

```bash
c8volt update job --key <job-key> --retries <n>
c8volt update job --key <job-key> --timeout <duration>
```

**Behavior**:

- Preserves current retry and timeout update behavior.
- Retry updates keep dry-run, confirmation, no-op detection, JSON guardrails, automation behavior, and no-wait behavior.
- Timeout updates keep submitted/accepted semantics without deadline equality confirmation.

## `c8volt update job --fail`

```bash
c8volt update job --key <job-key> --fail \
  --retries 0 \
  --message "no worker available for job type payment-service"

c8volt update job --key <job-key> --fail \
  --retries 2 \
  --retry-backoff 5m \
  --message "upstream unavailable"
```

**Behavior**:

- Submits a technical failure worker outcome.
- Supports retry count, optional retry backoff duration, optional message, and optional variables if implemented through the shared variables parser.
- Uses the same dry-run, confirmation, JSON, automation, and result rendering patterns as existing state-changing job updates.

**Validation**:

- `--fail` is mutually exclusive with `--throw-bpmn-error` and `--complete`.
- retry count must be non-negative.
- retry backoff must be a positive duration when supplied.

## `c8volt update job --throw-bpmn-error`

```bash
c8volt update job --key <job-key> --throw-bpmn-error PAYMENT_DECLINED \
  --message "card declined"
```

**Behavior**:

- Submits a modeled BPMN error worker outcome with the given error code.
- Supports optional message and optional variables if implemented through the shared variables parser.
- Uses the same dry-run, confirmation, JSON, automation, and result rendering patterns as existing state-changing job updates.

**Validation**:

- error code is required and non-empty.
- cannot be combined with `--fail`, `--complete`, `--retries`, or `--timeout`.

## `c8volt update job --complete`

```bash
c8volt update job --key <job-key> --complete \
  --vars '{"approved":true}'
```

**Behavior**:

- Submits a completion worker outcome.
- Completes with no additional variables when `--vars` is omitted.
- Uses the same dry-run, confirmation, JSON, automation, and result rendering patterns as existing state-changing job updates.

**Validation**:

- variable input must be valid JSON object input when supplied.
- cannot be combined with `--fail`, `--throw-bpmn-error`, `--retries`, or `--timeout`.

## Version Contract

- Camunda 8.8 and 8.9 support job search and worker outcomes through generated clients.
- Camunda 8.7 fails before unsupported new search or worker outcome mutation paths.
- Unsupported capability errors must be visible in human output and machine-readable JSON/error contracts.

## Documentation Contract

- README examples must include job list/search and worker outcome examples.
- Generated CLI docs must be regenerated from command metadata.
- Help text must describe version support, mutation safety, JSON guardrails, and unsupported combinations.
