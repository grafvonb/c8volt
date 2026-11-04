---
title: "c8volt resolve incident"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt resolve incident

Resolve incidents by key

### Synopsis

Resolve incidents by key.

Provide repeated --key values or newline-separated keys from stdin with '-'. Each unique incident is resolved once. By default c8volt waits until it is no longer active.

```
c8volt resolve incident [flags]
```

### Examples

```
  c8volt resolve incident --key <incident-key>
  c8volt resolve incident --key <incident-key> --key <another-incident-key>
```

### Options

```
      --dry-run           preview incident resolutions without submitting mutation
      --fail-fast         stop scheduling new incident resolutions after the first error
  -h, --help              help for incident
  -k, --key strings       incident key(s) to resolve; repeat or combine with stdin '-'
      --no-wait           return after the resolution request is accepted without incident confirmation
      --no-worker-limit   use all queued jobs as workers when --workers is unset
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

* [c8volt resolve]({{ "/cli/c8volt_resolve" | relative_url }})	 - Resolve operational incidents

