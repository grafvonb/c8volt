# Data Model: Process Instance Incident List Output

## Process Instance Result

Represents one process-instance row selected by keyed lookup or list/search mode.

- **Key**: process-instance key used for row identity and incident association.
- **Tenant/Definition/Version/State/Start**: existing row fields rendered by default process-instance output.
- **Incident Marker**: existing boolean marker rendered as `inc!` in human rows when the instance has an incident marker.
- **Selection Context**: keyed lookup, list/search filters, paging, and `--limit` determine which results are included before incident details are attached.

## Direct Incident

Represents an incident detail returned directly for one process instance.

- **Incident Key**: stable incident identifier rendered in human output.
- **Process Instance Key**: association back to the owning process instance.
- **Error Message**: full incident message preserved in data and JSON.
- **Rendered Message**: human-only message after optional truncation.

## Indirect Incident Marker

Represents a process instance where the row is marked with an incident but direct incident lookup returns no direct incidents.

- **Owning Process Instance Key**: the row that receives a short indented note.
- **Note**: short human line that says no direct incidents were found for the row.
- **List Warning**: one de-duplicated warning after the list explaining the process-instance tree inspection path.

## Incident Message Limit

Represents the optional human-output truncation setting.

- **Value**: non-negative integer character limit.
- **Default**: `0`, meaning unlimited.
- **Scope**: applies only to human incident error messages.
- **Validation**: invalid without `--with-incidents`; invalid when negative.
- **JSON Behavior**: JSON output always keeps full incident messages.

## Incident Output Association

Represents the invariant that incident details and notes belong to the row directly above them.

- Direct incident lines are rendered immediately below the owning process-instance row.
- Indirect marker notes are rendered immediately below the affected process-instance row.
- The final indirect-marker warning is list-level guidance, not row-specific data.
