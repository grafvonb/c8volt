// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

// appendRealStateCommandGapProposal records setup blocked by missing c8volt command support.
func appendRealStateCommandGapProposal(proposals []proposalRecord, requiredState string, coverageNeed string, affectedCommands []string, operatorValue string) []proposalRecord {
	return registerDirectCamundaSetupFallback(proposals,
		requiredState,
		coverageNeed,
		affectedCommands,
		[]string{realStateTargetVersion},
		operatorValue,
	)
}

// appendRealStateEmbeddedBPMNGapProposal records setup blocked by missing embedded BPMN behavior.
func appendRealStateEmbeddedBPMNGapProposal(proposals []proposalRecord, requiredState string, coverageNeed string, affectedCommands []string, operatorValue string) []proposalRecord {
	return registerMissingEmbeddedBPMNProposal(proposals,
		requiredState,
		coverageNeed,
		affectedCommands,
		[]string{realStateTargetVersion},
		operatorValue,
	)
}

// TestRealStateProposalFallbackHelpers verifies real-state gaps are version-scoped to C89.
func TestRealStateProposalFallbackHelpers(t *testing.T) {
	commandProposals := appendRealStateCommandGapProposal(nil,
		"active job requiring direct setup",
		"real-state job mutation coverage",
		[]string{"get job", "update job"},
		"Maintainers can replace direct setup with a c8volt command when available.",
	)
	embeddedProposals := appendRealStateEmbeddedBPMNGapProposal(nil,
		"listener-capable process model",
		"real-state listener coverage",
		[]string{"walk process-instance", "get element"},
		"Maintainers can cover listener paths with repository-owned embedded BPMN.",
	)

	requireProposalRecord(t, commandProposals[0], proposalKindCommand)
	requireProposalRecord(t, embeddedProposals[0], proposalKindEmbeddedBPMN)
	assertStringSlicesEqual(t, commandProposals[0].AffectedVersions, []string{realStateTargetVersion})
	assertStringSlicesEqual(t, embeddedProposals[0].AffectedVersions, []string{realStateTargetVersion})
}
