# Research: Preserve JSON Error Envelopes

## 1. Reuse the existing command error route

**Decision**: Route eligible execution failures through `handleCommandError` in `cmd/cmd_views_contract.go`, preserving error construction at the call site.

**Rationale**: The helper already checks JSON selection and `ContractSupportFull`, renders `resultEnvelopeForError`, and exits using `ferrors.ResolveExitCode`. Its fallback preserves ordinary logging. `resultEnvelopeForError` normalizes the message, selects outcome/class, and records the command path; the renderer attaches existing tenant context. Reimplementing these operations risks divergent semantics.

**Alternatives considered**: A new error framework, a global exit interceptor, and a root-level catch-all would broaden scope and affect bootstrap/parsing behavior. Returning errors through a redesigned command tree is unnecessary.

## 2. Audit scope

**Decision**: Correct the following execution paths. This inventory was checked against command declarations, not inferred from the presence of a JSON flag.

| Source | Execution failure | Disposition |
| --- | --- | --- |
| `cmd/delete_processinstance.go` | Search-flag validation | Route existing error through command handler |
| `cmd/get_cluster_topology.go` | Topology retrieval | Route existing wrapped error |
| `cmd/get_cluster_version.go` | Version retrieval | Route existing wrapped error |
| `cmd/get_cluster_license.go` | License retrieval | Route existing wrapped error |
| `cmd/cmd_cli.go` | Both stdin-key validation errors | Supply command context and route existing errors |
| `cmd/get_processdefinition.go` | XML option validation, by-key retrieval, selector lookup, paged search | Route existing errors; JSON/XML incompatibility remains invalid |
| `cmd/embed_list.go` | Embedded listing failure and no files for configured version | Route existing errors, retaining local-precondition classification for no files |

**Rationale**: Each command above already declares full contract support. The issue explicitly requests inspecting other direct exits, so the process-definition and embedded-list paths belong to the same correction.

**Alternatives considered**: Limiting work to the issue's examples leaves identical advertised-contract omissions. Replacing every direct exit would change non-full and bootstrap behavior.

### Excluded or unreachable paths

- Leave config/bootstrap handling, including context loading in `embed list`, unchanged.
- Leave direct handlers in non-full `config_*`, `embed deploy`, and `embed export` paths unchanged.
- Preserve `cmd_views_contract.go`'s direct fallback.
- Raw XML retrieval/write failures in `runGetProcessDefinitionXML` cannot be reached with JSON: XML validation rejects it first. Preserve these raw-output handlers.
- By-key/list process-definition renderers and embedded-list JSON renderer currently return nil; their defensive error branches cannot reproduce this omission today. Leave these branches unchanged rather than introduce output-write error semantics or artificial tests for them.
- `embed list`'s no-files error remains a failed local precondition, not a new successful empty result.

## 3. Shared stdin validation

**Decision**: Add `cmd *cobra.Command` as the first parameter of `mergeAndValidateKeys`, and update all 12 existing callers to pass their actual command. Keep the return type and caller-specific `.Unique()` choices.

| Command | Caller source |
| --- | --- |
| `get incident` | `cmd/get_incident.go` |
| `get process-instance` | `cmd/get_processinstance.go` |
| `expect process-instance` | `cmd/expect_processinstance.go` |
| `update process-instance` | `cmd/update_processinstance.go` |
| `cancel process-instance` | `cmd/cancel_processinstance.go` |
| `delete process-instance` | `cmd/delete_processinstance.go` |
| `resolve process-instance` | `cmd/resolve_processinstance.go` |
| `delete process-definition` | `cmd/delete_processdefinition.go` |
| `resolve incident` | `cmd/resolve_incident.go` |
| `ops analyse slow-process-instances` | `cmd/ops_analyse_slow_process_instances.go` |
| `ops repair incident` | `cmd/ops_repair_incident.go` |
| `ops repair process-instance` | `cmd/ops_repair_processinstance.go` |

**Rationale**: Every current caller is full-contract, but the existing eligibility gate also preserves fallback for future limited/unsupported callers. Passing context is the smallest change. The special `filter: ` error must terminate before the generic error, including with exit-code suppression. Generic indexes are zero-based indexes of processed stdin keys after blank lines are removed, not physical file line numbers.

**Alternatives considered**: Returning `(keys, error)` would require repeated handling at every caller. A global command pointer is unsafe. Supplying only a boolean loses command identity and contract metadata.

## 4. Validation strategy and hard-to-trigger paths

**Decision**: Use the existing subprocess execution model, `testx.RunCmdSubprocessInDirWithSeparateOutputs`, test config helpers, and local HTTP fixtures. Test real command wiring and process exit behavior; do not rely on view-only tests. For embedded listing failures, extract a small command-local runner accepting an explicit listing function, with production passing `embedded.List`. Subprocess tests invoke the same runner with an error-returning or empty function and a configured command context. Retain actual command success tests to verify production wiring.

**Rationale**: Error handlers call `os.Exit`, so in-process error tests cannot prove termination and code suppression. Separate streams expose diagnostics hidden by combined-output helpers. Embedded assets are compiled in, making missing assets and filesystem traversal failures hard to trigger naturally; a local function parameter provides deterministic coverage without global state or changing the embedded package.

**Alternatives considered**: Mutating global embedded filesystem state, requiring corrupted builds, or adding a new public injection interface would add risk. Real clusters add credentials, latency, and mutation hazards without improving command-handler coverage. No new dependency is required.

## 5. Compatibility and documentation

**Decision**: Preserve output precedence (JSON before keys-only), quiet/automation eligibility, all error wrappers, and all success paths. Update README error-contract guidance and relevant command help metadata, then regenerate references with `make docs-content` during implementation.

**Rationale**: `pickMode` and the existing contract helper already enforce these behaviors. `--no-err-codes` changes only exit status, never outcome or termination. The constitution requires documentation in the same implementation unit for user-visible behavior changes.

**Alternatives considered**: Expanding contract annotations or blanket promises about pre-execution errors would misrepresent the feature. Hand-editing generated references is prohibited.

## Resolution

All planning unknowns are resolved from repository sources. No external technology choice, new dependency, schema migration, or user clarification is required. Research includes an independent agent audit of direct exits and caller eligibility. Constitution gates passed before research; the Phase 1 plan records the post-design check.
