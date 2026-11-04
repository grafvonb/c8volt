---
title: "c8volt update element"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update element

Update variables at explicit element scopes

### Synopsis

Update variables owned directly by selected runtime elements.

Provide keys and one JSON object through --set-vars or --set-vars-file.
The plan changes only element-local values; parent and sibling scopes are excluded.
Use --dry-run to review additions and changes. Unchanged plans need no writes.
By default, wait until requested values are visible.

```
c8volt update element [flags]
```

### Examples

```
  c8volt update element --key <element-instance-key> --set-vars '{"approved":true}' --dry-run
  c8volt update element --key <element-instance-key> --set-vars-file vars.json
```

### Options

```
      --dry-run                preview exact local changes without mutation
      --fail-fast              stop scheduling new updates after the first error
  -h, --help                   help for element
  -k, --key strings            runtime element key(s); repeat, comma-separate or combine with stdin '-'
      --no-wait                return after accepted writes without local variable confirmation
      --no-worker-limit        use all queued jobs as workers when --workers is unset
      --set-vars string        JSON object with variables to set locally on each element
      --set-vars-file string   path to JSON object file with variables to set locally
  -w, --workers int            maximum concurrent workers; omit for an automatic limit
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

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

