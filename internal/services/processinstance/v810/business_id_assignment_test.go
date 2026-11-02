// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/grafvonb/c8volt/config"
	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// assignmentTransport captures native wire requests without opening network connections.
type assignmentTransport func(*http.Request) (*http.Response, error)

// RoundTrip gives the generated client only test-owned responses.
func (f assignmentTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// assignmentService builds bounded observation around the real generated serializer and parser.
func assignmentService(t *testing.T, transport assignmentTransport) *Service {
	t.Helper()
	cc, err := camunda.NewClientWithResponses("http://assignment.invalid", camunda.WithHTTPClient(&http.Client{Transport: transport}))
	require.NoError(t, err)
	cfg := &config.Config{}
	require.NoError(t, cfg.App.Backoff.Normalize())
	cfg.App.Backoff.InitialDelay = time.Millisecond
	cfg.App.Backoff.MaxRetries = 2
	return &Service{cc: cc, cfg: cfg, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// TestBusinessIDAssignmentWireEvidence proves exact key/body and one-shot status classification, including empty success and malformed/lost responses.
func TestBusinessIDAssignmentWireEvidence(t *testing.T) {
	t.Parallel()
	lost := errors.New("response lost")
	for _, tc := range []struct {
		name    string
		code    int
		body    string
		callErr error
		status  string
		cause   error
	}{
		{"accepted without JSON", 204, "", nil, "accepted", nil},
		{"child", 409, `{"detail":"child instance"}`, nil, "rejected", d.ErrConflict},
		{"inactive", 409, `{"detail":"inactive"}`, nil, "rejected", d.ErrConflict},
		{"identical existing BID", 409, `{"detail":"already has identical BID"}`, nil, "rejected", d.ErrConflict},
		{"uniqueness enabled", 409, `{"detail":"uniqueness enabled"}`, nil, "rejected", d.ErrConflict},
		{"rate limited no replay", 429, "", nil, "rejected", d.ErrRateLimited},
		{"proxy", 503, "{}", nil, "unknown", d.ErrUnavailable},
		{"lost response", 0, "", lost, "unknown", lost},
		{"malformed JSON rejection", 409, "{", nil, "unknown", nil},
		{"unexpected success", 200, "{}", nil, "unknown", d.ErrMalformedResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			value := "  Order,AbC001😀  "
			svc := assignmentService(t, func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "/process-instances/2251799813685249/business-id-assignment", req.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				require.Equal(t, map[string]any{"businessId": value}, body)
				if tc.callErr != nil {
					return nil, tc.callErr
				}
				return &http.Response{StatusCode: tc.code, Header: http.Header{"Content-Type": []string{"application/problem+json"}}, Body: io.NopCloser(strings.NewReader(tc.body)), Request: req}, nil
			})
			out, err := svc.AssignProcessInstanceBusinessID(context.Background(), "2251799813685249", &value, services.WithNoWait())
			require.Equal(t, 1, calls)
			require.Equal(t, tc.status, out.Status)
			require.Equal(t, "skipped", out.Confirmation)
			require.False(t, out.HasObservation)
			if tc.cause != nil {
				require.ErrorIs(t, err, tc.cause)
			} else if tc.status == "unknown" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, value, *out.RequestedBusinessID)
		})
	}
}

// TestBusinessIDAssignmentConfirmation separates acceptance from exact-value observations and never replays on failed confirmation.
func TestBusinessIDAssignmentConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		observed  string
		key       string
		readCode  int
		confirmed bool
		cause     error
	}{
		{"matching", "new", "2251799813685249", 200, true, nil},
		{"stale", "old", "2251799813685249", 200, false, d.ErrGatewayTimeout},
		{"wrong key", "new", "2251799813685250", 200, false, d.ErrMalformedResponse},
		{"read denied", "", "", 403, false, d.ErrForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				writes, reads := 0, 0
				svc := assignmentService(t, func(req *http.Request) (*http.Response, error) {
					code := 204
					body := ""
					require.Equal(t, "2251799813685249", strings.Split(req.URL.Path, "/")[2])
					if req.Method == http.MethodPost {
						writes++
					} else {
						reads++
						require.Equal(t, http.MethodGet, req.Method)
						code = tc.readCode
						body = `{"processInstanceKey":"` + tc.key + `","businessId":"` + tc.observed + `","state":"ACTIVE"}`
					}
					return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
				})
				out, err := svc.AssignProcessInstanceBusinessID(context.Background(), "2251799813685249", new("new"))
				require.Equal(t, 1, writes)
				require.Positive(t, reads)
				if tc.confirmed {
					require.NoError(t, err)
					require.Equal(t, "confirmed", out.Status)
					require.Equal(t, "confirmed", out.Confirmation)
					require.Equal(t, "new", *out.ObservedBusinessID)
				} else {
					require.ErrorIs(t, err, tc.cause)
					require.Equal(t, "accepted", out.Status)
					require.Equal(t, "unconfirmed", out.Confirmation)
				}
				if tc.name == "stale" {
					require.Equal(t, 2, reads)
					require.Equal(t, "old", *out.ObservedBusinessID)
				}
			})
		})
	}
}

// TestBusinessIDAssignmentLocalGuards rejects invalid input/timing before requests and preserves dry-run/no-wait semantics.
func TestBusinessIDAssignmentLocalGuards(t *testing.T) {
	t.Parallel()
	calls := 0
	svc := assignmentService(t, func(req *http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected request") })
	for _, value := range []*string{nil, new(""), new("\t "), new(string([]byte{0xff})), new(strings.Repeat("a", 257))} {
		_, err := svc.AssignProcessInstanceBusinessID(context.Background(), "2251799813685249", value)
		require.ErrorIs(t, err, d.ErrValidation)
	}
	_, err := svc.AssignProcessInstanceBusinessID(context.Background(), "invalid", new("x"))
	require.ErrorIs(t, err, d.ErrValidation)
	for _, value := range []string{"  Mixed,001  ", strings.Repeat("é", 256), strings.Repeat("😀", 128)} {
		out, err := svc.AssignProcessInstanceBusinessID(context.Background(), "2251799813685249", &value, services.WithDryRun())
		require.NoError(t, err)
		require.Equal(t, "not_submitted", out.Status)
		require.Equal(t, value, *out.RequestedBusinessID)
	}
	svc.cfg.App.Backoff.Timeout = 0
	_, err = svc.AssignProcessInstanceBusinessID(context.Background(), "2251799813685249", new("x"))
	require.ErrorIs(t, err, d.ErrValidation)
	_, err = svc.AssignProcessInstanceBusinessID(context.Background(), "2251799813685249", new("x"), services.WithDryRun(), services.WithNoWait())
	require.NoError(t, err)
	require.Zero(t, calls)
}
