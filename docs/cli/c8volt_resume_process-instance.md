---
title: "c8volt resume process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt resume process-instance

Resume suspended process instances by key or subtree

### Synopsis

Resume SUSPENDED process instances on Camunda 8.10 by explicit key or selected subtree.

Provide repeated or comma-separated --key values, or newline-separated keys from stdin with '-'. Keys are deduplicated in first-input order. Without --tree, each selected parent or child is independent: relatives are neither discovered nor resumed. Explicit keys use backend authorization without discovery tenant filtering.

When authorization is enabled, resumption requires SUSPEND_PROCESS_INSTANCE on PROCESS_DEFINITION. Default confirmation also requires permission to read the instance. Exact-key execution has no mandatory preflight read; only the submission response establishes acceptance. Active, terminal, terminating, missing or unauthorized targets remain failures when rejected.

Default execution waits for observed ACTIVE or immediate COMPLETED after acceptance within the configured backoff timeout (two minutes by default, adjustable with --backoff-timeout). --timeout bounds each HTTP request. --no-wait returns accepted submission without confirmation reads. Partial failures retain acceptance and observation evidence with a nonzero exit. Without --tree, --dry-run previews exact keys without reads, writes or confirmation prompts; eligibility is not checked, including with --no-wait.

Add --tree to include each selected instance and its descendants, merging overlapping subtrees without escalating to ancestors or siblings. Discovery traverses active and terminal parents before selecting only observed SUSPENDED members; independently suspended descendants remain eligible. All other states are excluded. Tree execution and tree dry-run require read/search permissions. Tree dry-run reports selected, discovered, eligible and excluded counts without mutation. Failed or incomplete planning stops before approval or submission. A complete plan with no suspended targets succeeds without approval, activity or mutation, including with --no-wait.

Tree execution asks once for the expanded eligible count and uses that frozen scope without rediscovery. Discovery retains the limit of up to 1,000 direct children per parent lookup; coverage beyond that limit is not guaranteed. The scope is not atomic: state races can cause authoritative rejection, and descendants created after discovery may be omitted. There is no automatic rollback or parent-before-child execution guarantee. --no-wait skips confirmation reads, not tree discovery.

Resumption releases buffered work. Failures while processing buffered work can leave the instance suspended. Observed CANCELED or TERMINATED is a cancellation failure, never confirmed resumption. Persistent suspension, absence, unknown state or failed reads remain unconfirmed within the wait bound; missing suspendedDate alone is not proof. Resumption does not guarantee incident-free execution, process completion, or immediate subscription readiness. Multiple-key execution is not atomic and partial failure has no automatic rollback.

Use --auto-confirm or --automation for unattended execution. JSON requires --dry-run, --auto-confirm or --automation and cannot be combined with --verbose. Keys-only previews list selected exact keys or eligible tree targets; execution keys indicate acceptance, not confirmed resumption, even when confirmation fails. Use JSON and the exit status for confirmation evidence. Use --verbose for per-instance submission, confirmation, and last observed state. Quiet preserves explicit machine results and required errors. Use --debug for redacted HTTP diagnostics independently of --verbose.

```
c8volt resume process-instance [flags]
```

### Examples

```
  ./c8volt resume process-instance --key <process-instance-key>
  ./c8volt resume pi --key <key-a>,<key-b> --key <key-a> --dry-run --json
  ./c8volt resume pi --key <process-instance-key> --auto-confirm --no-wait --json
  ./c8volt resume pi --key <process-instance-key> --tree --dry-run
  ./c8volt resume pi --key <parent-key>,<child-key> --tree --auto-confirm --json
  printf '%s\n' "$INSTANCE_A" "$INSTANCE_B" | ./c8volt resume pi --auto-confirm -
```

### Options

```
      --dry-run           preview exact keys locally or discover eligible tree targets; no mutations or approval
      --fail-fast         stop scheduling new instances after the first error
  -h, --help              help for process-instance
  -k, --key strings       process instance key(s); repeat, comma-separate or combine with stdin '-'
      --no-wait           return after accepted resumption without confirmation reads
      --no-worker-limit   use all queued jobs as workers when --workers is unset
      --tree              include selected instances and descendants; resume only observed SUSPENDED members
  -w, --workers int       maximum concurrent workers (default: min(count, 2*GOMAXPROCS, 32))
```

### Options inherited from parent commands

```
      --all-tenants          clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
  -y, --auto-confirm         auto-confirm prompts for non-interactive use
      --automation           enable non-interactive mode for commands that explicitly support it
      --config string        path to config file
      --debug                enable debug logging
  -j, --json                 output as JSON (where applicable)
      --keys-only            output keys only (where applicable)
      --log-level string     log level (debug, info, warn, error) (default "info")
      --no-indicator         disable transient terminal activity indicators
      --no-version-check     skip the automatic advisory gateway version check
      --profile string       config active profile name to use (e.g. dev, prod)
  -q, --quiet                suppress output except errors
      --request-rate float   maximum shared request starts per second (fractions allowed; 0 disables pacing)
      --tenant string        tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt resume]({{ "/cli/c8volt_resume" | relative_url }})	 - Resume suspended process instances

