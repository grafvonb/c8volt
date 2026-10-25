// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
)

// VariableUpdateElementAPI supplies exact runtime identities without discovery fallback.
type VariableUpdateElementAPI interface {
	GetElement(context.Context, string, ...services.CallOption) (d.Element, error)
}

// VariableUpdateAPI exposes collection-wide local planning and frozen execution.
type VariableUpdateAPI interface {
	ExecuteElementVariableUpdates(context.Context, d.ElementVariableUpdatePlan, int, ...services.CallOption) (d.ElementVariableUpdateResults, error)
	PlanElementVariableUpdates(context.Context, typex.Keys, map[string]any, ...services.CallOption) (d.ElementVariableUpdatePlan, error)
}

// VariableUpdateService owns identity validation, local reads and frozen classifications.
type VariableUpdateService struct {
	elements  VariableUpdateElementAPI
	variables scopeVariableSearcher
	config    *config.Config
	writer    VariableUpdateScopeAPI
}

// VariableUpdateScopeAPI supplies exact-local submissions with retained attempt evidence.
type VariableUpdateScopeAPI interface {
	UpdateScopeVariables(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error)
}

// NewVariableUpdates composes existing readers and an optional writer without new clients or limiters.
func NewVariableUpdates(elements VariableUpdateElementAPI, variables scopeVariableSearcher, cfg *config.Config) VariableUpdateAPI {
	writer, _ := variables.(VariableUpdateScopeAPI)
	return &VariableUpdateService{elements: elements, variables: variables, config: cfg, writer: writer}
}

// PlanElementVariableUpdates completes all authorized exact-local reads before
// publishing a plan. Errors discard all partial classifications; no writes occur.
func (s *VariableUpdateService) PlanElementVariableUpdates(ctx context.Context, keys typex.Keys, variables map[string]any, opts ...services.CallOption) (d.ElementVariableUpdatePlan, error) {
	requestedKeys := keys.Unique()
	if err := validateElementVariableUpdateInput(requestedKeys, variables); err != nil {
		return d.ElementVariableUpdatePlan{}, err
	}
	if err := ctx.Err(); err != nil {
		return d.ElementVariableUpdatePlan{}, err
	}
	if s.elements == nil || s.variables == nil {
		return d.ElementVariableUpdatePlan{}, fmt.Errorf("%w: element variable planning requires element and variable services", d.ErrPrecondition)
	}
	baseTenant, err := d.NewExplicitKeysTenantContext(elementUpdateConfiguredTenant(s.config))
	if err != nil {
		return d.ElementVariableUpdatePlan{}, err
	}
	planner := elementVariablePlanner{requested: copyElementVariableMap(variables)}
	opts = append(append([]services.CallOption(nil), opts...), services.WithIgnoreTenant())
	for _, key := range requestedKeys {
		if err := ctx.Err(); err != nil {
			return d.ElementVariableUpdatePlan{}, err
		}
		item, err := s.elements.GetElement(ctx, key, opts...)
		if err != nil {
			return d.ElementVariableUpdatePlan{}, fmt.Errorf("read selected element %s: %w", key, err)
		}
		if err := validateSelectedElement(key, item); err != nil {
			return d.ElementVariableUpdatePlan{}, err
		}
		query := d.ScopeVariableQuery{ProcessInstanceKey: item.ProcessInstanceKey, ScopeKey: item.ElementInstanceKey}
		if item.TenantId != "" {
			tenant := item.TenantId
			query.ExpectedTenantId = &tenant
		}
		if err := ctx.Err(); err != nil {
			return d.ElementVariableUpdatePlan{}, err
		}
		observed, err := s.variables.SearchScopeVariables(ctx, query, opts...)
		if err != nil {
			return d.ElementVariableUpdatePlan{}, fmt.Errorf("read local variables for element %s: %w", key, err)
		}
		if err := ctx.Err(); err != nil {
			return d.ElementVariableUpdatePlan{}, err
		}
		if err := planner.addElement(item, observed); err != nil {
			return d.ElementVariableUpdatePlan{}, err
		}
	}
	tenantContext, err := d.WithTenantEvidence(baseTenant, planner.tenantIDs, planner.unknownTenantCount)
	if err != nil {
		return d.ElementVariableUpdatePlan{}, err
	}
	return validateAndCopyElementVariableUpdatePlan(planner.plan(requestedKeys, tenantContext))
}

type elementVariablePlanner struct {
	requested          map[string]any
	elements           []d.ElementVariablePlan
	tenantIDs          []string
	unknownTenantCount int
}

// addElement validates and classifies one complete local-variable collection.
func (p *elementVariablePlanner) addElement(element d.Element, observed []d.ProcessInstanceVariable) error {
	localVariables, tenantID, err := validateLocalVariableObservations(element, observed)
	if err != nil {
		return err
	}
	if tenantID == "" {
		p.unknownTenantCount++
	} else {
		p.tenantIDs = append(p.tenantIDs, tenantID)
	}

	elementPlan := d.ElementVariablePlan{
		ElementKey:         element.ElementInstanceKey,
		Additions:          make([]d.ElementVariablePlannedValue, 0),
		Changes:            make([]d.ElementVariablePlannedChange, 0),
		UnchangedRequested: make([]d.ElementVariablePlannedValue, 0),
		Untouched:          make([]d.ElementVariablePlannedValue, 0),
		ProcessInstanceKey: element.ProcessInstanceKey,
		ScopeKey:           element.ElementInstanceKey,
		TenantId:           tenantID,
		MutationVariables:  make(map[string]any),
		VerificationValues: copyElementVariableMap(p.requested),
	}
	for _, name := range sortedElementVariableNames(p.requested) {
		after := p.requested[name]
		current, found := localVariables[name]
		if !found {
			elementPlan.Additions = append(elementPlan.Additions, d.ElementVariablePlannedValue{Name: name, Value: copyElementVariableValue(after)})
			elementPlan.MutationVariables[name] = copyElementVariableValue(after)
			continue
		}
		before := decodeElementObservedValue(current.Value)
		if !current.APITruncated && normalizedElementVariableValuesEqual(after, current.Value) {
			elementPlan.UnchangedRequested = append(elementPlan.UnchangedRequested, d.ElementVariablePlannedValue{Name: name, Value: before})
		} else {
			elementPlan.Changes = append(elementPlan.Changes, d.ElementVariablePlannedChange{
				Name: name, Before: before, After: copyElementVariableValue(after), APITruncated: current.APITruncated,
			})
			elementPlan.MutationVariables[name] = copyElementVariableValue(after)
		}
		delete(localVariables, name)
	}
	for _, name := range sortedElementVariableNames(localVariables) {
		current := localVariables[name]
		elementPlan.Untouched = append(elementPlan.Untouched, d.ElementVariablePlannedValue{
			Name: name, Value: decodeElementObservedValue(current.Value), APITruncated: current.APITruncated,
		})
	}
	if len(elementPlan.MutationVariables) == 0 {
		elementPlan.MutationVariables = nil
	}
	p.elements = append(p.elements, elementPlan)
	return nil
}

// plan assembles collection counts from the same per-element classifications
// that supply preview and eventual execution payloads.
func (p *elementVariablePlanner) plan(keys typex.Keys, tenantContext d.TenantContext) d.ElementVariableUpdatePlan {
	plan := d.ElementVariableUpdatePlan{
		RequestedKeys: append([]string(nil), keys...), RequestedValues: copyElementVariableMap(p.requested),
		Elements: p.elements, RequestedCount: len(keys), TenantContext: tenantContext,
	}
	if plan.Elements == nil {
		plan.Elements = make([]d.ElementVariablePlan, 0)
	}
	for _, element := range plan.Elements {
		plan.VariableAddCount += len(element.Additions)
		plan.VariableChangeCount += len(element.Changes)
		plan.VariableUnchangedCount += len(element.UnchangedRequested)
		plan.VariableUntouchedCount += len(element.Untouched)
		if len(element.MutationVariables) > 0 {
			plan.UpdateCount++
		}
	}
	return plan
}

// validateLocalVariableObservations keeps only the selected scope and rejects
// duplicate scope/name observations whose identity or value facts conflict.
func validateLocalVariableObservations(element d.Element, observed []d.ProcessInstanceVariable) (map[string]d.ProcessInstanceVariable, string, error) {
	tenantID := element.TenantId
	byName := make(map[string]d.ProcessInstanceVariable, len(observed))
	for _, variable := range observed {
		if strings.TrimSpace(variable.ProcessInstanceKey) == "" || strings.TrimSpace(variable.ScopeKey) == "" {
			return nil, "", fmt.Errorf("%w: element %s variable observation is missing owner or scope", d.ErrMalformedResponse, element.ElementInstanceKey)
		}
		if variable.ScopeKey != element.ElementInstanceKey {
			continue
		}
		if variable.ProcessInstanceKey != element.ProcessInstanceKey {
			return nil, "", elementPlanningValidationError("element %s variable %q has conflicting owner evidence", element.ElementInstanceKey, variable.Name)
		}
		if strings.TrimSpace(variable.Name) == "" {
			return nil, "", fmt.Errorf("%w: element %s returned a local variable with a blank name", d.ErrMalformedResponse, element.ElementInstanceKey)
		}
		if variable.TenantId != "" {
			if tenantID != "" && tenantID != variable.TenantId {
				return nil, "", elementPlanningValidationError("element %s variable %q has conflicting tenant evidence", element.ElementInstanceKey, variable.Name)
			}
			tenantID = variable.TenantId
		}
		canonical, ok := normalizeElementObservedValue(variable.Value)
		if !ok && !variable.APITruncated {
			return nil, "", fmt.Errorf("%w: element %s variable %q contains invalid complete JSON", d.ErrMalformedResponse, element.ElementInstanceKey, variable.Name)
		}
		previous, exists := byName[variable.Name]
		if !exists {
			byName[variable.Name] = variable
			continue
		}
		previousCanonical, previousOK := normalizeElementObservedValue(previous.Value)
		if !previousOK {
			previousCanonical = previous.Value
		}
		if !ok {
			canonical = variable.Value
		}
		if knownElementStringsConflict(previous.VariableKey, variable.VariableKey) ||
			knownElementStringsConflict(previous.TenantId, variable.TenantId) ||
			previous.APITruncated != variable.APITruncated || !reflect.DeepEqual(previousCanonical, canonical) {
			return nil, "", elementPlanningValidationError("element %s variable %q has conflicting duplicate observations", element.ElementInstanceKey, variable.Name)
		}
		// Retain newly observed optional identity so a later duplicate cannot
		// hide a conflict behind the first row's missing evidence.
		if previous.VariableKey == "" {
			previous.VariableKey = variable.VariableKey
		}
		if previous.TenantId == "" {
			previous.TenantId = variable.TenantId
		}
		byName[variable.Name] = previous
	}
	return byName, tenantID, nil
}

// validateElementVariableUpdateInput rejects blank identities and
// values that cannot be represented by the JSON variable payload contract.
func validateElementVariableUpdateInput(keys typex.Keys, variables map[string]any) error {
	if len(keys) == 0 {
		return elementPlanningValidationError("at least one element key is required")
	}
	if variables == nil {
		return elementPlanningValidationError("requested variables must be a non-null object")
	}
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			return elementPlanningValidationError("requested element key must not be blank")
		}
	}
	for name, value := range variables {
		if strings.TrimSpace(name) == "" {
			return elementPlanningValidationError("requested variable name must not be blank")
		}
		if _, err := json.Marshal(value); err != nil {
			return elementPlanningValidationError("requested variable %q is not JSON-compatible: %v", name, err)
		}
	}
	return nil
}

// validateSelectedElement prevents malformed lookup results from
// changing a selected scope destination.
func validateSelectedElement(requestedKey string, element d.Element) error {
	if strings.TrimSpace(element.ElementInstanceKey) == "" || element.ElementInstanceKey != requestedKey || strings.TrimSpace(element.ProcessInstanceKey) == "" {
		return elementPlanningValidationError("requested element %s returned invalid identity %q", requestedKey, element.ElementInstanceKey)
	}
	return nil
}

// normalizedElementVariableValuesEqual applies the same JSON data
// model comparison used by user-task variable planning.
func normalizedElementVariableValuesEqual(requested any, observedRaw string) bool {
	requestedValue, ok := normalizeElementRequestedValue(requested)
	if !ok {
		return false
	}
	observedValue, ok := normalizeElementObservedValue(observedRaw)
	return ok && reflect.DeepEqual(requestedValue, observedValue)
}

// normalizeElementRequestedValue round-trips requested JSON while
// preserving number spelling through json.Number.
func normalizeElementRequestedValue(value any) (any, bool) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	return decodeNormalizedElementJSON(raw)
}

// normalizeElementObservedValue decodes one raw backend JSON value.
func normalizeElementObservedValue(raw string) (any, bool) {
	return decodeNormalizedElementJSON([]byte(raw))
}

// decodeNormalizedElementJSON requires exactly one valid JSON value.
func decodeNormalizedElementJSON(raw []byte) (any, bool) {
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

// decodeElementObservedValue preserves truncated fragments while
// decoding valid raw JSON for preview values.
func decodeElementObservedValue(raw string) any {
	value, ok := normalizeElementObservedValue(raw)
	if !ok {
		return raw
	}
	return value
}

// sortedElementVariableNames keeps classifications deterministic.
func sortedElementVariableNames[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// copyElementVariableMap recursively isolates ordinary JSON maps and arrays.
func copyElementVariableMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for name, value := range in {
		out[name] = copyElementVariableValue(value)
	}
	return out
}

// copyElementVariableValue isolates mutable JSON storage while keeping
// ordinary interface maps and slices in their established representation.
func copyElementVariableValue(value any) any {
	switch value := value.(type) {
	case nil:
		return nil
	case map[string]any:
		return copyElementVariableMap(value)
	case []any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = copyElementVariableValue(value[i])
		}
		return out
	case json.RawMessage:
		return append(json.RawMessage(nil), value...)
	default:
		return snapshotTypedElementJSONValue(value)
	}
}

// snapshotTypedElementJSONValue retains concrete typed JSON values when they round-trip faithfully.
// Marshal-only types fall back to their decoded JSON value so custom storage
// cannot alias the caller or silently change the approved serialized payload.
func snapshotTypedElementJSONValue(value any) any {
	raw, err := json.Marshal(value)
	if err != nil {
		// Invalid input is rejected before copying.
		return value
	}
	target := reflect.New(reflect.TypeOf(value))
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target.Interface()); err == nil {
		snapshot := target.Elem().Interface()
		if encoded, err := json.Marshal(snapshot); err == nil && bytes.Equal(raw, encoded) {
			return snapshot
		}
	}
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var snapshot any
	_ = decoder.Decode(&snapshot) // json.Marshal above guarantees one valid JSON value.
	return snapshot
}

// knownElementStringsConflict treats missing optional evidence as unknown.
func knownElementStringsConflict(a, b string) bool {
	return a != "" && b != "" && a != b
}

// elementUpdateConfiguredTenant safely reads optional configuration evidence.
func elementUpdateConfiguredTenant(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.App.Tenant
}

// elementPlanningValidationError classifies unsafe input or discovery facts.
func elementPlanningValidationError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", d.ErrValidation, fmt.Sprintf(format, args...))
}
