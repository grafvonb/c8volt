---
title: "c8volt resume process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt resume process-instance

Resume suspended process instances by key, business ID or subtree

### Synopsis

Resume suspended process instances (only c8.10+).

Select exact keys or one exact --bid. Add --tree to include suspended descendants;
ancestors and siblings are excluded. The target set is fixed before confirmation.
By default, wait until ACTIVE or immediate COMPLETED is observed.

Use --dry-run to preview. An exact-key preview makes no requests; tree and BID
previews discover scope. Resumption releases buffered work and does not guarantee
incident-free execution. Partial failures do not roll back accepted changes.

```
c8volt resume process-instance [flags]
```

### Examples

```
  c8volt resume process-instance --key <process-instance-key> --dry-run
  c8volt resume process-instance --key <process-instance-key> --tree --dry-run
  c8volt resume process-instance --key <process-instance-key> --tree --auto-confirm
```

### Options

```
  -n, --batch-size int32   maximum process instances per discovery request (default 1000)
      --bid string         exact business ID to select in configured tenant scope (only c8.10+)
      --dry-run            preview exact keys locally or discover eligible tree targets; no mutations or approval
      --fail-fast          stop scheduling new instances after the first error
  -h, --help               help for process-instance
  -k, --key strings        process instance key(s); repeat, comma-separate or combine with stdin '-'
  -l, --limit int32        maximum distinct matching instances; zero selects all; tree expansion is not limited
      --no-wait            return after accepted resumption without confirmation reads
      --no-worker-limit    use all queued jobs as workers when --workers is unset
      --tree               include selected instances and descendants; resume only observed SUSPENDED members
  -w, --workers int        maximum concurrent workers; omit for an automatic limit
```

### Options inherited from parent commands

```
      --all-tenants          search all accessible tenants; cannot combine with --tenant
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
      --tenant string        tenant for discovery and creation; explicit keys use backend authorization
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt resume]({{ "/cli/c8volt_resume" | relative_url }})	 - Resume suspended process instances

