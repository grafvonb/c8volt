// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

func renderOpsExecuteRetentionPolicyResult(cmd *cobra.Command, result ops.RetentionPolicyResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	cmd.Printf("retention policy: %s\n", result.Outcome)
	cmd.Printf("retention days: %d\n", result.Request.RetentionDays)
	if result.Discovery.Status != "" {
		cmd.Printf("retention discovery: %s\n", result.Discovery.Status)
	}
	if result.DeletePlan.Status != "" {
		cmd.Printf("delete plan: %s\n", result.DeletePlan.Status)
	}
	if result.Deletion.Status != "" {
		cmd.Printf("deletion: %s\n", result.Deletion.Status)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}
