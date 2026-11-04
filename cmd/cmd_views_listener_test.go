// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestListenerTimestampColumns enforces independent lifecycle fields and the exact activated deadline rule.
func TestListenerTimestampColumns(t *testing.T) {
	creation := time.Date(2026, 9, 16, 13, 7, 16, 359000000, time.FixedZone("UTC+2", 2*60*60))
	end := time.Date(2026, 9, 16, 13, 7, 16, 842000000, time.FixedZone("UTC+2", 2*60*60))
	deadline := time.Date(2026, 9, 16, 13, 8, 0, 0, time.FixedZone("UTC+2", 2*60*60))

	tests := []struct {
		name       string
		state      string
		creation   *time.Time
		end        *time.Time
		deadline   *time.Time
		showOffset bool
		want       flatRow
	}{
		{name: "completed", state: "COMPLETED", creation: &creation, end: &end, deadline: &deadline, want: flatRow{"s:2026-09-16T13:07:16.359", "e:2026-09-16T13:07:16.842", ""}},
		{name: "activated", state: "ACTIVATED", creation: &creation, deadline: &deadline, want: flatRow{"s:2026-09-16T13:07:16.359", "", "d:2026-09-16T13:08:00.000"}},
		{name: "activated with end", state: "ACTIVATED", creation: &creation, end: &end, deadline: &deadline, want: flatRow{"s:2026-09-16T13:07:16.359", "e:2026-09-16T13:07:16.842", "d:2026-09-16T13:08:00.000"}},
		{name: "canceled", state: "CANCELED", creation: &creation, end: &end, deadline: &deadline, want: flatRow{"s:2026-09-16T13:07:16.359", "e:2026-09-16T13:07:16.842", ""}},
		{name: "created", state: "CREATED", creation: &creation, deadline: &deadline, want: flatRow{"s:2026-09-16T13:07:16.359", "", ""}},
		{name: "failed", state: "FAILED", end: &end, deadline: &deadline, want: flatRow{"", "e:2026-09-16T13:07:16.842", ""}},
		{name: "blank", deadline: &deadline, want: flatRow{"", "", ""}},
		{name: "unfamiliar", state: "PAUSED", creation: &creation, end: &end, deadline: &deadline, want: flatRow{"s:2026-09-16T13:07:16.359", "e:2026-09-16T13:07:16.842", ""}},
		{name: "noncanonical lowercase", state: "activated", creation: &creation, deadline: &deadline, want: flatRow{"s:2026-09-16T13:07:16.359", "", ""}},
		{name: "no timestamps", state: "ACTIVATED", want: flatRow{"", "", ""}},
		{name: "numeric offsets", state: "ACTIVATED", creation: &creation, end: &end, deadline: &deadline, showOffset: true, want: flatRow{"s:2026-09-16T13:07:16.359+02:00", "e:2026-09-16T13:07:16.842+02:00", "d:2026-09-16T13:08:00.000+02:00"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, listenerTimestampColumns(tc.creation, tc.end, tc.deadline, tc.state, tc.showOffset))
		})
	}
}

func TestRuntimeListenerDuration(t *testing.T) {
	creation := time.Date(2026, 9, 16, 5, 38, 52, 855000000, time.UTC)
	end := creation.Add(483 * time.Millisecond)
	now := creation.Add(55*time.Hour + 17*time.Minute + 44*time.Second + 701435*time.Microsecond)
	before := creation.Add(-time.Second)
	zero := time.Time{}
	for _, tc := range []struct {
		name       string
		state      string
		start, end *time.Time
		now        time.Time
		want       string
	}{
		{"created waiting", "CREATED", &creation, nil, now, "55h17m44.701435s"},
		{"activated", "ACTIVATED", &creation, nil, now, "55h17m44.701435s"},
		{"failed awaiting retry", "FAILED", &creation, nil, now, "55h17m44.701435s"},
		{"timed out", "TIMED_OUT", &creation, nil, now, "55h17m44.701435s"},
		{"retries updated", "RETRIES_UPDATED", &creation, nil, now, "55h17m44.701435s"},
		{"completed", "COMPLETED", &creation, &end, now, "483ms"},
		{"canceled", "CANCELED", &creation, &end, now, "483ms"},
		{"end takes precedence", "ACTIVATED", &creation, &end, now, "483ms"},
		{"zero duration", "COMPLETED", &creation, &creation, now, "0s"},
		{"missing creation", "COMPLETED", nil, &end, now, ""},
		{"zero creation", "CREATED", &zero, nil, now, ""},
		{"completed missing end", "COMPLETED", &creation, nil, now, ""},
		{"canceled missing end", "CANCELED", &creation, nil, now, ""},
		{"error thrown missing end", "ERROR_THROWN", &creation, nil, now, ""},
		{"unknown state", "UNKNOWN", &creation, nil, now, ""},
		{"end before creation", "COMPLETED", &creation, &before, now, ""},
		{"future creation", "CREATED", &creation, nil, before, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, runtimeListenerDuration(tc.start, tc.end, tc.state, tc.now))
		})
	}
}
