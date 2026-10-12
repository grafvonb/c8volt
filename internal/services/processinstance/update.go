// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strings"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	variablewaiter "github.com/grafvonb/c8volt/internal/services/variable/waiter"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// VariableUpdateProcessInstanceAPI supplies stable selected-instance facts for
// the composed variable update planner.
type VariableUpdateProcessInstanceAPI interface {
	GetProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstance, error)
}

// VariableUpdateRootVariableAPI supplies complete root-variable collections.
type VariableUpdateRootVariableAPI interface {
	SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error)
}

// VariableUpdateScopeAPI supplies exact local-scope mutation submissions.
type VariableUpdateScopeAPI interface {
	UpdateScopeVariables(ctx context.Context, scopeKey string, variables map[string]any, opts ...services.CallOption) (d.ScopeVariableUpdateResponse, error)
}

// VariableUpdateAPI exposes collection-wide process-instance variable planning.
type VariableUpdateAPI interface {
	PlanProcessInstanceVariableUpdates(ctx context.Context, keys typex.Keys, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdatePlan, error)
	ExecuteProcessInstanceVariableUpdates(ctx context.Context, plan d.ProcessInstanceVariableUpdatePlan, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResults, error)
}

// VariableUpdateService composes process-instance identity reads with complete
// root-variable discovery. Mutation dependencies are added by the execution work unit.
type VariableUpdateService struct {
	processInstances VariableUpdateProcessInstanceAPI
	variables        VariableUpdateRootVariableAPI
	writer           VariableUpdateScopeAPI
	config           *config.Config
	log              *slog.Logger
}

// NewVariableUpdates creates the version-neutral process-instance variable planner.
func NewVariableUpdates(processInstances VariableUpdateProcessInstanceAPI, variables VariableUpdateRootVariableAPI, cfg *config.Config, log *slog.Logger) VariableUpdateAPI {
	if log == nil {
		log = slog.Default()
	}
	writer, _ := variables.(VariableUpdateScopeAPI)
	return &VariableUpdateService{processInstances: processInstances, variables: variables, writer: writer, config: cfg, log: log}
}

// ExecuteProcessInstanceVariableUpdates validates and freezes the complete
// collection plan before submitting exact root payloads and optionally waiting once.
func (s *VariableUpdateService) ExecuteProcessInstanceVariableUpdates(ctx context.Context, plan d.ProcessInstanceVariableUpdatePlan, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResults, error) {
	frozen, err := validateAndCopyProcessInstanceVariableUpdatePlan(plan)
	if err != nil {
		return d.ProcessInstanceVariableUpdateResults{}, err
	}
	cfg := services.ApplyCallOptions(opts)
	if cfg.DryRun || len(frozen.Instances) == 0 {
		return d.ProcessInstanceVariableUpdateResults{Items: make([]d.ProcessInstanceVariableUpdateResult, 0)}, nil
	}

	items := make([]d.ProcessInstanceVariableUpdateResult, len(frozen.Instances))
	changedIndexes := make([]int, 0, frozen.UpdateCount)
	for i, instance := range frozen.Instances {
		items[i] = d.ProcessInstanceVariableUpdateResult{
			Key: instance.ProcessInstanceKey, Variables: copyProcessInstanceVariableMap(instance.VerificationValues), ConfirmationStatus: "skipped",
		}
		if len(instance.MutationVariables) == 0 {
			items[i].Status = d.ProcessInstanceVariableUpdateStatusUnchanged
			continue
		}
		changedIndexes = append(changedIndexes, i)
	}
	if len(changedIndexes) == 0 {
		return d.ProcessInstanceVariableUpdateResults{Items: items}, nil
	}
	if s.writer == nil {
		return d.ProcessInstanceVariableUpdateResults{}, fmt.Errorf("%w: process-instance variable execution requires a scope variable service", d.ErrPrecondition)
	}
	if !cfg.NoWait && s.variables == nil {
		return d.ProcessInstanceVariableUpdateResults{}, fmt.Errorf("%w: process-instance variable confirmation requires a variable service", d.ErrPrecondition)
	}

	workers := toolx.DetermineNoOfWorkers(len(changedIndexes), wantedWorkers, cfg.NoWorkerLimit)
	mutations, mutationErr := pool.ExecuteSlice[int, processInstanceVariableExecution](ctx, changedIndexes, workers, cfg.FailFast,
		func(ctx context.Context, instanceIndex int, _ int) (processInstanceVariableExecution, error) {
			instance := frozen.Instances[instanceIndex]
			response, err := s.writer.UpdateScopeVariables(ctx, instance.ScopeKey, copyProcessInstanceVariableMap(instance.MutationVariables), opts...)
			result := items[instanceIndex]
			result.MutationAccepted = response.Accepted
			result.StatusCode = response.StatusCode
			result.Message = response.Status
			if err != nil {
				result.Status = d.ProcessInstanceVariableUpdateStatusMutationFailed
				result.Error = err.Error()
				return processInstanceVariableExecution{attempted: true, result: result}, err
			}
			if !response.Accepted {
				err = fmt.Errorf("%w: process-instance %s variable update was not accepted", d.ErrUpstream, instance.ProcessInstanceKey)
				result.Status = d.ProcessInstanceVariableUpdateStatusMutationFailed
				result.Error = err.Error()
				return processInstanceVariableExecution{attempted: true, result: result}, err
			}
			result.Status = d.ProcessInstanceVariableUpdateStatusSubmitted
			return processInstanceVariableExecution{attempted: true, result: result}, nil
		})
	mutationErr = joinProcessInstanceExecutionContextError(mutationErr, ctx.Err())
	for i, instanceIndex := range changedIndexes {
		if mutations[i].attempted {
			items[instanceIndex] = mutations[i].result
			continue
		}
		items[instanceIndex].Status = d.ProcessInstanceVariableUpdateStatusSkipped
	}
	results := d.ProcessInstanceVariableUpdateResults{Items: items}
	if cfg.NoWait {
		return results, mutationErr
	}

	confirmationIndexes := make([]int, 0, len(changedIndexes))
	for _, instanceIndex := range changedIndexes {
		if results.Items[instanceIndex].Status == d.ProcessInstanceVariableUpdateStatusSubmitted {
			confirmationIndexes = append(confirmationIndexes, instanceIndex)
		}
	}
	confirmationWorkers := toolx.DetermineNoOfWorkers(len(confirmationIndexes), wantedWorkers, cfg.NoWorkerLimit)
	confirmations, confirmationErr := pool.ExecuteSlice[int, processInstanceVariableConfirmation](ctx, confirmationIndexes, confirmationWorkers, cfg.FailFast,
		func(ctx context.Context, instanceIndex int, _ int) (processInstanceVariableConfirmation, error) {
			instance := frozen.Instances[instanceIndex]
			missing, err := variablewaiter.WaitForProcessInstanceVariables(ctx, s.variables, s.executionConfig(), instance.ScopeKey, copyProcessInstanceVariableMap(instance.VerificationValues), opts...)
			if err == nil && len(missing) > 0 {
				err = fmt.Errorf("%w: variables %s not confirmed on process-instance %s", d.ErrUpstream, strings.Join(missing, ", "), instance.ProcessInstanceKey)
			}
			return processInstanceVariableConfirmation{attempted: true, err: err}, err
		})
	confirmationErr = joinProcessInstanceExecutionContextError(confirmationErr, ctx.Err())
	for i, instanceIndex := range confirmationIndexes {
		if !confirmations[i].attempted {
			continue
		}
		if confirmations[i].err != nil {
			results.Items[instanceIndex].Status = d.ProcessInstanceVariableUpdateStatusConfirmationFailed
			results.Items[instanceIndex].ConfirmationStatus = "failed"
			results.Items[instanceIndex].Error = confirmations[i].err.Error()
			continue
		}
		results.Items[instanceIndex].Status = d.ProcessInstanceVariableUpdateStatusConfirmed
		results.Items[instanceIndex].ConfirmationStatus = "confirmed"
	}
	return results, errors.Join(mutationErr, confirmationErr)
}

type processInstanceVariableExecution struct {
	attempted bool
	result    d.ProcessInstanceVariableUpdateResult
}

type processInstanceVariableConfirmation struct {
	attempted bool
	err       error
}

// executionConfig guarantees waiter defaults even for manually composed services.
func (s *VariableUpdateService) executionConfig() *config.Config {
	if s.config == nil {
		return config.New()
	}
	return s.config
}

// joinProcessInstanceExecutionContextError retains cancellation that stopped scheduling.
func joinProcessInstanceExecutionContextError(err, contextErr error) error {
	if contextErr == nil || errors.Is(err, contextErr) {
		return err
	}
	return errors.Join(err, contextErr)
}

// PlanProcessInstanceVariableUpdates completes every selected-instance and
// root-variable read before returning a usable collection plan.
func (s *VariableUpdateService) PlanProcessInstanceVariableUpdates(ctx context.Context, keys typex.Keys, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdatePlan, error) {
	requestedKeys := keys.Unique()
	if err := validateProcessInstanceVariableUpdateInput(requestedKeys, variables); err != nil {
		return d.ProcessInstanceVariableUpdatePlan{}, err
	}
	baseTenant, err := d.NewExplicitKeysTenantContext(processInstanceUpdateConfiguredTenant(s.config))
	if err != nil {
		return d.ProcessInstanceVariableUpdatePlan{}, err
	}
	requestedValues := copyProcessInstanceVariableMap(variables)
	if len(requestedKeys) == 0 {
		return emptyProcessInstanceVariableUpdatePlan(requestedValues, baseTenant), nil
	}
	if s.processInstances == nil {
		return d.ProcessInstanceVariableUpdatePlan{}, fmt.Errorf("%w: process-instance variable planning requires a process-instance service", d.ErrPrecondition)
	}
	if s.variables == nil {
		return d.ProcessInstanceVariableUpdatePlan{}, fmt.Errorf("%w: process-instance variable planning requires a variable service", d.ErrPrecondition)
	}

	instances, err := s.processInstances.GetProcessInstances(ctx, requestedKeys, 0, opts...)
	if err != nil {
		return d.ProcessInstanceVariableUpdatePlan{}, fmt.Errorf("read selected process instances: %w", err)
	}
	if len(instances) != len(requestedKeys) {
		return d.ProcessInstanceVariableUpdatePlan{}, processInstancePlanningValidationError("requested %d process instances but received %d", len(requestedKeys), len(instances))
	}

	planner := processInstanceVariablePlanner{requested: requestedValues}
	for i, instance := range instances {
		key := requestedKeys[i]
		if err := validateSelectedProcessInstance(key, instance); err != nil {
			return d.ProcessInstanceVariableUpdatePlan{}, err
		}
		observed, err := s.variables.SearchProcessInstanceVariables(ctx, key, opts...)
		if err != nil {
			return d.ProcessInstanceVariableUpdatePlan{}, fmt.Errorf("read root variables for process-instance %s: %w", key, err)
		}
		if err := planner.addInstance(instance, observed); err != nil {
			return d.ProcessInstanceVariableUpdatePlan{}, err
		}
	}

	tenantContext, err := d.WithTenantEvidence(baseTenant, planner.tenantIDs, planner.unknownTenantCount)
	if err != nil {
		return d.ProcessInstanceVariableUpdatePlan{}, err
	}
	return planner.plan(requestedKeys, tenantContext), nil
}

type processInstanceVariablePlanner struct {
	requested          map[string]any
	instances          []d.ProcessInstanceVariablePlan
	tenantIDs          []string
	unknownTenantCount int
}

// addInstance validates and classifies one complete root-variable collection.
func (p *processInstanceVariablePlanner) addInstance(instance d.ProcessInstance, observed []d.ProcessInstanceVariable) error {
	rootVariables, tenantID, err := validateRootVariableObservations(instance, observed)
	if err != nil {
		return err
	}
	if tenantID == "" {
		p.unknownTenantCount++
	} else {
		p.tenantIDs = append(p.tenantIDs, tenantID)
	}

	instancePlan := d.ProcessInstanceVariablePlan{
		ProcessInstanceKey: instance.Key,
		ScopeKey:           instance.Key,
		TenantId:           tenantID,
		MutationVariables:  make(map[string]any),
		VerificationValues: copyProcessInstanceVariableMap(p.requested),
	}
	for _, name := range sortedProcessInstanceVariableNames(p.requested) {
		after := p.requested[name]
		current, found := rootVariables[name]
		if !found {
			instancePlan.Additions = append(instancePlan.Additions, d.ProcessInstanceVariablePlannedValue{Name: name, Value: copyProcessInstanceVariableValue(after)})
			instancePlan.MutationVariables[name] = copyProcessInstanceVariableValue(after)
			continue
		}
		before := decodeProcessInstanceObservedValue(current.Value)
		if !current.APITruncated && normalizedProcessInstanceVariableValuesEqual(after, current.Value) {
			instancePlan.UnchangedRequested = append(instancePlan.UnchangedRequested, d.ProcessInstanceVariablePlannedValue{Name: name, Value: before})
		} else {
			instancePlan.Changes = append(instancePlan.Changes, d.ProcessInstanceVariablePlannedChange{
				Name: name, Before: before, After: copyProcessInstanceVariableValue(after), APITruncated: current.APITruncated,
			})
			instancePlan.MutationVariables[name] = copyProcessInstanceVariableValue(after)
		}
		delete(rootVariables, name)
	}
	for _, name := range sortedProcessInstanceVariableNames(rootVariables) {
		current := rootVariables[name]
		instancePlan.Untouched = append(instancePlan.Untouched, d.ProcessInstanceVariablePlannedValue{
			Name: name, Value: decodeProcessInstanceObservedValue(current.Value), APITruncated: current.APITruncated,
		})
	}
	if len(instancePlan.MutationVariables) == 0 {
		instancePlan.MutationVariables = nil
	}
	p.instances = append(p.instances, instancePlan)
	return nil
}

// plan assembles collection counts from the same per-instance classifications
// that supply preview and eventual execution payloads.
func (p *processInstanceVariablePlanner) plan(keys typex.Keys, tenantContext d.TenantContext) d.ProcessInstanceVariableUpdatePlan {
	plan := d.ProcessInstanceVariableUpdatePlan{
		RequestedKeys: append([]string(nil), keys...), RequestedValues: copyProcessInstanceVariableMap(p.requested),
		Instances: p.instances, RequestedCount: len(keys), TenantContext: tenantContext,
	}
	if plan.Instances == nil {
		plan.Instances = make([]d.ProcessInstanceVariablePlan, 0)
	}
	for _, instance := range plan.Instances {
		plan.VariableAddCount += len(instance.Additions)
		plan.VariableChangeCount += len(instance.Changes)
		plan.VariableUnchangedCount += len(instance.UnchangedRequested)
		plan.VariableUntouchedCount += len(instance.Untouched)
		if len(instance.MutationVariables) > 0 {
			plan.UpdateCount++
		}
	}
	return plan
}

// validateRootVariableObservations keeps only the selected root and rejects
// duplicate root/name observations whose identity or value facts conflict.
func validateRootVariableObservations(instance d.ProcessInstance, observed []d.ProcessInstanceVariable) (map[string]d.ProcessInstanceVariable, string, error) {
	tenantID := instance.TenantId
	byName := make(map[string]d.ProcessInstanceVariable, len(observed))
	for _, variable := range observed {
		if variable.ProcessInstanceKey != instance.Key || variable.ScopeKey != instance.Key {
			continue
		}
		if strings.TrimSpace(variable.Name) == "" {
			return nil, "", fmt.Errorf("%w: process-instance %s returned a root variable with a blank name", d.ErrMalformedResponse, instance.Key)
		}
		if variable.TenantId != "" {
			if tenantID != "" && tenantID != variable.TenantId {
				return nil, "", processInstancePlanningValidationError("process-instance %s variable %q has conflicting tenant evidence", instance.Key, variable.Name)
			}
			tenantID = variable.TenantId
		}
		canonical, ok := normalizeProcessInstanceObservedValue(variable.Value)
		if !ok && !variable.APITruncated {
			return nil, "", fmt.Errorf("%w: process-instance %s variable %q contains invalid complete JSON", d.ErrMalformedResponse, instance.Key, variable.Name)
		}
		previous, exists := byName[variable.Name]
		if !exists {
			byName[variable.Name] = variable
			continue
		}
		previousCanonical, previousOK := normalizeProcessInstanceObservedValue(previous.Value)
		if !previousOK {
			previousCanonical = previous.Value
		}
		if !ok {
			canonical = variable.Value
		}
		if knownProcessInstanceStringsConflict(previous.VariableKey, variable.VariableKey) ||
			knownProcessInstanceStringsConflict(previous.TenantId, variable.TenantId) ||
			previous.APITruncated != variable.APITruncated || !reflect.DeepEqual(previousCanonical, canonical) {
			return nil, "", processInstancePlanningValidationError("process-instance %s variable %q has conflicting duplicate observations", instance.Key, variable.Name)
		}
	}
	return byName, tenantID, nil
}

// validateProcessInstanceVariableUpdateInput rejects blank identities and
// values that cannot be represented by the JSON variable payload contract.
func validateProcessInstanceVariableUpdateInput(keys typex.Keys, variables map[string]any) error {
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			return processInstancePlanningValidationError("requested process-instance key must not be blank")
		}
	}
	for name, value := range variables {
		if strings.TrimSpace(name) == "" {
			return processInstancePlanningValidationError("requested variable name must not be blank")
		}
		if _, err := json.Marshal(value); err != nil {
			return processInstancePlanningValidationError("requested variable %q is not JSON-compatible: %v", name, err)
		}
	}
	return nil
}

// validateSelectedProcessInstance prevents malformed bulk results from
// changing a selected root destination.
func validateSelectedProcessInstance(requestedKey string, instance d.ProcessInstance) error {
	if strings.TrimSpace(instance.Key) == "" || instance.Key != requestedKey {
		return processInstancePlanningValidationError("requested process-instance %s returned invalid identity %q", requestedKey, instance.Key)
	}
	return nil
}

// normalizedProcessInstanceVariableValuesEqual applies the same JSON data
// model comparison used by user-task variable planning.
func normalizedProcessInstanceVariableValuesEqual(requested any, observedRaw string) bool {
	requestedValue, ok := normalizeProcessInstanceRequestedValue(requested)
	if !ok {
		return false
	}
	observedValue, ok := normalizeProcessInstanceObservedValue(observedRaw)
	return ok && reflect.DeepEqual(requestedValue, observedValue)
}

// normalizeProcessInstanceRequestedValue round-trips requested JSON while
// preserving number spelling through json.Number.
func normalizeProcessInstanceRequestedValue(value any) (any, bool) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	return decodeNormalizedProcessInstanceJSON(raw)
}

// normalizeProcessInstanceObservedValue decodes one raw backend JSON value.
func normalizeProcessInstanceObservedValue(raw string) (any, bool) {
	return decodeNormalizedProcessInstanceJSON([]byte(raw))
}

// decodeNormalizedProcessInstanceJSON requires exactly one valid JSON value.
func decodeNormalizedProcessInstanceJSON(raw []byte) (any, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, false
	}
	return value, true
}

// decodeProcessInstanceObservedValue preserves truncated fragments while
// decoding valid raw JSON for preview values.
func decodeProcessInstanceObservedValue(raw string) any {
	value, ok := normalizeProcessInstanceObservedValue(raw)
	if !ok {
		return raw
	}
	return value
}

// sortedProcessInstanceVariableNames keeps classifications deterministic.
func sortedProcessInstanceVariableNames[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// copyProcessInstanceVariableMap recursively isolates ordinary JSON maps and arrays.
func copyProcessInstanceVariableMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for name, value := range in {
		out[name] = copyProcessInstanceVariableValue(value)
	}
	return out
}

// copyProcessInstanceVariableValue recursively copies mutable JSON values.
func copyProcessInstanceVariableValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		return copyProcessInstanceVariableMap(value)
	case []any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = copyProcessInstanceVariableValue(value[i])
		}
		return out
	default:
		return value
	}
}

// knownProcessInstanceStringsConflict treats missing optional evidence as unknown.
func knownProcessInstanceStringsConflict(a, b string) bool {
	return a != "" && b != "" && a != b
}

// processInstanceUpdateConfiguredTenant safely reads optional configuration evidence.
func processInstanceUpdateConfiguredTenant(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.App.Tenant
}

// emptyProcessInstanceVariableUpdatePlan returns initialized collections for a
// valid internal no-work request.
func emptyProcessInstanceVariableUpdatePlan(requested map[string]any, tenantContext d.TenantContext) d.ProcessInstanceVariableUpdatePlan {
	return d.ProcessInstanceVariableUpdatePlan{
		RequestedKeys: make([]string, 0), RequestedValues: copyProcessInstanceVariableMap(requested),
		Instances: make([]d.ProcessInstanceVariablePlan, 0), TenantContext: tenantContext,
	}
}

// processInstancePlanningValidationError classifies unsafe input or discovery facts.
func processInstancePlanningValidationError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", d.ErrValidation, fmt.Sprintf(format, args...))
}
