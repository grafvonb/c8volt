// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"time"

	"github.com/grafvonb/c8volt/typex"
)

// OpsRepairTarget identifies the repair workflow entry point.
type OpsRepairTarget string

const (
	// OpsRepairTargetIncident repairs incidents selected directly or by incident filters.
	OpsRepairTargetIncident OpsRepairTarget = "incident"
	// OpsRepairTargetProcessInstance repairs active incidents associated with selected process instances.
	OpsRepairTargetProcessInstance OpsRepairTarget = "process_instance"
)

// OpsRepairDiscoveryMode records how the workflow found its initial target set.
type OpsRepairDiscoveryMode string

const (
	// OpsRepairDiscoveryModeKeyed records targets supplied by repeated key flags.
	OpsRepairDiscoveryModeKeyed OpsRepairDiscoveryMode = "keyed"
	// OpsRepairDiscoveryModeStdin records targets supplied through stdin.
	OpsRepairDiscoveryModeStdin OpsRepairDiscoveryMode = "stdin"
	// OpsRepairDiscoveryModeSearch records targets discovered from filters.
	OpsRepairDiscoveryModeSearch OpsRepairDiscoveryMode = "search"
)

// OpsRepairOutcome is the final aggregate state of a repair workflow.
type OpsRepairOutcome string

const (
	// OpsRepairOutcomePlanned means discovery and validation completed without mutation.
	OpsRepairOutcomePlanned OpsRepairOutcome = "planned"
	// OpsRepairOutcomeRepaired means every requested repair target completed successfully.
	OpsRepairOutcomeRepaired OpsRepairOutcome = "repaired"
	// OpsRepairOutcomePartiallyFailed means at least one repair target failed while another succeeded or remained planned.
	OpsRepairOutcomePartiallyFailed OpsRepairOutcome = "partially_failed"
	// OpsRepairOutcomeFailed means discovery, planning, or repair failed before a successful target result.
	OpsRepairOutcomeFailed OpsRepairOutcome = "failed"
)

// OpsRepairRequest captures one operator repair invocation.
type OpsRepairRequest struct {
	CommandName              string
	Target                   OpsRepairTarget
	DiscoveryMode            OpsRepairDiscoveryMode
	InputKeys                typex.Keys
	IncidentSelection        IncidentFilter
	ProcessInstanceSelection ProcessInstanceFilter
	DirectIncidentsOnly      bool
	BatchSize                int32
	Limit                    int32
	Workers                  int
	FailFast                 bool
	NoWorkerLimit            bool
	DryRun                   bool
	AutoConfirm              bool
	Automation               bool
	NoWait                   bool
	OutputMode               string
	Variables                map[string]any
	VariablesFile            string
	RequestedRetries         *int32
	RequestedJobTimeout      time.Duration
	ReportFile               string
	ReportFormat             string
	StartedAt                time.Time
}

// OpsRepairFrozenSet captures the immutable target data discovered before mutation.
type OpsRepairFrozenSet struct {
	Status                     OpsWorkflowStepStatus
	Target                     OpsRepairTarget
	DiscoveryMode              OpsRepairDiscoveryMode
	InputKeys                  typex.Keys
	IncidentKeys               typex.Keys
	ProcessInstanceKeys        typex.Keys
	SkippedProcessInstanceKeys typex.Keys
	RootProcessKeys            typex.Keys
	JobKeys                    typex.Keys
	VariableScopes             typex.Keys
	OriginalIncidents          []ProcessInstanceIncidentDetail
	IncidentFilters            IncidentFilter
	ProcessFilters             ProcessInstanceFilter
	Errors                     []string
}

// OpsRepairPlanItem describes the planned or executed steps for one incident.
type OpsRepairPlanItem struct {
	IncidentKey            string
	ProcessInstanceKey     string
	RootProcessInstanceKey string
	JobKey                 string
	VariableScopeKey       string
	RequestedVariableNames []string
	RequestedRetries       *int32
	RequestedTimeout       string
	VariableUpdateStatus   OpsWorkflowStepStatus
	RetryUpdateStatus      OpsWorkflowStepStatus
	TimeoutUpdateStatus    OpsWorkflowStepStatus
	ResolutionStatus       OpsWorkflowStepStatus
	ConfirmationStatus     OpsWorkflowStepStatus
	Notices                []OpsRepairWorkflowNotice
	Errors                 []string
}

// OpsRepairVariableScopeUpdate represents one deduped process-instance variable mutation scope.
type OpsRepairVariableScopeUpdate struct {
	ScopeKey              string
	VariableNames         []string
	Payload               map[string]any
	DependentIncidentKeys typex.Keys
	Status                OpsWorkflowStepStatus
	Errors                []string
}

// OpsRepairJobApplicability records whether job repair steps apply to one incident.
type OpsRepairJobApplicability struct {
	IncidentKey        string
	JobKey             string
	RetryStatus        OpsWorkflowStepStatus
	TimeoutStatus      OpsWorkflowStepStatus
	Reason             string
	RequestedRetries   *int32
	RequestedTimeout   string
	UnsupportedVersion bool
}

// OpsRepairRemainingIncidentSummary captures post-repair incident context for audit output.
type OpsRepairRemainingIncidentSummary struct {
	Status     OpsWorkflowStepStatus
	ActiveKeys typex.Keys
	Incidents  []ProcessInstanceIncidentDetail
	Checked    bool
	Errors     []string
}

// OpsRepairWorkflowNotice represents semantic repair context for compact and structured output.
type OpsRepairWorkflowNotice struct {
	Code     string
	Severity string
	Message  string
	Details  map[string]string
}

// OpsRepairAuditReport is the stable report model for repair output and audit files.
type OpsRepairAuditReport struct {
	SchemaVersion    string
	CommandName      string
	StartedAt        time.Time
	FinishedAt       time.Time
	Duration         string
	DryRun           bool
	C8voltVersion    string
	CamundaVersion   string
	ProfileIdentity  string
	TenantID         string
	Request          OpsRepairRequest
	FrozenSet        OpsRepairFrozenSet
	Plan             []OpsRepairPlanItem
	VariableUpdates  []OpsRepairVariableScopeUpdate
	JobApplicability []OpsRepairJobApplicability
	Remaining        OpsRepairRemainingIncidentSummary
	AutoConfirm      bool
	Automation       bool
	NoWait           bool
	FailFast         bool
	NoWorkerLimit    bool
	Errors           []string
	Notices          []OpsRepairWorkflowNotice
	Outcome          OpsRepairOutcome
}

// OpsRepairResult carries the full repair workflow result across service boundaries.
type OpsRepairResult struct {
	Request          OpsRepairRequest
	FrozenSet        OpsRepairFrozenSet
	Plan             []OpsRepairPlanItem
	VariableUpdates  []OpsRepairVariableScopeUpdate
	JobApplicability []OpsRepairJobApplicability
	Remaining        OpsRepairRemainingIncidentSummary
	Report           OpsRepairAuditReport
	Outcome          OpsRepairOutcome
	Errors           []string
	Notices          []OpsRepairWorkflowNotice
}
