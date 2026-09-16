// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRuntimeListenerJobFromJobPreservesLifecycleTimestamps verifies that
// listener projection keeps independently optional recorded times unchanged.
func TestRuntimeListenerJobFromJobPreservesLifecycleTimestamps(t *testing.T) {
	creationTime := time.Date(2026, time.September, 16, 13, 7, 16, 359000000, time.FixedZone("UTC+05:30", 5*60*60+30*60))
	endTime := time.Date(2026, time.September, 16, 13, 7, 16, 842000000, time.FixedZone("UTC-07:00", -7*60*60))
	deadline := time.Date(2026, time.September, 16, 13, 8, 0, 0, time.UTC)

	tests := []struct {
		name         string
		creationTime *time.Time
		endTime      *time.Time
	}{
		{name: "both timestamps", creationTime: &creationTime, endTime: &endTime},
		{name: "creation time only", creationTime: &creationTime},
		{name: "end time only", endTime: &endTime},
		{name: "neither timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := Job{
				Key:          "job-1",
				State:        "COMPLETED",
				CreationTime: tt.creationTime,
				EndTime:      tt.endTime,
				Deadline:     &deadline,
			}

			listener := RuntimeListenerJobFromJob(job)

			require.Equal(t, tt.creationTime, listener.CreationTime)
			require.Equal(t, tt.endTime, listener.EndTime)
			require.Equal(t, &deadline, listener.Deadline)
			if tt.creationTime != nil {
				_, sourceOffset := tt.creationTime.Zone()
				_, projectedOffset := listener.CreationTime.Zone()
				require.True(t, listener.CreationTime.Equal(*tt.creationTime))
				require.Equal(t, sourceOffset, projectedOffset)
			}
			if tt.endTime != nil {
				_, sourceOffset := tt.endTime.Zone()
				_, projectedOffset := listener.EndTime.Zone()
				require.True(t, listener.EndTime.Equal(*tt.endTime))
				require.Equal(t, sourceOffset, projectedOffset)
			}
		})
	}
}

// TestJobLifecycleTimestampJSON verifies that domain job representations use
// the stable optional field names without suppressing non-active deadlines.
func TestJobLifecycleTimestampJSON(t *testing.T) {
	creationTime := time.Date(2026, time.September, 16, 13, 7, 16, 359000000, time.FixedZone("UTC+05:30", 5*60*60+30*60))
	endTime := time.Date(2026, time.September, 16, 13, 7, 16, 842000000, time.FixedZone("UTC-07:00", -7*60*60))
	deadline := time.Date(2026, time.September, 16, 13, 8, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name: "job values",
			value: Job{
				State:        "COMPLETED",
				CreationTime: &creationTime,
				EndTime:      &endTime,
				Deadline:     &deadline,
			},
			want: `{
				"state":"COMPLETED",
				"retries":0,
				"creationTime":"2026-09-16T13:07:16.359+05:30",
				"endTime":"2026-09-16T13:07:16.842-07:00",
				"deadline":"2026-09-16T13:08:00Z"
			}`,
		},
		{
			name: "listener values",
			value: RuntimeListenerJob{
				State:        "CANCELED",
				CreationTime: &creationTime,
				EndTime:      &endTime,
				Deadline:     &deadline,
			},
			want: `{
				"state":"CANCELED",
				"retries":0,
				"creationTime":"2026-09-16T13:07:16.359+05:30",
				"endTime":"2026-09-16T13:07:16.842-07:00",
				"deadline":"2026-09-16T13:08:00Z"
			}`,
		},
		{name: "job omissions", value: Job{}, want: `{"retries":0}`},
		{name: "listener omissions", value: RuntimeListenerJob{}, want: `{"retries":0}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.value)
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(got))
		})
	}
}

func TestJobUpdateRequestHasUpdates(t *testing.T) {
	retries := int32(3)
	timeout := int64(300000)

	tests := []struct {
		name string
		req  JobUpdateRequest
		want bool
	}{
		{name: "none", req: JobUpdateRequest{}, want: false},
		{name: "retries", req: JobUpdateRequest{Retries: &retries}, want: true},
		{name: "timeout", req: JobUpdateRequest{TimeoutMillis: &timeout}, want: true},
		{name: "both", req: JobUpdateRequest{Retries: &retries, TimeoutMillis: &timeout}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.req.HasUpdates())
		})
	}
}
