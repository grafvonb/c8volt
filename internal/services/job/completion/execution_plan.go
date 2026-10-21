// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package completion

import (
	"fmt"
	"reflect"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
)

// snapshotExecutionBatch detaches mutable plan data before workers or normalization can access it.
func snapshotExecutionBatch(b d.PreparedCompletionBatch) (d.PreparedCompletionBatch, error) {
	b.Selection.Keys = slices.Clone(b.Selection.Keys)
	b.Selection.Filter.Retries = toolx.CopyPtr(b.Selection.Filter.Retries)
	b.Candidates = slices.Clone(b.Candidates)
	b.TenantIDs = slices.Clone(b.TenantIDs)
	variables, err := toolx.CopyJSONMap(b.Variables)
	b.Variables = variables
	if err != nil {
		return b, fmt.Errorf("%w: invalid completion variables: %v", d.ErrValidation, err)
	}
	return b, nil
}

// validateExecutionBatch rejects inconsistent or non-normalized plans before any remote call.
// These invariants detect accidental edits; plans are not durable authorization tokens.
func validateExecutionBatch(b d.PreparedCompletionBatch, tenant string) error {
	invalid := func(detail string) error {
		return fmt.Errorf("%w: invalid prepared completion batch: %s", d.ErrValidation, detail)
	}
	if b.Status != d.CompletionPreparationComplete {
		return invalid("preparation is incomplete")
	}
	if b.DiscoveryTenant != tenant {
		return invalid("discovery tenant changed")
	}
	normalized := b.Selection
	normalized.Keys = slices.Clone(b.Selection.Keys)
	if err := normalizeSelection(&normalized); err != nil {
		return err
	}
	if !reflect.DeepEqual(normalized, b.Selection) {
		return invalid("selection is not normalized")
	}
	if b.PageCount < 0 || b.ObservedCount < int64(len(b.Candidates)) {
		return invalid("inconsistent preparation counts")
	}
	seen := map[string]bool{}
	keys := make([]string, 0, len(b.Candidates))
	tenants := []string{}
	for _, c := range b.Candidates {
		if !validKey(c.Key) || canonicalKey(c.Key) != c.Key || seen[c.Key] {
			return invalid("invalid or duplicate candidate key")
		}
		seen[c.Key] = true
		keys = append(keys, c.Key)
		job := d.Job{Key: c.Key, State: c.State, Type: c.Type, ProcessInstanceKey: c.ProcessInstanceKey, ElementInstanceKey: c.ElementInstanceKey, ElementId: c.ElementId, Worker: c.Worker, Retries: c.Retries, Kind: c.Kind, ListenerEventType: c.ListenerEventType, TenantId: c.TenantId}
		if !Matches(job, b.Selection, b.DiscoveryTenant) {
			return invalid("candidate outside prepared scope")
		}
		if err := common.ValidateJobCompletionPayload(job, b.Variables); err != nil {
			return err
		}
		if !slices.Contains(tenants, c.TenantId) {
			tenants = append(tenants, c.TenantId)
		}
	}
	if !slices.Equal(tenants, b.TenantIDs) {
		return invalid("inconsistent candidate tenants")
	}
	if b.Selection.Mode == d.CompletionSelectionExplicit {
		if b.CompletionReason != d.CompletionReasonExplicitInputsRead || b.Limited || b.PageCount != 0 || b.ObservedCount != int64(len(keys)) || !slices.Equal(toolx.UniqueSlice(b.Selection.Keys), keys) {
			return invalid("inconsistent explicit selection")
		}
	} else {
		if b.PageCount == 0 {
			return invalid("missing discovery evidence")
		}
		limit := int(b.Selection.Limit)
		if b.Limited {
			if b.CompletionReason != d.CompletionReasonLimitReached || limit <= 0 || len(keys) != limit {
				return invalid("inconsistent bounded selection")
			}
		} else if b.CompletionReason != d.CompletionReasonExhausted || (limit > 0 && len(keys) >= limit) {
			return invalid("inconsistent exhausted selection")
		}
	}
	return nil
}
