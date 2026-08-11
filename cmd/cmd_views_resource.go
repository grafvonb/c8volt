// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/spf13/cobra"
)

// resourceView renders a resource lookup result using the active get-output mode.
func resourceView(cmd *cobra.Command, item resource.Resource) error {
	return resourceItemView(cmd, item, pickMode())
}

// resourceItemView renders one resource through the shared item view contract.
func resourceItemView(cmd *cobra.Command, item resource.Resource, mode RenderMode) error {
	return itemView(cmd, item, mode, oneLineResource, func(it resource.Resource) string { return it.ID })
}

// oneLineResource formats a compact resource row for human lookup output.
func oneLineResource(it resource.Resource) string {
	return compactFlatRow(flatRowResource(it))
}

// flatRowResource keeps resource names in the same human position while aligning IDs and keys in list output.
func flatRowResource(it resource.Resource) flatRow {
	vTag := ""
	if it.VersionTag != "" {
		vTag = "/" + it.VersionTag
	}
	return flatRow{it.ID, "k:" + it.Key, it.TenantId, it.Name, fmt.Sprintf("v%d%s", it.Version, vTag)}
}
