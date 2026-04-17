# Data Model: Preserve Concise CLI Error Breadcrumbs

## Shared Error Class Prefix

- **Purpose**: Represents the normalized class prefix rendered by `c8volt/ferrors`, such as `resource not found:` or `unsupported capability:`.
- **Key attributes**:
  - Class sentinel
  - Rendered prefix text
  - Exit-code mapping
- **Invariants**:
  - The prefix is owned by `ferrors` normalization.
  - This feature must not change class selection or exit-code resolution.

## Breadcrumb Segment

- **Purpose**: Represents one contextual stage label in the composed user-facing error path.
- **Key attributes**:
  - Stage name
  - Optional resource or operation key reference
  - Source layer: command, helper, service, or client wrapper
  - Whether the label was preserved verbatim or shortened equivalently
- **Invariants**:
  - Breadcrumbs must remain ordered from outermost stage to innermost stage.
  - A shortened breadcrumb must preserve the same stage meaning as the original label.
  - Breadcrumbs may add context, but must not restate the root failure meaning.

## Root Failure Detail

- **Purpose**: Represents the deepest specific failure detail that explains the underlying resource or operation problem.
- **Key attributes**:
  - Failure subject, such as resource key or operation target
  - Failure meaning, such as not found or unsupported condition
  - Producing layer
- **Invariants**:
  - The final rendered error should include the root failure detail once.
  - Upper layers must not repeat the same identifier or failure meaning when the root detail already contains it.

## Error Composition Chain

- **Purpose**: Represents the full ordered composition of a final CLI error from the normalized prefix through breadcrumbs to the root failure detail.
- **Fields**:
  - Shared error class prefix
  - Ordered breadcrumb segments
  - Root failure detail
  - Final rendered string
  - Shared helper owner for classification and exit behavior
- **Behavioral rules**:
  - The shared class prefix appears once at the start.
  - Breadcrumbs remain concise and stage-identifying.
  - The rendered string must not contain redundant repetitions of the same root meaning.
  - Shared helpers may classify and render the chain, but they must not rewrite the breadcrumb or root-detail content.

## Duplication-Pattern Family

- **Purpose**: Groups error paths that share the same duplication shape and can therefore share one representative regression strategy.
- **Current family variants**:
  - Process-instance lookup and traversal
  - Process-instance mutation and wait follow-up
  - Single-resource command fetch wrappers
  - Resource/client orchestration wrappers
  - Cluster license/topology fetch wrappers for unavailable and malformed-response failures
- **Invariants**:
  - Each affected family needs at least one representative regression anchor.
  - Fixes should be applied consistently across all paths in a family once the pattern is confirmed.

## Rendering Contract Decision

- **Purpose**: Captures the clarified user-facing rules that constrain implementation.
- **Fields**:
  - Scope rule: audit every duplicated CLI error path found during investigation
  - Prefix rule: preserve the existing shared class prefix
  - Cross-class rule: apply the same dedup contract to other shared classes when the pattern matches
  - Breadcrumb rule: shortening allowed only when meaning remains equivalent
  - Helper-boundary rule: `ferrors` and command/bootstrap normalization helpers keep classification and exit behavior stable and do not perform string deduplication
  - Test rule: representative regression per affected pattern family
- **Invariants**:
  - Implementation must not silently narrow scope back to only the original walk path.
  - Any change to these rules requires updating the feature spec and plan before code changes continue.
