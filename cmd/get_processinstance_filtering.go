// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// registerPISharedDateRangeFlags attaches the absolute and relative date
// filters reused by get/cancel/delete process-instance search commands.
func registerPISharedDateRangeFlags(fs *pflag.FlagSet) {
	fs.StringVar(&flagGetPIStartDateAfter, "start-date-after", "", "only include process instances with start date >= YYYY-MM-DD")
	fs.StringVar(&flagGetPIStartDateBefore, "start-date-before", "", "only include process instances with start date <= YYYY-MM-DD")
	fs.StringVar(&flagGetPIEndDateAfter, "end-date-after", "", "only include process instances with end date >= YYYY-MM-DD")
	fs.StringVar(&flagGetPIEndDateBefore, "end-date-before", "", "only include process instances with end date <= YYYY-MM-DD")

	fs.IntVar(&flagGetPIStartAfterDays, "start-date-older-days", -1, "only include process instances N days old or older")
	fs.IntVar(&flagGetPIStartBeforeDays, "start-date-newer-days", -1, "only include process instances N days old or newer (0 means today)")
	fs.IntVar(&flagGetPIEndAfterDays, "end-date-older-days", -1, "only include process instances with end date N days old or older")
	fs.IntVar(&flagGetPIEndBeforeDays, "end-date-newer-days", -1, "only include process instances with end date N days old or newer (0 means today)")
}

// registerPISharedProcessDefinitionFilterFlags attaches the process-definition
// selector flags that are validated before process-instance search starts.
func registerPISharedProcessDefinitionFilterFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&flagGetPIBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.Int32Var(&flagGetPIProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagGetPIProcessVersionTag, "pd-version-tag", "", "process definition version tag")
}

// populatePISearchFilterOpts converts validated command flags into the service
// filter model. Relative-day inputs are resolved here so lower layers receive
// concrete date bounds independent of the CLI flag style.
func populatePISearchFilterOpts() process.ProcessInstanceFilter {
	f := process.ProcessInstanceFilter{
		ParentKey:            flagGetPIParentKey,
		BpmnProcessId:        flagGetPIBpmnProcessID,
		ProcessVersion:       flagGetPIProcessVersion,
		ProcessVersionTag:    flagGetPIProcessVersionTag,
		ProcessDefinitionKey: flagGetPIProcessDefinitionKey,
		StartDateAfter:       pickPIDateBound(flagGetPIStartDateAfter, flagGetPIStartBeforeDays),
		StartDateBefore:      pickPIDateUpperBound(flagGetPIStartDateBefore, flagGetPIStartAfterDays),
		EndDateAfter:         pickPIDateBound(flagGetPIEndDateAfter, flagGetPIEndBeforeDays),
		EndDateBefore:        pickPIDateUpperBound(flagGetPIEndDateBefore, flagGetPIEndAfterDays),
	}

	if s := flagGetPIState; s != "" && s != "all" {
		if st, ok := process.ParseState(s); ok {
			f.State = st
		}
	}
	if flagGetPIChildrenOnly {
		f.HasParent = new(true)
	}
	if flagGetPIRootsOnly {
		f.HasParent = new(false)
	}
	if flagGetPIIncidentsOnly {
		f.HasIncident = new(true)
	}
	if flagGetPINoIncidentsOnly {
		f.HasIncident = new(false)
	}
	return f
}

// hasPISearchFilterFlags reports whether list/search selectors beyond the
// default state are present. Keyed and user-task modes use this to reject
// ambiguous combinations before any lookup work starts.
func hasPISearchFilterFlags() bool {
	return flagGetPIParentKey != "" ||
		flagGetPIBpmnProcessID != "" ||
		flagGetPIProcessVersion != 0 ||
		flagGetPIProcessVersionTag != "" ||
		flagGetPIProcessDefinitionKey != "" ||
		hasPIDateFilterFlags() ||
		hasPIRelativeDayFilterFlags() ||
		(flagGetPIState != "" && flagGetPIState != "all")
}

// hasPIDateFilterFlags isolates absolute date filters from relative-day filters
// so validation can reject mixed styles with targeted diagnostics.
func hasPIDateFilterFlags() bool {
	return flagGetPIStartDateAfter != "" ||
		flagGetPIStartDateBefore != "" ||
		flagGetPIEndDateAfter != "" ||
		flagGetPIEndDateBefore != ""
}

// hasPIRelativeDayFilterFlags reports whether any relative-day selector was set
// from its sentinel value. The sentinel distinction matters because zero is a
// valid operator input meaning "today".
func hasPIRelativeDayFilterFlags() bool {
	return flagGetPIStartAfterDays >= 0 ||
		flagGetPIStartBeforeDays >= 0 ||
		flagGetPIEndAfterDays >= 0 ||
		flagGetPIEndBeforeDays >= 0
}

// applyPISearchResultFilters applies relationship and incident filters that may
// need to run locally after a backend page is fetched. This preserves behavior
// for Camunda versions or filter combinations where request-side support is not
// reliable enough to trust alone.
func applyPISearchResultFilters(cmd *cobra.Command, cli process.API, pis process.ProcessInstances) (process.ProcessInstances, error) {
	var err error
	// Keep the local fallback path in place so versions without reliable
	// request-side support still preserve the existing filter semantics.
	if flagGetPIChildrenOnly {
		pis = pis.FilterChildrenOnly()
	}
	if flagGetPIRootsOnly {
		pis = pis.FilterRootsOnly()
	}
	if flagGetPIOrphanChildrenOnly {
		stopActivity := startCommandActivity(cmd, fmt.Sprintf("checking orphan parents for %d process instance(s)", len(pis.Items)))
		pis.Items, err = cli.FilterProcessInstanceWithOrphanParent(cmd.Context(), pis.Items, collectOptions()...)
		stopActivity()
		if err != nil {
			return process.ProcessInstances{}, fmt.Errorf("error filtering orphan children: %w", err)
		}
		pis.Total = int32(len(pis.Items))
	}
	if flagGetPIIncidentsOnly {
		pis = pis.FilterByHavingIncidents(true)
	}
	if flagGetPIDirectIncidentsOnly {
		pis, err = filterProcessInstancesWithDirectIncidents(cmd, cli, pis)
		if err != nil {
			return process.ProcessInstances{}, err
		}
	}
	if flagGetPINoIncidentsOnly {
		pis = pis.FilterByHavingIncidents(false)
	}
	return pis, nil
}

func filterProcessInstancesWithDirectIncidents(cmd *cobra.Command, cli process.API, pis process.ProcessInstances) (process.ProcessInstances, error) {
	if len(pis.Items) == 0 {
		pis.Total = 0
		return pis, nil
	}
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("checking direct incidents for %d process instance(s)", len(pis.Items)))
	enriched, err := cli.EnrichProcessInstancesWithIncidents(cmd.Context(), pis, collectIncidentEnrichmentOptions()...)
	stopActivity()
	if err != nil {
		return process.ProcessInstances{}, fmt.Errorf("error filtering direct incidents: %w", err)
	}
	out := make([]process.ProcessInstance, 0, len(enriched.Items))
	for _, item := range enriched.Items {
		if len(item.Incidents) > 0 {
			out = append(out, item.Item)
		}
	}
	return process.ProcessInstances{Total: int32(len(out)), Items: out}, nil
}
