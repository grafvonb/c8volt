// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProcessInstanceResumptionSerialization fixes the evidence vocabulary and
// distinguishes absent observations from UNKNOWN without inventing timestamps.
func TestProcessInstanceResumptionSerialization(t *testing.T) {
	t.Parallel()
	preview := ProcessInstanceResumptionResult{
		Key: "2251799813685249", Status: ProcessInstanceResumptionStatusPreview,
		Submission:   ProcessInstanceResumptionSubmissionNotSubmitted,
		Confirmation: ProcessInstanceResumptionConfirmationNotRequested,
	}
	data, err := json.Marshal(preview)
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"2251799813685249","status":"preview","submission":"not_submitted","confirmation":"not_requested","hasObservation":false}`, string(data))
	observed := ProcessInstanceResumptionResult{
		Key: preview.Key, Status: ProcessInstanceResumptionStatusTimedOut,
		Submission:   ProcessInstanceResumptionSubmissionAccepted,
		Confirmation: ProcessInstanceResumptionConfirmationUnconfirmed,
		StatusCode:   204, HasObservation: true, LastObservedState: StateUnknown,
		SuspendedDate: "supplied-date", Error: "machine evidence",
	}
	data, err = json.Marshal(observed)
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"2251799813685249","status":"timed_out","submission":"accepted","confirmation":"unconfirmed","statusCode":204,"hasObservation":true,"lastObservedState":"UNKNOWN","suspendedDate":"supplied-date","error":"machine evidence"}`, string(data))
	var roundTrip ProcessInstanceResumptionResult
	require.NoError(t, json.Unmarshal(data, &roundTrip))
	require.Equal(t, observed, roundTrip)
	observed.SuspendedDate = ""
	data, err = json.Marshal(observed)
	require.NoError(t, err)
	require.NotContains(t, string(data), "suspendedDate")
	data, err = json.Marshal(ProcessInstanceResumptionResults{Items: []ProcessInstanceResumptionResult{}})
	require.NoError(t, err)
	require.JSONEq(t, `{"items":[]}`, string(data))

	data, err = json.Marshal([]ProcessInstanceResumptionStatus{
		ProcessInstanceResumptionStatusPreview, ProcessInstanceResumptionStatusAccepted,
		ProcessInstanceResumptionStatusConfirmed, ProcessInstanceResumptionStatusFailed,
		ProcessInstanceResumptionStatusTimedOut, ProcessInstanceResumptionStatusInterrupted,
		ProcessInstanceResumptionStatusNotAttempted,
	})
	require.NoError(t, err)
	require.JSONEq(t, `["preview","accepted","confirmed","failed","timed_out","interrupted","not_attempted"]`, string(data))
	data, err = json.Marshal([]ProcessInstanceResumptionSubmission{
		ProcessInstanceResumptionSubmissionNotSubmitted, ProcessInstanceResumptionSubmissionAccepted,
		ProcessInstanceResumptionSubmissionRejected, ProcessInstanceResumptionSubmissionUnknown,
	})
	require.NoError(t, err)
	require.JSONEq(t, `["not_submitted","accepted","rejected","unknown"]`, string(data))
	data, err = json.Marshal([]ProcessInstanceResumptionConfirmation{
		ProcessInstanceResumptionConfirmationNotRequested, ProcessInstanceResumptionConfirmationUnconfirmed,
		ProcessInstanceResumptionConfirmationConfirmed,
	})
	require.NoError(t, err)
	require.JSONEq(t, `["not_requested","unconfirmed","confirmed"]`, string(data))
}
