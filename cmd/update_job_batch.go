// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

// parseUpdateJobBatchRequest maps CLI inputs into the public preparation contract.
// The services own normalization of key spellings, traversal and frozen membership.
func parseUpdateJobBatchRequest(cmd *cobra.Command, explicit *job.CompletionSelection) (job.CompletionRequest, error) {
	if pickMode() == RenderModeKeysOnly {
		return job.CompletionRequest{}, invalidFlagValuef("--keys-only is not supported for job completion")
	}
	for _, name := range []string{"fail", "throw-bpmn-error", "timeout", "retry-backoff", "message"} {
		if cmd.Flags().Changed(name) {
			return job.CompletionRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --%s", name)
		}
	}
	fs := cmd.Flags()
	if fs.Changed("workers") && flagWorkers < 1 {
		return job.CompletionRequest{}, invalidFlagValuef("--workers must be positive")
	}
	request := job.CompletionRequest{}
	if explicit != nil {
		if fs.Changed("retries") {
			return request, mutuallyExclusiveFlagsf("keyed --complete cannot be combined with --retries")
		}
		request.Selection = *explicit
	} else {
		selection, err := parseUpdateJobSearchSelection(cmd)
		if err != nil {
			return request, err
		}
		request.Selection = selection
	}
	if fs.Changed("vars") {
		variables, err := parseUpdateJobVariables(flagUpdateJobVariables)
		if err != nil {
			return request, err
		}
		request.Variables = variables
	}
	return request, nil
}

// parseUpdateJobSearchSelection reuses read-command predicate validation without
// letting search controls silently truncate explicit inputs.
func parseUpdateJobSearchSelection(cmd *cobra.Command) (job.CompletionSelection, error) {
	fs := cmd.Flags()
	value := func(name string) string { v, _ := fs.GetString(name); return strings.TrimSpace(v) }
	filter := job.CompletionFilter{State: value("state"), Type: value("type"), ProcessInstanceKey: value("pi-key"), ElementInstanceKey: value("element-instance-key"), ElementId: value("element-id"), Worker: value("worker"), Kind: value("kind"), ListenerEventType: value("listener-event-type")}
	if fs.Changed("retries") {
		retries := flagUpdateJobRetries
		filter.Retries = &retries
	}
	if filter == (job.CompletionFilter{}) {
		return job.CompletionSelection{}, invalidFlagValuef("completion requires --key or at least one job search predicate")
	}
	if err := validateJobSearchPredicates(job.SearchRequest{State: filter.State, ProcessInstanceKey: filter.ProcessInstanceKey, ElementInstanceKey: filter.ElementInstanceKey, Retries: filter.Retries, Kind: filter.Kind, ListenerEventType: filter.ListenerEventType}); err != nil {
		return job.CompletionSelection{}, err
	}
	filter.State = normalizedJobState(filter.State)
	filter.Kind = normalizedJobKind(filter.Kind)
	filter.ListenerEventType = normalizedJobListenerEventType(filter.ListenerEventType)
	size, _ := fs.GetInt32("batch-size")
	limit, _ := fs.GetInt32("limit")
	if size <= 0 || size > consts.MaxPISearchSize {
		return job.CompletionSelection{}, invalidFlagValuef("--batch-size must be positive and at most %d", consts.MaxPISearchSize)
	}
	if limit < 0 {
		return job.CompletionSelection{}, invalidFlagValuef("--limit must be non-negative")
	}
	return job.CompletionSelection{Mode: job.CompletionSelectionSearch, InputForm: job.CompletionInputFilters, Filter: filter, BatchSize: size, Limit: limit}, nil
}

// parseUpdateJobExplicitSelection reads and validates all input before preparation.
// Keep duplicates and decimal strings here; the service owns canonical membership.
func parseUpdateJobExplicitSelection(cmd *cobra.Command, inputs []string, stdin bool) (job.CompletionSelection, error) {
	selection := job.CompletionSelection{Mode: job.CompletionSelectionExplicit, InputForm: job.CompletionInputCollection}
	for _, input := range inputs {
		selection.Keys = append(selection.Keys, strings.Split(input, ",")...)
	}
	if stdin {
		selection.InputForm = job.CompletionInputStdin
		scanner := bufio.NewScanner(cmd.InOrStdin())
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			if key := strings.TrimSpace(scanner.Text()); key != "" {
				selection.Keys = append(selection.Keys, key)
			}
		}
		if err := scanner.Err(); err != nil {
			return selection, invalidFlagValuef("reading job keys from stdin: %v", err)
		}
	}
	for i, raw := range selection.Keys {
		key := strings.TrimSpace(raw)
		value, err := strconv.ParseInt(key, 10, 64)
		if err != nil || value <= 0 || strings.IndexFunc(key, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			return selection, invalidFlagValuef("invalid job key %q: expected a positive signed 64-bit decimal integer", raw)
		}
		selection.Keys[i] = key
	}
	return selection, nil
}

// runUpdateJobBatch prepares once, confirms the frozen scope once, then renders
// report plus error once. Callers must use exitAfterRenderedResult on failure.
func runUpdateJobBatch(cmd *cobra.Command, cli job.API, cfg *config.Config, request job.CompletionRequest) error {
	batch, err := cli.PrepareJobCompletionBatch(cmd.Context(), request, collectOptions()...)
	tenantContext := newDiscoveryTenantContext(configuredTenantID(cfg))
	if request.Selection.Mode == job.CompletionSelectionExplicit {
		tenantContext = newExplicitKeysTenantContext(configuredTenantID(cfg))
	}
	unknownTenants := 0
	for _, candidate := range batch.Candidates {
		if candidate.TenantId == "" {
			unknownTenants++
		}
	}
	attachTenantContext(cmd, withTenantContextEvidence(tenantContext, batch.TenantIDs, unknownTenants))
	if err != nil {
		return finishUpdateJobBatch(cmd, jobCompletionPreparationReport(batch, job.CompletionBatchPreparationFailed), err)
	}
	if !flagDryRun && len(batch.Candidates) > 0 && !shouldImplicitlyConfirm(cmd) {
		prompt := jobCompletionConfirmationPrompt(batch)
		if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, prompt); err != nil {
			report := jobCompletionPreparationReport(batch, job.CompletionBatchAborted)
			report.Candidates = nil
			report.Counts.NotStarted = len(batch.Candidates)
			for _, candidate := range batch.Candidates {
				report.Outcomes = append(report.Outcomes, job.CompletionOutcome{Candidate: candidate, Disposition: job.CompletionDispositionNotStarted, Submission: job.CompletionSubmission{Status: job.CompletionSubmissionNotSubmitted}, Observation: job.CompletionObservation{Status: job.CompletionObservationSkipped}, Reason: job.CompletionOutcomeDeclined})
			}
			return finishUpdateJobBatch(cmd, report, err)
		}
	}
	report, err := cli.ExecuteJobCompletionBatch(cmd.Context(), batch, flagWorkers, collectOptions()...)
	return finishUpdateJobBatch(cmd, report, err)
}

// jobCompletionPreparationReport preserves partial evidence or declined selection
// without fabricating successful submission or observation.
func jobCompletionPreparationReport(batch job.PreparedCompletionBatch, status job.CompletionBatchStatus) job.CompletionBatchReport {
	return job.CompletionBatchReport{Status: status, Selection: batch.Selection, DiscoveryTenant: batch.DiscoveryTenant, TenantIDs: batch.TenantIDs, CompletionReason: batch.CompletionReason, PageCount: batch.PageCount, ObservedCount: batch.ObservedCount, Limited: batch.Limited, Counts: job.CompletionCounts{Selected: len(batch.Candidates)}, Candidates: batch.Candidates}
}

// finishUpdateJobBatch retains the operation error after rendering its partial payload.
func finishUpdateJobBatch(cmd *cobra.Command, report job.CompletionBatchReport, runErr error) error {
	if err := jobCompletionBatchView(cmd, report, runErr); err != nil {
		return fmt.Errorf("render job completion result: %w", err)
	}
	return runErr
}
