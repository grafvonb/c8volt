// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"encoding/json"
	"testing"
	"time"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestConvertProcessInstanceSuspendedDate preserves fractional timestamps in UTC
// and treats absent, null and zero dates as missing without inferring state.
func TestConvertProcessInstanceSuspendedDate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, field, want string }{
		{"non-UTC fractional", `,"suspendedDate":"2026-10-17T10:30:00.123456789+02:00"`, "2026-10-17T08:30:00.123456789Z"},
		{"UTC", `,"suspendedDate":"2026-10-17T08:30:00Z"`, "2026-10-17T08:30:00Z"},
		{"absent", "", ""},
		{"null", `,"suspendedDate":null`, ""},
		{"zero", `,"suspendedDate":"0001-01-01T00:00:00Z"`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var source camundav810.ProcessInstanceResult
			require.NoError(t, json.Unmarshal([]byte(`{"processInstanceKey":"123","state":"SUSPENDED"`+tc.field+`}`), &source))
			got := fromProcessInstanceResult(source)
			require.Equal(t, tc.want, got.SuspendedDate)
			require.Equal(t, d.StateSuspended, got.State)
			require.Empty(t, got.EndDate)
		})
	}
	// Replacement must clear a prior value, including when the state stays suspended.
	source := camundav810.ProcessInstanceResult{State: camundav810.ProcessInstanceStateEnum("SUSPENDED"), SuspendedDate: new(time.Date(2026, 10, 17, 8, 30, 0, 0, time.UTC))}
	got := fromProcessInstanceResult(source)
	require.NotEmpty(t, got.SuspendedDate)
	source.SuspendedDate = nil
	got = fromProcessInstanceResult(source)
	require.Empty(t, got.SuspendedDate)
	require.Equal(t, d.StateSuspended, got.State)
}

// TestCombinedStatePageOverflow retains capped lower-bound evidence even on a
// short page; neither a boundary cursor nor a short capped page proves completion.
func TestCombinedStatePageOverflow(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		total  int64
		capped bool
		from   int32
		count  int
		want   d.ProcessInstanceOverflowState
	}{
		{"exact final", 3, false, 2, 1, d.ProcessInstanceOverflowStateNoMore},
		{"exact remaining", 4, false, 2, 1, d.ProcessInstanceOverflowStateHasMore},
		{"capped short at bound", 3, true, 2, 1, d.ProcessInstanceOverflowStateHasMore},
		{"capped full past bound", 2, true, 2, 2, d.ProcessInstanceOverflowStateHasMore},
		{"capped short past bound", 2, true, 2, 1, d.ProcessInstanceOverflowStateIndeterminate},
		{"inconsistent zero", 0, false, 0, 1, d.ProcessInstanceOverflowStateIndeterminate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := camundav810.SearchQueryPageResponse{TotalItems: tc.total, HasMoreTotalItems: tc.capped, EndCursor: new(camundav810.EndCursor("next"))}
			require.Equal(t, tc.want, pickProcessInstanceOverflowState(page, d.ProcessInstancePageRequest{From: tc.from, Size: 2, After: "prev"}, tc.count))
		})
	}
}
