package process

type ProcessDefinition struct {
	BpmnProcessId     string                       `json:"bpmnProcessId,omitempty"`
	Key               string                       `json:"key,omitempty"`
	Name              string                       `json:"name,omitempty"`
	TenantId          string                       `json:"tenantId,omitempty"`
	ProcessVersion    int32                        `json:"processVersion,omitempty"`
	ProcessVersionTag string                       `json:"processVersionTag,omitempty"`
	Statistics        *ProcessDefinitionStatistics `json:"statistics,omitempty"`
}

type ProcessDefinitionStatistics struct {
	Active    int64 `json:"active"`
	Canceled  int64 `json:"canceled"`
	Completed int64 `json:"completed"`
	Incidents int64 `json:"incidents"`
}

type ProcessDefinitions struct {
	Total int32               `json:"total,omitempty"`
	Items []ProcessDefinition `json:"items,omitempty"`
}

type ProcessDefinitionFilter struct {
	Key               string `json:"key,omitempty"`
	BpmnProcessId     string `json:"bpmnProcessId,omitempty"`
	ProcessVersion    int32  `json:"processVersion,omitempty"`
	ProcessVersionTag string `json:"processVersionTag,omitempty"`
}

type ProcessInstanceData struct {
	BpmnProcessId               string                 `json:"bpmnProcessId,omitempty"`               // ProcessDefinitionId in API
	ProcessDefinitionSpecificId string                 `json:"processDefinitionSpecificId,omitempty"` // ProcessDefinitionKey in API
	ProcessDefinitionVersion    int32                  `json:"processDefinitionVersion,omitempty"`
	Variables                   map[string]interface{} `json:"variables,omitempty"`
	TenantId                    string                 `json:"tenantId,omitempty"`
}

type ProcessInstance struct {
	BpmnProcessId             string                 `json:"bpmnProcessId,omitempty"`
	EndDate                   string                 `json:"endDate,omitempty"`
	Incident                  bool                   `json:"incident,omitempty"`
	Key                       string                 `json:"key,omitempty"`
	ParentFlowNodeInstanceKey string                 `json:"parentFlowNodeInstanceKey,omitempty"`
	ParentKey                 string                 `json:"parentKey,omitempty"`
	ParentProcessInstanceKey  string                 `json:"parentProcessInstanceKey,omitempty"`
	ProcessDefinitionKey      string                 `json:"processDefinitionKey,omitempty"`
	ProcessVersion            int32                  `json:"processVersion,omitempty"`
	ProcessVersionTag         string                 `json:"processVersionTag,omitempty"`
	StartDate                 string                 `json:"startDate,omitempty"`
	State                     State                  `json:"state,omitempty"`
	TenantId                  string                 `json:"tenantId,omitempty"`
	Variables                 map[string]interface{} `json:"variables,omitempty"`
}

type ProcessInstances struct {
	Total int32             `json:"total,omitempty"`
	Items []ProcessInstance `json:"items,omitempty"`
}

type ProcessInstanceFilter struct {
	Key                  string `json:"key,omitempty"`
	BpmnProcessId        string `json:"bpmnProcessId,omitempty"`
	ProcessVersion       int32  `json:"version,omitempty"`
	ProcessVersionTag    string `json:"versionTag,omitempty"`
	ProcessDefinitionKey string `json:"processDefinitionKey,omitempty"`
	State                State  `json:"state,omitempty"`
	ParentKey            string `json:"parentKey,omitempty"`
}

type Reporter struct {
	Key        string `json:"key,omitempty"`
	Ok         bool   `json:"ok,omitempty"`
	StatusCode int    `json:"statusCode,omitempty"`
	Status     string `json:"status,omitempty"`
}

func (r Reporter) OK() bool {
	return r.Ok
}

type CancelReport = Reporter

type CancelReports struct {
	Items []CancelReport `json:"items,omitempty"`
}

func (c CancelReports) Totals() (total int, oks int, noks int) {
	return TotalsOf(c.Items)
}

type DeleteReport = Reporter

type DeleteReports struct {
	Items []DeleteReport `json:"items,omitempty"`
}

func (c DeleteReports) Totals() (total int, oks int, noks int) {
	return TotalsOf(c.Items)
}

type OKer interface {
	OK() bool
}

func TotalsOf[T OKer](items []T) (total, oks, noks int) {
	for _, r := range items {
		if r.OK() {
			oks++
		}
	}
	total = len(items)
	noks = total - oks
	return
}
