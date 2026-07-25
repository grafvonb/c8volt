// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestOpsExecuteFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "ops execute", []string{
		"ops execute",
		"ops execute retention-policy",
		"ops execute smoke-test",
	})
}

// appendOpsExecuteCommandGapProposals records setup gaps for retention and incident/job-state workflows.
func appendOpsExecuteCommandGapProposals(proposals []proposalRecord) []proposalRecord {
	proposals = registerDirectCamundaSetupFallback(proposals,
		"ended process instances old enough for retention-policy selection",
		"ops execute retention-policy aged data",
		[]string{"ops execute retention-policy", "delete process-instance"},
		supportedProposalVersions(),
		"Operators can prepare aged disposable data through c8volt before validating retention execution.",
	)
	return registerDirectCamundaSetupFallback(proposals,
		"process instances with active incidents and controllable job states",
		"ops execute smoke-test incident and job-state coverage",
		[]string{"ops execute smoke-test", "get incident", "get job", "ops repair incident"},
		supportedProposalVersions(),
		"Smoke-test coverage can validate incident and job-state paths without direct API seeding.",
	)
}

// appendOpsExecuteEmbeddedBPMNGapProposals records missing fixtures for retention and incident/job-state setup.
func appendOpsExecuteEmbeddedBPMNGapProposals(proposals []proposalRecord) []proposalRecord {
	proposals = registerMissingEmbeddedBPMNProposal(proposals,
		"process model that can complete and age deterministically for retention tests",
		"retention-policy aged process-instance coverage",
		[]string{"ops execute retention-policy"},
		supportedProposalVersions(),
		"Retention-policy validation can use embedded BPMN with predictable lifecycle state.",
	)
	return registerMissingEmbeddedBPMNProposal(proposals,
		"process model that can create observable incidents and jobs on demand",
		"incident and job-state workflow coverage",
		[]string{"ops execute smoke-test", "get incident", "get job", "ops repair incident"},
		supportedProposalVersions(),
		"Incident and job-state scenarios can be seeded through embedded fixtures instead of direct API setup.",
	)
}
