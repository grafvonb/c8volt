package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

//nolint:unused
func printFilter(cmd *cobra.Command) {
	var filters []string
	if flagGetPIParentKey != "" {
		filters = append(filters, fmt.Sprintf("parent-key=%s", flagGetPIParentKey))
	}
	if flagGetPIState != "" && flagGetPIState != "all" {
		filters = append(filters, fmt.Sprintf("state=%s", flagGetPIState))
	}
	if flagGetPIParentsOnly {
		filters = append(filters, "parents-only=true")
	}
	if flagGetPIChildrenOnly {
		filters = append(filters, "children-only=true")
	}
	if flagGetPIOrphanParentsOnly {
		filters = append(filters, "orphan-parents-only=true")
	}
	if flagGetPIIncidentsOnly {
		filters = append(filters, "incidents-only=true")
	}
	if flagGetPINoIncidentsOnly {
		filters = append(filters, "no-incidents-only=true")
	}
	if len(filters) > 0 {
		cmd.Println("filter: " + strings.Join(filters, ", "))
	}
}
