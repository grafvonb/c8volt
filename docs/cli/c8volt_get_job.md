---
title: "c8volt get job"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get job

Inspect a job by key

### Synopsis

Inspect a Camunda job by key.

Use the jobKey exposed by incident-aware process-instance output to inspect the matching runtime job directly. Use --json for the stable job payload, or --error-message-limit to shorten long error messages. Getting jobs by key is supported for Camunda 8.8 and 8.9; Camunda 8.7 returns an unsupported-version error.

```
c8volt get job [flags]
```

### Examples

```
  ./c8volt get job --key 2251799813711967
  ./c8volt --json get job --key 2251799813711967
```

### Options

```
      --error-message-limit int   maximum characters to show for error messages; 0 keeps full messages
  -h, --help                      help for job
      --key string                job key to inspect
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

* [c8volt get](c8volt_get)	 - Inspect cluster, process, incident, tenant, and resource state

