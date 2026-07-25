// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestWalkFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "walk", []string{
		"walk",
		"walk process-instance",
	})
	runBehavioralCoverageScenarios(t, "walk")
}

// appendWalkCommandGapProposals records listener-state setup that currently needs direct API preparation.
func appendWalkCommandGapProposals(proposals []proposalRecord) []proposalRecord {
	return registerDirectCamundaSetupFallback(proposals,
		"listener job attached to a runtime element",
		"walk process-instance --with-listeners",
		[]string{"walk process-instance", "get element"},
		supportedProposalVersions(),
		"Operators can inspect listener state through c8volt without relying on direct Camunda setup.",
	)
}

// appendWalkEmbeddedBPMNGapProposals records the missing embedded listener-oriented fixture.
func appendWalkEmbeddedBPMNGapProposals(proposals []proposalRecord) []proposalRecord {
	return registerMissingEmbeddedBPMNProposal(proposals,
		"process model with execution listeners that leave observable listener records",
		"listener-oriented walk and element coverage",
		[]string{"walk process-instance", "get element"},
		supportedProposalVersions(),
		"Release validators can cover listener traversal with repository-owned embedded BPMN.",
	)
}
