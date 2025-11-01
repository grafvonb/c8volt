package v87

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/grafvonb/c8volt/config"
	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/processinstance/waiter"
	"github.com/grafvonb/c8volt/internal/services/processinstance/walker"
	"github.com/grafvonb/c8volt/toolx"
)

const wrongStateMessage400 = "Process instances needs to be in one of the states [COMPLETED, CANCELED]"

type Service struct {
	cc  GenClusterClientCamunda
	oc  GenClusterClientOperate
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	cc, err := camundav87.NewClientWithResponses(
		cfg.APIs.Camunda.BaseURL,
		camundav87.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	co, err := operatev87.NewClientWithResponses(
		cfg.APIs.Operate.BaseURL,
		operatev87.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{oc: co, cc: cc, cfg: cfg, log: log}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

func (s *Service) GetProcessInstanceByKey(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
	oldKey, err := toolx.StringToInt64(key)
	if err != nil {
		return d.ProcessInstance{}, fmt.Errorf("converting process instance key %q to int64: %w", key, err)
	}
	s.log.Debug(fmt.Sprintf("fetching process instance with key %d", oldKey))
	resp, err := s.oc.GetProcessInstanceByKeyWithResponse(ctx, oldKey)
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
	return fromProcessInstanceResponse(*resp.JSON200), nil
}

func (s *Service) GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
	filter := d.ProcessInstanceSearchFilterOpts{
		ParentKey: key,
	}
	resp, err := s.SearchForProcessInstances(ctx, filter, 1000)
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
		_, err := s.GetProcessInstanceByKey(ctx, it.ParentKey)
		if err != nil && strings.Contains(err.Error(), "404") {
			result = append(result, it)
		} else if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *Service) SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceSearchFilterOpts, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching for process instances with filter: %+v", filter))
	st := operatev87.ProcessInstanceState(filter.State)
	pk, err := toolx.StringToInt64Ptr(filter.ParentKey)
	if err != nil {
		return nil, fmt.Errorf("parsing parent key %q to int64: %w", filter.ParentKey, err)
	}
	f := operatev87.ProcessInstance{
		TenantId:          &s.cfg.App.Tenant,
		BpmnProcessId:     &filter.BpmnProcessId,
		ProcessVersion:    toolx.PtrIfNonZero(filter.ProcessVersion),
		ProcessVersionTag: &filter.ProcessVersionTag,
		State:             &st,
		ParentKey:         pk,
	}
	body := operatev87.SearchProcessInstancesJSONRequestBody{
		Filter: &f,
		Size:   &size,
	}
	resp, err := s.oc.SearchProcessInstancesWithResponse(ctx, body)
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
	return toolx.DerefSlicePtr(resp.JSON200.Items, fromProcessInstanceResponse), nil
}

func (s *Service) CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, error) {
	cCfg := services.ApplyCallOptions(opts)
	if !cCfg.NoStateCheck {
		s.log.Debug(fmt.Sprintf("checking if process instance with key %s is in allowable state to cancel", key))
		st, err := s.GetProcessInstanceStateByKey(ctx, key)
		if err != nil {
			return d.CancelResponse{}, err
		}
		if st.IsTerminal() {
			s.log.Info(fmt.Sprintf("process instance with key %s is already in state %s, no need to cancel", key, st))
			return d.CancelResponse{
				StatusCode: http.StatusOK,
				Status:     fmt.Sprintf("process instance with key %s is already in state %s, no need to cancel", key, st),
			}, nil
		}
	} else {
		s.log.Debug(fmt.Sprintf("skipping state check for process instance with key %s before cancellation", key))
	}
	s.log.Debug(fmt.Sprintf("cancelling process instance with key %s", key))
	resp, err := s.cc.PostProcessInstancesProcessInstanceKeyCancellationWithResponse(ctx, key,
		camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody{})
	if err != nil {
		return d.CancelResponse{}, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.CancelResponse{}, err
	}
	if cCfg.Wait {
		s.log.Info(fmt.Sprintf("waiting for process instance with key %s to be cancelled by workflow engine...", key))
		states := []d.State{d.StateCanceled}
		if _, err = waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, key, states, opts...); err != nil {
			return d.CancelResponse{}, fmt.Errorf("waiting for canceled state failed for %s: %w", key, err)
		}
	}
	s.log.Info(fmt.Sprintf("process instance with key %s was successfully cancelled", key))
	return d.CancelResponse{
		StatusCode: resp.StatusCode(),
		Status:     resp.Status(),
	}, nil
}

func (s *Service) GetProcessInstanceStateByKey(ctx context.Context, key string, opts ...services.CallOption) (d.State, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("checking state of process instance with key %s", key))
	oldKey, err := toolx.StringToInt64(key)
	if err != nil {
		return "", fmt.Errorf("converting process instance key %q to int64: %w", key, err)
	}
	pi, err := s.oc.GetProcessInstanceByKeyWithResponse(ctx, oldKey)
	if err != nil {
		return "", fmt.Errorf("fetching process instance with key %s: %w", key, err)
	}
	if err = httpc.HttpStatusErr(pi.HTTPResponse, pi.Body); err != nil {
		return "", fmt.Errorf("fetching process instance with key %s: %w", key, err)
	}
	if pi.JSON200 == nil {
		return "", fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(pi.Body))
	}
	st := d.State(*pi.JSON200.State)
	s.log.Debug(fmt.Sprintf("process instance with key %s is in state %s", key, st))
	return st, nil
}

func (s *Service) DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ChangeStatus, error) {
	cCfg := services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("deleting process instance with key %s", key))
	oldKey, err := toolx.StringToInt64(key)
	if err != nil {
		return d.ChangeStatus{}, fmt.Errorf("parsing process instance key %q to int64: %w", key, err)
	}
	resp, err := s.oc.DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx, oldKey)
	if resp.StatusCode() == http.StatusBadRequest &&
		resp.ApplicationproblemJSON400 != nil &&
		*resp.ApplicationproblemJSON400.Message == wrongStateMessage400 {
		if cCfg.Cancel {
			s.log.Info(fmt.Sprintf("process instance with key %s not in one of terminated states; cancelling it first", key))
			_, err = s.CancelProcessInstance(ctx, key)
			if err != nil {
				return d.ChangeStatus{}, fmt.Errorf("error cancelling process instance with key %s: %w", key, err)
			}
			s.log.Info(fmt.Sprintf("waiting for process instance with key %s to be cancelled by workflow engine...", key))
			states := []d.State{d.StateCanceled, d.StateTerminated}
			if _, err = waiter.WaitForProcessInstanceState(ctx, s, s.cfg, s.log, key, states); err != nil {
				return d.ChangeStatus{}, fmt.Errorf("waiting for canceled state failed for %s: %w", key, err)
			}
			s.log.Info(fmt.Sprintf("retrying deletion of process instance with key %d", oldKey))
			resp, err = s.oc.DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx, oldKey)
		}
	}
	if err != nil {
		return d.ChangeStatus{}, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.ChangeStatus{}, err
	}
	s.log.Info(fmt.Sprintf("process instance with key %s was successfully deleted", key))
	return d.ChangeStatus{
		Deleted: toolx.Deref(resp.JSON200.Deleted, 0),
		Message: toolx.Deref(resp.JSON200.Message, ""),
	}, nil
}

func (s *Service) WaitForProcessInstanceState(ctx context.Context, key string, desired d.States, opts ...services.CallOption) (d.State, error) {
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
