// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

func jobView(cmd *cobra.Command, item job.Job) error {
	return itemView(cmd, item, pickMode(), oneLineJobWithTimezoneForMode(cmd), jobKey)
}

// jobsView renders searched jobs using the same row format as keyed lookup and
// leaves JSON output as one collection payload.
func jobsView(cmd *cobra.Command, result job.SearchResult) error {
	return listOrJSONFlat(cmd, result, result.Items, pickMode(), flatRowJobWithTimezoneForMode(cmd), jobKey)
}

// oneLineJobWithTimezoneForMode binds the current timezone display setting for
// keyed job rendering.
func oneLineJobWithTimezoneForMode(cmd *cobra.Command) func(job.Job) string {
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	return func(item job.Job) string {
		return oneLineJobWithTimezone(item, showTimezoneOffset)
	}
}

// flatRowJobWithTimezoneForMode binds the current timezone display setting for
// aligned list/search rows.
func flatRowJobWithTimezoneForMode(cmd *cobra.Command) func(job.Job) flatRow {
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	return func(item job.Job) flatRow {
		return flatRowJobWithTimezone(item, showTimezoneOffset)
	}
}

// jobKey returns the stable pipeline key for keyed and searched job output.
func jobKey(item job.Job) string {
	return item.Key
}

func jobUpdateResultView(cmd *cobra.Command, result job.UpdateResult) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, result)
	}
	renderAttachedTenantContext(cmd)
	switch result.Status {
	case "confirmed":
		parts := []string{fmt.Sprintf("confirmed retries=%d", derefInt32(result.ConfirmedRetries))}
		if result.SubmittedTimeoutMS != nil {
			parts = append(parts, fmt.Sprintf("timeout=%dms submitted", *result.SubmittedTimeoutMS))
		}
		renderHumanLine(cmd, "updated job %s: %s", result.Key, strings.Join(parts, "; "))
	case "submitted":
		parts := []string{"submitted"}
		if result.SubmittedRetries != nil {
			parts = append(parts, fmt.Sprintf("retries=%d", *result.SubmittedRetries))
		}
		if result.SubmittedTimeoutMS != nil {
			parts = append(parts, fmt.Sprintf("timeout=%dms", *result.SubmittedTimeoutMS))
		}
		renderHumanLine(cmd, "updated job %s: %s", result.Key, strings.Join(parts, " "))
	case "confirmation_failed":
		renderHumanLine(cmd, "updated job %s: confirmation failed: %s", result.Key, result.Error)
	case "mutation_failed":
		renderHumanLine(cmd, "updated job %s: mutation failed: %s", result.Key, result.Error)
	default:
		renderHumanLine(cmd, "updated job %s: %s", result.Key, result.Status)
	}
	return nil
}

func jobWorkerOutcomeResultView(cmd *cobra.Command, result job.WorkerOutcomeResult) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, result)
	}
	renderAttachedTenantContext(cmd)
	switch result.Status {
	case "submitted":
		parts := []string{fmt.Sprintf("submitted %s", formatJobMutationMode(job.MutationMode(result.Mode)))}
		if result.SubmittedRetries != nil {
			parts = append(parts, fmt.Sprintf("retries=%d", *result.SubmittedRetries))
		}
		if result.SubmittedBackoffMS != nil {
			parts = append(parts, fmt.Sprintf("retryBackoff=%dms", *result.SubmittedBackoffMS))
		}
		if result.SubmittedErrorCode != "" {
			parts = append(parts, fmt.Sprintf("errorCode=%s", result.SubmittedErrorCode))
		}
		renderHumanLine(cmd, "updated job %s: %s", result.Key, strings.Join(parts, " "))
	case "mutation_failed":
		renderHumanLine(cmd, "updated job %s: mutation failed: %s", result.Key, result.Error)
	default:
		renderHumanLine(cmd, "updated job %s: %s", result.Key, result.Status)
	}
	return nil
}

func jobUpdatePlanView(cmd *cobra.Command, plan job.UpdatePlan, label string) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, plan)
	}
	renderAttachedTenantContext(cmd)
	status := jobUpdatePlanHumanStatus(plan, label)
	if !plan.HasMaterialChange() {
		renderHumanLine(cmd, "%s: update job %s: nothing to update; %s", label, plan.Key, status)
		return nil
	}
	if status != "" {
		renderHumanLine(cmd, "%s: update job %s: %s; %s", label, plan.Key, formatJobUpdatePlanItems(plan.Items), status)
		return nil
	}
	renderHumanLine(cmd, "%s: update job %s: %s", label, plan.Key, formatJobUpdatePlanItems(plan.Items))
	return nil
}

func jobUpdatePlanHumanStatus(plan job.UpdatePlan, label string) string {
	if label == "dry run" {
		return "no changes applied"
	}
	if !plan.HasMaterialChange() {
		return "no confirmation required"
	}
	return ""
}

func oneLineJob(item job.Job) string {
	return oneLineJobWithTimezone(item, false)
}

func oneLineJobWithTimezone(item job.Job, showTimezoneOffset bool) string {
	return compactFlatRow(flatRowJobWithTimezone(item, showTimezoneOffset))
}

func flatRowJob(item job.Job) flatRow {
	return flatRowJobWithTimezone(item, false)
}

func flatRowJobWithTimezone(item job.Job, showTimezoneOffset bool) flatRow {
	parts := flatRow{
		item.Key,
		item.TenantId,
		item.Kind,
		item.ElementId,
		item.State,
		prefixedJobField("tp", item.Type),
		prefixedJobField("lsnr", item.ListenerEventType),
		prefixedJobField("worker", item.Worker),
	}
	if item.ProcessInstanceKey != "" {
		parts = append(parts, "pi:"+item.ProcessInstanceKey)
	} else {
		parts = append(parts, "")
	}
	if item.ElementInstanceKey != "" {
		parts = append(parts, "ei:"+item.ElementInstanceKey)
	} else {
		parts = append(parts, "")
	}
	parts = append(parts, "r:"+strconv.FormatInt(int64(item.Retries), 10))
	if item.Deadline != nil {
		parts = append(parts, "d:"+toolx.FormatTime(*item.Deadline, showTimezoneOffset))
	} else {
		parts = append(parts, "")
	}
	if item.ErrorCode != "" {
		parts = append(parts, "ec:"+item.ErrorCode)
	} else {
		parts = append(parts, "")
	}
	if item.ErrorMessage != "" {
		parts = append(parts, "err:"+truncateHumanMessage(item.ErrorMessage, flagGetErrorMessageLimit))
	} else {
		parts = append(parts, "")
	}
	return parts
}

func prefixedJobField(prefix string, value string) string {
	if value == "" {
		return ""
	}
	return prefix + ":" + value
}

func formatJobUpdatePlanItems(items []job.UpdatePlanItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		switch item.Name {
		case "retries":
			if item.Status == string(job.RetryChangeUnchanged) {
				parts = append(parts, fmt.Sprintf("retries: %s (unchanged)", item.After))
				continue
			}
			if item.Before == "" {
				parts = append(parts, fmt.Sprintf("retries: %s", item.After))
				continue
			}
			parts = append(parts, fmt.Sprintf("retries: %s -> %s", item.Before, item.After))
		case "timeout":
			parts = append(parts, fmt.Sprintf("timeout: set to %s", item.After))
		case string(job.MutationModeTechnicalFailure):
			parts = append(parts, "technical failure: submit")
		case string(job.MutationModeBPMNError):
			parts = append(parts, "BPMN error: submit")
		case "retryBackoff":
			parts = append(parts, fmt.Sprintf("retry backoff: %s", item.After))
		case "errorCode":
			parts = append(parts, fmt.Sprintf("error code: %s", item.After))
		case "message":
			parts = append(parts, fmt.Sprintf("message: %s", item.After))
		case "variables":
			parts = append(parts, "variables: submit")
		default:
			parts = append(parts, fmt.Sprintf("%s: %s", item.Name, item.After))
		}
	}
	return strings.Join(parts, "; ")
}

func formatJobMutationMode(mode job.MutationMode) string {
	switch mode {
	case job.MutationModeTechnicalFailure:
		return "technical failure"
	case job.MutationModeBPMNError:
		return "BPMN error"
	case job.MutationModeCompletion:
		return "completion"
	default:
		return string(mode)
	}
}

func derefInt32(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

// jobCompletionListenerConsequence is shared by preview and confirmation so the
// operator sees the same consequence before either reviewing or approving work.
const jobCompletionListenerConsequence = "Completing a listener job acknowledges its work without executing its business logic."

// jobCompletionConfirmationPrompt describes the exact frozen count and scope.
func jobCompletionConfirmationPrompt(batch job.PreparedCompletionBatch) string {
	scope := tenantContextPrimaryHumanLine(newDiscoveryTenantContext(batch.DiscoveryTenant))
	if batch.Selection.Mode == job.CompletionSelectionExplicit {
		scope = tenantContextPrimaryHumanLine(newExplicitKeysTenantContext(batch.DiscoveryTenant))
		if len(batch.TenantIDs) > 0 {
			scope += "; affected tenants: " + strings.Join(batch.TenantIDs, ", ")
		}
		return fmt.Sprintf("You are about to complete %d explicitly selected job(s); %s. %s Do you want to proceed?", len(batch.Candidates), scope, jobCompletionListenerConsequence)
	}
	return fmt.Sprintf("You are about to complete %d selected job(s); %s; filters: %s. %s Do you want to proceed?", len(batch.Candidates), scope, jobCompletionFilterText(batch.Selection.Filter), jobCompletionListenerConsequence)
}

// jobCompletionFilterText renders normalized AND predicates in stable CLI order.
func jobCompletionFilterText(filter job.CompletionFilter) string {
	parts := []string{}
	for _, pair := range [][2]string{{"state", filter.State}, {"type", filter.Type}, {"pi-key", filter.ProcessInstanceKey}, {"element-instance-key", filter.ElementInstanceKey}, {"element-id", filter.ElementId}, {"worker", filter.Worker}, {"kind", filter.Kind}, {"listener-event-type", filter.ListenerEventType}} {
		if pair[1] != "" {
			parts = append(parts, pair[0]+"="+pair[1])
		}
	}
	if filter.Retries != nil {
		parts = append(parts, fmt.Sprintf("retries=%d", *filter.Retries))
	}
	return strings.Join(parts, "; ")
}

// jobCompletionBatchView renders authoritative empty, preview and partial outcomes
// through the same mode-aware path. No-wait acceptance requires actual acceptance.
func jobCompletionBatchView(cmd *cobra.Command, report job.CompletionBatchReport, runErr error) error {
	if pickMode() == RenderModeJSON {
		envelope := ResultEnvelope[any]{Outcome: OutcomeSucceeded, Command: commandPath(cmd), Payload: report}
		if runErr != nil {
			envelope = resultEnvelopeForError(cmd, runErr)
			envelope.Payload = report
		} else if flagNoWait && report.Counts.Accepted > 0 {
			envelope.Outcome = OutcomeAccepted
		}
		return renderResultEnvelope(cmd, envelope)
	}
	if runErr != nil {
		cmd.PrintErrf("job completion: %s\n", runErr)
	}
	if flagQuiet {
		return nil
	}
	label := "job completion"
	if report.Status == job.CompletionBatchPreview {
		label = "dry run: job completion"
	}
	renderOutputLine(cmd, "%s", label)
	if report.Selection.Mode == job.CompletionSelectionExplicit {
		renderOutputLine(cmd, "selection: explicit keys")
	} else {
		renderOutputLine(cmd, "filters: %s", jobCompletionFilterText(report.Selection.Filter))
	}
	if ctx, ok := attachedTenantContext(cmd); ok {
		for _, line := range tenantContextHumanLines(cmd, *ctx) {
			renderOutputLine(cmd, "%s", line.Text)
		}
	}
	if report.Status == job.CompletionBatchPreparationFailed {
		renderOutputLine(cmd, "preparation incomplete: %d observed candidate job(s); no changes applied", report.Counts.Selected)
	} else {
		renderOutputLine(cmd, "selected job(s): %d", report.Counts.Selected)
	}
	if report.Limited {
		renderOutputLine(cmd, "discovery user-limited: limit %d; pages %d; batch size %d", report.Selection.Limit, report.PageCount, report.Selection.BatchSize)
	} else if flagVerbose && report.Selection.Mode == job.CompletionSelectionSearch {
		renderOutputLine(cmd, "discovery: pages %d; batch size %d", report.PageCount, report.Selection.BatchSize)
	}
	switch report.Status {
	case job.CompletionBatchPreview:
		renderOutputLine(cmd, "%s", jobCompletionListenerConsequence)
		renderOutputLine(cmd, "outcome: preview; no changes applied")
	case job.CompletionBatchPreparationFailed:
		renderOutputLine(cmd, "outcome: preparation failed; no changes applied")
	default:
		if report.Counts.Selected == 0 && runErr == nil {
			if report.Selection.Mode == job.CompletionSelectionExplicit {
				renderOutputLine(cmd, "outcome: no selected jobs; no changes applied")
			} else {
				renderOutputLine(cmd, "outcome: no matching jobs; no changes applied")
			}
		} else {
			status := string(report.Status)
			if runErr != nil && report.Status == job.CompletionBatchFinished {
				status = "failed"
			}
			renderOutputLine(cmd, "outcome: %s; accepted %d; observed completed %d; unsatisfied %d; not started %d", status, report.Counts.Accepted, report.Counts.Completed, report.Counts.Unsatisfied, report.Counts.NotStarted)
			if report.Counts.Accepted > 0 {
				renderOutputLine(cmd, "accepted completions: independent confirmation skipped; process termination not verified")
			}
		}
	}
	if flagVerbose {
		for _, candidate := range report.Candidates {
			renderOutputLine(cmd, "%s", jobCompletionCandidateRow(candidate))
		}
		for _, outcome := range report.Outcomes {
			renderOutputLine(cmd, "%s disposition:%s submission:%s observation:%s reason:%s", jobCompletionCandidateRow(outcome.Candidate), outcome.Disposition, outcome.Submission.Status, outcome.Observation.Status, outcome.Reason)
		}
	}
	return nil
}

// jobCompletionCandidateRow reuses ordinary job rows with only observed facts.
func jobCompletionCandidateRow(candidate job.CompletionCandidate) string {
	return oneLineJob(job.Job{Key: candidate.Key, ProcessInstanceKey: candidate.ProcessInstanceKey, ElementInstanceKey: candidate.ElementInstanceKey, ElementId: candidate.ElementId, TenantId: candidate.TenantId, Type: candidate.Type, Kind: candidate.Kind, ListenerEventType: candidate.ListenerEventType, State: candidate.State, Worker: candidate.Worker, Retries: candidate.Retries})
}
