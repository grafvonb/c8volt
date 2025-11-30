package v88

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	operatev88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/processinstance/waiter"
	"github.com/grafvonb/c8volt/internal/services/processinstance/walker"
	"github.com/grafvonb/c8volt/toolx"
)

const wrongStateMessage400 = "Process instances needs to be in one of the states [COMPLETED, CANCELED]"

type Service struct {
	cc  GenProcessInstanceClientCamunda
	co  GenProcessInstanceClientOperate
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	cc, err := camundav88.NewClientWithResponses(
		cfg.APIs.Camunda.BaseURL,
		camundav88.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	co, err := operatev88.NewClientWithResponses(
		cfg.APIs.Operate.BaseURL,
		operatev88.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{co: co, cc: cc, cfg: cfg, log: log}
	for _, opt := range opts {
		opt(s)
	}
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
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.ProcessInstanceCreation{}, err
	}
	if resp.JSON200 == nil {
		return d.ProcessInstanceCreation{}, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}
	pi := fromPostProcessInstancesResponse(*resp.JSON200)
	s.log.Debug(fmt.Sprintf("created new process instance %s using process definition id %s, %s, v%d, tenant: %s", pi.Key, pi.ProcessDefinitionKey, pi.BpmnProcessId, pi.ProcessDefinitionVersion, pi.TenantId))
	if !cCfg.NoWait {
		s.log.Info(fmt.Sprintf("waiting for process instance of %s with key %s to be started by workflow engine...", pi.ProcessDefinitionKey, pi.Key))
		states := []d.State{d.StateActive}
		_, created, err := waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, pi.Key, states, opts...)
		if err != nil {
			return d.ProcessInstanceCreation{}, fmt.Errorf("waiting for started state failed for %s: %w", pi.Key, err)
		}
		pi.StartDate = created.StartDate
		pi.StartConfirmedAt = time.Now().UTC().Format(time.RFC3339)
		s.log.Info(fmt.Sprintf("process instance %s succesfully created (start registered at %s and confirmed at %s) using process definition id %s, %s, v%d, tenant: %s", pi.Key, pi.StartDate, pi.StartConfirmedAt, pi.ProcessDefinitionKey, pi.BpmnProcessId, pi.ProcessDefinitionVersion, pi.TenantId))
	} else {
		pi.StartDate = time.Now().UTC().Format(time.RFC3339)
		s.log.Info(fmt.Sprintf("process instance creation with the key %s requested at %s (run not confirmed, as no-wait is set) using process definition id %s, %s, v%d, tenant: %s", pi.Key, pi.StartDate, pi.ProcessDefinitionKey, pi.BpmnProcessId, pi.ProcessDefinitionVersion, pi.TenantId))
	}
	return pi, nil
}

func (s *Service) GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
	filter := d.ProcessInstanceFilter{
		ParentKey: key,
	}
	resp, err := s.SearchForProcessInstances(ctx, filter, 1000, opts...)
	if err != nil {
		return nil, fmt.Errorf("searching for children of process instance with key %s: %w", key, err)
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
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching for process instances with filter: %+v", filter))

	bodyFilter := camundav88.ProcessInstanceFilter{
		TenantId:                    common.NewStringEqFilterPtr(s.cfg.App.Tenant),
		ProcessInstanceKey:          common.NewProcessInstanceKeyEqFilterPtr(filter.Key),
		ProcessDefinitionId:         common.NewStringEqFilterPtr(filter.BpmnProcessId),
		ProcessDefinitionVersion:    common.NewIntegerEqFilterPtr(filter.ProcessVersion),
		ProcessDefinitionVersionTag: common.NewStringEqFilterPtr(filter.ProcessVersionTag),
		State:                       common.NewProcessInstanceStateEqFilterPtr(string(filter.State)),
		ParentProcessInstanceKey:    common.NewProcessInstanceKeyEqFilterPtr(filter.ParentKey),
	}
	page := camundav88.SearchQueryPageRequest{}
	from := int32(0)
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  &from,
		Limit: &size,
	})
	orderDesc := camundav88.DESC
	orderAsc := camundav88.ASC
	sort := []camundav88.ProcessInstanceSearchQuerySortRequest{
		{
			Field: camundav88.ProcessInstanceSearchQuerySortRequestFieldProcessDefinitionName,
			Order: &orderDesc,
		},
		{
			Field: camundav88.ProcessInstanceSearchQuerySortRequestFieldProcessDefinitionVersion,
			Order: &orderAsc,
		},
	}
	body := camundav88.SearchProcessInstancesJSONRequestBody{
		Filter: &bodyFilter,
		Page:   &page,
		Sort:   &sort,
	}
	resp, err := s.cc.SearchProcessInstancesWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}
	return toolx.MapSlice(resp.JSON200.Items, fromProcessInstanceResult), nil
}

func (s *Service) CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, error) {
	cCfg := services.ApplyCallOptions(opts)
	if !cCfg.NoStateCheck {
		s.log.Debug(fmt.Sprintf("getting state and parent of process instance with key %s before cancellation", key))
		st, pi, err := s.GetProcessInstanceStateByKey(ctx, key, opts...)

		if err != nil {
			return d.CancelResponse{}, err
		}
		s.log.Debug(fmt.Sprintf("checking if process instance with key %s is in allowable state to cancel", key))
		if st.IsTerminal() {
			s.log.Info(fmt.Sprintf("process instance with key %s is already in state %s, no need to cancel", key, st))
			return d.CancelResponse{
				StatusCode: http.StatusOK,
				Status:     fmt.Sprintf("process instance with key %s is already in state %s, no need to cancel", key, st),
			}, nil
		}
		s.log.Debug(fmt.Sprintf("checking if process instance with key %s is a child process", key))
		if pi.ParentKey != "" {
			s.log.Debug("child process, looking up root process instance in ancestry")
			rootPIKey, _, _, erra := walker.Ancestry(ctx, s, key, opts...)
			if erra != nil {
				return d.CancelResponse{}, fmt.Errorf("fetching ancestry for process instance with key %s: %w", key, erra)
			}
			s.log.Info(fmt.Sprintf("cannot cancel, process instance with key %s is a child process of a root parent with key %s", key, rootPIKey))
			if cCfg.Force {
				s.log.Info(fmt.Sprintf("force flag is set, cancelling root process instance with key %s and all its child processes", rootPIKey))
				return s.CancelProcessInstance(ctx, rootPIKey, opts...)
			} else {
				s.log.Info(fmt.Sprintf("use --force flag to cancel the root process instance with key %s and all its child processes", rootPIKey))
				return d.CancelResponse{StatusCode: http.StatusConflict}, nil
			}
		}
	} else {
		s.log.Debug(fmt.Sprintf("skipping state check and parent for process instance with key %s before cancellation", key))
	}
	s.log.Debug(fmt.Sprintf("cancelling process instance with key %s", key))
	resp, err := s.cc.CancelProcessInstanceWithResponse(ctx, key,
		camundav88.CancelProcessInstanceJSONRequestBody{})
	if err != nil {
		return d.CancelResponse{}, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.CancelResponse{}, err
	}
	if !cCfg.NoWait {
		s.log.Info(fmt.Sprintf("waiting for process instance with key %s to be cancelled by workflow engine...", key))
		states := []d.State{d.StateCanceled, d.StateTerminated}
		if _, _, err = waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, key, states, opts...); err != nil {
			return d.CancelResponse{}, fmt.Errorf("waiting for canceled state failed for %s: %w", key, err)
		}
		s.log.Info(fmt.Sprintf("process instance with key %s was successfully (confirmed) cancelled", key))
	} else {
		s.log.Info(fmt.Sprintf("process instance with key %s cancellation requested (not confirmed, as no-wait is set)", key))
	}
	return d.CancelResponse{
		Ok:         true,
		StatusCode: resp.StatusCode(),
		Status:     resp.Status(),
	}, nil
}

func (s *Service) GetProcessInstanceStateByKey(ctx context.Context, key string, opts ...services.CallOption) (d.State, d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("checking state of process instance with key %s", key))
	pi, err := s.cc.GetProcessInstanceWithResponse(ctx, key)
	if err != nil {
		return "", d.ProcessInstance{}, fmt.Errorf("fetching process instance with key %s: %w", key, err)
	}
	if err = httpc.HttpStatusErr(pi.HTTPResponse, pi.Body); err != nil {
		return "", d.ProcessInstance{}, fmt.Errorf("fetching process instance with key %s: %w", key, err)
	}
	if pi.JSON200 == nil {
		return "", d.ProcessInstance{}, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(pi.Body))
	}
	st := d.State(pi.JSON200.State)
	s.log.Debug(fmt.Sprintf("process instance with key %s is in state %s", key, st))
	return st, fromProcessInstanceResult(*pi.JSON200), nil
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
	orphans := edges[key]
	if len(edges[key]) > 0 {
		if cCfg.NoStateCheck {
			s.log.Warn(fmt.Sprintf("deleting process instance with key %s, will cause creation of %d orphaned child process instance(s): %v", key, len(orphans), orphans))
		} else {
			s.log.Info(fmt.Sprintf("cannot delete, process instance with key %s has %d child process instance(s): %v; use --no-state-check to ignore and delete anyway", key, len(orphans), orphans))
			return d.DeleteResponse{StatusCode: http.StatusConflict}, nil
		}
	}

	resp, err := s.co.DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx, oldKey)
	if resp.StatusCode() == http.StatusBadRequest &&
		resp.ApplicationproblemJSON400 != nil &&
		*resp.ApplicationproblemJSON400.Message == wrongStateMessage400 {
		if cCfg.Force {
			s.log.Info(fmt.Sprintf("process instance with key %s not in one of terminated states; cancelling it first", key))
			_, err = s.CancelProcessInstance(ctx, key, opts...)
			if err != nil {
				return d.DeleteResponse{}, fmt.Errorf("error cancelling process instance with key %s: %w", key, err)
			}
			s.log.Info(fmt.Sprintf("waiting for process instance with key %s to be cancelled by workflow engine...", key))
			states := []d.State{d.StateCanceled, d.StateTerminated}
			if _, _, err = waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, key, states, opts...); err != nil {
				return d.DeleteResponse{}, fmt.Errorf("waiting for canceled state failed for %s: %w", key, err)
			}
			s.log.Info(fmt.Sprintf("retrying deletion of process instance with key %d", oldKey))
			resp, err = s.co.DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx, oldKey)
		} else {
			s.log.Info(fmt.Sprintf("cannot delete, process instance %s is not in one of terminated states; use --force flag to cancel and then delete the process instance", key))
			return d.DeleteResponse{StatusCode: http.StatusConflict}, nil
		}
	}
	if err != nil {
		return d.DeleteResponse{}, err
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

func (s *Service) GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)

	resp, err := s.cc.GetProcessInstanceWithResponse(ctx, key)
	if err != nil {
		return d.ProcessInstance{}, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.ProcessInstance{}, err
	}
	if resp.JSON200 == nil {
		return d.ProcessInstance{}, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}
	return fromProcessInstanceResult(*resp.JSON200), nil
}

func (s *Service) WaitForProcessInstanceState(ctx context.Context, key string, desired d.States, opts ...services.CallOption) (d.State, d.ProcessInstance, error) {
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
