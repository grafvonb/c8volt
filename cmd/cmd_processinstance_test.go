package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// newProcessInstanceSearchCaptureServer starts an IPv4 test server that captures
// search request bodies and returns an empty process-instance search result.
func newProcessInstanceSearchCaptureServer(t *testing.T, requests *[]string) *httptest.Server {
	t.Helper()

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		*requests = append(*requests, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	return srv
}

// decodeCapturedPISearchFilter returns the JSON search filter captured by the
// shared process-instance search server so command tests can assert canonical
// absolute-date fields regardless of how inputs were provided.
func decodeCapturedPISearchFilter(t *testing.T, requests []string) map[string]any {
	t.Helper()

	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok, "expected search request filter object")
	return filter
}

func requireCapturedPISearchDateBound(t *testing.T, filter map[string]any, field string, operator string, want string) {
	t.Helper()

	dateFilter, ok := filter[field].(map[string]any)
	require.True(t, ok, "expected %s filter object", field)
	require.Equal(t, want, dateFilter[operator])
}

func requireCapturedPISearchDateExists(t *testing.T, filter map[string]any, field string) {
	t.Helper()

	dateFilter, ok := filter[field].(map[string]any)
	require.True(t, ok, "expected %s filter object", field)
	require.Equal(t, true, dateFilter["$exists"])
}

func decodeCapturedPISearchPage(t *testing.T, requests []string) map[string]any {
	t.Helper()

	body := decodeSingleRequestJSON(t, requests)
	page, ok := body["page"].(map[string]any)
	require.True(t, ok, "expected search request page object")
	return page
}

func decodeCapturedPISearchRequest(t *testing.T, request string) map[string]any {
	t.Helper()

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(request), &got))
	return got
}
