// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"fmt"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

func newJobKeyEqFilterPtr(v string) (*camundav89.JobKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.JobKeyFilterProperty
	if err := f.FromJobKeyFilterProperty0(camundav89.JobKey(v)); err != nil {
		return nil, err
	}
	return &f, nil
}

func newSearchQueryPageRequest(limit int32) camundav89.SearchQueryPageRequest {
	var page camundav89.SearchQueryPageRequest
	from := int32(0)
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	return page
}

func fromJobSearchResult(r camundav89.JobSearchResult) d.Job {
	return d.Job{
		Key:                string(r.JobKey),
		State:              string(r.State),
		Retries:            r.Retries,
		Deadline:           r.Deadline,
		ProcessInstanceKey: string(r.ProcessInstanceKey),
		ElementInstanceKey: string(r.ElementInstanceKey),
		ErrorCode:          stringPtrValue(r.ErrorCode),
		ErrorMessage:       stringPtrValue(r.ErrorMessage),
		TenantId:           string(r.TenantId),
	}
}

func requireSingleJob(items []camundav89.JobSearchResult, key string) (d.Job, error) {
	switch len(items) {
	case 0:
		return d.Job{}, fmt.Errorf("%w: job %s was not found or is not visible to the configured tenant", d.ErrNotFound, key)
	case 1:
		return fromJobSearchResult(items[0]), nil
	default:
		return d.Job{}, fmt.Errorf("%w: get job for key %s returned %d matches", d.ErrMalformedResponse, key, len(items))
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
