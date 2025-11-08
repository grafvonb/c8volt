package processdefinition

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/processdefinition/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/processdefinition/v88"
)

var MaxResultSize int32 = 1000

type API interface {
	SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error)
}

var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
