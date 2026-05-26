// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestOneLineJob_RendersCompactRowWithFullErrorMessageByDefault(t *testing.T) {
	deadline := time.Date(2026, 4, 23, 1, 7, 49, 0, time.UTC)
	message := "Process instance could not be deleted. Error: Failed DELETE to https://example.invalid/orchestration/v1/process-instances/6755399441384051"
	flagGetErrorMessageLimit = 0
	t.Cleanup(func() { flagGetErrorMessageLimit = 0 })

	line := oneLineJob(job.Job{
		Key:                "2251799814014237",
		State:              "FAILED",
		Retries:            0,
		Deadline:           &deadline,
		ProcessInstanceKey: "2251799814014230",
		ElementInstanceKey: "2251799814014236",
		ErrorMessage:       message,
		TenantId:           "tenant-a",
	})

	require.Equal(t, "2251799814014237 tenant-a FAILED pi:2251799814014230 ei:2251799814014236 r:0 d:2026-04-23T01:07:49.000 err:"+message, line)
}

func TestOneLineJob_RendersDiscoveryFields(t *testing.T) {
	line := oneLineJob(job.Job{
		Key:                "2251799817814347",
		State:              "CREATED",
		Retries:            0,
		Type:               "C88StabilityServiceTaskWorker",
		Kind:               "BPMN_ELEMENT",
		ListenerEventType:  "UNSPECIFIED",
		ProcessInstanceKey: "2251799817814342",
		ElementInstanceKey: "2251799817814346",
		ElementId:          "StabilityServiceTask_ServiceTask",
		TenantId:           "<default>",
	})

	require.Equal(t, "2251799817814347 <default> BPMN_ELEMENT StabilityServiceTask_ServiceTask CREATED tp:C88StabilityServiceTaskWorker lsnr:UNSPECIFIED pi:2251799817814342 ei:2251799817814346 r:0", line)
}

func TestOneLineJob_TruncatesErrorMessageOnlyWhenLimitIsSet(t *testing.T) {
	flagGetErrorMessageLimit = 16
	t.Cleanup(func() { flagGetErrorMessageLimit = 0 })

	line := oneLineJob(job.Job{
		Key:          "2251799814014237",
		Retries:      0,
		ErrorMessage: "Process instance could not be deleted",
	})

	require.Equal(t, "2251799814014237 r:0 err:Process instance...", line)
}

// TestJobsView_RendersSearchRowsAndFoundCount verifies list/search rows use the
// compact job row format and include a final bounded count.
func TestJobsView_RendersSearchRowsAndFoundCount(t *testing.T) {
	cmd, buf := newJobRenderTestCommand()

	err := jobsView(cmd, job.SearchResult{Items: []job.Job{
		{Key: "2251799813711967", State: "FAILED", Retries: 0, TenantId: "tenant-a"},
		{Key: "2251799813711968", State: "CREATED", Retries: 3, TenantId: "tenant-b"},
	}, Limit: 2})

	require.NoError(t, err)
	require.Equal(t, "2251799813711967 tenant-a FAILED  r:0\n2251799813711968 tenant-b CREATED r:3\nfound: 2\n", buf.String())
}

func TestJobsView_AlignsDiscoveryColumns(t *testing.T) {
	cmd, buf := newJobRenderTestCommand()

	err := jobsView(cmd, job.SearchResult{Items: []job.Job{
		{
			Key:                "2251799817814347",
			State:              "CREATED",
			Retries:            0,
			Type:               "C88StabilityServiceTaskWorker",
			Kind:               "BPMN_ELEMENT",
			ListenerEventType:  "UNSPECIFIED",
			ProcessInstanceKey: "2251799817814342",
			ElementInstanceKey: "2251799817814346",
			ElementId:          "StabilityServiceTask_ServiceTask",
			TenantId:           "<default>",
		},
		{
			Key:                "22",
			State:              "FAILED",
			Retries:            3,
			Type:               "short-worker",
			Kind:               "TASK_LISTENER",
			ListenerEventType:  "COMPLETING",
			ProcessInstanceKey: "99",
			ElementInstanceKey: "100",
			ElementId:          "Task",
			TenantId:           "tenant-b",
		},
	}, Limit: 2})

	require.NoError(t, err)
	require.Equal(t, ""+
		"2251799817814347 <default> BPMN_ELEMENT  StabilityServiceTask_ServiceTask CREATED tp:C88StabilityServiceTaskWorker lsnr:UNSPECIFIED pi:2251799817814342 ei:2251799817814346 r:0\n"+
		"22               tenant-b  TASK_LISTENER Task                             FAILED  tp:short-worker                  lsnr:COMPLETING  pi:99               ei:100              r:3\n"+
		"found: 2\n", buf.String())
}

func TestRenderJobSearchPage_AlignsDiscoveryColumns(t *testing.T) {
	cmd, buf := newJobRenderTestCommand()

	err := renderJobSearchPage(cmd, []job.Job{
		{
			Key:                "2251799817814347",
			State:              "CREATED",
			Retries:            0,
			Type:               "C88StabilityServiceTaskWorker",
			Kind:               "BPMN_ELEMENT",
			ListenerEventType:  "UNSPECIFIED",
			ProcessInstanceKey: "2251799817814342",
			ElementInstanceKey: "2251799817814346",
			ElementId:          "StabilityServiceTask_ServiceTask",
			TenantId:           "<default>",
		},
		{
			Key:                "22",
			State:              "FAILED",
			Retries:            3,
			Type:               "short-worker",
			Kind:               "TASK_LISTENER",
			ListenerEventType:  "COMPLETING",
			ProcessInstanceKey: "99",
			ElementInstanceKey: "100",
			ElementId:          "Task",
			TenantId:           "tenant-b",
		},
	})

	require.NoError(t, err)
	require.Equal(t, ""+
		"2251799817814347 <default> BPMN_ELEMENT  StabilityServiceTask_ServiceTask CREATED tp:C88StabilityServiceTaskWorker lsnr:UNSPECIFIED pi:2251799817814342 ei:2251799817814346 r:0\n"+
		"22               tenant-b  TASK_LISTENER Task                             FAILED  tp:short-worker                  lsnr:COMPLETING  pi:99               ei:100              r:3\n", buf.String())
}

// TestJobsView_RendersSearchJSONPayload keeps JSON output as the search result
// document rather than separate per-row JSON fragments.
func TestJobsView_RendersSearchJSONPayload(t *testing.T) {
	cmd, buf := newJobRenderTestCommand()
	flagViewAsJson = true
	t.Cleanup(func() { flagViewAsJson = false })

	err := jobsView(cmd, job.SearchResult{Items: []job.Job{{Key: "2251799813711967", State: "FAILED"}}, Limit: 1})

	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &payload))
	require.Equal(t, float64(1), payload["limit"])
	require.Len(t, payload["items"], 1)
}

// TestJobsView_RendersSearchKeysOnly verifies searched job keys are printed one
// per line with no found-count footer.
func TestJobsView_RendersSearchKeysOnly(t *testing.T) {
	cmd, buf := newJobRenderTestCommand()
	flagViewKeysOnly = true
	t.Cleanup(func() { flagViewKeysOnly = false })

	err := jobsView(cmd, job.SearchResult{Items: []job.Job{
		{Key: "2251799813711967"},
		{Key: "2251799813711968"},
	}, Limit: 2})

	require.NoError(t, err)
	require.Equal(t, "2251799813711967\n2251799813711968\n", buf.String())
}

// newJobRenderTestCommand captures renderer output without constructing the
// whole command tree.
func newJobRenderTestCommand() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{Use: "job"}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd, buf
}
