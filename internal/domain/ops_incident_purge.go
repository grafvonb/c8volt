// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"time"

	"github.com/grafvonb/c8volt/typex"
)

// IncidentPurgeOutcome is the final state of an incident-based process-instance purge workflow.
type IncidentPurgeOutcome string

const (
	// IncidentPurgeOutcomePlanned means the workflow produced a plan without mutating anything.
	IncidentPurgeOutcomePlanned IncidentPurgeOutcome = "planned"
	// IncidentPurgeOutcomeDeleted means every submitted deletion completed successfully.
	IncidentPurgeOutcomeDeleted IncidentPurgeOutcome = "deleted"
	// IncidentPurgeOutcomePartiallyFailed means some submitted deletions failed.
	IncidentPurgeOutcomePartiallyFailed IncidentPurgeOutcome = "partially_failed"
	// IncidentPurgeOutcomeFailed means discovery, planning, or deletion failed.
	IncidentPurgeOutcomeFailed IncidentPurgeOutcome = "failed"
)

// IncidentPurgeRequest captures one requested incident-based purge run.
type IncidentPurgeRequest struct {
	CommandName                            string
	DryRun                                 bool
	AutoConfirm                            bool
	Automation                             bool
	OutputMode                             string
	Selection                              IncidentFilter
	BatchSize                              int32
	Limit                                  int32
	Workers                                int
	FailFast                               bool
	NoWorkerLimit                          bool
	NoWait                                 bool
	Force                                  bool
	ReportFile                             string
	ReportFormat                           string
	DiscoveredCandidateProcessInstanceKeys typex.Keys
	StartedAt                              time.Time
}

// IncidentPurgeSkippedIncident records a matching incident that could not produce a delete candidate.
type IncidentPurgeSkippedIncident struct {
	Incident ProcessInstanceIncidentDetail
	Reason   string
}

// IncidentDiscoveryResult captures immutable incident discovery and candidate extraction output.
type IncidentDiscoveryResult struct {
	Status                                OpsWorkflowStepStatus
	Filters                               IncidentFilter
	CandidateIncidents                    []ProcessInstanceIncidentDetail
	IncidentKeys                          typex.Keys
	CandidateProcessInstanceKeys          typex.Keys
	DuplicateCandidateProcessInstanceKeys typex.Keys
	SkippedIncidents                      []IncidentPurgeSkippedIncident
	IncidentCount                         int
	CandidateProcessInstanceCount         int
	Notices                               []IncidentPurgeWorkflowNotice
	Errors                                []string
}

// IncidentPurgeDeletePlan captures the validated delete plan for frozen incident candidates.
type IncidentPurgeDeletePlan struct {
	Status                                OpsWorkflowStepStatus
	CandidateProcessInstanceKeys          typex.Keys
	ResolvedRootKeys                      typex.Keys
	AffectedKeys                          typex.Keys
	DuplicateCandidateProcessInstanceKeys typex.Keys
	FinalStateItems                       []ProcessInstance
	NonFinalAffectedItems                 []ProcessInstance
	MissingAncestors                      []MissingAncestor
	TraversalWarnings                     []string
	RequiresConfirmation                  bool
	Errors                                []string
}

// IncidentPurgeDeletionResult captures mutation submission and confirmation output.
type IncidentPurgeDeletionResult struct {
	Status            OpsWorkflowStepStatus
	SubmittedRootKeys typex.Keys
	Items             []Reporter
	Submitted         bool
	Confirmed         bool
	NoWait            bool
	Errors            []string
}

// IncidentPurgeReport is the stable audit model for output and report files.
type IncidentPurgeReport struct {
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
	SelectionFilters IncidentFilter
	Discovery        IncidentDiscoveryResult
	DeletePlan       IncidentPurgeDeletePlan
	Deletion         IncidentPurgeDeletionResult
	AutoConfirm      bool
	Automation       bool
	NoWait           bool
	Force            bool
	FailFast         bool
	NoWorkerLimit    bool
	Errors           []string
	Notices          []IncidentPurgeWorkflowNotice
	Outcome          IncidentPurgeOutcome
}

// IncidentPurgeResult carries the full workflow result across the service and facade boundary.
type IncidentPurgeResult struct {
	Request    IncidentPurgeRequest
	Discovery  IncidentDiscoveryResult
	DeletePlan IncidentPurgeDeletePlan
	Deletion   IncidentPurgeDeletionResult
	Report     IncidentPurgeReport
	Outcome    IncidentPurgeOutcome
	Errors     []string
	Notices    []IncidentPurgeWorkflowNotice
}

// IncidentPurgeWorkflowNotice represents semantic workflow context for compact and structured output.
type IncidentPurgeWorkflowNotice struct {
	Code     string
	Severity string
	Message  string
	Details  map[string]string
}
