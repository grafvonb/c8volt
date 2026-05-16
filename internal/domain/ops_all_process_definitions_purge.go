// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"time"

	"github.com/grafvonb/c8volt/typex"
)

// AllProcessDefinitionsPurgeOutcome is the final state of the all-process-definitions purge workflow.
type AllProcessDefinitionsPurgeOutcome string

const (
	AllProcessDefinitionsPurgeOutcomePlanned         AllProcessDefinitionsPurgeOutcome = "planned"
	AllProcessDefinitionsPurgeOutcomeDeleted         AllProcessDefinitionsPurgeOutcome = "deleted"
	AllProcessDefinitionsPurgeOutcomePartiallyFailed AllProcessDefinitionsPurgeOutcome = "partially_failed"
	AllProcessDefinitionsPurgeOutcomeFailed          AllProcessDefinitionsPurgeOutcome = "failed"
)

// AllProcessDefinitionsPurgeRequest captures one requested all-process-definitions purge run.
type AllProcessDefinitionsPurgeRequest struct {
	CommandName                              string
	DryRun                                   bool
	AutoConfirm                              bool
	Automation                               bool
	OutputMode                               string
	Selection                                ProcessDefinitionFilter
	Workers                                  int
	FailFast                                 bool
	NoWorkerLimit                            bool
	NoWait                                   bool
	Force                                    bool
	ReportFile                               string
	ReportFormat                             string
	DiscoveredCandidateProcessDefinitionKeys typex.Keys
	StartedAt                                time.Time
}

// ProcessDefinitionDiscoveryResult captures immutable process-definition discovery output.
type ProcessDefinitionDiscoveryResult struct {
	Status                                  OpsWorkflowStepStatus
	Filters                                 ProcessDefinitionFilter
	CandidateProcessDefinitionKeys          typex.Keys
	CandidateProcessDefinitions             []ProcessDefinition
	DuplicateCandidateProcessDefinitionKeys typex.Keys
	CandidateProcessDefinitionCount         int
	LatestOnly                              bool
	Notices                                 []AllProcessDefinitionsPurgeWorkflowNotice
	Errors                                  []string
}

// AllProcessDefinitionsPurgeDeletePlan captures the validated delete plan for frozen candidates.
type AllProcessDefinitionsPurgeDeletePlan struct {
	Status                                  OpsWorkflowStepStatus
	CandidateProcessDefinitionKeys          typex.Keys
	Items                                   []DeleteProcessDefinitionPlanItem
	DuplicateCandidateProcessDefinitionKeys typex.Keys
	AffectedProcessInstanceCount            int64
	ActiveProcessInstanceCount              int64
	RequiresConfirmation                    bool
	RequiresForce                           bool
	Errors                                  []string
}

// AllProcessDefinitionsPurgeDeletionResult captures mutation submission and confirmation output.
type AllProcessDefinitionsPurgeDeletionResult struct {
	Status                         OpsWorkflowStepStatus
	SubmittedProcessDefinitionKeys typex.Keys
	Items                          []ResourceDeleteResponse
	Submitted                      bool
	Confirmed                      bool
	NoWait                         bool
	Errors                         []string
}

// AllProcessDefinitionsPurgeReport is the stable audit model for output and report files.
type AllProcessDefinitionsPurgeReport struct {
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
	SelectionFilters ProcessDefinitionFilter
	Discovery        ProcessDefinitionDiscoveryResult
	DeletePlan       AllProcessDefinitionsPurgeDeletePlan
	Deletion         AllProcessDefinitionsPurgeDeletionResult
	AutoConfirm      bool
	Automation       bool
	NoWait           bool
	Force            bool
	FailFast         bool
	NoWorkerLimit    bool
	Errors           []string
	Notices          []AllProcessDefinitionsPurgeWorkflowNotice
	Outcome          AllProcessDefinitionsPurgeOutcome
}

// AllProcessDefinitionsPurgeResult carries the full workflow result across the service and facade boundary.
type AllProcessDefinitionsPurgeResult struct {
	Request    AllProcessDefinitionsPurgeRequest
	Discovery  ProcessDefinitionDiscoveryResult
	DeletePlan AllProcessDefinitionsPurgeDeletePlan
	Deletion   AllProcessDefinitionsPurgeDeletionResult
	Report     AllProcessDefinitionsPurgeReport
	Outcome    AllProcessDefinitionsPurgeOutcome
	Errors     []string
	Notices    []AllProcessDefinitionsPurgeWorkflowNotice
}

// AllProcessDefinitionsPurgeWorkflowNotice represents semantic workflow context for compact and structured output.
type AllProcessDefinitionsPurgeWorkflowNotice struct {
	Code     string
	Severity string
	Message  string
	Details  map[string]string
}
