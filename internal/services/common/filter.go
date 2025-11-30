package common

import (
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
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

func newFilterPtr[T any, D any](v D, init func(*T, D) error) *T {
	var f T
	if err := init(&f, v); err != nil {
		panic(err)
	}
	return &f
}
