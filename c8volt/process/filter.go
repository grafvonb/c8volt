package process

func (r ProcessInstances) FilterByHavingIncidents(has bool) ProcessInstances {
	return r.filterByBool(func(pi ProcessInstance) bool { return pi.Incident }, has)
}

func (r ProcessInstances) FilterChildrenOnly() ProcessInstances {
	return r.filterByBool(func(pi ProcessInstance) bool { return pi.ParentKey != "" }, true)
}

func (r ProcessInstances) FilterParentsOnly() ProcessInstances {
	return r.filterByBool(func(pi ProcessInstance) bool { return pi.ParentKey == "" }, true)
}

func (r ProcessInstances) filterByBool(pred func(ProcessInstance) bool, want bool) ProcessInstances {
	if len(r.Items) == 0 {
		return r
	}
	out := make([]ProcessInstance, 0, len(r.Items))
	for _, it := range r.Items {
		if pred(it) == want {
			out = append(out, it)
		}
	}
	r.Items = out
	r.Total = int32(len(out))
	return r
}
