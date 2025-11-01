package processdefinition

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/processdefinition/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/processdefinition/v88"
)

type API interface {
	SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionSearchFilterOpts, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	GetProcessDefinitionsLatest(ctx context.Context, opts ...services.CallOption) ([]d.ProcessDefinition, error)

	GetProcessDefinitionByKey(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error)
	GetProcessDefinitionByBpmnProcessIdLatest(ctx context.Context, bpmnProcessId string, opts ...services.CallOption) (d.ProcessDefinition, error)
	GetProcessDefinitionVersionsByBpmnProcessId(ctx context.Context, bpmnProcessId string, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	GetProcessDefinitionByBpmnProcessIdAndVersion(ctx context.Context, bpmnProcessId string, version int32, opts ...services.CallOption) (d.ProcessDefinition, error)
}

var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
