// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

// searchOrphanProcessInstancesWithSharedDiscovery keeps `get pi
// --orphan-children-only` on the same orphan discovery primitive used by ops
// purge workflows. This avoids two subtly different meanings for --limit.
func searchOrphanProcessInstancesWithSharedDiscovery(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (process.ProcessInstances, bool, error) {
	limit := flagGetPILimit
	if flagGetPIDirectIncidentsOnly || hasPIIncidentDetailFilters() {
		limit = 0
	}
	stopActivity := startCommandActivity(cmd, "discovering orphan child process instances")
	defer stopActivity()
	discovery, err := cli.DiscoverOrphanProcessInstances(cmd.Context(), process.OrphanDiscoveryRequest{
		Filter:    filter,
		BatchSize: resolvePISearchSize(cmd, cfg),
		Limit:     limit,
		Progress:  updateOrphanDiscoveryActivity(cmd),
	}, collectOptions()...)
	if err != nil {
		return process.ProcessInstances{}, false, err
	}
	pis := process.ProcessInstances{
		Total: int32(len(discovery.Items)),
		Items: discovery.Items,
	}
	if flagGetPIDirectIncidentsOnly || hasPIIncidentDetailFilters() {
		pis, err = filterProcessInstancesWithDirectIncidents(cmd, cli, pis)
		if err != nil {
			return process.ProcessInstances{}, false, err
		}
		pis.Items = limitPIItems(pis.Items, 0)
		pis.Total = int32(len(pis.Items))
	}
	return pis, false, nil
}

func updateOrphanDiscoveryActivity(cmd *cobra.Command) func(process.OrphanDiscoveryProgress) {
	if cmd == nil {
		return func(process.OrphanDiscoveryProgress) {}
	}
	return func(event process.OrphanDiscoveryProgress) {
		msg := formatOrphanDiscoveryProgress(event)
		logging.UpdateActivity(cmd.Context(), msg)
		printOrphanDiscoveryProgress(cmd, msg)
	}
}

func printOrphanDiscoveryProgress(cmd *cobra.Command, msg string) {
	if cmd == nil || !flagVerbose || pickMode() != RenderModeOneLine {
		return
	}
	fmt.Fprintln(cmd.ErrOrStderr(), msg)
}

func formatOrphanDiscoveryProgress(event process.OrphanDiscoveryProgress) string {
	switch event.Phase {
	case "checking":
		return fmt.Sprintf(
			"orphan search: page %d checking %d child process instance(s) for missing parents; checked %d, found %d orphan child process instance(s)",
			event.Page,
			event.CurrentPageCandidates,
			event.CandidatesChecked,
			event.OrphansFound,
		)
	default:
		msg := fmt.Sprintf(
			"orphan search: page %d checked %d child process instance(s), found %d on page, %d total",
			event.Page,
			event.CandidatesChecked,
			event.CurrentPageOrphans,
			event.OrphansFound,
		)
		if event.Limit > 0 && event.OrphansFound >= int(event.Limit) {
			msg += fmt.Sprintf(", limit %d reached", event.Limit)
		}
		return msg
	}
}
