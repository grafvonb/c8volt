// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strings"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
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

// VariableUpdateAPI exposes collection-wide process-instance variable planning.
type VariableUpdateAPI interface {
	PlanProcessInstanceVariableUpdates(ctx context.Context, keys typex.Keys, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdatePlan, error)
}

// VariableUpdateService composes process-instance identity reads with complete
// root-variable discovery. Mutation dependencies are added by the execution work unit.
type VariableUpdateService struct {
	processInstances VariableUpdateProcessInstanceAPI
	variables        VariableUpdateRootVariableAPI
	config           *config.Config
	log              *slog.Logger
}

// NewVariableUpdates creates the version-neutral process-instance variable planner.
func NewVariableUpdates(processInstances VariableUpdateProcessInstanceAPI, variables VariableUpdateRootVariableAPI, cfg *config.Config, log *slog.Logger) VariableUpdateAPI {
	if log == nil {
		log = slog.Default()
	}
	return &VariableUpdateService{processInstances: processInstances, variables: variables, config: cfg, log: log}
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
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
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
