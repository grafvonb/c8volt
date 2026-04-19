package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testRelativeDayNowEnv = "C8VOLT_TEST_RELATIVE_DAY_NOW"

func applyRelativeDayNowOverrideFromEnv(t *testing.T) {
	t.Helper()

	now := os.Getenv(testRelativeDayNowEnv)
	if now == "" {
		return
	}

	parsed, err := time.Parse(time.RFC3339, now)
	require.NoError(t, err)

	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return parsed
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})
}

// newProcessInstanceSearchCaptureServer starts an IPv4 test server that captures
// search request bodies and returns an empty process-instance search result.
func newProcessInstanceSearchCaptureServer(t *testing.T, requests *[]string) *httptest.Server {
	t.Helper()

	return newProcessInstanceSearchCaptureServerWithResponses(t, requests, `{"items":[]}`)
}

// newProcessInstanceSearchCaptureServerWithResponses captures each search request
// body and returns the provided JSON responses in order so paging tests can
// assert sequential page fetch behavior without duplicating server scaffolding.
func newProcessInstanceSearchCaptureServerWithResponses(t *testing.T, requests *[]string, responses ...string) *httptest.Server {
	t.Helper()

	served := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		*requests = append(*requests, string(body))
		require.Less(t, served, len(responses), "unexpected extra process-instance search request")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[served]))
		served++
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

func decodeCapturedPISearchRequests(t *testing.T, requests []string) []map[string]any {
	t.Helper()

	decoded := make([]map[string]any, 0, len(requests))
	for _, request := range requests {
		decoded = append(decoded, decodeCapturedPISearchRequest(t, request))
	}
	return decoded
}

func decodeCapturedPISearchPages(t *testing.T, requests []string) []map[string]any {
	t.Helper()

	decoded := decodeCapturedPISearchRequests(t, requests)
	pages := make([]map[string]any, 0, len(decoded))
	for _, request := range decoded {
		page, ok := request["page"].(map[string]any)
		require.True(t, ok, "expected search request page object")
		pages = append(pages, page)
	}
	return pages
}

func decodeCapturedTopLevelPISearchFilters(t *testing.T, requests []string) []map[string]any {
	t.Helper()

	decoded := decodeCapturedPISearchRequests(t, requests)
	filters := make([]map[string]any, 0, len(decoded))
	for _, request := range decoded {
		filter, _ := request["filter"].(map[string]any)
		if filter != nil {
			if _, hasKey := filter["processInstanceKey"]; hasKey {
				continue
			}
			if key, hasKey := filter["key"]; hasKey && key != nil {
				continue
			}
			if _, hasParent := filter["parentProcessInstanceKey"]; hasParent {
				continue
			}
			if parentKey, hasParent := filter["parentKey"]; hasParent && parentKey != nil {
				continue
			}
		}
		filters = append(filters, filter)
	}
	return filters
}

func decodeCapturedTopLevelPISearchPages(t *testing.T, requests []string) []map[string]any {
	t.Helper()

	decoded := decodeCapturedPISearchRequests(t, requests)
	pages := make([]map[string]any, 0, len(decoded))
	for _, request := range decoded {
		filter, _ := request["filter"].(map[string]any)
		if filter != nil {
			if _, hasKey := filter["processInstanceKey"]; hasKey {
				continue
			}
			if key, hasKey := filter["key"]; hasKey && key != nil {
				continue
			}
			if _, hasParent := filter["parentProcessInstanceKey"]; hasParent {
				continue
			}
			if parentKey, hasParent := filter["parentKey"]; hasParent && parentKey != nil {
				continue
			}
		}
		page, ok := request["page"].(map[string]any)
		require.True(t, ok, "expected search request page object")
		pages = append(pages, page)
	}
	return pages
}

func decodeCapturedTopLevelPISearchSizes(t *testing.T, requests []string) []float64 {
	t.Helper()

	decoded := decodeCapturedPISearchRequests(t, requests)
	sizes := make([]float64, 0, len(decoded))
	for _, request := range decoded {
		filter, _ := request["filter"].(map[string]any)
		if filter != nil {
			if _, hasKey := filter["processInstanceKey"]; hasKey {
				continue
			}
			if key, hasKey := filter["key"]; hasKey && key != nil {
				continue
			}
			if _, hasParent := filter["parentProcessInstanceKey"]; hasParent {
				continue
			}
			if parentKey, hasParent := filter["parentKey"]; hasParent && parentKey != nil {
				continue
			}
		}
		size, ok := request["size"].(float64)
		require.True(t, ok, "expected search request size value")
		sizes = append(sizes, size)
	}
	return sizes
}
