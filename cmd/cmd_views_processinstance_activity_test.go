// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

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
