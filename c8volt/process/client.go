// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	types "github.com/grafvonb/c8volt/typex"
)

type client struct {
	pdApi  pdsvc.API
	piApi  pisvc.API
	incApi incsvc.API
	log    *slog.Logger
}

// New creates a process facade with incident lookup routed through the incident service layer.
func New(pdApi pdsvc.API, piApi pisvc.API, incApi incsvc.API, log *slog.Logger) API {
	return &client{
		pdApi:  pdApi,
		piApi:  piApi,
		incApi: incApi,
		log:    log,
	}
}

func (c *client) SearchProcessDefinitions(ctx context.Context, filter ProcessDefinitionFilter, opts ...options.FacadeOption) (ProcessDefinitions, error) {
	pds, err := c.pdApi.SearchProcessDefinitions(ctx, toDomainProcessDefinitionFilter(filter), pdsvc.MaxResultSize, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinitions{}, ferr.FromDomain(err)
	}
	return fromDomainProcessDefinitions(pds), nil
}

func (c *client) SearchProcessDefinitionsLatest(ctx context.Context, filter ProcessDefinitionFilter, opts ...options.FacadeOption) (ProcessDefinitions, error) {
	pds, err := c.pdApi.SearchProcessDefinitionsLatest(ctx, toDomainProcessDefinitionFilter(filter), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinitions{}, ferr.FromDomain(err)
	}
	return fromDomainProcessDefinitions(pds), nil
}

func (c *client) GetProcessDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessDefinition, error) {
	pd, err := c.pdApi.GetProcessDefinition(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinition{}, ferr.FromDomain(err)
	}
	return fromDomainProcessDefinition(pd), nil
}

func (c *client) GetProcessDefinitionXML(ctx context.Context, key string, opts ...options.FacadeOption) (string, error) {
	xml, err := c.pdApi.GetProcessDefinitionXML(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return "", ferr.FromDomain(err)
	}
	return xml, nil
}

func (c *client) GetProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstance, error) {
	return c.LookupProcessInstance(ctx, key, opts...)
}

func (c *client) LookupProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstance, error) {
	pi, err := pisvc.LookupProcessInstance(ctx, c.piApi, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstance{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstance(pi), nil
}

// GetIncident exposes direct incident lookup through the public facade error model.
func (c *client) GetIncident(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstanceIncidentDetail, error) {
	incident, err := c.incApi.GetIncident(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstanceIncidentDetail{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstanceIncidentDetail(incident), nil
}

// GetIncidents fetches unique incident keys with bounded parallelism.
func (c *client) GetIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (Incidents, error) {
	cfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	workers := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("getting %d incident(s) using %d worker(s)", len(ukeys), workers), c.log, cfg.Verbose)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("getting %d incident(s)", len(ukeys)))
	defer stopActivity()
	items, err := pool.ExecuteSlice[string, ProcessInstanceIncidentDetail](ctx, ukeys, workers, cfg.FailFast, func(ctx context.Context, key string, _ int) (ProcessInstanceIncidentDetail, error) {
		return c.GetIncident(ctx, key, opts...)
	})
	if err != nil {
		return Incidents{}, err
	}
	return Incidents{
		Total: int32(len(items)),
		Items: items,
	}, nil
}

// SearchIncidents exposes top-level incident list/search through the incident service boundary.
func (c *client) SearchIncidents(ctx context.Context, filter IncidentFilter, size int32, opts ...options.FacadeOption) (Incidents, error) {
	if incidentSearchNeedsPagedLocalFiltering(filter) {
		return c.searchIncidentPagesUntilLimit(ctx, filter, size, opts...)
	}
	incidents, err := c.incApi.SearchIncidents(ctx, toDomainIncidentFilter(filter), size, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Incidents{}, ferr.FromDomain(err)
	}
	return fromDomainIncidents(incidents), nil
}

func (c *client) searchIncidentPagesUntilLimit(ctx context.Context, filter IncidentFilter, size int32, opts ...options.FacadeOption) (Incidents, error) {
	if size <= 0 {
		return Incidents{}, nil
	}
	req := IncidentPageRequest{Size: size}
	out := make([]ProcessInstanceIncidentDetail, 0, size)
	for {
		page, err := c.SearchIncidentsPage(ctx, filter, req, opts...)
		if err != nil {
			return Incidents{}, err
		}
		for _, item := range page.Items {
			if int32(len(out)) >= size {
				return Incidents{Total: int32(len(out)), Items: out}, nil
			}
			out = append(out, item)
		}
		if page.OverflowState == ProcessInstanceOverflowStateNoMore {
			return Incidents{Total: int32(len(out)), Items: out}, nil
		}
		req = nextIncidentFacadePageRequest(req, page)
	}
}

func incidentSearchNeedsPagedLocalFiltering(filter IncidentFilter) bool {
	return filter.ErrorMessage != ""
}

func nextIncidentFacadePageRequest(current IncidentPageRequest, page IncidentPage) IncidentPageRequest {
	if page.EndCursor != "" {
		return IncidentPageRequest{Size: current.Size, After: page.EndCursor}
	}
	return IncidentPageRequest{From: current.From + current.Size, Size: current.Size}
}

// SearchIncidentsPage exposes one top-level incident search page with service-owned request semantics.
func (c *client) SearchIncidentsPage(ctx context.Context, filter IncidentFilter, page IncidentPageRequest, opts ...options.FacadeOption) (IncidentPage, error) {
	incidents, err := c.incApi.SearchIncidentsPage(ctx, toDomainIncidentFilter(filter), toDomainIncidentPageRequest(page), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return IncidentPage{}, ferr.FromDomain(err)
	}
	return fromDomainIncidentPage(incidents), nil
}

// SearchProcessInstanceIncidents exposes the tenant-safe service incident lookup through the facade error model.
func (c *client) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...options.FacadeOption) ([]ProcessInstanceIncidentDetail, error) {
	incidents, err := c.incApi.SearchProcessInstanceIncidents(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return fromDomainProcessInstanceIncidentDetails(incidents), nil
}

// SearchProcessInstanceVariables exposes the tenant-safe service variable lookup through the facade error model.
func (c *client) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...options.FacadeOption) ([]ProcessInstanceVariable, error) {
	variables, err := c.piApi.SearchProcessInstanceVariables(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return fromDomainProcessInstanceVariables(variables), nil
}

func (c *client) UpdateProcessInstanceVariables(ctx context.Context, request ProcessInstanceVariableUpdateRequest, opts ...options.FacadeOption) (ProcessInstanceVariableUpdateResult, error) {
	cfg := options.ApplyFacadeOptions(opts)
	dreq := toDomainProcessInstanceVariableUpdateRequest(request)
	resp, err := c.piApi.UpdateProcessInstanceVariables(ctx, dreq.Key, dreq.Variables, options.MapFacadeOptionsToCallOptions(opts)...)
	result := fromDomainProcessInstanceVariableUpdateResponse(resp, request.Variables)
	if result.Key == "" {
		result.Key = request.Key
	}
	if err != nil {
		updateErr := ferr.FromDomain(err)
		result.Error = updateErr.Error()
		if result.MutationAccepted {
			result.Status = ProcessInstanceVariableUpdateStatusConfirmationFailed
			result.ConfirmationStatus = "failed"
			return result, updateErr
		}
		result.Status = ProcessInstanceVariableUpdateStatusMutationFailed
		result.MutationAccepted = false
		if cfg.NoWait {
			result.ConfirmationStatus = "skipped"
			return result, nil
		}
		return result, updateErr
	}
	if cfg.NoWait {
		result.Status = ProcessInstanceVariableUpdateStatusSubmitted
		result.ConfirmationStatus = "skipped"
		return result, nil
	}
	result.Status = ProcessInstanceVariableUpdateStatusConfirmed
	result.ConfirmationStatus = "confirmed"
	return result, nil
}

// EnrichProcessInstancesWithIncidents attaches direct incident details to selected process-instance results without reordering them.
func (c *client) EnrichProcessInstancesWithIncidents(ctx context.Context, pis ProcessInstances, opts ...options.FacadeOption) (IncidentEnrichedProcessInstances, error) {
	items := make([]IncidentEnrichedProcessInstance, 0, len(pis.Items))
	for _, pi := range pis.Items {
		incidents, err := c.SearchProcessInstanceIncidents(ctx, pi.Key, opts...)
		if err != nil {
			return IncidentEnrichedProcessInstances{}, err
		}
		items = append(items, IncidentEnrichedProcessInstance{
			Item:      pi,
			Incidents: incidentsForProcessInstance(pi.Key, incidents),
		})
	}
	return IncidentEnrichedProcessInstances{
		Total: int32(len(items)),
		Items: items,
	}, nil
}

// EnrichProcessInstancesWithVariables attaches process-scope variables to selected process-instance results without reordering them.
func (c *client) EnrichProcessInstancesWithVariables(ctx context.Context, pis ProcessInstances, opts ...options.FacadeOption) (VariableEnrichedProcessInstances, error) {
	items := make([]VariableEnrichedProcessInstance, 0, len(pis.Items))
	for _, pi := range pis.Items {
		variables, err := c.SearchProcessInstanceVariables(ctx, pi.Key, opts...)
		if err != nil {
			return VariableEnrichedProcessInstances{}, err
		}
		items = append(items, VariableEnrichedProcessInstance{
			Item:      pi,
			Variables: variablesForProcessInstance(pi.Key, variables),
		})
	}
	return VariableEnrichedProcessInstances{
		Total: int32(len(items)),
		Items: items,
	}, nil
}

// EnrichTraversalWithIncidents overlays incident details onto walked items while preserving traversal metadata and warnings.
func (c *client) EnrichTraversalWithIncidents(ctx context.Context, result TraversalResult, opts ...options.FacadeOption) (IncidentEnrichedTraversalResult, error) {
	items := make([]IncidentEnrichedTraversalItem, 0, len(result.Keys))
	for _, key := range result.Keys {
		pi, ok := result.Chain[key]
		if !ok {
			continue
		}
		incidents, err := c.SearchProcessInstanceIncidents(ctx, key, opts...)
		if err != nil {
			return IncidentEnrichedTraversalResult{}, err
		}
		items = append(items, IncidentEnrichedTraversalItem{
			Item:      pi,
			Incidents: incidentsForProcessInstance(key, incidents),
		})
	}
	return IncidentEnrichedTraversalResult{
		Mode:             result.Mode,
		Outcome:          result.Outcome,
		StartKey:         result.StartKey,
		RootKey:          result.RootKey,
		Keys:             append([]string(nil), result.Keys...),
		Edges:            result.Edges,
		Items:            items,
		MissingAncestors: append([]MissingAncestor(nil), result.MissingAncestors...),
		Warning:          result.Warning,
	}, nil
}

// incidentsForProcessInstance keeps only details owned by the requested key, guarding against broad backend incident responses.
func incidentsForProcessInstance(key string, incidents []ProcessInstanceIncidentDetail) []ProcessInstanceIncidentDetail {
	out := make([]ProcessInstanceIncidentDetail, 0, len(incidents))
	for _, incident := range incidents {
		if incident.ProcessInstanceKey == key {
			out = append(out, incident)
		}
	}
	return out
}

// variablesForProcessInstance keeps only process-scope variables owned by the requested key.
func variablesForProcessInstance(key string, variables []ProcessInstanceVariable) []ProcessInstanceVariable {
	out := make([]ProcessInstanceVariable, 0, len(variables))
	for _, variable := range variables {
		if variable.ProcessInstanceKey == key && variable.ScopeKey == key {
			out = append(out, variable)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (c *client) LookupProcessInstanceStateByKey(ctx context.Context, key string, opts ...options.FacadeOption) (StateReport, ProcessInstance, error) {
	got, pi, err := pisvc.LookupProcessInstanceStateByKey(ctx, c.piApi, key, options.MapFacadeOptionsToCallOptions(opts)...)
	pgot, _ := ParseState(got.String())
	if err != nil {
		return StateReport{State: pgot}, ProcessInstance{}, ferr.FromDomain(err)
	}
	return StateReport{State: pgot, Status: got.String(), Key: pi.Key}, fromDomainProcessInstance(pi), nil
}

func (c *client) CreateProcessInstance(ctx context.Context, data ProcessInstanceData, opts ...options.FacadeOption) (ProcessInstance, error) {
	pic, err := c.piApi.CreateProcessInstance(ctx, toProcessInstanceData(data), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstance{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstanceCreation(pic), nil
}

func (c *client) CreateProcessInstances(ctx context.Context, datas []ProcessInstanceData, opts ...options.FacadeOption) ([]ProcessInstance, error) {
	pis := make([]ProcessInstance, 0, len(datas))
	for _, data := range datas {
		pic, err := c.piApi.CreateProcessInstance(ctx, toProcessInstanceData(data), options.MapFacadeOptionsToCallOptions(opts)...)
		if err != nil {
			return nil, ferr.FromDomain(err)
		}
		pis = append(pis, fromDomainProcessInstanceCreation(pic))
	}
	return pis, nil
}

func (c *client) SearchProcessInstances(ctx context.Context, filter ProcessInstanceFilter, size int32, opts ...options.FacadeOption) (ProcessInstances, error) {
	page, err := c.SearchProcessInstancesPage(ctx, filter, ProcessInstancePageRequest{Size: size}, opts...)
	if err != nil {
		return ProcessInstances{}, ferr.FromDomain(err)
	}
	return ProcessInstances{
		Total: int32(len(page.Items)),
		Items: page.Items,
	}, nil
}

func (c *client) SearchProcessInstancesPage(ctx context.Context, filter ProcessInstanceFilter, page ProcessInstancePageRequest, opts ...options.FacadeOption) (ProcessInstancePage, error) {
	pis, err := c.piApi.SearchForProcessInstancesPage(ctx, toDomainProcessInstanceFilter(filter), toDomainProcessInstancePageRequest(page), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstancePage{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstancePage(pis), nil
}

func (c *client) CancelProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (CancelReport, ProcessInstances, error) {
	resp, pis, err := c.piApi.CancelProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return CancelReport{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, ProcessInstances{}, ferr.FromDomain(err)
	}
	return CancelReport{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, fromDomainProcessInstances(pis), nil
}

func (c *client) DeleteProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (DeleteReport, error) {
	resp, err := c.piApi.DeleteProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return DeleteReport{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, ferr.FromDomain(err)
	}
	return DeleteReport{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, nil
}

func (c *client) GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstances, error) {
	children, err := c.piApi.GetDirectChildrenOfProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstances{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstances(children), nil
}

func (c *client) FilterProcessInstanceWithOrphanParent(ctx context.Context, items []ProcessInstance, opts ...options.FacadeOption) ([]ProcessInstance, error) {
	in := toolx.MapSlice(items, toDomainProcessInstance)
	// Orphan detection remains a follow-up lookup flow instead of becoming part
	// of the initial process-instance search request.
	out, err := c.piApi.FilterProcessInstanceWithOrphanParent(ctx, in, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return toolx.MapSlice(out, fromDomainProcessInstance), nil
}

func (c *client) WaitForProcessInstanceState(ctx context.Context, key string, desired States, opts ...options.FacadeOption) (StateReport, ProcessInstance, error) {
	got, pi, err := c.piApi.WaitForProcessInstanceState(ctx, key, toolx.MapSlice(desired, func(s State) d.State { return d.State(s) }), options.MapFacadeOptionsToCallOptions(opts)...)
	pgot, _ := ParseState(got.State.String())
	if err != nil {
		return StateReport{State: pgot, Status: got.Status}, ProcessInstance{}, ferr.FromDomain(err)
	}
	return StateReport{State: pgot, Status: got.Status, Key: pi.Key}, fromDomainProcessInstance(pi), nil
}

// WaitForProcessInstanceExpectation preserves the selected key in failure reports before mapping combined state/incident waits.
func (c *client) WaitForProcessInstanceExpectation(ctx context.Context, key string, request ProcessInstanceExpectationRequest, opts ...options.FacadeOption) (ProcessInstanceExpectationReport, ProcessInstance, error) {
	got, pi, err := c.piApi.WaitForProcessInstanceExpectation(ctx, key, toDomainProcessInstanceExpectationRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	report := fromDomainProcessInstanceExpectationResponse(got)
	if report.Key == "" {
		report.Key = key
	}
	if err != nil {
		return report, ProcessInstance{}, ferr.FromDomain(err)
	}
	if report.Key == "" {
		report.Key = pi.Key
	}
	return report, fromDomainProcessInstance(pi), nil
}

func MapStateResponseToReport(in d.StateResponse) StateReport {
	return StateReport{
		State:  State(in.State.String()),
		Status: in.Status,
	}
}

func MapStateResponsesToReports(in d.StateResponses) StateReports {
	out := StateReports{
		Items: make([]StateReport, len(in.Items)),
	}
	for i, r := range in.Items {
		out.Items[i] = MapStateResponseToReport(r)
	}
	return out
}

func mapDryRunTraversalWarning(results []TraversalResult) (warning string, missing []MissingAncestor, outcome TraversalOutcome) {
	outcome = TraversalOutcomeComplete
	for _, result := range results {
		if len(result.MissingAncestors) > 0 {
			missing = append(missing, result.MissingAncestors...)
		}
		if result.Warning != "" && warning == "" {
			warning = result.Warning
		}
		switch result.Outcome {
		case TraversalOutcomeUnresolved:
			if outcome == TraversalOutcomeComplete {
				outcome = TraversalOutcomeUnresolved
			}
		case TraversalOutcomePartial:
			outcome = TraversalOutcomePartial
		}
	}
	if len(missing) > 0 && warning == "" {
		warning = "one or more parent process instances were not found"
	}
	if len(missing) == 0 {
		return warning, nil, outcome
	}
	return warning, uniqueMissingAncestors(missing), outcome
}

func uniqueMissingAncestors(items []MissingAncestor) []MissingAncestor {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]MissingAncestor, 0, len(items))
	for _, item := range items {
		key := item.StartKey + ":" + item.Key
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}
