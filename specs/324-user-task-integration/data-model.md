# Data Model: Integration Scenario State

No production models, persisted schemas, or public envelopes change. Reuse existing integration records and public result types.

| Entity | Required data | Rules and relationships |
| --- | --- | --- |
| Profile | name, expected/observed minor, readiness | Existing integrationProfile; prefix matches 8.8/C88, 8.9/C89, 8.10/C810; 8.7 unsupported |
| Definition selection | embedded path, BPMN ID, deployed key/version | Existing fixture/deployment records; matching parent-child family |
| Seed values | run marker, hasIncident=false, incident=99, customer string, structured/Unicode payload | Preserve typed JSON; marker cannot be overridden; no alteration of shared defaults |
| Owned dataset | definition keys, root/child PI keys, discovered task and element-instance keys | Small scenario-local structure only if useful; all task ownership validated |
| Expected scopes | process incident=99; task incident=1; process customer/payload | Effective scopeKey for local values equals element-instance key, process values equal owning PI; no assumption that caller PI remains the variable owner in child |
| Volume baseline | bounded preexisting task keys, seeded task keys, observed matching keys | Keep ownership classes separate; stable expected union; no duplicate seeded keys |
| Evidence | profile/version, command arguments, streams, exit, duration, keys, outcomes | Reuse evidenceRecord and existing family report; retain last readiness observation and assertion failures |

Lifecycle: profile-ready -> deployed -> seeded -> task/variable-ready -> asserted -> reported. Required setup or assertion failure ends in failed evidence; no-profile/prerequisite skips and unavailable versions never enter verified coverage. An expected-unsupported assertion is a verified compatibility outcome, not successful supported-feature execution.

Result decoding uses existing `task.UserTasks` and `task.VariableEnrichedUserTasks`; require one shared command envelope followed by EOF. Preserve initialized collection and null/omission contracts already exercised by existing integration tests.
