---
title: "Camunda CLI Options"
permalink: /camunda-cli/
nav_order: 4
has_toc: false
description: "Choose a command-line tool for Camunda operations or application integration."
---

# Camunda CLI options

`c8volt` is an independent operations CLI. Its v5 documentation targets Camunda 8.10; 8.9 and 8.8 remain supported.

| Need | Starting point |
| --- | --- |
| Repair incidents, preview bulk changes, or clean up history | [c8volt ops playbooks](/ops/) |
| Inspect resources or automate an operational task | [c8volt CLI reference](/cli/) |
| Use Camunda's own command-line tooling | [Camunda c8ctl documentation](https://docs.camunda.io/docs/apis-tools/c8ctl/getting-started/) |
| Build a long-running application integration | [Camunda APIs and tools](https://docs.camunda.io/docs/apis-tools/) |

For existing `zbctl` scripts, review their required APIs before migrating. c8volt v5 uses its own command and flag grammar; it is not a drop-in replacement.

Start with the [quick start](/#fast-start) and [configuration](/configuration/). Use `c8volt capabilities --json` to inspect supported commands, flags, output modes, and automation support.
