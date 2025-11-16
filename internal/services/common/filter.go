package common

import camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"

func NewStringEqFilterPtr(v string) *camundav88.StringFilterProperty {
	if v == "" {
		return nil
	}
	var f camundav88.StringFilterProperty
	_ = f.FromStringFilterProperty0(v)
	return &f
}
