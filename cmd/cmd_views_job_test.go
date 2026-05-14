// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/job"
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

	require.Equal(t, "2251799814014237 tenant-a FAILED pi:2251799814014230 ei:2251799814014236 r:0 d:2026-04-23T01:07:49+00:00 err:"+message, line)
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
