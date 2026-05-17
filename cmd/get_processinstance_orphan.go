// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
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
	discovery, err := cli.DiscoverOrphanProcessInstances(cmd.Context(), process.OrphanDiscoveryRequest{
		Filter:    filter,
		BatchSize: resolvePISearchSize(cmd, cfg),
		Limit:     limit,
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
