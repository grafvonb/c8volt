// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package completion

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
)

// ReadPorts is the read-only adapter boundary; preparation has no mutation capability.
type ReadPorts struct {
	Get    func(context.Context, string, ...services.CallOption) (d.Job, error)
	Search func(context.Context, d.JobSearchQuery, d.JobPageRequest, ...services.CallOption) (d.JobSearchPage, error)
}

// Prepare freezes complete, unique membership before any execution is possible.
// Errors retain partial observations with incomplete status and no completion reason.
func Prepare(ctx context.Context, request d.CompletionRequest, tenant string, backoff config.BackoffConfig, ports ReadPorts, opts ...services.CallOption) (d.PreparedCompletionBatch, error) {
	_ = services.ApplyCallOptions(opts)
	batch := d.PreparedCompletionBatch{Selection: request.Selection, DiscoveryTenant: tenant, Status: d.CompletionPreparationIncomplete, Candidates: []d.CompletionCandidate{}, TenantIDs: []string{}}
	batch.Selection.Keys = slices.Clone(request.Selection.Keys)
	batch.Selection.Filter.Retries = toolx.CopyPtr(request.Selection.Filter.Retries)
	if err := ctx.Err(); err != nil {
		return batch, err
	}
	if err := normalizeSelection(&batch.Selection); err != nil {
		return batch, err
	}
	if err := common.ValidateJobCompletionPayload(d.Job{}, request.Variables); err != nil {
		return batch, err
	}
	variables, err := toolx.CopyJSONMap(request.Variables)
	if err != nil {
		return batch, fmt.Errorf("%w: invalid completion variables: %v", d.ErrValidation, err)
	}
	batch.Variables = variables
	if batch.Selection.Mode == d.CompletionSelectionExplicit {
		err = prepareExplicit(ctx, &batch, ports, opts...)
	} else {
		err = prepareSearch(ctx, &batch, backoff, ports, opts...)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		batch.Status = d.CompletionPreparationIncomplete
		batch.CompletionReason = ""
		return batch, err
	}
	batch.Status = d.CompletionPreparationComplete
	return batch, nil
}

// normalizeSelection validates all explicit keys before reads and keeps controls
// separate from the nine AND predicates. Empty stdin is a deliberate selection.
func normalizeSelection(s *d.CompletionSelection) error {
	invalid := func(detail string) error { return fmt.Errorf("%w: %s", d.ErrValidation, detail) }
	if s.BatchSize < 0 || s.Limit < 0 {
		return invalid("search controls must be non-negative")
	}
	f := &s.Filter
	for _, field := range []struct {
		value *string
		valid []string
	}{
		{&f.State, []string{"CANCELED", "COMPLETED", "CREATED", "ERROR_THROWN", "FAILED", "MIGRATED", "RETRIES_UPDATED", "TIMED_OUT"}},
		{&f.Kind, []string{"AD_HOC_SUB_PROCESS", "BPMN_ELEMENT", "EXECUTION_LISTENER", "TASK_LISTENER"}},
		{&f.ListenerEventType, []string{"ASSIGNING", "CANCELING", "COMPLETING", "CREATING", "END", "START", "UNSPECIFIED", "UPDATING"}},
	} {
		if *field.value != "" {
			value, ok := toolx.CanonicalEnumString(*field.value, field.valid)
			if !ok {
				return invalid("invalid job search enum: " + *field.value)
			}
			*field.value = value
		}
	}
	if f.Retries != nil && *f.Retries < 0 {
		return invalid("retries must be non-negative")
	}
	for _, key := range []*string{&f.ProcessInstanceKey, &f.ElementInstanceKey} {
		if *key != "" {
			if !validKey(*key) {
				return invalid("invalid job scope key: " + *key)
			}
			*key = canonicalKey(*key)
		}
	}
	q := searchQuery(*s)
	switch s.Mode {
	case d.CompletionSelectionExplicit:
		if q.HasSearchFilters() {
			return invalid("keys and filter selection are mutually exclusive")
		}
		if s.BatchSize != 0 || s.Limit != 0 {
			return invalid("search controls cannot accompany explicit keys")
		}
		if s.InputForm != d.CompletionInputScalar && s.InputForm != d.CompletionInputCollection && s.InputForm != d.CompletionInputStdin {
			return invalid("invalid explicit input form")
		}
		if len(s.Keys) == 0 && s.InputForm != d.CompletionInputStdin {
			return invalid("explicit completion requires keys or stdin")
		}
		if s.InputForm == d.CompletionInputScalar && len(s.Keys) != 1 {
			return invalid("scalar completion requires one key")
		}
		for i, key := range s.Keys {
			if !validKey(key) {
				return invalid("invalid job key: " + key)
			}
			s.Keys[i] = canonicalKey(key)
		}
	case d.CompletionSelectionSearch:
		if len(s.Keys) > 0 {
			return invalid("keys and filter selection are mutually exclusive")
		}
		if s.InputForm != d.CompletionInputFilters || !q.HasSearchFilters() {
			return invalid("search completion requires job predicates")
		}
		if s.BatchSize == 0 {
			s.BatchSize = consts.MaxPISearchSize
		}
	default:
		return invalid("completion selection mode must be explicit or search")
	}
	return nil
}

// validKey preserves the caller's decimal string without float conversion.
func validKey(key string) bool {
	if key == "" || strings.IndexFunc(key, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return false
	}
	value, err := strconv.ParseInt(key, 10, 64)
	return err == nil && value > 0
}

// canonicalKey removes equivalent decimal spellings after validation, without
// floating-point conversion or losing integer precision.
func canonicalKey(key string) string {
	value, _ := strconv.ParseInt(key, 10, 64)
	return strconv.FormatInt(value, 10)
}

// searchQuery maps provenance to adapter predicates; unique limits belong here,
// not in the adapter's raw-row limit handling.
func searchQuery(s d.CompletionSelection) d.JobSearchQuery {
	f := s.Filter
	return d.JobSearchQuery{State: f.State, Type: f.Type, ProcessInstanceKey: f.ProcessInstanceKey, ElementInstanceKey: f.ElementInstanceKey, ElementId: f.ElementId, Worker: f.Worker, Retries: toolx.CopyPtr(f.Retries), Kind: f.Kind, ListenerEventType: f.ListenerEventType, BatchSize: s.BatchSize, SortByKey: true}
}

// Matches verifies all original predicates and named discovery tenant locally.
// Explicit keys intentionally retain cross-tenant backend authorization semantics.
func Matches(job d.Job, selection d.CompletionSelection, tenant string) bool {
	if selection.Mode == d.CompletionSelectionExplicit {
		return true
	}
	f := selection.Filter
	return (tenant == "" || job.TenantId == tenant) &&
		(f.State == "" || job.State == f.State) && (f.Type == "" || job.Type == f.Type) &&
		(f.ProcessInstanceKey == "" || job.ProcessInstanceKey == f.ProcessInstanceKey) &&
		(f.ElementInstanceKey == "" || job.ElementInstanceKey == f.ElementInstanceKey) &&
		(f.ElementId == "" || job.ElementId == f.ElementId) && (f.Worker == "" || job.Worker == f.Worker) &&
		(f.Retries == nil || job.Retries == *f.Retries) && (f.Kind == "" || job.Kind == f.Kind) &&
		(f.ListenerEventType == "" || job.ListenerEventType == f.ListenerEventType)
}

// appendCandidate records only available context and validates payload before freezing membership.
func appendCandidate(batch *d.PreparedCompletionBatch, job d.Job) error {
	if !validKey(job.Key) {
		return fmt.Errorf("%w: invalid candidate key", d.ErrMalformedResponse)
	}
	if err := common.ValidateJobCompletionPayload(job, batch.Variables); err != nil {
		return err
	}
	batch.Candidates = append(batch.Candidates, d.CompletionCandidate{Key: job.Key, ProcessInstanceKey: job.ProcessInstanceKey, ElementInstanceKey: job.ElementInstanceKey, ElementId: job.ElementId, TenantId: job.TenantId, Type: job.Type, Kind: job.Kind, ListenerEventType: job.ListenerEventType, State: job.State, Worker: job.Worker, Retries: job.Retries})
	if !slices.Contains(batch.TenantIDs, job.TenantId) {
		batch.TenantIDs = append(batch.TenantIDs, job.TenantId)
	}
	return nil
}

// prepareExplicit completes key-only reads in input order; any failure blocks the whole batch.
func prepareExplicit(ctx context.Context, batch *d.PreparedCompletionBatch, ports ReadPorts, opts ...services.CallOption) error {
	seen := map[string]bool{}
	for _, key := range batch.Selection.Keys {
		if seen[key] {
			continue
		}
		seen[key] = true
		if err := ctx.Err(); err != nil {
			return err
		}
		if ports.Get == nil {
			return fmt.Errorf("%w: missing job lookup port", d.ErrUnsupported)
		}
		job, err := ports.Get(ctx, key, opts...)
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		batch.ObservedCount++
		if job.Key != key {
			return fmt.Errorf("%w: job lookup returned a different key", d.ErrMalformedResponse)
		}
		if err = appendCandidate(batch, job); err != nil {
			return err
		}
	}
	batch.CompletionReason = d.CompletionReasonExplicitInputsRead
	return nil
}

// prepareSearch advances by raw evidence and bounds empty cursor runs. Deduplication
// occurs only after scope matching, and every unique match consumes the target cap.
func prepareSearch(ctx context.Context, batch *d.PreparedCompletionBatch, backoff config.BackoffConfig, ports ReadPorts, opts ...services.CallOption) error {
	if ports.Search == nil {
		return fmt.Errorf("%w: missing job search port", d.ErrUnsupported)
	}
	_ = backoff.Normalize()
	query := searchQuery(batch.Selection)
	request := d.JobPageRequest{Size: batch.Selection.BatchSize}
	seen, cursors := map[string]bool{}, map[string]bool{}
	pageCtx := ctx
	var cancelAmbiguity context.CancelFunc
	defer func() {
		if cancelAmbiguity != nil {
			cancelAmbiguity()
		}
	}()
	incomplete := func(detail string) error {
		return fmt.Errorf("%w: incomplete job search: %s", d.ErrMalformedResponse, detail)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, err := ports.Search(pageCtx, query, request, opts...)
		if pageCtx.Err() != nil {
			return fmt.Errorf("%w: incomplete job search: %w", d.ErrMalformedResponse, pageCtx.Err())
		}
		if err != nil {
			return err
		}
		batch.PageCount++
		batch.ObservedCount += int64(page.RawItemCount)
		if page.RawItemCount < 0 || page.NextFrom < request.From || page.Request != request || len(page.Items) > int(page.RawItemCount) {
			return incomplete("invalid raw page evidence")
		}
		consumed := page.NextFrom - request.From
		if consumed > min(page.RawItemCount, request.Size) ||
			(page.RawItemCount <= request.Size && consumed != page.RawItemCount) ||
			len(page.Items) > int(consumed) {
			return incomplete("inconsistent raw advancement")
		}
		if page.OverflowState != d.ProcessInstanceOverflowStateHasMore && page.OverflowState != d.ProcessInstanceOverflowStateNoMore {
			return incomplete("unknown continuation")
		}
		if total := page.ReportedTotal; total != nil {
			switch total.Kind {
			case d.JobReportedTotalKindExact:
				if total.Count < int64(page.NextFrom) ||
					(page.OverflowState == d.ProcessInstanceOverflowStateNoMore && total.Count != int64(page.NextFrom)) {
					return incomplete("contradictory exact total")
				}
			case d.JobReportedTotalKindLowerBound:
				if total.Count < 0 || (page.OverflowState == d.ProcessInstanceOverflowStateNoMore &&
					(consumed != 0 || int64(page.NextFrom) <= total.Count)) {
					return incomplete("unproven lower-bound exhaustion")
				}
			default:
				return incomplete("unknown total certainty")
			}
		}
		if page.EndCursor != "" {
			if page.EndCursor == request.After || cursors[page.EndCursor] {
				return incomplete("repeated cursor")
			}
			cursors[page.EndCursor] = true
		}
		if page.OverflowState == d.ProcessInstanceOverflowStateHasMore && page.EndCursor == "" && page.NextFrom <= request.From {
			return incomplete("nonadvancing continuation")
		}
		if page.RawItemCount > 0 && cancelAmbiguity != nil {
			cancelAmbiguity()
			cancelAmbiguity = nil
			pageCtx = ctx
		}
		for _, job := range page.Items {
			if !Matches(job, batch.Selection, batch.DiscoveryTenant) || seen[job.Key] {
				continue
			}
			if err = appendCandidate(batch, job); err != nil {
				return err
			}
			seen[job.Key] = true
			if batch.Selection.Limit > 0 && len(batch.Candidates) >= int(batch.Selection.Limit) {
				batch.Limited = true
				batch.CompletionReason = d.CompletionReasonLimitReached
				return nil
			}
		}
		if page.OverflowState == d.ProcessInstanceOverflowStateNoMore {
			batch.CompletionReason = d.CompletionReasonExhausted
			return nil
		}
		if page.RawItemCount == 0 && cancelAmbiguity == nil {
			ambiguityCtx, cancel := context.WithTimeout(ctx, backoff.Timeout)
			pageCtx, cancelAmbiguity = ambiguityCtx, cancel
		}
		request.From, request.After = page.NextFrom, page.EndCursor
	}
}
