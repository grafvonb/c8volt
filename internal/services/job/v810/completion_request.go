// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
)

// completeJobWithResponse omits unset optional fields that the pinned generated
// 8.10 model serializes as null, which the backend can reject. Explicit values,
// including empty strings, and all other generated fields remain unchanged.
// The generated custom-body method retains response parsing and transport errors;
// this helper never retries, replaces a token, or converts an error into success.
func (s *Service) completeJobWithResponse(ctx context.Context, key camunda.JobKey, body camunda.CompleteJobJSONRequestBody) (*camunda.CompleteJobResponse, error) {
	payload, err := json.Marshal(struct {
		camunda.CompleteJobJSONRequestBody
		BusinessId    *camunda.BusinessId    `json:"businessId,omitempty"`
		JobLeaseToken *camunda.JobLeaseToken `json:"jobLeaseToken,omitempty"`
	}{
		CompleteJobJSONRequestBody: body,
		BusinessId:                 body.BusinessId,
		JobLeaseToken:              body.JobLeaseToken,
	})
	if err != nil {
		return nil, fmt.Errorf("encode job completion request: %w", err)
	}
	return s.c.CompleteJobWithBodyWithResponse(ctx, key, "application/json", bytes.NewReader(payload))
}
