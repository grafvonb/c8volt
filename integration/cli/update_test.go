// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestUpdateFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "update", []string{
		"update",
		"update job",
		"update process-instance",
	})
	runBehavioralCoverageScenarios(t, "update")
}

// appendUpdateCommandGapProposals records setup gaps for job BPMN errors and richer variable shapes.
func appendUpdateCommandGapProposals(proposals []proposalRecord) []proposalRecord {
	proposals = registerDirectCamundaSetupFallback(proposals,
		"active job that can throw a modeled BPMN error",
		"update job --throw-bpmn-error",
		[]string{"update job"},
		supportedProposalVersions(),
		"Operators can prepare BPMN error repair/update scenarios through c8volt commands.",
	)
	return registerDirectCamundaSetupFallback(proposals,
		"process instance carrying nested object, array, boolean, numeric, and null variables",
		"update process-instance variable-shape coverage",
		[]string{"update process-instance", "get process-instance"},
		supportedProposalVersions(),
		"Operators can create representative variable payloads without direct API setup.",
	)
}

// appendUpdateEmbeddedBPMNGapProposals records missing fixtures for update-oriented product states.
func appendUpdateEmbeddedBPMNGapProposals(proposals []proposalRecord) []proposalRecord {
	proposals = registerMissingEmbeddedBPMNProposal(proposals,
		"service task model with a boundary BPMN error catch path",
		"BPMN error job coverage",
		[]string{"update job"},
		supportedProposalVersions(),
		"Maintainers can exercise BPMN error update flows using an embedded model.",
	)
	return registerMissingEmbeddedBPMNProposal(proposals,
		"process model that initializes representative scalar and structured variables",
		"variable-shape process-instance coverage",
		[]string{"update process-instance", "get process-instance"},
		supportedProposalVersions(),
		"Variable rendering and mutation coverage can use stable embedded data shapes.",
	)
}
