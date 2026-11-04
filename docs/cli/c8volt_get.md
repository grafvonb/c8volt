---
title: "c8volt get"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get

Inspect cluster, process, job, element, incident, tenant, and resource state

### Synopsis

Inspect cluster health, deployed definitions, process instances, tasks, jobs,
elements, and incidents without changing them.

Choose a resource command to look up a key or search with filters.

```
c8volt get [flags]
```

### Examples

```
  c8volt get cluster topology
  c8volt get process-instance --state active --limit 10
  c8volt get incident --state active --limit 10
```

### Options

```
  -h, --help   help for get
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

* [c8volt]({{ "/cli/c8volt/" | relative_url }})	 - Operate Camunda 8 workflows from the command line
* [c8volt get cluster]({{ "/cli/c8volt_get_cluster" | relative_url }})	 - Inspect cluster-wide topology, version, and license information
* [c8volt get element]({{ "/cli/c8volt_get_element" | relative_url }})	 - List or fetch runtime element instances
* [c8volt get incident]({{ "/cli/c8volt_get_incident" | relative_url }})	 - List or fetch incidents
* [c8volt get job]({{ "/cli/c8volt_get_job" | relative_url }})	 - Inspect or search jobs
* [c8volt get process-definition]({{ "/cli/c8volt_get_process-definition" | relative_url }})	 - List or fetch deployed process definitions
* [c8volt get process-instance]({{ "/cli/c8volt_get_process-instance" | relative_url }})	 - List or fetch process instances
* [c8volt get resource]({{ "/cli/c8volt_get_resource" | relative_url }})	 - Get a resource by key
* [c8volt get tenant]({{ "/cli/c8volt_get_tenant" | relative_url }})	 - List tenants
* [c8volt get user-task]({{ "/cli/c8volt_get_user-task" | relative_url }})	 - Fetch or search native user tasks

