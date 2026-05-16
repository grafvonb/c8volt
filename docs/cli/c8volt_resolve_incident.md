---
title: "c8volt resolve incident"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt resolve incident

Resolve incidents by key

### Synopsis

Resolve incidents by key.

The command accepts repeated --key values or newline-separated keys from stdin with '-'. Each unique incident key is submitted for resolution and reported independently.

By default c8volt waits until each incident is no longer active by polling incident lookup through the incident service.

```
c8volt resolve incident [flags]
```

### Examples

```
  ./c8volt resolve incident --key <incident-key>
  ./c8volt resolve inc --key <incident-key> --key <another-incident-key>
  printf '%s\n' "$INCIDENT_KEY_A" "$INCIDENT_KEY_B" | ./c8volt resolve incident -
  printf '%s\n' "$INCIDENT_KEY_A" | ./c8volt resolve inc --key "$INCIDENT_KEY_B" -
```

### Options

```
      --dry-run           preview incident resolutions without submitting mutation
      --fail-fast         stop scheduling new incident resolutions after the first error
  -h, --help              help for incident
  -k, --key strings       incident key(s) to resolve; repeat or combine with stdin '-'
      --no-wait           return after the resolution request is accepted without incident confirmation
      --no-worker-limit   use all queued jobs as workers when --workers is unset
  -w, --workers int       maximum concurrent workers when resolving multiple incidents (default: min(count, 2*GOMAXPROCS, 32))
```

### Options inherited from parent commands

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
      --config string      path to config file
      --debug              enable debug logging
  -j, --json               output as JSON (where applicable)
      --keys-only          output keys only (where applicable)
      --log-level string   log level (debug, info, warn, error) (default "info")
      --no-indicator       disable transient terminal activity indicators
      --profile string     config active profile name to use (e.g. dev, prod)
  -q, --quiet              suppress output except errors
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt resolve](c8volt_resolve)	 - Resolve operational incidents

