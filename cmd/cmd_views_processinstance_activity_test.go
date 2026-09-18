// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

func TestProcessInstanceListenerRowsUseSharedTimestampGrammar(t *testing.T) {
	offset := time.FixedZone("UTC+2", 2*60*60)
	creation := time.Date(2026, 9, 16, 13, 7, 16, 359000000, offset)
	end := time.Date(2026, 9, 16, 13, 7, 16, 842000000, offset)
	deadline := time.Date(2026, 9, 16, 13, 8, 0, 0, offset)
	listeners := []process.RuntimeListenerJob{
		{JobKey: "job-complete", Kind: "EXECUTION_LISTENER", ListenerEventType: "END", State: "COMPLETED", Type: "audit", CreationTime: &creation, EndTime: &end, Deadline: &deadline},
		{JobKey: "job-active", Kind: "TASK_LISTENER", ListenerEventType: "COMPLETING", State: "ACTIVATED", Type: "notify", Worker: "worker-a", CreationTime: &creation, Deadline: &deadline, ErrorCode: "E1", ErrorMessage: "failed"},
	}

	lines := formatProcessInstanceElementListenerRows(&listeners, true, deadline)

	require.Len(t, lines, 2)
	require.True(t, strings.HasSuffix(lines[0], "dur:483ms"))
	require.True(t, strings.HasSuffix(lines[1], "dur:43.641s"))
	require.Contains(t, lines[0], "s:2026-09-16T13:07:16.359+02:00 e:2026-09-16T13:07:16.842+02:00")
	require.NotContains(t, lines[0], "d:")
	require.Contains(t, lines[1], "s:2026-09-16T13:07:16.359+02:00")
	require.Contains(t, lines[1], "d:2026-09-16T13:08:00.000+02:00")
	require.Less(t, strings.Index(lines[1], "worker:"), strings.Index(lines[1], "s:"))
	require.Less(t, strings.Index(lines[1], "d:"), strings.Index(lines[1], "ec:"))
}

func TestFormatProcessInstanceActivityElementListenersNestUnderOwningElement(t *testing.T) {
	capturedNow := time.Date(2026, 7, 15, 10, 13, 0, 0, time.UTC)
	listeners := []process.RuntimeListenerJob{
		{JobKey: "job-1", Kind: "TASK_LISTENER", ListenerEventType: "COMPLETING", Type: "audit-user-task", State: "CREATED", Retries: 3, Worker: "audit-worker", ProcessInstanceKey: "pi-1", ElementInstanceKey: "el-1", ElementId: "ReviewOrder"},
	}

	lines, needsWarning := formatProcessInstanceActivityLinesWithElementsWithTimezone("", nil, nil, []process.ProcessInstanceElement{
		{ElementInstanceKey: "el-1", ElementId: "ReviewOrder", Type: "USER_TASK", State: "ACTIVE", StartDate: "2026-07-15T10:12:00Z", ProcessInstanceKey: "pi-1", Listeners: &listeners},
		{ElementInstanceKey: "el-2", ElementId: "ShipOrder", Type: "SERVICE_TASK", State: "ACTIVE", StartDate: "2026-07-15T10:12:30Z", ProcessInstanceKey: "pi-1"},
	}, false, false, 0, false, capturedNow)

	require.False(t, needsWarning)
	require.Len(t, lines, 5)
	require.Equal(t, "└─ elements:", lines[0])
	require.Contains(t, lines[1], "   ├─ el-1")
	require.Equal(t, "   │  └─ listeners:", lines[2])
	require.Contains(t, lines[3], "   │     └─ job-1")
	require.Contains(t, lines[3], "TASK_LISTENER")
	require.Contains(t, lines[3], "lsnr:COMPLETING")
	require.Contains(t, lines[3], "r:3")
	require.Contains(t, lines[4], "└─ el-2")
}

func TestProcessInstanceActivityJSONIncludesEmptyListenersOnlyWhenRequested(t *testing.T) {
	requested := []process.RuntimeListenerJob{}
	payload := processInstanceActivityItem{
		Item: process.ProcessInstance{Key: "pi-1"},
		Elements: []process.ProcessInstanceElement{
			{ElementInstanceKey: "el-1", Listeners: &requested},
			{ElementInstanceKey: "el-2"},
		},
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	require.JSONEq(t, `{
		"item": {"key": "pi-1"},
		"elements": [
			{"elementInstanceKey": "el-1", "hasIncident": false, "listeners": []},
			{"elementInstanceKey": "el-2", "hasIncident": false}
		]
	}`, string(data))
}
