package testx

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// Predefined responses for collection endpoints
var collectionResponses = map[string]string{
	"/App/System": `{
    }`,
}

// Predefined responses for single-object endpoints
var objectResponses = map[string]string{
	"/v2/topology": `{
	  "Brokers": [
		{
		  "Host": "camunda-platform-c88-zeebe-0.camunda-platform-c88-zeebe",
		  "NodeId": 0,
		  "Partitions": [
			{
			  "Health": "healthy",
			  "PartitionId": 1,
			  "Role": "leader"
			}
		  ],
		  "Port": 26501,
		  "Version": "8.8.0"
		}
	  ],
	  "ClusterSize": 1,
	  "GatewayVersion": "8.8.0",
	  "PartitionsCount": 1,
	  "ReplicationFactor": 1,
	  "LastCompletedChangeId": ""
	}`,
}

var createResponses = map[string]string{
	"/v2/deployments": `{
	  "tenantId": "customer-service",
	  "deploymentKey": "key-2251799813686749",
	  "deployments": [
		{
		  "processDefinition": {
			"processDefinitionId": "new-account-onboarding-workflow",
			"processDefinitionVersion": 0,
			"resourceName": "string",
			"tenantId": "customer-service",
			"processDefinitionKey": "2251799813686749"
		  },
		  "decisionDefinition": {
			"decisionDefinitionId": "new-hire-onboarding-workflow",
			"version": 0,
			"name": "string",
			"tenantId": "customer-service",
			"decisionRequirementsId": "string",
			"decisionDefinitionKey": "2251799813326547",
			"decisionRequirementsKey": "2251799813683346"
		  },
		  "decisionRequirements": {
			"decisionRequirementsId": "string",
			"version": 0,
			"decisionRequirementsName": "string",
			"tenantId": "customer-service",
			"resourceName": "string",
			"decisionRequirementsKey": "2251799813683346"
		  },
		  "form": {
			"formId": "Form_1nx5hav",
			"version": 0,
			"resourceName": "string",
			"tenantId": "customer-service",
			"formKey": "2251799813684365"
		  },
		  "resource": {
			"resourceId": "string",
			"version": 0,
			"resourceName": "string",
			"tenantId": "customer-service",
			"resourceKey": "2251799813686749"
		  }
		}
	  ]
	}`,
}

var (
	onceFS   sync.Once
	sharedFS *FakeServer
)

type FakeServer struct {
	FS      *httptest.Server
	BaseURL string
}

func NewFakeServer(t *testing.T) *FakeServer {
	t.Helper()
	onceFS.Do(func() {
		fs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				if resp, ok := collectionResponses[r.URL.Path]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(resp))
					return
				}
				if resp, ok := objectResponses[r.URL.Path]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(resp))
					return
				}
				http.NotFound(w, r)
			case http.MethodPost:
				// accept multipart or json; no parsing needed for tests
				if resp, ok := createResponses[r.URL.Path]; ok {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK) // Camunda returns 200 for deployments
					_, _ = w.Write([]byte(resp))
					return
				}
				http.NotFound(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))
		sharedFS = &FakeServer{
			FS:      fs,
			BaseURL: fs.URL,
		}
	})

	require.NotNil(t, sharedFS)
	return sharedFS
}
