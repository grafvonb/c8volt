package v89

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromProcessInstanceResult(r camundav89.ProcessInstanceResult) d.ProcessInstance {
	return d.ProcessInstance{
		BpmnProcessId:             r.ProcessDefinitionId,
		EndDate:                   formatTimePtr(r.EndDate),
		Incident:                  r.HasIncident,
		Key:                       r.ProcessInstanceKey,
		ParentFlowNodeInstanceKey: valueOrEmpty(r.ParentElementInstanceKey),
		ParentKey:                 valueOrEmpty(r.ParentProcessInstanceKey),
		ProcessDefinitionKey:      r.ProcessDefinitionKey,
		ProcessVersion:            r.ProcessDefinitionVersion,
		ProcessVersionTag:         valueOrEmpty(r.ProcessDefinitionVersionTag),
		StartDate:                 formatTime(r.StartDate),
		State:                     d.State(r.State),
		TenantId:                  r.TenantId,
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func formatTimePtr(p *time.Time) string {
	if p == nil {
		return ""
	}
	return formatTime(*p)
}

func toProcessInstanceCreationInstruction(in d.ProcessInstanceData) (camundav89.ProcessInstanceCreationInstruction, error) {
	var instr camundav89.ProcessInstanceCreationInstruction
	switch {
	case in.BpmnProcessId != "":
		err := instr.FromProcessInstanceCreationInstructionById(
			camundav89.ProcessInstanceCreationInstructionById{
				ProcessDefinitionId:      in.BpmnProcessId,
				ProcessDefinitionVersion: normalizeVersion(in.ProcessDefinitionVersion),
				Variables:                toolx.PtrCopyMap(in.Variables),
				TenantId:                 toolx.PtrIf(in.TenantId, ""),
			},
		)
		return instr, err
	case in.ProcessDefinitionSpecificId != "":
		err := instr.FromProcessInstanceCreationInstructionByKey(
			camundav89.ProcessInstanceCreationInstructionByKey{
				ProcessDefinitionKey: in.ProcessDefinitionSpecificId,
				Variables:            toolx.PtrCopyMap(in.Variables),
				TenantId:             toolx.PtrIf(in.TenantId, ""),
			},
		)
		return instr, err
	default:
		return instr, errors.New("provide ProcessDefinitionId or ProcessDefinitionKey")
	}
}

func normalizeVersion(v int32) *int32 {
	latest := int32(-1)
	switch {
	case v == -1:
		return new(latest)
	case v > 0:
		return new(v)
	default:
		return new(latest)
	}
}

func fromCreateProcessInstanceResult(r camundav89.CreateProcessInstanceResult) d.ProcessInstanceCreation {
	return d.ProcessInstanceCreation{
		Key:                      r.ProcessInstanceKey,
		BpmnProcessId:            r.ProcessDefinitionId,
		ProcessDefinitionKey:     r.ProcessDefinitionKey,
		ProcessDefinitionVersion: r.ProcessDefinitionVersion,
		TenantId:                 r.TenantId,
		Variables:                toolx.CopyMap(r.Variables),
		StartConfirmedAt:         "<not available>",
	}
}

type processInstanceSearchQuery struct {
	Filter *processInstanceFilter                              `json:"filter,omitempty"`
	Page   *camundav89.SearchQueryPageRequest                  `json:"page,omitempty"`
	Sort   *[]camundav89.ProcessInstanceSearchQuerySortRequest `json:"sort,omitempty"`
}

type processInstanceFilter struct {
	TenantId                    *camundav89.StringFilterProperty               `json:"tenantId,omitempty"`
	ProcessInstanceKey          *camundav89.ProcessInstanceKeyFilterProperty   `json:"processInstanceKey,omitempty"`
	ProcessDefinitionId         *camundav89.StringFilterProperty               `json:"processDefinitionId,omitempty"`
	ProcessDefinitionVersion    *camundav89.IntegerFilterProperty              `json:"processDefinitionVersion,omitempty"`
	ProcessDefinitionVersionTag *camundav89.StringFilterProperty               `json:"processDefinitionVersionTag,omitempty"`
	StartDate                   *camundav89.DateTimeFilterProperty             `json:"startDate,omitempty"`
	EndDate                     *camundav89.DateTimeFilterProperty             `json:"endDate,omitempty"`
	State                       *camundav89.ProcessInstanceStateFilterProperty `json:"state,omitempty"`
	HasIncident                 *bool                                          `json:"hasIncident,omitempty"`
	ParentProcessInstanceKey    *camundav89.ProcessInstanceKeyFilterProperty   `json:"parentProcessInstanceKey,omitempty"`
}

func (f *processInstanceFilter) isEmpty() bool {
	return f != nil &&
		f.TenantId == nil &&
		f.ProcessInstanceKey == nil &&
		f.ProcessDefinitionId == nil &&
		f.ProcessDefinitionVersion == nil &&
		f.ProcessDefinitionVersionTag == nil &&
		f.StartDate == nil &&
		f.EndDate == nil &&
		f.State == nil &&
		f.HasIncident == nil &&
		f.ParentProcessInstanceKey == nil
}

type processInstanceSearchQueryResult struct {
	Items []camundav89.ProcessInstanceResult `json:"items"`
	Page  camundav89.SearchQueryPageResponse `json:"page"`
}

func decodeSearchProcessInstancesResponse(body []byte, page *camundav89.ProcessInstanceSearchQueryResult) (processInstanceSearchQueryResult, error) {
	if len(bytesTrimSpace(body)) == 0 {
		return processInstanceSearchQueryResult{}, d.ErrMalformedResponse
	}
	var result processInstanceSearchQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return processInstanceSearchQueryResult{}, err
	}
	result.Page = page.Page
	return result, nil
}

func newStringEqFilterPtr(v string) *camundav89.StringFilterProperty {
	if v == "" {
		return nil
	}
	var f camundav89.StringFilterProperty
	if err := f.FromStringFilterProperty0(v); err != nil {
		panic(err)
	}
	return &f
}

func newIntegerEqFilterPtr(v int32) *camundav89.IntegerFilterProperty {
	if v == 0 {
		return nil
	}
	var f camundav89.IntegerFilterProperty
	if err := f.FromIntegerFilterProperty0(v); err != nil {
		panic(err)
	}
	return &f
}

func newProcessInstanceKeyEqFilterPtr(v string) *camundav89.ProcessInstanceKeyFilterProperty {
	if v == "" {
		return nil
	}
	var f camundav89.ProcessInstanceKeyFilterProperty
	if err := f.FromProcessInstanceKeyFilterProperty0(v); err != nil {
		panic(err)
	}
	return &f
}

func newProcessInstanceKeyExistsFilterPtr(exists *bool) *camundav89.ProcessInstanceKeyFilterProperty {
	if exists == nil {
		return nil
	}
	var f camundav89.ProcessInstanceKeyFilterProperty
	if err := f.FromAdvancedProcessInstanceKeyFilter(camundav89.AdvancedProcessInstanceKeyFilter{
		Exists: exists,
	}); err != nil {
		panic(err)
	}
	return &f
}

func newProcessInstanceStateEqFilterPtr(v string) *camundav89.ProcessInstanceStateFilterProperty {
	if v == "" {
		return nil
	}
	var f camundav89.ProcessInstanceStateFilterProperty
	if err := f.FromProcessInstanceStateFilterProperty0(camundav89.ProcessInstanceStateEnum(v)); err != nil {
		panic(err)
	}
	return &f
}

func newDateTimeRangeFilterPtr(after, before *time.Time, exists *bool) *camundav89.DateTimeFilterProperty {
	if after == nil && before == nil && exists == nil {
		return nil
	}
	var f camundav89.DateTimeFilterProperty
	if err := f.FromAdvancedDateTimeFilter(camundav89.AdvancedDateTimeFilter{
		Gte:    after,
		Lte:    before,
		Exists: exists,
	}); err != nil {
		panic(err)
	}
	return &f
}

func valueOrEmpty[T ~string](v *T) T {
	if v == nil {
		return ""
	}
	return *v
}

func bytesTrimSpace(b []byte) []byte {
	return bytes.TrimSpace(b)
}
