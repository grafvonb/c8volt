// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"log/slog"
	"net/http"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
)

// AssignUserTask submits one protected native assignment under backend authorization, without preflight reads.
func (s *Service) AssignUserTask(ctx context.Context, key string, assignee string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
	options := services.ApplyCallOptions(opts)
	var log *slog.Logger
	if options.Verbose && !options.SuppressWorkflowDetailLogs {
		log = s.log
	}
	allowOverride := options.UserTaskAssignmentOverride
	return common.AssignUserTaskSubmission(ctx, key, log, func(ctx context.Context) (*http.Response, []byte, error) {
		// Each attempt owns its pointers; a client cannot change the frozen policy or identity for a retry.
		identity, override := assignee, allowOverride
		body := camunda.AssignUserTaskJSONRequestBody{Assignee: &identity, AllowOverride: &override}
		response, err := s.cc.AssignUserTaskWithResponse(ctx, camunda.UserTaskKey(key), body)
		if response == nil {
			return nil, nil, err
		}
		return response.HTTPResponse, response.Body, err
	})
}

// UnassignUserTask submits native removal without reads, assignment, completion or polling.
func (s *Service) UnassignUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
	options := services.ApplyCallOptions(opts)
	var log *slog.Logger
	if options.Verbose && !options.SuppressWorkflowDetailLogs {
		log = s.log
	}
	return common.UnassignUserTaskSubmission(ctx, key, log, func(ctx context.Context) (*http.Response, []byte, error) {
		response, err := s.cc.UnassignUserTaskWithResponse(ctx, camunda.UserTaskKey(key))
		if response == nil {
			return nil, nil, err
		}
		return response.HTTPResponse, response.Body, err
	})
}
