package domain

type ProcessInstance struct {
	BpmnProcessId             string
	EndDate                   string
	Incident                  bool
	Key                       string
	ParentFlowNodeInstanceKey string
	ParentKey                 string
	ProcessDefinitionKey      string
	ProcessVersion            int32
	ProcessVersionTag         string
	StartDate                 string
	State                     State
	TenantId                  string
	Variables                 map[string]interface{}
}

type ProcessInstanceFilter struct {
	Key                  string
	BpmnProcessId        string
	ProcessVersion       int32
	ProcessVersionTag    string
	ProcessDefinitionKey string
	State                State
	ParentKey            string
}

type CancelResponse struct {
	StatusCode int
	Status     string
}

type DeleteResponse struct {
	StatusCode int
	Status     string
}

type ProcessInstanceData struct {
	BpmnProcessId               string // ProcessDefinitionId in API
	ProcessDefinitionSpecificId string // ProcessDefinitionKey in API
	ProcessDefinitionVersion    int32
	Variables                   map[string]interface{}
	TenantId                    string
}

type ProcessInstanceCreation struct {
	Key                      string                 `json:"key,omitempty"`
	BpmnProcessId            string                 `json:"bpmnProcessId,omitempty"`        // ProcessDefinitionId in API
	ProcessDefinitionKey     string                 `json:"processDefinitionKey,omitempty"` // ProcessDefinitionKey in API
	ProcessDefinitionVersion int32                  `json:"processDefinitionVersion,omitempty"`
	TenantId                 string                 `json:"tenantId,omitempty"`
	Variables                map[string]interface{} `json:"variables,omitempty"`
	StartDate                string                 `json:"startDate,omitempty"`
	StartConfirmedAt         string                 `json:"startConfirmedAt,omitempty"`
}
