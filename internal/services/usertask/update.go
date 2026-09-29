// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

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
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// ScopeVariableAPI is the submission primitive required by the composed
// user-task update workflow. Planning deliberately never calls it.
type ScopeVariableAPI interface {
	UpdateScopeVariables(ctx context.Context, scopeKey string, variables map[string]any, opts ...services.CallOption) (d.ScopeVariableUpdateResponse, error)
}

// VariableUpdateAPI exposes the two-stage frozen-plan update workflow.
type VariableUpdateAPI interface {
	PlanUserTaskVariableUpdates(ctx context.Context, keys typex.Keys, variables map[string]any, opts ...services.CallOption) (d.UserTaskVariableUpdatePlan, error)
	ExecuteUserTaskVariableUpdates(ctx context.Context, plan d.UserTaskVariableUpdatePlan, wantedWorkers int, opts ...services.CallOption) (d.UserTaskVariableUpdateResults, error)
}

// VariableUpdateService composes native task reads with scope-local writes.
// The writer, backoff configuration, and logger are retained for execution.
type VariableUpdateService struct {
	userTasks API
	variables ScopeVariableAPI
	config    *config.Config
	log       *slog.Logger
}

// NewVariableUpdates creates the version-neutral user-task variable workflow.
func NewVariableUpdates(userTasks API, variables ScopeVariableAPI, cfg *config.Config, log *slog.Logger) VariableUpdateAPI {
	if log == nil {
		log = slog.Default()
	}
	return &VariableUpdateService{userTasks: userTasks, variables: variables, config: cfg, log: log}
}

// PlanUserTaskVariableUpdates completes every selected task and variable read
// before returning a frozen, deduplicated scope plan.
func (s *VariableUpdateService) PlanUserTaskVariableUpdates(ctx context.Context, keys typex.Keys, variables map[string]any, opts ...services.CallOption) (d.UserTaskVariableUpdatePlan, error) {
	requestedKeys := keys.Unique()
	baseTenant, err := d.NewExplicitKeysTenantContext(configuredTenant(s.config))
	if err != nil {
		return d.UserTaskVariableUpdatePlan{}, err
	}
	if len(requestedKeys) == 0 {
		return emptyUserTaskVariableUpdatePlan(baseTenant), nil
	}
	if err := validateRequestedVariables(variables); err != nil {
		return d.UserTaskVariableUpdatePlan{}, err
	}
	if s.userTasks == nil {
		return d.UserTaskVariableUpdatePlan{}, fmt.Errorf("%w: user-task variable planning requires a user-task service", d.ErrPrecondition)
	}

	tasks, err := GetUserTasks(ctx, s.userTasks, requestedKeys, 0, opts...)
	if err != nil {
		return d.UserTaskVariableUpdatePlan{}, fmt.Errorf("read selected user tasks: %w", err)
	}

	planner := userTaskVariablePlanner{
		requested:     copyVariableMap(variables),
		targetByScope: make(map[string]int),
		observed:      make(map[scopeVariableIdentity]observedScopeVariable),
	}
	for i, task := range tasks {
		if err := validateSelectedUserTask(requestedKeys[i], task); err != nil {
			return d.UserTaskVariableUpdatePlan{}, err
		}
		effective, err := SearchUserTaskEffectiveVariables(ctx, s.userTasks, task.Key, opts...)
		if err != nil {
			return d.UserTaskVariableUpdatePlan{}, fmt.Errorf("read effective variables for user task %s: %w", task.Key, err)
		}
		if err := planner.addTask(task, effective); err != nil {
			return d.UserTaskVariableUpdatePlan{}, err
		}
	}

	tenantContext, err := d.WithTenantEvidence(baseTenant, planner.tenantIDs, planner.unknownTenantCount)
	if err != nil {
		return d.UserTaskVariableUpdatePlan{}, err
	}
	return planner.plan(requestedKeys, tenantContext), nil
}

// ExecuteUserTaskVariableUpdates validates and freezes a supplied plan before
// submitting each unique scope once and confirming fully accepted tasks.
func (s *VariableUpdateService) ExecuteUserTaskVariableUpdates(ctx context.Context, plan d.UserTaskVariableUpdatePlan, wantedWorkers int, opts ...services.CallOption) (d.UserTaskVariableUpdateResults, error) {
	frozen, err := validateAndCopyUserTaskVariableUpdatePlan(plan)
	if err != nil {
		return d.UserTaskVariableUpdateResults{}, err
	}
	cfg := services.ApplyCallOptions(opts)
	if cfg.DryRun || len(frozen.Targets) == 0 {
		return d.UserTaskVariableUpdateResults{Items: make([]d.UserTaskVariableUpdateResult, 0)}, nil
	}
	if s.variables == nil {
		return d.UserTaskVariableUpdateResults{}, fmt.Errorf("%w: user-task variable execution requires a scope variable service", d.ErrPrecondition)
	}
	if !cfg.NoWait && s.userTasks == nil {
		return d.UserTaskVariableUpdateResults{}, fmt.Errorf("%w: user-task variable confirmation requires a user-task service", d.ErrPrecondition)
	}

	workers := toolx.DetermineNoOfWorkers(len(frozen.Targets), wantedWorkers, cfg.NoWorkerLimit)
	outcomes, mutationErr := pool.ExecuteSlice[d.ScopeVariableUpdateTarget, d.ScopeVariableUpdateOutcome](ctx, frozen.Targets, workers, cfg.FailFast,
		func(ctx context.Context, target d.ScopeVariableUpdateTarget, _ int) (d.ScopeVariableUpdateOutcome, error) {
			response, err := s.variables.UpdateScopeVariables(ctx, target.ScopeKey, copyVariableMap(target.Variables), opts...)
			outcome := d.ScopeVariableUpdateOutcome{
				ScopeKey: target.ScopeKey, Names: sortedStringMapKeys(target.Variables),
				Accepted: response.Accepted, StatusCode: response.StatusCode, Message: response.Status,
			}
			if err != nil {
				outcome.Status = d.ScopeVariableUpdateStatusMutationFailed
				outcome.Error = err.Error()
				return outcome, err
			}
			if !response.Accepted {
				err = fmt.Errorf("%w: scope %s variable update was not accepted", d.ErrUpstream, target.ScopeKey)
				outcome.Status = d.ScopeVariableUpdateStatusMutationFailed
				outcome.Error = err.Error()
				return outcome, err
			}
			outcome.Status = d.ScopeVariableUpdateStatusSubmitted
			return outcome, nil
		})
	mutationErr = joinContextError(mutationErr, ctx.Err())
	for i := range frozen.Targets {
		if outcomes[i].ScopeKey == "" {
			outcomes[i] = skippedScopeVariableUpdateOutcome(frozen.Targets[i])
		}
	}

	results := resultsFromScopeVariableUpdateOutcomes(frozen, outcomes)
	if cfg.NoWait {
		return results, mutationErr
	}

	confirmationIndexes := make([]int, 0, len(results.Items))
	for i := range results.Items {
		if results.Items[i].Status == d.UserTaskVariableUpdateStatusSubmitted {
			confirmationIndexes = append(confirmationIndexes, i)
		}
	}
	if len(confirmationIndexes) == 0 {
		return results, mutationErr
	}
	confirmationWorkers := toolx.DetermineNoOfWorkers(len(confirmationIndexes), wantedWorkers, cfg.NoWorkerLimit)
	confirmations, confirmationErr := pool.ExecuteSlice[int, userTaskVariableConfirmation](ctx, confirmationIndexes, confirmationWorkers, cfg.FailFast,
		func(ctx context.Context, resultIndex int, _ int) (userTaskVariableConfirmation, error) {
			err := waitForUserTaskVariableUpdate(ctx, s.userTasks, s.config, frozen.UserTasks[resultIndex], opts...)
			return userTaskVariableConfirmation{attempted: true, err: err}, err
		})
	confirmationErr = joinContextError(confirmationErr, ctx.Err())
	for i, resultIndex := range confirmationIndexes {
		if !confirmations[i].attempted {
			continue
		}
		if confirmations[i].err != nil {
			results.Items[resultIndex].Status = d.UserTaskVariableUpdateStatusConfirmationFailed
			results.Items[resultIndex].ConfirmationStatus = "failed"
			results.Items[resultIndex].Error = confirmations[i].err.Error()
			continue
		}
		results.Items[resultIndex].Status = d.UserTaskVariableUpdateStatusConfirmed
		results.Items[resultIndex].ConfirmationStatus = "confirmed"
	}
	return results, errors.Join(mutationErr, confirmationErr)
}

type userTaskVariableConfirmation struct {
	attempted bool
	err       error
}

// skippedScopeVariableUpdateOutcome represents a frozen target that the pool
// did not start after fail-fast or caller cancellation.
func skippedScopeVariableUpdateOutcome(target d.ScopeVariableUpdateTarget) d.ScopeVariableUpdateOutcome {
	return d.ScopeVariableUpdateOutcome{
		ScopeKey: target.ScopeKey, Names: sortedStringMapKeys(target.Variables),
		Status: d.ScopeVariableUpdateStatusSkipped,
	}
}

// resultsFromScopeVariableUpdateOutcomes propagates each physical scope fact
// to all associated task results without losing stable requested-key order.
func resultsFromScopeVariableUpdateOutcomes(plan d.UserTaskVariableUpdatePlan, outcomes []d.ScopeVariableUpdateOutcome) d.UserTaskVariableUpdateResults {
	byScope := make(map[string]d.ScopeVariableUpdateOutcome, len(outcomes))
	for _, outcome := range outcomes {
		byScope[outcome.ScopeKey] = outcome
	}
	items := make([]d.UserTaskVariableUpdateResult, len(plan.UserTasks))
	for i, task := range plan.UserTasks {
		result := d.UserTaskVariableUpdateResult{
			Key: task.UserTaskKey, Variables: requestedVariablesForTaskPlan(task), ConfirmationStatus: "skipped",
		}
		if len(task.TargetScopeKeys) == 0 {
			result.Status = d.UserTaskVariableUpdateStatusUnchanged
			items[i] = result
			continue
		}
		var hasFailed, hasSkipped bool
		var taskErrors []error
		for _, scopeKey := range task.TargetScopeKeys {
			outcome := byScope[scopeKey]
			outcome.Names = associatedNamesForTask(plan.Targets, scopeKey, task.UserTaskKey)
			result.Scopes = append(result.Scopes, outcome)
			result.MutationAccepted = result.MutationAccepted || outcome.Accepted
			switch outcome.Status {
			case d.ScopeVariableUpdateStatusMutationFailed:
				hasFailed = true
				if outcome.Error != "" {
					taskErrors = append(taskErrors, errors.New(outcome.Error))
				}
			case d.ScopeVariableUpdateStatusSkipped:
				hasSkipped = true
			}
		}
		switch {
		case hasFailed:
			result.Status = d.UserTaskVariableUpdateStatusMutationFailed
		case hasSkipped:
			result.Status = d.UserTaskVariableUpdateStatusSkipped
		default:
			result.Status = d.UserTaskVariableUpdateStatusSubmitted
		}
		if err := errors.Join(taskErrors...); err != nil {
			result.Error = err.Error()
		}
		items[i] = result
	}
	return d.UserTaskVariableUpdateResults{Items: items}
}

// requestedVariablesForTaskPlan reconstructs the task-level requested payload
// from its mutually exclusive plan categories.
func requestedVariablesForTaskPlan(task d.UserTaskVariablePlan) map[string]any {
	variables := make(map[string]any, len(task.Additions)+len(task.Changes)+len(task.UnchangedRequested))
	for _, value := range task.Additions {
		variables[value.Name] = copyVariableValue(value.Value)
	}
	for _, value := range task.Changes {
		variables[value.Name] = copyVariableValue(value.After)
	}
	for _, value := range task.UnchangedRequested {
		variables[value.Name] = copyVariableValue(value.Value)
	}
	return variables
}

// associatedNamesForTask narrows a shared scope outcome to the logical names
// on the selected task while preserving the planned sorted order.
func associatedNamesForTask(targets []d.ScopeVariableUpdateTarget, scopeKey, taskKey string) []string {
	for _, target := range targets {
		if target.ScopeKey != scopeKey {
			continue
		}
		for _, association := range target.Associations {
			if association.UserTaskKey == taskKey {
				return append([]string(nil), association.Names...)
			}
		}
	}
	return nil
}

// joinContextError retains cancellation when the pool stopped before it could
// attach the caller's context error to a started work item.
func joinContextError(err, contextErr error) error {
	if contextErr == nil || errors.Is(err, contextErr) {
		return err
	}
	return errors.Join(err, contextErr)
}

type scopeVariableIdentity struct {
	scopeKey string
	name     string
}

type observedScopeVariable struct {
	variableKey        string
	processInstanceKey string
	tenantID           string
	canonicalValue     any
	apiTruncated       bool
}

type userTaskVariablePlanner struct {
	requested          map[string]any
	tasks              []d.UserTaskVariablePlan
	targets            []d.ScopeVariableUpdateTarget
	targetByScope      map[string]int
	observed           map[scopeVariableIdentity]observedScopeVariable
	tenantIDs          []string
	unknownTenantCount int
}

// addTask validates one complete effective view and adds its classifications
// only after all cross-task identity evidence remains consistent.
func (p *userTaskVariablePlanner) addTask(task d.UserTask, variables []d.ProcessInstanceVariable) error {
	tenantID, err := p.validateEffectiveVariables(task, variables)
	if err != nil {
		return err
	}
	if tenantID == "" {
		p.unknownTenantCount++
	} else {
		p.tenantIDs = append(p.tenantIDs, tenantID)
	}

	byName := make(map[string]d.ProcessInstanceVariable, len(variables))
	for _, variable := range variables {
		byName[variable.Name] = variable
	}

	taskPlan := d.UserTaskVariablePlan{
		UserTaskKey:        task.Key,
		ElementInstanceKey: task.ElementInstanceKey,
		TenantId:           tenantID,
	}
	for _, name := range sortedStringMapKeys(p.requested) {
		after := p.requested[name]
		variable, found := byName[name]
		if !found {
			taskPlan.Additions = append(taskPlan.Additions, d.UserTaskVariablePlannedValue{
				Name: name, ScopeKey: task.ElementInstanceKey, Value: copyVariableValue(after),
			})
			if err := p.addTarget(&taskPlan, task.ElementInstanceKey, tenantID, name, after); err != nil {
				return err
			}
			continue
		}

		before := decodeObservedVariableValue(variable.Value)
		inherited := variable.ScopeKey != task.ElementInstanceKey
		if !variable.APITruncated && normalizedJSONValuesEqual(after, variable.Value) {
			taskPlan.UnchangedRequested = append(taskPlan.UnchangedRequested, d.UserTaskVariablePlannedValue{
				Name: name, ScopeKey: variable.ScopeKey, Inherited: inherited, Value: before,
			})
		} else {
			taskPlan.Changes = append(taskPlan.Changes, d.UserTaskVariablePlannedChange{
				Name: name, ScopeKey: variable.ScopeKey, Inherited: inherited,
				Before: before, After: copyVariableValue(after), APITruncated: variable.APITruncated,
			})
			if err := p.addTarget(&taskPlan, variable.ScopeKey, tenantID, name, after); err != nil {
				return err
			}
		}
		delete(byName, name)
	}

	for _, name := range sortedStringMapKeys(byName) {
		variable := byName[name]
		taskPlan.Untouched = append(taskPlan.Untouched, d.UserTaskVariablePlannedValue{
			Name: name, ScopeKey: variable.ScopeKey,
			Inherited: variable.ScopeKey != task.ElementInstanceKey,
			Value:     decodeObservedVariableValue(variable.Value), APITruncated: variable.APITruncated,
		})
	}
	p.tasks = append(p.tasks, taskPlan)
	return nil
}

// validateEffectiveVariables rejects incomplete identities and contradictory
// scope/name observations before any plan can become executable.
func (p *userTaskVariablePlanner) validateEffectiveVariables(task d.UserTask, variables []d.ProcessInstanceVariable) (string, error) {
	tenantID := task.TenantId
	for _, variable := range variables {
		if strings.TrimSpace(variable.Name) == "" {
			return "", planningValidationError("user task %s returned a variable with a blank name", task.Key)
		}
		if strings.TrimSpace(variable.ScopeKey) == "" {
			return "", planningValidationError("user task %s variable %q has no scope key", task.Key, variable.Name)
		}
		if variable.ProcessInstanceKey != "" && task.ProcessInstanceKey != "" && variable.ProcessInstanceKey != task.ProcessInstanceKey {
			return "", planningValidationError("user task %s variable %q has conflicting process-instance identity", task.Key, variable.Name)
		}
		if variable.TenantId != "" {
			if tenantID != "" && tenantID != variable.TenantId {
				return "", planningValidationError("user task %s variable %q has conflicting tenant evidence", task.Key, variable.Name)
			}
			tenantID = variable.TenantId
		}

		canonical, ok := normalizeObservedJSONValue(variable.Value)
		if !ok && !variable.APITruncated {
			return "", fmt.Errorf("%w: user task %s variable %q contains invalid complete JSON", d.ErrMalformedResponse, task.Key, variable.Name)
		}
		if !ok {
			canonical = variable.Value
		}
		identity := scopeVariableIdentity{scopeKey: variable.ScopeKey, name: variable.Name}
		observed := observedScopeVariable{
			variableKey: variable.VariableKey, processInstanceKey: variable.ProcessInstanceKey,
			tenantID: variable.TenantId, canonicalValue: canonical, apiTruncated: variable.APITruncated,
		}
		if previous, exists := p.observed[identity]; exists {
			if conflictingScopeVariableEvidence(previous, observed) {
				return "", planningValidationError("scope %s variable %q has conflicting identity or value evidence", variable.ScopeKey, variable.Name)
			}
		} else {
			p.observed[identity] = observed
		}
	}
	return tenantID, nil
}

// addTarget groups all changed names for a scope and preserves per-task name
// associations without duplicating shared inherited writes.
func (p *userTaskVariablePlanner) addTarget(taskPlan *d.UserTaskVariablePlan, scopeKey, tenantID, name string, value any) error {
	index, exists := p.targetByScope[scopeKey]
	if !exists {
		index = len(p.targets)
		p.targetByScope[scopeKey] = index
		p.targets = append(p.targets, d.ScopeVariableUpdateTarget{
			ScopeKey: scopeKey, TenantId: tenantID, Variables: make(map[string]any),
		})
	}
	target := &p.targets[index]
	if knownStringsConflict(target.TenantId, tenantID) {
		return planningValidationError("scope %s has conflicting tenant evidence", scopeKey)
	}
	if target.TenantId == "" {
		target.TenantId = tenantID
	}
	if previous, exists := target.Variables[name]; exists && !normalizedRequestedValuesEqual(previous, value) {
		return planningValidationError("scope %s variable %q has conflicting requested values", scopeKey, name)
	}
	target.Variables[name] = copyVariableValue(value)
	if len(target.Associations) == 0 || target.Associations[len(target.Associations)-1].UserTaskKey != taskPlan.UserTaskKey {
		target.Associations = append(target.Associations, d.ScopeVariableUpdateAssociation{UserTaskKey: taskPlan.UserTaskKey})
		taskPlan.TargetScopeKeys = append(taskPlan.TargetScopeKeys, scopeKey)
	}
	association := &target.Associations[len(target.Associations)-1]
	association.Names = append(association.Names, name)
	return nil
}

// plan assembles aggregate task-variable counts without replacing them with
// the smaller number of deduplicated physical targets.
func (p *userTaskVariablePlanner) plan(keys typex.Keys, tenantContext d.TenantContext) d.UserTaskVariableUpdatePlan {
	plan := d.UserTaskVariableUpdatePlan{
		RequestedKeys: append([]string(nil), keys...), UserTasks: p.tasks, Targets: p.targets,
		RequestedCount: len(keys), TenantContext: tenantContext, MutationSubmitted: false,
	}
	if plan.UserTasks == nil {
		plan.UserTasks = make([]d.UserTaskVariablePlan, 0)
	}
	if plan.Targets == nil {
		plan.Targets = make([]d.ScopeVariableUpdateTarget, 0)
	}
	for _, task := range p.tasks {
		plan.VariableAddCount += len(task.Additions)
		plan.VariableChangeCount += len(task.Changes)
		plan.VariableUnchangedCount += len(task.UnchangedRequested)
		plan.VariableUntouchedCount += len(task.Untouched)
		if len(task.Additions)+len(task.Changes) > 0 {
			plan.UpdateCount++
		}
	}
	return plan
}

// validateRequestedVariables rejects values that cannot be represented in the
// JSON payload contract before initiating remote discovery.
func validateRequestedVariables(variables map[string]any) error {
	for name, value := range variables {
		if strings.TrimSpace(name) == "" {
			return planningValidationError("requested variable name must not be blank")
		}
		if _, err := json.Marshal(value); err != nil {
			return planningValidationError("requested variable %q is not JSON-compatible: %v", name, err)
		}
	}
	return nil
}

// validateSelectedUserTask prevents a malformed direct read from changing the
// explicit identity or selecting an invalid local creation scope.
func validateSelectedUserTask(requestedKey string, task d.UserTask) error {
	if strings.TrimSpace(task.Key) == "" || task.Key != requestedKey {
		return planningValidationError("requested user task %s returned invalid task identity %q", requestedKey, task.Key)
	}
	if strings.TrimSpace(task.ElementInstanceKey) == "" {
		return planningValidationError("user task %s has no element-instance scope", task.Key)
	}
	return nil
}

// conflictingScopeVariableEvidence detects observations that cannot safely be
// merged into one logical (scope, name) mutation target.
func conflictingScopeVariableEvidence(a, b observedScopeVariable) bool {
	return knownStringsConflict(a.variableKey, b.variableKey) ||
		knownStringsConflict(a.processInstanceKey, b.processInstanceKey) ||
		knownStringsConflict(a.tenantID, b.tenantID) ||
		a.apiTruncated != b.apiTruncated || !reflect.DeepEqual(a.canonicalValue, b.canonicalValue)
}

// knownStringsConflict treats missing optional evidence as unknown rather than
// contradictory while rejecting two different known identities.
func knownStringsConflict(a, b string) bool {
	return a != "" && b != "" && a != b
}

// normalizedJSONValuesEqual compares complete backend JSON using stable number
// handling so Go input number types do not create false changes.
func normalizedJSONValuesEqual(requested any, observedRaw string) bool {
	requestedValue, ok := normalizeJSONValue(requested)
	if !ok {
		return false
	}
	observedValue, ok := normalizeObservedJSONValue(observedRaw)
	return ok && reflect.DeepEqual(requestedValue, observedValue)
}

// normalizeJSONValue round-trips a requested value with json.Number semantics.
func normalizeJSONValue(value any) (any, bool) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	return decodeNormalizedJSON(raw)
}

// normalizeObservedJSONValue decodes one raw backend value without losing
// numeric spelling through float conversion.
func normalizeObservedJSONValue(raw string) (any, bool) {
	return decodeNormalizedJSON([]byte(raw))
}

// decodeNormalizedJSON requires exactly one valid JSON value.
func decodeNormalizedJSON(raw []byte) (any, bool) {
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

// normalizedRequestedValuesEqual compares two JSON-compatible requested values
// independently of their concrete Go number types.
func normalizedRequestedValuesEqual(a, b any) bool {
	normalizedA, okA := normalizeJSONValue(a)
	normalizedB, okB := normalizeJSONValue(b)
	return okA && okB && reflect.DeepEqual(normalizedA, normalizedB)
}

// decodeObservedVariableValue preserves truncated fragments while decoding
// complete or still-valid raw JSON into operator-facing values.
func decodeObservedVariableValue(raw string) any {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	return value
}

// sortedStringMapKeys returns deterministic category and payload ordering.
func sortedStringMapKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// copyVariableMap isolates the frozen plan from later mutations of the input
// map and its ordinary JSON object and array values.
func copyVariableMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for name, value := range in {
		out[name] = copyVariableValue(value)
	}
	return out
}

// copyVariableValue recursively copies the mutable collection shapes produced
// by JSON decoding while leaving scalar JSON values unchanged.
func copyVariableValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		return copyVariableMap(value)
	case []any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = copyVariableValue(value[i])
		}
		return out
	default:
		return value
	}
}

// configuredTenant safely reads the optional discovery configuration evidence.
func configuredTenant(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.App.Tenant
}

// emptyUserTaskVariableUpdatePlan returns initialized collections for a valid
// service-level no-work request.
func emptyUserTaskVariableUpdatePlan(tenantContext d.TenantContext) d.UserTaskVariableUpdatePlan {
	return d.UserTaskVariableUpdatePlan{
		RequestedKeys: make([]string, 0), UserTasks: make([]d.UserTaskVariablePlan, 0),
		Targets: make([]d.ScopeVariableUpdateTarget, 0), TenantContext: tenantContext,
	}
}

// planningValidationError classifies inconsistent input or discovered identity
// facts before any mutation workflow can begin.
func planningValidationError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", d.ErrValidation, fmt.Sprintf(format, args...))
}
