# Data Model: Add Process Definition XML Command

## Overview

This feature adds a focused retrieval mode to the existing process-definition CLI flow. The data involved is lightweight and request-oriented rather than persistent.

## Entities

### Process Definition XML Request

- **Purpose**: Represents the operator's request to retrieve one process definition as XML through the CLI.
- **Fields**:
  - `key`: required process definition identifier used to retrieve one definition
  - `xmlRequested`: boolean derived from the `--xml` flag
  - `renderFlags`: inherited output modifiers such as `--json` and `--keys-only`
  - `searchFilters`: existing process-definition filters that remain valid only for non-XML list/detail retrieval
- **Validation rules**:
  - `xmlRequested=true` requires `key` to be present.
  - `xmlRequested=true` cannot be combined with list-style retrieval semantics.
  - `xmlRequested=true` cannot be combined with output modes that conflict with raw XML stdout.

### Process Definition XML Payload

- **Purpose**: The raw BPMN XML content returned by the versioned processdefinition service for a single definition.
- **Fields**:
  - `key`: the process definition key used for retrieval
  - `content`: XML string returned from the supported Camunda API version
- **Validation rules**:
  - Retrieval succeeds only when the service returns a successful response with a non-nil XML payload.
  - The payload is written to stdout without summary decorations in XML mode.

### XML Retrieval Outcome

- **Purpose**: Captures the user-visible result of the XML retrieval workflow.
- **States**:
  - `requested`: CLI accepted the flags and is attempting retrieval
  - `succeeded`: XML payload was retrieved and written to stdout
  - `failed`: retrieval or validation failed and the command returns a non-success exit
- **Validation rules**:
  - `succeeded` must not include list summaries, key-only output, or JSON wrappers.
  - `failed` must preserve the repository's normal error handling and exit-code behavior.

## Relationships

- A `Process Definition XML Request` targets exactly one `Process Definition XML Payload`.
- A `Process Definition XML Request` produces one `XML Retrieval Outcome`.
- Existing list/detail process-definition models remain unchanged for non-XML flows.

## Notes

- No new persistent storage or long-lived domain entity is required.
- The feature should avoid expanding the general rendering system unless later work creates a broader reusable need.
