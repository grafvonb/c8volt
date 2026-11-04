---
title: "c8volt expect process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt expect process-instance

Wait for process instances to satisfy expectations

### Synopsis

Wait for selected process instances to satisfy state or incident expectations.

Use after a mutation when an observed outcome matters. Repeat --state to accept
any listed state. Suspended-state expectations are available only c8.10+.

```
c8volt expect process-instance [flags]
```

### Examples

```
  c8volt expect process-instance --key <process-instance-key> --state completed
  c8volt expect process-instance --key <process-instance-key> --incident false
```

### Options

```
      --fail-fast         stop scheduling new instances after the first error
  -h, --help              help for process-instance
      --incident string   incident expectation; valid values are: [true, false]
  -k, --key strings       process instance key(s) to watch
      --no-worker-limit   use all queued jobs as workers when --workers is unset
  -s, --state strings     state expectation; valid values are: [active, suspended, completed, canceled, terminated, absent]; suspended: only c8.10+
  -w, --workers int       maximum concurrent workers; omit for an automatic limit
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

* [c8volt expect]({{ "/cli/c8volt_expect" | relative_url }})	 - Wait for process instances to satisfy expectations

