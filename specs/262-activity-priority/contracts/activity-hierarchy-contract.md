# Contract: Activity Hierarchy

## Purpose

Define the observable contract for transient terminal activity so operators see the highest useful active context during nested Camunda work.

## Activity Importance Order

```text
workflow > batch > wait > http
```

## Required Behavior

- When a workflow activity is active, lower-importance activity MUST NOT replace the visible activity message.
- When no workflow activity is active, batch activity MAY be visible.
- When no workflow or batch activity is active, wait activity MAY be visible.
- When no workflow, batch, or wait activity is active, HTTP fallback activity MAY be visible.
- When the visible activity scope stops, the display MUST fall back to the highest-importance remaining active scope.
- When an equal-importance scope starts, the newest equal-importance scope MAY become visible.
- Activity writes MUST clear visible spinner text before durable output is written.
- Disabled, quiet, automation, non-interactive, JSON, and keys-only modes MUST remain free of transient activity output according to existing output-mode rules.

## Representative Examples

### Process-Instance Delete

```text
active workflow: deleting process-instance trees 48/800
nested wait: waiting for pi 123 state
nested http: loading process instance
visible: deleting process-instance trees 48/800
```

### Simple Lookup

```text
active workflow: none
nested http: loading process instance
visible: loading process instance
```

### Workflow Completes Before Wait

```text
workflow stops
wait remains active: waiting for 8 pi state
visible: waiting for 8 pi state
```

## Non-Goals

- This contract does not define durable `INFO`, `WARN`, prompt, debug, or final outcome wording.
- This contract does not require additional Camunda requests.
- This contract does not require every command to create a bespoke activity message.
