---
title: "c8volt"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt

Operate Camunda 8 with guided help and script-safe output modes

### Synopsis

c8volt: Camunda 8 Operations CLI.

Built for Camunda 8 operators and developers who need confirmation, not guesses.
c8volt focuses on operational workflows such as deploying BPMN models, starting process instances,
waiting for state transitions, walking process trees, cancelling safely, and deleting thoroughly.

Start with "c8volt <group> --help" when choosing an operator workflow, or use
"c8volt capabilities --json" when a script, CI job, or AI caller needs the public command inventory,
flag metadata, output modes, mutation behavior, and automation support without scraping prose help.
Human-oriented command families remain the primary interactive surface; JSON and keys-only modes layer onto
the same public Cobra tree for script-safe automation on supported commands.
Prefer --json where a command exposes structured output, and use --automation only when that command's
capabilities entry reports automation:full for the canonical non-interactive contract.

Strict single-resource lookups keep their normal not-found behavior. The newer orphan-parent warning
contract is limited to traversal and dependency-expansion flows such as walk, cancel, and delete when
actionable process-instance data was still resolved.

Tenant-aware process-instance flows use one effective tenant context per command execution.
Supported wrong-tenant lookups resolve as not found. Current process-instance runtime support
is implemented for Camunda 8.7, 8.8, and 8.9 through the repository's versioned service
factories and facades, with the same repository command-family coverage on 8.9 that already
exists on 8.8.

Refer to the documentation at https://c8volt.info for more information.

```
c8volt [flags]
```

### Examples

```
  ./c8volt get --help
  ./c8volt run process-instance --help
  ./c8volt capabilities --json
  ./c8volt --config ./config.yaml config show --validate
```

### Options

```
  -y, --auto-confirm        auto-confirm prompts for non-interactive use
      --automation          enable the canonical non-interactive contract for commands that explicitly support it
      --config string       path to config file
      --debug               enable debug logging, overwrites and is shorthand for --log-level=debug
  -h, --help                help for c8volt
  -j, --json                output as JSON (where applicable)
      --keys-only           output as keys only (where applicable), can be used for piping to other commands
      --log-format string   log format (json, plain, text) (default "plain")
      --log-level string    log level (debug, info, warn, error) (default "info")
      --log-with-source     include source file and line number in logs
      --no-err-codes        suppress error codes in error outputs
      --profile string      config active profile name to use (e.g. dev, prod)
  -q, --quiet               suppress all output, except errors, overrides --log-level
      --tenant string       tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose             adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt cancel](c8volt_cancel)	 - Cancel running work with explicit confirmation semantics
* [c8volt capabilities](c8volt_capabilities)	 - Describe the public CLI contract for automation and discovery
* [c8volt config](c8volt_config)	 - Manage application configuration
* [c8volt delete](c8volt_delete)	 - Delete resources with explicit destructive confirmation
* [c8volt deploy](c8volt_deploy)	 - Deploy state-changing resources such as BPMN definitions
* [c8volt embed](c8volt_embed)	 - Inspect, export, or deploy embedded BPMN resources
* [c8volt expect](c8volt_expect)	 - Wait for verification targets to reach the expected state
* [c8volt get](c8volt_get)	 - Read cluster, process, and resource state without changing it
* [c8volt run](c8volt_run)	 - Start state-changing work such as process instances
* [c8volt version](c8volt_version)	 - Print version information
* [c8volt walk](c8volt_walk)	 - Inspect parent and child relationships for verification follow-up

