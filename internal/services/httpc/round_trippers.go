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
		stopActivity := startHTTPActivity(t.Activity, httpActivityMessage(req))
		defer stopActivity()
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

// startHTTPActivity records request activity as the lowest-priority fallback scope.
func startHTTPActivity(activity logging.ActivitySink, msg string) func() {
	if priorityActivity, ok := activity.(logging.PriorityActivitySink); ok {
		return priorityActivity.StartActivityWithImportance(msg, logging.ActivityImportanceHTTP)
	}
	activity.StartActivity(msg)
	return activity.StopActivity
}

// httpActivityMessage returns compact resource-aware activity text for known Camunda paths.
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
		if len(parts) == 1 && method == http.MethodGet {
			return "checking cluster topology"
		}
	case "license":
		if len(parts) == 1 && method == http.MethodGet {
			return "loading license"
		}
	case "deployments":
		if len(parts) == 1 && method == http.MethodPost {
			return "deploying resources"
		}
	case "process-instances":
		return processInstanceHTTPActivityMessage(method, parts)
	case "incidents":
		return incidentHTTPActivityMessage(method, parts)
	case "jobs":
		return jobHTTPActivityMessage(method, parts)
	case "batch-operations":
		return batchOperationHTTPActivityMessage(method, parts)
	case "element-instances":
		return elementInstanceHTTPActivityMessage(method, parts)
	case "variables":
		return variableHTTPActivityMessage(method, parts)
	case "user-tasks":
		return userTaskHTTPActivityMessage(method, parts)
	case "tenants":
		return tenantHTTPActivityMessage(method, parts)
	case "process-definitions":
		return processDefinitionHTTPActivityMessage(method, parts)
	case "resources":
		return resourceHTTPActivityMessage(method, parts)
	}
	return httpActivityFallbackMessage(method)
}

// processInstanceHTTPActivityMessage labels process-instance request families without exposing raw request details.
func processInstanceHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" && method == http.MethodPost {
		return "searching process instances"
	}
	if len(parts) == 1 && method == http.MethodPost {
		return "creating process instance"
	}
	if len(parts) == 4 && parts[2] == "incidents" && parts[3] == "search" && method == http.MethodPost {
		return "searching process-instance incidents"
	}
	if len(parts) == 3 && parts[2] == "deletion" && method == http.MethodPost {
		return "submitting process-instance deletion"
	}
	if len(parts) == 3 && parts[2] == "cancellation" && method == http.MethodPost {
		return "submitting process-instance cancellation"
	}
	if len(parts) == 2 && method == http.MethodGet {
		return "loading process instance"
	}
	return httpActivityFallbackMessage(method)
}

// incidentHTTPActivityMessage labels incident request families without exposing raw request details.
func incidentHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" && method == http.MethodPost {
		return "searching incidents"
	}
	if len(parts) == 3 && parts[2] == "resolution" && method == http.MethodPost {
		return "submitting incident resolution"
	}
	if len(parts) == 2 && method == http.MethodGet {
		return "loading incident"
	}
	return httpActivityFallbackMessage(method)
}

// jobHTTPActivityMessage labels job request families without exposing raw request details.
func jobHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" && method == http.MethodPost {
		return "searching jobs"
	}
	if len(parts) == 2 && method == http.MethodPatch {
		return "updating job"
	}
	if len(parts) == 2 && method == http.MethodGet {
		return "loading job"
	}
	return httpActivityFallbackMessage(method)
}

// batchOperationHTTPActivityMessage labels batch-operation request families without exposing raw request details.
func batchOperationHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" && method == http.MethodPost {
		return "searching batch operations"
	}
	if len(parts) == 2 && method == http.MethodGet {
		return "loading batch operation"
	}
	if len(parts) == 2 && parts[1] == "cancellation" && method == http.MethodPost {
		return "submitting batch-operation cancellation"
	}
	return httpActivityFallbackMessage(method)
}

// elementInstanceHTTPActivityMessage labels element-instance request families without exposing raw request details.
func elementInstanceHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" && method == http.MethodPost {
		return "searching element instances"
	}
	if len(parts) == 2 && method == http.MethodGet {
		return "loading element instance"
	}
	if len(parts) == 4 && parts[2] == "incidents" && parts[3] == "search" && method == http.MethodPost {
		return "searching element-instance incidents"
	}
	if len(parts) == 3 && parts[2] == "variables" && method == http.MethodPut {
		return "setting element variables"
	}
	return httpActivityFallbackMessage(method)
}

// variableHTTPActivityMessage labels variable request families without exposing raw request details.
func variableHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" && method == http.MethodPost {
		return "searching variables"
	}
	if len(parts) == 2 && method == http.MethodGet {
		return "loading variable"
	}
	return httpActivityFallbackMessage(method)
}

// userTaskHTTPActivityMessage labels user-task request families without exposing raw request details.
func userTaskHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" && method == http.MethodPost {
		return "searching user tasks"
	}
	if len(parts) == 2 && method == http.MethodGet {
		return "loading user task"
	}
	if len(parts) == 2 && method == http.MethodPatch {
		return "updating user task"
	}
	return httpActivityFallbackMessage(method)
}

// tenantHTTPActivityMessage labels tenant request families without exposing raw request details.
func tenantHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" && method == http.MethodPost {
		return "searching tenants"
	}
	if len(parts) == 2 && method == http.MethodGet {
		return "loading tenant"
	}
	return httpActivityFallbackMessage(method)
}

// processDefinitionHTTPActivityMessage labels process-definition request families without exposing raw request details.
func processDefinitionHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 2 && parts[1] == "search" && method == http.MethodPost {
		return "searching process definitions"
	}
	if len(parts) == 3 && parts[2] == "xml" && method == http.MethodGet {
		return "loading process-definition XML"
	}
	if len(parts) == 2 && method == http.MethodGet {
		return "loading process definition"
	}
	if len(parts) == 3 && parts[2] == "deletion" && method == http.MethodPost {
		return "submitting process-definition deletion"
	}
	return httpActivityFallbackMessage(method)
}

// resourceHTTPActivityMessage labels resource request families without exposing raw request details.
func resourceHTTPActivityMessage(method string, parts []string) string {
	if len(parts) == 1 && method == http.MethodGet {
		return "loading resources"
	}
	if len(parts) == 2 && method == http.MethodGet {
		return "loading resource"
	}
	if len(parts) == 3 && parts[2] == "deletion" && method == http.MethodPost {
		return "submitting resource deletion"
	}
	return httpActivityFallbackMessage(method)
}

// httpActivityFallbackMessage keeps generic request wording for unknown Camunda operations.
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
