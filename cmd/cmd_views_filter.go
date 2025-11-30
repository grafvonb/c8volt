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
	if flagGetPIRootsOnly {
		filters = append(filters, "roots-only=true")
	}
	if flagGetPIChildrenOnly {
		filters = append(filters, "children-only=true")
	}
	if flagGetPIOrphanChildrenOnly {
		filters = append(filters, "orphan-children-only=true")
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
