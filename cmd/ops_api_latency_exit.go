// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ops"
)

// opsAPILatencyCommandError maps diagnostic outcomes onto the existing command error contract.
func opsAPILatencyCommandError(commandName string, result ops.APILatencyResult, err error) error {
	if opsAPILatencyOutcomeExitsSuccessfully(result.Outcome) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", commandName, err)
	}
	if result.Outcome == "" {
		return nil
	}
	return fmt.Errorf("%s: API latency diagnostic ended with outcome %s", commandName, result.Outcome)
}

// opsAPILatencyOutcomeExitsSuccessfully defines the successful terminal states shared by both latency leaves.
func opsAPILatencyOutcomeExitsSuccessfully(outcome ops.APILatencyOutcome) bool {
	switch outcome {
	case ops.APILatencyOutcomePlanned, ops.APILatencyOutcomeCompleted, ops.APILatencyOutcomeCompletedRetained:
		return true
	default:
		return false
	}
}
