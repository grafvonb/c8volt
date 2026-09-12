// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package payload

import (
	"context"
	"net/http"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx/poller"
	"github.com/stretchr/testify/require"
)

// TestReportDeploymentProcessDefinitionCompletions emits one wording-free
// completion fact per returned process-definition key.
func TestReportDeploymentProcessDefinitionCompletions(t *testing.T) {
	var events []d.OpsProgressEvent

	ReportDeploymentProcessDefinitionCompletions(func(event d.OpsProgressEvent) {
		events = append(events, event)
	}, []string{"pd-1", "", "pd-2"}, d.OpsCompletionDispositionSubmitted)

	require.Equal(t, []d.OpsProgressEvent{
		{
			Kind: d.OpsProgressEventKindCompletion,
			Completion: &d.OpsCompletionProgress{
				Phase:        DeploymentProcessDefinitionsPhase,
				CoreResource: DeploymentProcessDefinitionsCoreResource,
				Total:        2,
				Identity:     "pd-1",
				Disposition:  d.OpsCompletionDispositionSubmitted,
			},
		},
		{
			Kind: d.OpsProgressEventKindCompletion,
			Completion: &d.OpsCompletionProgress{
				Phase:        DeploymentProcessDefinitionsPhase,
				CoreResource: DeploymentProcessDefinitionsCoreResource,
				Total:        2,
				Identity:     "pd-2",
				Disposition:  d.OpsCompletionDispositionSubmitted,
			},
		},
	}, events)
}

// TestProcessDefinitionVisibilityPollerReportsFirstVisibility verifies the
// callback uses existing visibility checks and fires at most once per key.
func TestProcessDefinitionVisibilityPollerReportsFirstVisibility(t *testing.T) {
	ctx := context.Background()
	visibleByAttempt := []map[string]int{
		{"pd-1": http.StatusOK, "pd-2": http.StatusNotFound},
		{"pd-1": http.StatusOK, "pd-2": http.StatusOK},
	}
	attempt := 0
	var requested []string
	var visible []string
	poll := NewProcessDefinitionVisibilityPollerWithCallback([]string{"pd-1", "pd-2"}, func(_ context.Context, key string) (*http.Response, error) {
		requested = append(requested, key)
		return &http.Response{StatusCode: visibleByAttempt[attempt][key]}, nil
	}, func(key string) {
		visible = append(visible, key)
	})

	status, err := poll(ctx)
	require.NoError(t, err)
	require.Equal(t, poller.JobPollStatus{
		Success: false,
		Message: "pd not visible; waiting [pd-2]",
	}, status)
	require.Equal(t, []string{"pd-1"}, visible)

	attempt++
	status, err = poll(ctx)
	require.NoError(t, err)
	require.Equal(t, poller.JobPollStatus{
		Success: true,
		Message: "pd visible [pd-1 pd-2]",
	}, status)
	require.Equal(t, []string{"pd-1", "pd-2", "pd-1", "pd-2"}, requested)
	require.Equal(t, []string{"pd-1", "pd-2"}, visible)
}
