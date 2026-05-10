// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import d "github.com/grafvonb/c8volt/internal/domain"

func summarizeIncidentResolutionResults(items []d.IncidentResolutionResult) d.IncidentResolutionResults {
	out := d.IncidentResolutionResults{Operation: d.ResolutionOperationIncident, Items: items, Total: len(items)}
	for _, item := range items {
		out.DryRun = out.DryRun || item.DryRun
		out.MutationSubmitted = out.MutationSubmitted || item.MutationSubmitted
		switch item.Status {
		case d.IncidentResolutionStatusSubmitted:
			out.Submitted++
		case d.IncidentResolutionStatusConfirmed:
			out.Confirmed++
		case d.IncidentResolutionStatusSkipped, d.IncidentResolutionStatusPlanned:
			out.Skipped++
		case d.IncidentResolutionStatusMutationFailed, d.IncidentResolutionStatusConfirmationFailed:
			out.Failed++
		}
	}
	return out
}

func summarizeProcessInstanceResolutionResults(items []d.ProcessInstanceResolutionResult) d.ProcessInstanceResolutionResults {
	out := d.ProcessInstanceResolutionResults{Operation: d.ResolutionOperationProcessInstance, Items: items, Total: len(items)}
	for _, item := range items {
		out.DryRun = out.DryRun || item.DryRun
		out.MutationSubmitted = out.MutationSubmitted || item.MutationSubmitted
		switch item.Status {
		case d.ProcessInstanceResolutionStatusSubmitted:
			out.Submitted++
		case d.ProcessInstanceResolutionStatusConfirmed:
			out.Confirmed++
		case d.ProcessInstanceResolutionStatusSkipped, d.ProcessInstanceResolutionStatusPlanned:
			out.Skipped++
		case d.ProcessInstanceResolutionStatusFailed, d.ProcessInstanceResolutionStatusPartialFailed:
			out.Failed++
		}
	}
	return out
}

func compactIncidentResolutionResults(items []d.IncidentResolutionResult) []d.IncidentResolutionResult {
	out := items[:0]
	for _, item := range items {
		if item.IncidentKey == "" && item.Status == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func compactProcessInstanceResolutionResults(items []d.ProcessInstanceResolutionResult) []d.ProcessInstanceResolutionResult {
	out := items[:0]
	for _, item := range items {
		if item.ProcessInstanceKey == "" && item.Status == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}
