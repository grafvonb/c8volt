---
title: "c8volt get process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-definition

List or fetch deployed process definitions

### Synopsis

Inspect deployed process definitions. Use --latest for the newest version of
each tenant and BPMN process ID, or --key for one exact definition.

Add --with-stats for instance and incident counts. Lifecycle state and suspended
counts are available only c8.10+. DRAINING blocks new starts while unfinished
instances remain; active and suspended counts are separate.

Use --watch to refresh until interrupted, or --xml to retrieve the BPMN model.
Watch requires human output and cannot be combined with --json, --keys-only,
--xml, --quiet, or --automation.

```
c8volt get process-definition [flags]
```

### Examples

```
  c8volt get process-definition --latest
  c8volt get process-definition --bpmn-process-id order --with-stats
  c8volt get process-definition --bpmn-process-id order --latest --watch
```

### Options

```
  -n, --batch-size int32          process definitions per discovery page (server maximum 1000) (default 1000)
  -b, --bpmn-process-id string    BPMN process ID to filter process instances
  -h, --help                      help for process-definition
  -k, --key string                process definition key to fetch
      --latest                    only include the latest matching process-definition version per exact tenant/BPMN process group
      --pd-version int32          process definition version
      --pd-version-tag string     process definition version tag
      --watch                     repeat the process-definition lookup until stopped
      --watch-interval duration   interval between process-definition watch refreshes after the immediate first refresh (default 1s)
      --watch-timeout duration    maximum watch session duration (0 = unlimited; requires --watch)
      --with-stats                include process definition statistics, incident counts, and suspended counts when supported
      --xml                       output the selected process definition as raw XML (requires --key and no other filters)
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

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

