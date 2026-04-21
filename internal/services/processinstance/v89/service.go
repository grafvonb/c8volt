package v89

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/processinstance/waiter"
	"github.com/grafvonb/c8volt/internal/services/processinstance/walker"
	"github.com/grafvonb/c8volt/toolx"
)

type Service struct {
	cc  GenProcessInstanceClientCamunda
	cfg *config.Config
	log *slog.Logger
}

func (s *Service) ClientCamunda() GenProcessInstanceClientCamunda { return s.cc }
func (s *Service) Config() *config.Config                         { return s.cfg }
func (s *Service) Logger() *slog.Logger                           { return s.log }

type Option func(*Service)

func WithClientCamunda(c GenProcessInstanceClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.cc = c
		}
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.log = logger
		}
	}
}

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	cc, err := camundav89.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav89.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{cc: cc, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.cc)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

func (s *Service) CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
	cCfg := services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("creating new process instance with process definition id %s", data.ProcessDefinitionSpecificId))
	body, err := toProcessInstanceCreationInstruction(data)
	if err != nil {
		return d.ProcessInstanceCreation{}, fmt.Errorf("building process instance creation instruction: %w", err)
	}
	resp, err := s.cc.CreateProcessInstanceWithResponse(ctx, body)
	if err != nil {
		return d.ProcessInstanceCreation{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessInstanceCreation{}, err
	}
	pi := fromCreateProcessInstanceResult(*payload)
	s.log.Debug(fmt.Sprintf("created new process instance %s using process definition id %s, %s, v%d, tenant: %s", pi.Key, pi.ProcessDefinitionKey, pi.BpmnProcessId, pi.ProcessDefinitionVersion, pi.TenantId))
	if !cCfg.NoWait {
		s.log.Info(fmt.Sprintf("waiting for process instance of %s with key %s to be started by workflow engine...", pi.ProcessDefinitionKey, pi.Key))
		states := []d.State{d.StateActive}
		_, created, err := waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, pi.Key, states, opts...)
		if err != nil {
			return d.ProcessInstanceCreation{}, fmt.Errorf("wait for started state: %w", err)
		}
		pi.StartDate = created.StartDate
		pi.StartConfirmedAt = time.Now().UTC().Format(time.RFC3339)
		s.log.Info(fmt.Sprintf("process instance %s successfully created (start registered at %s and confirmed at %s) using process definition id %s, %s, v%d, tenant: %s", pi.Key, pi.StartDate, pi.StartConfirmedAt, pi.ProcessDefinitionKey, pi.BpmnProcessId, pi.ProcessDefinitionVersion, pi.TenantId))
	} else {
		pi.StartDate = time.Now().UTC().Format(time.RFC3339)
		s.log.Info(fmt.Sprintf("process instance creation with the key %s requested at %s (run not confirmed, as no-wait is set) using process definition id %s, %s, v%d, tenant: %s", pi.Key, pi.StartDate, pi.ProcessDefinitionKey, pi.BpmnProcessId, pi.ProcessDefinitionVersion, pi.TenantId))
	}
	return pi, nil
}

func (s *Service) GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("performing tenant-safe lookup for process instance with key %s", key))
	items, err := s.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{Key: key}, 2, opts...)
	if err != nil {
		return d.ProcessInstance{}, fmt.Errorf("get process instance: %w", err)
	}
	pi, err := common.RequireSingleProcessInstance(items, key)
	if err != nil {
		return d.ProcessInstance{}, fmt.Errorf("get process instance: %w", err)
	}
	return pi, nil
}

func (s *Service) GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
	resp, err := s.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{ParentKey: key}, 1000, opts...)
	if err != nil {
		return nil, fmt.Errorf("search child process instances: %w", err)
	}
	return resp, nil
}

func (s *Service) FilterProcessInstanceWithOrphanParent(ctx context.Context, items []d.ProcessInstance, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
	if items == nil {
		return nil, nil
	}
	var result []d.ProcessInstance
	for _, it := range items {
		if it.ParentKey == "" {
			continue
		}
		_, err := s.GetProcessInstance(ctx, it.ParentKey, opts...)
		if err != nil && strings.Contains(err.Error(), "404") {
			result = append(result, it)
		} else if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *Service) SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	page, err := s.SearchForProcessInstancesPage(ctx, filter, d.ProcessInstancePageRequest{Size: size}, opts...)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

func (s *Service) SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, pageReq d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching for process instances with filter: %+v", filter))

	startDateAfter, err := parseInclusiveDateLowerBound(filter.StartDateAfter)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building start-date filter: %w", err)
	}
	startDateBefore, err := parseInclusiveDateUpperBound(filter.StartDateBefore)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building start-date filter: %w", err)
	}
	endDateAfter, err := parseInclusiveDateLowerBound(filter.EndDateAfter)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building end-date filter: %w", err)
	}
	endDateBefore, err := parseInclusiveDateUpperBound(filter.EndDateBefore)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building end-date filter: %w", err)
	}
	tenantFilter, err := newStringEqFilterPtr(s.cfg.App.Tenant)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building tenant filter: %w", err)
	}
	processInstanceKeyFilter, err := newProcessInstanceKeyEqFilterPtr(filter.Key)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building process-instance-key filter: %w", err)
	}
	processDefinitionIDFilter, err := newStringEqFilterPtr(filter.BpmnProcessId)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building process-definition-id filter: %w", err)
	}
	processDefinitionVersionFilter, err := newIntegerEqFilterPtr(filter.ProcessVersion)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building process-definition-version filter: %w", err)
	}
	processDefinitionVersionTagFilter, err := newStringEqFilterPtr(filter.ProcessVersionTag)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building process-definition-version-tag filter: %w", err)
	}
	startDateFilter, err := newDateTimeRangeFilterPtr(startDateAfter, startDateBefore, nil)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building start-date filter: %w", err)
	}
	endDateFilter, err := newDateTimeRangeFilterPtr(endDateAfter, endDateBefore, endDateExistsFilter(filter))
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building end-date filter: %w", err)
	}
	stateFilter, err := newProcessInstanceStateEqFilterPtr(string(filter.State))
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building state filter: %w", err)
	}
	parentProcessInstanceKeyFilter, err := newParentProcessInstanceKeyFilter(filter)
	if err != nil {
		return d.ProcessInstancePage{}, fmt.Errorf("building parent-process-instance-key filter: %w", err)
	}

	bodyFilter := &processInstanceFilter{
		TenantId:                    tenantFilter,
		ProcessInstanceKey:          processInstanceKeyFilter,
		ProcessDefinitionId:         processDefinitionIDFilter,
		ProcessDefinitionVersion:    processDefinitionVersionFilter,
		ProcessDefinitionVersionTag: processDefinitionVersionTagFilter,
		StartDate:                   startDateFilter,
		EndDate:                     endDateFilter,
		State:                       stateFilter,
		HasIncident:                 filter.HasIncident,
		ParentProcessInstanceKey:    parentProcessInstanceKeyFilter,
	}
	if bodyFilter.isEmpty() {
		bodyFilter = nil
	}

	page := camundav89.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &pageReq.From,
		Limit: &pageReq.Size,
	})
	sort := []camundav89.ProcessInstanceSearchQuerySortRequest{
		{
			Field: camundav89.ProcessInstanceSearchQuerySortRequestFieldProcessDefinitionName,
			Order: new(camundav89.DESC),
		},
		{
			Field: camundav89.ProcessInstanceSearchQuerySortRequestFieldProcessDefinitionVersion,
			Order: new(camundav89.ASC),
		},
	}
	body := processInstanceSearchQuery{
		Filter: bodyFilter,
		Page:   &page,
		Sort:   &sort,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return d.ProcessInstancePage{}, err
	}
	resp, err := s.cc.SearchProcessInstancesWithBodyWithResponse(ctx, "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		return d.ProcessInstancePage{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessInstancePage{}, err
	}
	result, err := decodeSearchProcessInstancesResponse(resp.Body, payload)
	if err != nil {
		return d.ProcessInstancePage{}, err
	}

	return d.ProcessInstancePage{
		Items:         toolx.MapSlice(result.Items, fromProcessInstanceResult),
		Request:       pageReq,
		OverflowState: pickProcessInstanceOverflowState(result.Page, pageReq, len(result.Items)),
	}, nil
}

func newParentProcessInstanceKeyFilter(filter d.ProcessInstanceFilter) (*camundav89.ProcessInstanceKeyFilterProperty, error) {
	if filter.ParentKey != "" {
		return newProcessInstanceKeyEqFilterPtr(filter.ParentKey)
	}
	return newProcessInstanceKeyExistsFilterPtr(filter.HasParent)
}

func pickProcessInstanceOverflowState(page camundav89.SearchQueryPageResponse, req d.ProcessInstancePageRequest, itemCount int) d.ProcessInstanceOverflowState {
	visibleCount := int64(req.From) + int64(itemCount)
	if page.HasMoreTotalItems {
		return d.ProcessInstanceOverflowStateHasMore
	}
	if page.TotalItems > visibleCount {
		return d.ProcessInstanceOverflowStateHasMore
	}
	if page.TotalItems == 0 && itemCount > 0 {
		return d.ProcessInstanceOverflowStateIndeterminate
	}
	return d.ProcessInstanceOverflowStateNoMore
}

func parseInclusiveDateLowerBound(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return nil, fmt.Errorf("parse %q as YYYY-MM-DD: %w", raw, err)
	}
	return new(t), nil
}

func parseInclusiveDateUpperBound(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return nil, fmt.Errorf("parse %q as YYYY-MM-DD: %w", raw, err)
	}
	t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
	return new(t), nil
}

func endDateExistsFilter(filter d.ProcessInstanceFilter) *bool {
	if filter.EndDateAfter == "" && filter.EndDateBefore == "" {
		return nil
	}
	return new(true)
}

func (s *Service) CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
	cCfg := services.ApplyCallOptions(opts)
	var pis []d.ProcessInstance
	if !cCfg.NoStateCheck {
		s.log.Debug(fmt.Sprintf("getting state and parent of process instance with key %s before cancellation", key))
		st, pi, err := s.GetProcessInstanceStateByKey(ctx, key, opts...)
		if err != nil {
			return d.CancelResponse{}, nil, err
		}
		s.log.Debug(fmt.Sprintf("checking if process instance with key %s is in allowable state to cancel", key))
		if st.IsTerminal() {
			s.log.Info(fmt.Sprintf("process instance with key %s is already in state %s, no need to cancel", key, st))
			return d.CancelResponse{
				StatusCode: http.StatusOK,
				Status:     fmt.Sprintf("process instance with key %s is already in state %s, no need to cancel", key, st),
			}, pis, nil
		}
		s.log.Debug(fmt.Sprintf("checking if process instance with key %s is a child process", key))
		if pi.ParentKey != "" {
			s.log.Debug("child process, looking up root process instance in ancestry")
			rootPIKey, _, _, erra := walker.Ancestry(ctx, s, key, opts...)
			if erra != nil {
				return d.CancelResponse{}, pis, fmt.Errorf("cancel ancestry: %w", erra)
			}
			if cCfg.Force {
				keys, _, family, err := walker.Descendants(ctx, s, rootPIKey, opts...)
				if err != nil {
					return d.CancelResponse{}, pis, fmt.Errorf("cancel descendants: %w", err)
				}
				for i := range family {
					pis = append(pis, family[i])
				}
				if cCfg.DryRun {
					s.log.Debug(fmt.Sprintf("dry-run: would cancel %d process instances with keys: %v", len(keys), keys))
					return d.CancelResponse{
						StatusCode: http.StatusOK,
						Status:     fmt.Sprintf("dry-run: would cancel %d process instances with keys %v", len(keys), keys),
					}, pis, nil
				}
				s.log.Info(fmt.Sprintf("force flag is set, cancelling %d process instances with keys %v", len(keys), keys))
				return s.CancelProcessInstance(ctx, rootPIKey, opts...)
			}
			s.log.Info(fmt.Sprintf("cannot cancel: process instance with key %s is a child of root %s; use --force to cancel the root and its child instances", key, rootPIKey))
			return d.CancelResponse{StatusCode: http.StatusConflict}, pis, nil
		}
		pis = append(pis, pi)
	} else {
		s.log.Debug(fmt.Sprintf("skipping state check and parent for process instance with key %s before cancellation", key))
	}

	s.log.Debug(fmt.Sprintf("cancelling process instance with key %s", key))
	resp, err := s.cc.CancelProcessInstanceWithResponse(ctx, key, camundav89.CancelProcessInstanceJSONRequestBody{})
	if err != nil {
		return d.CancelResponse{}, nil, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.CancelResponse{}, nil, err
	}
	if !cCfg.NoWait {
		keys, _, _, err := s.Family(ctx, key, opts...)
		if err != nil {
			return d.CancelResponse{}, nil, fmt.Errorf("cancel family: %w", err)
		}
		s.log.Info(fmt.Sprintf("waiting for process instance with key %s to be cancelled by workflow engine...", key))
		states := []d.State{d.StateCanceled, d.StateTerminated}
		if _, err = waiter.WaitForProcessInstancesState(ctx, s, s.cfg, s.log, keys, states, len(keys), opts...); err != nil {
			return d.CancelResponse{}, nil, fmt.Errorf("cancel wait: %w", err)
		}
		s.log.Info(fmt.Sprintf("process instance with key %s was successfully (confirmed) cancelled", key))
	} else {
		s.log.Info(fmt.Sprintf("process instance with key %s cancellation requested (not confirmed, as no-wait is set)", key))
	}
	return d.CancelResponse{
		Ok:         true,
		StatusCode: resp.StatusCode(),
		Status:     resp.Status(),
	}, pis, nil
}

func (s *Service) GetProcessInstanceStateByKey(ctx context.Context, key string, opts ...services.CallOption) (d.State, d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("checking tenant-safe state of process instance with key %s", key))
	pi, err := s.GetProcessInstance(ctx, key, opts...)
	if err != nil {
		return "", d.ProcessInstance{}, fmt.Errorf("process instance state: %w", err)
	}
	st := pi.State
	s.log.Debug(fmt.Sprintf("process instance with key %s is in state %s", key, st))
	return st, pi, nil
}

func (s *Service) DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
	cCfg := services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("deleting process instance with key %s", key))

	s.log.Debug(fmt.Sprintf("checking children of process instance with key %s before deletion", key))
	_, edges, _, err := s.Descendants(ctx, key, opts...)
	if err != nil {
		return d.DeleteResponse{}, err
	}
	children := edges[key]
	if len(children) > 0 {
		for _, ch := range children {
			s.log.Debug(fmt.Sprintf("found child process instance with key %s of process instance with key %s, deleting...", ch, key))
			if _, err = s.DeleteProcessInstance(ctx, ch, opts...); err != nil {
				return d.DeleteResponse{}, fmt.Errorf("deleting child process instance with key %s of process instance with key %s: %w", ch, key, err)
			}
		}
	}

	resp, err := s.cc.DeleteProcessInstanceWithResponse(ctx, key, camundav89.DeleteProcessInstanceJSONRequestBody{})
	if err != nil {
		return d.DeleteResponse{}, err
	}
	if resp.StatusCode() == http.StatusConflict {
		if cCfg.Force {
			s.log.Info(fmt.Sprintf("process instance with key %s not in one of terminated states; cancelling it first", key))
			if _, _, err = s.CancelProcessInstance(ctx, key, opts...); err != nil {
				return d.DeleteResponse{}, fmt.Errorf("delete cancel: %w", err)
			}
			s.log.Info(fmt.Sprintf("waiting for process instance with key %s to be cancelled by workflow engine...", key))
			states := []d.State{d.StateCanceled, d.StateTerminated}
			if _, _, err = waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, key, states, opts...); err != nil {
				return d.DeleteResponse{}, fmt.Errorf("delete wait canceled: %w", err)
			}
			s.log.Info(fmt.Sprintf("retrying deletion of process instance with key %s", key))
			resp, err = s.cc.DeleteProcessInstanceWithResponse(ctx, key, camundav89.DeleteProcessInstanceJSONRequestBody{})
			if err != nil {
				return d.DeleteResponse{}, err
			}
		} else {
			s.log.Info(fmt.Sprintf("cannot delete, process instance %s is not in one of terminated states; use --force flag to cancel and then delete the process instance", key))
			return d.DeleteResponse{StatusCode: http.StatusConflict}, nil
		}
	}
	if !cCfg.NoWait {
		s.log.Info(fmt.Sprintf("waiting for process instance with key %s to be deleted by workflow engine...", key))
		states := []d.State{d.StateAbsent}
		if _, _, err = waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, key, states, opts...); err != nil {
			return d.DeleteResponse{}, fmt.Errorf("delete wait absent: %w", err)
		}
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.DeleteResponse{}, err
	}
	s.log.Info(fmt.Sprintf("process instance with key %s was successfully deleted", key))
	return d.DeleteResponse{
		Ok:         true,
		StatusCode: resp.StatusCode(),
	}, nil
}

func (s *Service) WaitForProcessInstanceState(ctx context.Context, key string, desired d.States, opts ...services.CallOption) (d.StateResponse, d.ProcessInstance, error) {
	return waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, key, desired, opts...)
}

func (s *Service) Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (rootKey string, path []string, chain map[string]d.ProcessInstance, err error) {
	return walker.Ancestry(ctx, s, startKey, opts...)
}

func (s *Service) Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) (desc []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error) {
	return walker.Descendants(ctx, s, rootKey, opts...)
}

func (s *Service) Family(ctx context.Context, startKey string, opts ...services.CallOption) (fam []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error) {
	return walker.Family(ctx, s, startKey, opts...)
}
