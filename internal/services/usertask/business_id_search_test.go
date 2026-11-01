// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask_test

import (
	"context"
	"encoding/json"
	"fmt"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/usertask"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// TestBusinessIDSearchPolicyAndNativePayload checks all public service search routes, omitted input, literal serialization and zero-I/O rejection.
func TestBusinessIDSearchPolicyAndNativePayload(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, tc := range []struct {
			name    string
			value   *string
			invalid bool
		}{
			{"omitted", nil, false}, {"opaque", new(" ORDER,零*? "), false}, {"numeric", new("000123"), false},
			{"long", new(strings.Repeat("界", 300)), false}, {"empty", new(""), true}, {"invalid UTF8", new(string([]byte{0xff})), true},
		} {
			t.Run(string(version)+"/"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					var body struct {
						Filter map[string]any `json:"filter"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					if tc.value == nil {
						require.NotContains(t, body.Filter, "businessId")
					} else {
						require.Equal(t, map[string]any{"businessId": *tc.value}, body.Filter)
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
				}))
				defer server.Close()
				cfg := testConfig()
				cfg.App.CamundaVersion = version
				cfg.APIs.Camunda.BaseURL = server.URL
				api, err := usertask.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
				require.NoError(t, err)
				var want error
				if tc.value != nil && version != toolx.V810 {
					want = d.ErrUnsupported
				} else if tc.invalid {
					want = d.ErrValidation
				}
				ctx := context.Background()

				query := d.UserTaskSearchQuery{BusinessId: tc.value, BatchSize: 1}
				checks := []func() error{
					func() error { _, e := api.SearchUserTasksPage(ctx, query, d.UserTaskPageRequest{Size: 1}); return e },
					func() error { _, e := usertask.SearchUserTasks(ctx, api, query); return e },
					func() error { _, e := usertask.SearchUserTasksPages(ctx, api, query, nil); return e },
					func() error { _, e := usertask.SearchUserTasksTotal(ctx, api, query); return e },
				}

				for _, check := range checks {
					require.ErrorIs(t, check(), want)
				}
				if want != nil {
					require.Zero(t, calls.Load())
				} else {
					require.EqualValues(t, len(checks), calls.Load())
				}

			})
		}
	}
}
