// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestOpsAnalyseFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "ops analyse", []string{
		"ops analyse",
		"ops analyse slow-process-instances",
	})
}

// appendOpsAnalyseCommandGapProposals records setup gaps for slow-duration and listener timeline analysis.
func appendOpsAnalyseCommandGapProposals(proposals []proposalRecord) []proposalRecord {
	proposals = registerDirectCamundaSetupFallback(proposals,
		"process instance with element duration exceeding configured thresholds",
		"ops analyse slow-process-instances duration filters",
		[]string{"ops analyse slow-process-instances"},
		supportedProposalVersions(),
		"Operators can create slow-process fixtures through c8volt before running duration analysis.",
	)
	return registerDirectCamundaSetupFallback(proposals,
		"runtime element timeline that includes listener records",
		"ops analyse slow-process-instances --with-listeners",
		[]string{"ops analyse slow-process-instances", "walk process-instance"},
		supportedProposalVersions(),
		"Listener-aware analysis can be validated without direct Camunda setup.",
	)
}

// appendOpsAnalyseEmbeddedBPMNGapProposals records missing slow-process and listener fixtures.
func appendOpsAnalyseEmbeddedBPMNGapProposals(proposals []proposalRecord) []proposalRecord {
	proposals = registerMissingEmbeddedBPMNProposal(proposals,
		"process model that reliably waits long enough to satisfy duration filters",
		"slow duration analysis coverage",
		[]string{"ops analyse slow-process-instances"},
		supportedProposalVersions(),
		"Duration analysis can use deterministic embedded BPMN instead of environment timing assumptions.",
	)
	return registerMissingEmbeddedBPMNProposal(proposals,
		"process model with listener events visible in analysis timelines",
		"listener timeline analysis coverage",
		[]string{"ops analyse slow-process-instances"},
		supportedProposalVersions(),
		"Listener timeline analysis can use repository-owned fixtures.",
	)
}
