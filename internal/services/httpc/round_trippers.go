// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
	"github.com/grafvonb/c8volt/toolx/logging"
)

type LogTransport struct {
	base     http.RoundTripper
	WithBody bool
	Log      *slog.Logger
	Activity logging.ActivitySink
}

func (t *LogTransport) rt() http.RoundTripper {
	if t.base != nil {
		return t.base
	}
	if t.Log == nil {
		t.Log = slog.Default()
	}
	return http.DefaultTransport
}

func (t *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Activity != nil {
		t.Activity.StartActivity(httpActivityMessage(req))
		defer t.Activity.StopActivity()
	}
	if t.WithBody {
		// clone body to avoid consuming it
		var bodyCopy []byte
		if req.Body != nil {
			bodyCopy, _ = httputil.DumpRequestOut(req, true)
		} else {
			bodyCopy, _ = httputil.DumpRequestOut(req, false)
		}
		// restore body if needed
		if req.Body != nil && len(bodyCopy) > 0 {
			// DumpRequestOut already reads body, so rebuild it
			req.Body = io.NopCloser(bytes.NewReader(extractBody(bodyCopy)))
		}
		t.Log.Debug(string(bodyCopy))
		return t.rt().RoundTrip(req)
	}
	t.Log.Debug(fmt.Sprintf("calling: %s %s", req.Method, req.URL.String()))
	return t.rt().RoundTrip(req)
}

func httpActivityMessage(req *http.Request) string {
	if req == nil {
		return "calling Camunda API"
	}
	method := req.Method
	path := ""
	if req.URL != nil {
		path = strings.Trim(req.URL.Path, "/")
	}
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		return httpActivityFallbackMessage(method)
	}
	if parts[0] == "v2" {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return httpActivityFallbackMessage(method)
	}
	switch parts[0] {
	case "topology":
		return "checking cluster topology"
	case "process-instances":
		return processInstanceHTTPActivityMessage(method, parts)
	case "incidents":
		return incidentHTTPActivityMessage(method, parts)
	case "jobs":
		return jobHTTPActivityMessage(method, parts)
	case "process-definitions":
		return processDefinitionHTTPActivityMessage(method, parts)
	case "resources":
		return resourceHTTPActivityMessage(method, parts)
	default:
		return httpActivityFallbackMessage(method)
	}
}

func processInstanceHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" {
		return "searching process instances"
	}
	if len(parts) == 1 && method == http.MethodPost {
		return "creating process instance"
	}
	if len(parts) >= 3 && parts[2] == "incidents" {
		return "searching process-instance incidents"
	}
	if len(parts) >= 3 && parts[2] == "deletion" {
		return "submitting process-instance deletion"
	}
	if len(parts) >= 3 && parts[2] == "cancellation" {
		return "submitting process-instance cancellation"
	}
	if len(parts) >= 2 && method == http.MethodGet {
		return "loading process instance"
	}
	return httpActivityFallbackMessage(method)
}

func incidentHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" {
		return "searching incidents"
	}
	if len(parts) >= 3 && parts[2] == "resolution" {
		return "submitting incident resolution"
	}
	if len(parts) >= 2 && method == http.MethodGet {
		return "loading incident"
	}
	return httpActivityFallbackMessage(method)
}

func jobHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" {
		return "searching jobs"
	}
	if len(parts) >= 2 && method == http.MethodPatch {
		return "updating job"
	}
	if len(parts) >= 2 && method == http.MethodGet {
		return "loading job"
	}
	return httpActivityFallbackMessage(method)
}

func processDefinitionHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" {
		return "searching process definitions"
	}
	if len(parts) >= 3 && parts[2] == "xml" {
		return "loading process-definition XML"
	}
	if len(parts) >= 2 && method == http.MethodGet {
		return "loading process definition"
	}
	if len(parts) >= 3 && parts[2] == "deletion" {
		return "submitting process-definition deletion"
	}
	return httpActivityFallbackMessage(method)
}

func resourceHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 1 {
		return "loading resources"
	}
	if len(parts) >= 2 && method == http.MethodGet {
		return "loading resource"
	}
	return httpActivityFallbackMessage(method)
}

func httpActivityFallbackMessage(method string) string {
	switch method {
	case http.MethodGet:
		return "loading Camunda API data"
	case http.MethodPost:
		return "submitting Camunda API request"
	case http.MethodPatch, http.MethodPut:
		return "updating Camunda API resource"
	case http.MethodDelete:
		return "deleting Camunda API resource"
	default:
		return "calling Camunda API"
	}
}

// helper to extract body part from DumpRequestOut output
func extractBody(dump []byte) []byte {
	parts := bytes.SplitN(dump, []byte("\r\n\r\n"), 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return nil
}

type AuthTransport struct {
	base   http.RoundTripper
	Editor authenticator.RequestEditor
}

func (t *AuthTransport) rt() http.RoundTripper {
	if t.base != nil {
		return t.base
	}
	return http.DefaultTransport
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Editor != nil {
		if err := t.Editor(req.Context(), req); err != nil {
			return nil, err
		}
	}
	return t.rt().RoundTrip(req)
}
