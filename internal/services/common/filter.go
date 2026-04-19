package common

import (
	"time"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

func NewStringEqFilterPtr(v string) *camundav88.StringFilterProperty {
	if v == "" {
		return nil
	}
	return newFilterPtr(v, (*camundav88.StringFilterProperty).FromStringFilterProperty0)
}

func NewIntegerEqFilterPtr(v int32) *camundav88.IntegerFilterProperty {
	if v == 0 {
		return nil
	}
	return newFilterPtr(v, (*camundav88.IntegerFilterProperty).FromIntegerFilterProperty0)
}

func NewProcessInstanceKeyEqFilterPtr(v string) *camundav88.ProcessInstanceKeyFilterProperty {
	if v == "" {
		return nil
	}
	return newFilterPtr(v, (*camundav88.ProcessInstanceKeyFilterProperty).FromProcessInstanceKeyFilterProperty0)
}

func NewProcessInstanceKeyExistsFilterPtr(exists *bool) *camundav88.ProcessInstanceKeyFilterProperty {
	if exists == nil {
		return nil
	}
	return newFilterPtr(camundav88.AdvancedProcessInstanceKeyFilter{
		Exists: exists,
	}, (*camundav88.ProcessInstanceKeyFilterProperty).FromAdvancedProcessInstanceKeyFilter)
}

func NewProcessInstanceStateEqFilterPtr(v string) *camundav88.ProcessInstanceStateFilterProperty {
	if v == "" {
		return nil
	}
	return newFilterPtr(v, func(f *camundav88.ProcessInstanceStateFilterProperty, s string) error {
		return f.FromProcessInstanceStateFilterProperty0(
			camundav88.ProcessInstanceStateEnum(s),
		)
	})
}

// NewDateTimeRangeFilterPtr builds a datetime range filter from optional lower/upper bounds and exists flag.
// Example: after=2026-01-01T00:00:00Z and before=2026-01-31T23:59:59.999999999Z sets Gte/Lte on the returned filter.
func NewDateTimeRangeFilterPtr(after, before *time.Time, exists *bool) *camundav88.DateTimeFilterProperty {
	if after == nil && before == nil && exists == nil {
		return nil
	}
	return newFilterPtr(camundav88.AdvancedDateTimeFilter{
		Gte:    after,
		Lte:    before,
		Exists: exists,
	}, (*camundav88.DateTimeFilterProperty).FromAdvancedDateTimeFilter)
}

func ProcessInstanceFilterHasTenantSafeLookupFields(filter d.ProcessInstanceFilter) bool {
	return filter.Key != "" ||
		filter.BpmnProcessId != "" ||
		filter.ProcessVersion != 0 ||
		filter.ProcessVersionTag != "" ||
		filter.ParentKey != "" ||
		filter.State != ""
}

func newFilterPtr[T any, D any](v D, init func(*T, D) error) *T {
	var f T
	if err := init(&f, v); err != nil {
		panic(err)
	}
	return &f
}
