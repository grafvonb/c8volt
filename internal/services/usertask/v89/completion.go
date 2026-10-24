// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
)

// CompleteUserTask submits native completion under backend authorization, with
// optional data and acceptance evidence only. It never reads, assigns or polls a task.
func (s *Service) CompleteUserTask(ctx context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.UserTaskCompletionResult, error) {
	options := services.ApplyCallOptions(opts)
	snapshot, err := toolx.CopyJSONMap(variables)
	if err != nil {
		err = fmt.Errorf("%w: invalid completion variables: %v", d.ErrValidation, err)
		return d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusNotAttempted, Submission: d.UserTaskCompletionSubmissionNotSubmitted, Error: err.Error()}, err
	}
	body := camunda.CompleteUserTaskJSONRequestBody{}
	if snapshot != nil {
		body.Variables = &snapshot
	}
	var log *slog.Logger
	if options.Verbose && !options.SuppressWorkflowDetailLogs {
		log = s.log
	}
	return common.CompleteUserTaskSubmission(ctx, key, log, func(ctx context.Context) (*http.Response, []byte, error) {
		response, err := s.cc.CompleteUserTaskWithResponse(ctx, camunda.UserTaskKey(key), body)
		if response == nil {
			return nil, nil, err
		}
		return response.HTTPResponse, response.Body, err
	})
}
