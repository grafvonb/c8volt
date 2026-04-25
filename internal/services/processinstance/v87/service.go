package v87

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/internal/services/processinstance/waiter"
	"github.com/grafvonb/c8volt/internal/services/processinstance/walker"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
)

const wrongStateMessage400 = "Process instances needs to be in one of the states [COMPLETED, CANCELED]"

type Service struct {
	cc  GenProcessInstanceClientCamunda
	co  GenProcessInstanceClientOperate
	cfg *config.Config
	log *slog.Logger
}

type traversalAdapter struct {
	*Service
}

func (s *Service) ClientCamunda() GenProcessInstanceClientCamunda { return s.cc }
func (s *Service) ClientOperate() GenProcessInstanceClientOperate { return s.co }
func (s *Service) Config() *config.Config                         { return s.cfg }
func (s *Service) Logger() *slog.Logger                           { return s.log }
func (s *Service) LegacyDryRunTraversalOnly() bool                { return true }

type Option func(*Service)

func WithClientCamunda(c GenProcessInstanceClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.cc = c
		}
	}
}

func WithClientOperate(c GenProcessInstanceClientOperate) Option {
	return func(s *Service) {
		if c != nil {
			s.co = c
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
	cc, err := camundav87.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav87.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	co, err := operatev87.NewClientWithResponses(
		deps.Config.APIs.Operate.BaseURL,
		operatev87.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{co: co, cc: cc, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.cc, s.co)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

func (s *Service) CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
	cCfg := services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("creating new process instance with process definition id %s", data.ProcessDefinitionSpecificId))
	body := toProcessInstanceCreationInstruction(data)
	resp, err := s.cc.PostProcessInstancesWithResponse(ctx, body)
	if err != nil {
		return d.ProcessInstanceCreation{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessInstanceCreation{}, err
	}
	pi := fromPostProcessInstancesResponse(*payload)
	s.log.Debug(fmt.Sprintf("created new process instance %s using process definition id %s, %s, v%d, tenant: %s", pi.Key, pi.ProcessDefinitionKey, pi.BpmnProcessId, pi.ProcessDefinitionVersion, pi.TenantId))
	if !cCfg.NoWait {
		if pi.Key == "" || pi.Key == unknownProcessInstanceKeyV87 {
			pi.StartDate = time.Now().UTC().Format(time.RFC3339)
			s.log.Info(fmt.Sprintf("process instance creation requested at %s (run not confirmed, as v8.7 returns no process-instance key) using process definition id %s, %s, v%d, tenant: %s", pi.StartDate, pi.ProcessDefinitionKey, pi.BpmnProcessId, pi.ProcessDefinitionVersion, pi.TenantId))
			return pi, nil
		}
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
	_, err := processInstanceKeyInt64(key)
	if err != nil {
		return d.ProcessInstance{}, err
	}
	return d.ProcessInstance{}, fmt.Errorf("%w: process-instance direct lookup by key is not tenant-safe in Camunda 8.7", d.ErrUnsupported)
}

func (s *Service) GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
	filter := d.ProcessInstanceFilter{
		ParentKey: key,
	}
	resp, err := s.SearchForProcessInstances(ctx, filter, 1000, opts...)
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
	// Preserve the existing orphan-detection strategy: inspect candidate children
	// and verify each parent through a follow-up lookup instead of broadening the
	// initial search request with an approximation.
	var result []d.ProcessInstance
	for _, it := range items {
		if it.ParentKey == "" {
			continue
		}
		_, err := s.GetProcessInstance(ctx, it.ParentKey, opts...)
		if errors.Is(err, d.ErrNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found") {
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
	if hasDateFilterBounds(filter) {
		return d.ProcessInstancePage{}, fmt.Errorf("%w: process-instance date filters require Camunda 8.8", d.ErrUnsupported)
	}
	fetchSize := pickProcessInstanceSearchFetchSize(pageReq)
	body, err := searchProcessInstancesRequest(s.cfg.App.Tenant, filter, fetchSize)
	if err != nil {
		return d.ProcessInstancePage{}, err
	}
	resp, err := s.co.SearchProcessInstancesWithResponse(ctx, body)
	if err != nil {
		return d.ProcessInstancePage{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessInstancePage{}, err
	}
	items := toolx.DerefSlicePtr(payload.Items, fromProcessInstanceResponse)
	window, overflow := trimProcessInstancePageWindow(items, payload.Total, pageReq, fetchSize)
	return d.ProcessInstancePage{
		Items:         window,
		Request:       pageReq,
		OverflowState: overflow,
		ReportedTotal: newProcessInstanceReportedTotal(payload.Total),
	}, nil
}

func newProcessInstanceReportedTotal(total *int64) *d.ProcessInstanceReportedTotal {
	if total == nil {
		return nil
	}
	return &d.ProcessInstanceReportedTotal{
		Count: *total,
		Kind:  d.ProcessInstanceReportedTotalKindExact,
	}
}

func pickProcessInstanceSearchFetchSize(pageReq d.ProcessInstancePageRequest) int32 {
	if pageReq.Size <= 0 {
		return 0
	}
	fetchSize := pageReq.From + pageReq.Size
	if fetchSize <= 0 {
		return pageReq.Size
	}
	if fetchSize > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return fetchSize
}

func trimProcessInstancePageWindow(items []d.ProcessInstance, total *int64, pageReq d.ProcessInstancePageRequest, fetchSize int32) ([]d.ProcessInstance, d.ProcessInstanceOverflowState) {
	if pageReq.From < 0 {
		pageReq.From = 0
	}
	start := int(pageReq.From)
	if start >= len(items) {
		return nil, pickProcessInstanceOverflowState(total, pageReq, 0, len(items), fetchSize)
	}
	end := start + int(pageReq.Size)
	if end > len(items) {
		end = len(items)
	}
	window := items[start:end]
	return window, pickProcessInstanceOverflowState(total, pageReq, len(window), len(items), fetchSize)
}

func pickProcessInstanceOverflowState(total *int64, pageReq d.ProcessInstancePageRequest, windowCount int, fetchedCount int, fetchSize int32) d.ProcessInstanceOverflowState {
	visibleThrough := int64(pageReq.From) + int64(windowCount)
	if total != nil {
		if *total > visibleThrough {
			return d.ProcessInstanceOverflowStateHasMore
		}
		return d.ProcessInstanceOverflowStateNoMore
	}
	if pageReq.From+pageReq.Size > consts.MaxPISearchSize {
		return d.ProcessInstanceOverflowStateIndeterminate
	}
	if int32(fetchedCount) < fetchSize {
		return d.ProcessInstanceOverflowStateNoMore
	}
	return d.ProcessInstanceOverflowStateIndeterminate
}

func hasDateFilterBounds(filter d.ProcessInstanceFilter) bool {
	return filter.StartDateAfter != "" ||
		filter.StartDateBefore != "" ||
		filter.EndDateAfter != "" ||
		filter.EndDateBefore != ""
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
			} else {
				s.log.Info(fmt.Sprintf("cannot cancel: process instance with key %s is a child of root %s; use --force to cancel the root and its child instances", key, rootPIKey))
				return d.CancelResponse{StatusCode: http.StatusConflict}, pis, nil
			}
		}
		pis = append(pis, pi)
	} else {
		s.log.Debug(fmt.Sprintf("skipping state check and parent for process instance with key %s before cancellation", key))
	}
	s.log.Debug(fmt.Sprintf("cancelling process instance with key %s", key))
	resp, err := s.cc.PostProcessInstancesProcessInstanceKeyCancellationWithResponse(ctx, key,
		camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody{})
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
	s.log.Debug(fmt.Sprintf("checking state of process instance with key %s", key))
	_, err := processInstanceKeyInt64(key)
	if err != nil {
		return "", d.ProcessInstance{}, err
	}
	return "", d.ProcessInstance{}, fmt.Errorf("process instance state: %w", fmt.Errorf("%w: process-instance state lookup by key is not tenant-safe in Camunda 8.7", d.ErrUnsupported))
}

func (s *Service) DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
	cCfg := services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("deleting process instance with key %s", key))
	oldKey, err := toolx.StringToInt64(key)
	if err != nil {
		return d.DeleteResponse{}, fmt.Errorf("parsing process instance key %q to int64: %w", key, err)
	}

	s.log.Debug(fmt.Sprintf("checking children of process instance with key %s before deletion", key))
	_, edges, _, err := s.Descendants(ctx, key, opts...)
	if err != nil {
		return d.DeleteResponse{}, err
	}
	children := edges[key]
	if len(children) > 0 {
		for _, ch := range children {
			s.log.Debug(fmt.Sprintf("found child process instance with key %s of process instance with key %s, deleting...", ch, key))
			_, err = s.DeleteProcessInstance(ctx, ch, opts...)
			if err != nil {
				return d.DeleteResponse{}, fmt.Errorf("deleting child process instance with key %s of process instance with key %s: %w", ch, key, err)
			}
		}
		/*
			if cCfg.NoStateCheck {
				s.log.Warn(fmt.Sprintf("deleting process instance with key %s, will cause creation of %d orphaned child process instance(s): %v", key, len(orphans), orphans))
			} else {
				s.log.Info(fmt.Sprintf("cannot delete, process instance with key %s has %d child process instance(s): %v; use --no-state-check to ignore and delete anyway", key, len(orphans), orphans))
				return d.DeleteResponse{StatusCode: http.StatusConflict}, nil
			}
		*/
	}

	resp, err := s.co.DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx, oldKey)
	if isDeleteWrongStateResponse(resp) {
		if cCfg.Force {
			s.log.Info(fmt.Sprintf("process instance with key %s not in one of terminated states; cancelling it first", key))
			_, _, err = s.CancelProcessInstance(ctx, key, opts...)
			if err != nil {
				return d.DeleteResponse{}, fmt.Errorf("delete cancel: %w", err)
			}
			s.log.Info(fmt.Sprintf("waiting for process instance with key %s to be cancelled by workflow engine...", key))
			states := []d.State{d.StateCanceled, d.StateTerminated}
			if _, _, err = waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, key, states, opts...); err != nil {
				return d.DeleteResponse{}, fmt.Errorf("delete wait canceled: %w", err)
			}
			s.log.Info(fmt.Sprintf("retrying deletion of process instance with key %d", oldKey))
			resp, err = s.co.DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx, oldKey)
		} else {
			logging.InfoIfVerbose(fmt.Sprintf("cannot delete, process instance %s is not in one of terminated states; use --force flag to cancel and then delete the process instance", key), s.log, cCfg.Verbose)
			return d.DeleteResponse{StatusCode: http.StatusConflict}, nil
		}
	}
	if err != nil {
		return d.DeleteResponse{}, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.DeleteResponse{}, err
	}
	if !cCfg.NoWait {
		s.log.Info(fmt.Sprintf("waiting for process instance with key %s to be deleted by workflow engine...", key))
		states := []d.State{d.StateAbsent}
		if _, _, err = waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, key, states, opts...); err != nil {
			return d.DeleteResponse{}, fmt.Errorf("delete wait absent: %w", err)
		}
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

func (s *Service) AncestryResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	return pitraversal.BuildAncestryResult(ctx, traversalAdapter{s}, startKey, opts...)
}

func (s *Service) DescendantsResult(ctx context.Context, rootKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	return pitraversal.BuildDescendantsResult(ctx, traversalAdapter{s}, rootKey, opts...)
}

func (s *Service) FamilyResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	return pitraversal.BuildFamilyResult(ctx, traversalAdapter{s}, startKey, opts...)
}

func (a traversalAdapter) GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	items, err := a.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{Key: key}, 2, opts...)
	if err != nil {
		return d.ProcessInstance{}, err
	}
	return common.RequireSingleProcessInstance(items, key)
}

func (a traversalAdapter) Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (rootKey string, path []string, chain map[string]d.ProcessInstance, err error) {
	return walker.Ancestry(ctx, a, startKey, opts...)
}

func (a traversalAdapter) Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) (desc []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error) {
	return walker.Descendants(ctx, a, rootKey, opts...)
}

func (a traversalAdapter) Family(ctx context.Context, startKey string, opts ...services.CallOption) (fam []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error) {
	return walker.Family(ctx, a, startKey, opts...)
}

func processInstanceKeyInt64(key string) (int64, error) {
	oldKey, err := toolx.StringToInt64(key)
	if err != nil {
		return 0, fmt.Errorf("converting process instance key %q to int64: %w", key, err)
	}
	return oldKey, nil
}

func isDeleteWrongStateResponse(resp *operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse) bool {
	if resp == nil || resp.StatusCode() != http.StatusBadRequest || resp.ApplicationproblemJSON400 == nil || resp.ApplicationproblemJSON400.Message == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(*resp.ApplicationproblemJSON400.Message))
	if msg == strings.ToLower(wrongStateMessage400) {
		return true
	}
	return strings.Contains(msg, "process") &&
		strings.Contains(msg, "state") &&
		(strings.Contains(msg, "completed") || strings.Contains(msg, "canceled") || strings.Contains(msg, "cancelled") || strings.Contains(msg, "terminated"))
}

func searchProcessInstancesRequest(tenant string, filter d.ProcessInstanceFilter, size int32) (operatev87.SearchProcessInstancesJSONRequestBody, error) {
	parentKey, err := toolx.StringToInt64Ptr(filter.ParentKey)
	if err != nil {
		return operatev87.SearchProcessInstancesJSONRequestBody{}, fmt.Errorf("parsing parent key %q to int64: %w", filter.ParentKey, err)
	}
	// Camunda 8.7 only supports the existing equality-style request fields here.
	// Parent/incident presence semantics stay on the client-side fallback path.
	bodyFilter := operatev87.ProcessInstance{
		TenantId:          toolx.PtrIf(tenant, ""),
		BpmnProcessId:     &filter.BpmnProcessId,
		ProcessVersion:    toolx.PtrIfNonZero(filter.ProcessVersion),
		ProcessVersionTag: &filter.ProcessVersionTag,
		State:             new(operatev87.ProcessInstanceState(filter.State)),
		ParentKey:         parentKey,
	}
	return operatev87.SearchProcessInstancesJSONRequestBody{
		Filter: &bodyFilter,
		Size:   &size,
	}, nil
}
