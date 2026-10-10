# Contract: Supported Camunda Policy

## Selection and errors

- Supported canonical values are exactly `8.8`, `8.9`, and `8.10`; default is `8.9`.
- Retained aliases, normalization, flag/environment/profile/file precedence and maturity labels are unchanged.
- Any unsupported selection fails through existing error classes before remote requests or mutation confirmation; no implicit remapping or dedicated retirement message.
- The retired public constant is removed without a replacement compatibility symbol. Other public facade signatures and output schemas remain unchanged.

## Retained operations

8.8 process-instance deletion retains its Operate request path. 8.8 and 8.9 user-task lookup retain Tasklist fallback after a primary miss, tenant/identity validation and no fallback after primary success or unrelated server failure. Existing restrictions, including full process-definition history deletion requiring 8.9 or newer, remain.

## Output and operational proof

Preserve human wording for retained workflows, JSON envelope shapes and omission/null/empty-array rules, one-key-per-line output, zero-byte empty keys output, quiet/machine-output precedence and configured stderr prompt routing. Preserve dry-run preview semantics, prompt-free empty scopes, automation and auto-confirm behavior, wait/no-wait semantics, and distinction between accepted and confirmed mutations. No additional discovery requests may be introduced by rendering.

## Generation and documentation

Normal generation cannot recreate retired support. Version output, help, README and generated docs describe the same retained policy. Historical planning content contains no obsolete support requirements. No dedicated migration/deprecation document or retired-release test suite is added.
