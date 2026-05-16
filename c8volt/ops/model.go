// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"time"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/typex"
)

type WorkflowStepStatus string

const (
	WorkflowStepStatusPlanned            WorkflowStepStatus = "planned"
	WorkflowStepStatusSkipped            WorkflowStepStatus = "skipped"
	WorkflowStepStatusSubmitted          WorkflowStepStatus = "submitted"
	WorkflowStepStatusConfirmed          WorkflowStepStatus = "confirmed"
	WorkflowStepStatusConfirmationFailed WorkflowStepStatus = "confirmation_failed"
	WorkflowStepStatusBlocked            WorkflowStepStatus = "blocked"
	WorkflowStepStatusFailed             WorkflowStepStatus = "failed"
)

type OrphanPurgeOutcome string

const (
	OrphanPurgeOutcomePlanned         OrphanPurgeOutcome = "planned"
	OrphanPurgeOutcomeDeleted         OrphanPurgeOutcome = "deleted"
	OrphanPurgeOutcomePartiallyFailed OrphanPurgeOutcome = "partially_failed"
	OrphanPurgeOutcomeFailed          OrphanPurgeOutcome = "failed"
)

type OrphanPurgeRequest struct {
	CommandName    string                        `json:"commandName,omitempty"`
	DryRun         bool                          `json:"dryRun,omitempty"`
	AutoConfirm    bool                          `json:"autoConfirm,omitempty"`
	Automation     bool                          `json:"automation,omitempty"`
	NoWait         bool                          `json:"noWait,omitempty"`
	OutputMode     string                        `json:"outputMode,omitempty"`
	Selection      process.ProcessInstanceFilter `json:"selection,omitempty"`
	BatchSize      int32                         `json:"batchSize,omitempty"`
	Limit          int32                         `json:"limit,omitempty"`
	Workers        int                           `json:"workers,omitempty"`
	ReportFile     string                        `json:"reportFile,omitempty"`
	ReportFormat   string                        `json:"reportFormat,omitempty"`
	DiscoveredKeys typex.Keys                    `json:"discoveredKeys,omitempty"`
	StartedAt      time.Time                     `json:"startedAt,omitempty"`
}

type OrphanDiscoveryResult struct {
	Status  WorkflowStepStatus            `json:"status,omitempty"`
	Filters process.ProcessInstanceFilter `json:"filters,omitempty"`
	Keys    typex.Keys                    `json:"keys,omitempty"`
	Count   int                           `json:"count"`
	Errors  []string                      `json:"errors,omitempty"`
}

type DeletionPlan struct {
	Status               WorkflowStepStatus           `json:"status,omitempty"`
	RequestedKeys        typex.Keys                   `json:"requestedKeys,omitempty"`
	AffectedKeys         typex.Keys                   `json:"affectedKeys,omitempty"`
	RootKeys             typex.Keys                   `json:"rootKeys,omitempty"`
	RequiresConfirmation bool                         `json:"requiresConfirmation,omitempty"`
	DryRunPreview        process.DryRunPIKeyExpansion `json:"dryRunPreview,omitempty"`
	Errors               []string                     `json:"errors,omitempty"`
}

type DeletionResult struct {
	Status    WorkflowStepStatus     `json:"status,omitempty"`
	Items     []process.DeleteReport `json:"items,omitempty"`
	Errors    []string               `json:"errors,omitempty"`
	Submitted bool                   `json:"submitted,omitempty"`
	Confirmed bool                   `json:"confirmed,omitempty"`
	NoWait    bool                   `json:"noWait,omitempty"`
}

type OrphanPurgeReport struct {
	SchemaVersion    string                        `json:"schemaVersion,omitempty"`
	CommandName      string                        `json:"commandName,omitempty"`
	StartedAt        time.Time                     `json:"startedAt,omitempty"`
	FinishedAt       time.Time                     `json:"finishedAt,omitempty"`
	Duration         string                        `json:"duration,omitempty"`
	DryRun           bool                          `json:"dryRun,omitempty"`
	C8voltVersion    string                        `json:"c8voltVersion,omitempty"`
	CamundaVersion   string                        `json:"camundaVersion,omitempty"`
	ProfileIdentity  string                        `json:"profileIdentity,omitempty"`
	SelectionFilters process.ProcessInstanceFilter `json:"selectionFilters,omitempty"`
	Discovery        OrphanDiscoveryResult         `json:"discovery,omitempty"`
	DeletionPlan     DeletionPlan                  `json:"deletionPlan,omitempty"`
	Deletion         DeletionResult                `json:"deletion,omitempty"`
	DeleteRequested  bool                          `json:"deleteRequested,omitempty"`
	AutoConfirm      bool                          `json:"autoConfirm,omitempty"`
	Automation       bool                          `json:"automation,omitempty"`
	NoWait           bool                          `json:"noWait,omitempty"`
	Errors           []string                      `json:"errors,omitempty"`
	Outcome          OrphanPurgeOutcome            `json:"outcome,omitempty"`
}

type OrphanPurgeResult struct {
	Request         OrphanPurgeRequest    `json:"request,omitempty"`
	Discovery       OrphanDiscoveryResult `json:"discovery,omitempty"`
	DeletionPlan    DeletionPlan          `json:"deletionPlan,omitempty"`
	Deletion        DeletionResult        `json:"deletion,omitempty"`
	Report          OrphanPurgeReport     `json:"report,omitempty"`
	DeleteRequested bool                  `json:"deleteRequested,omitempty"`
	Outcome         OrphanPurgeOutcome    `json:"outcome,omitempty"`
	Errors          []string              `json:"errors,omitempty"`
}

type RetentionPolicyOutcome string

const (
	RetentionPolicyOutcomePlanned         RetentionPolicyOutcome = "planned"
	RetentionPolicyOutcomeDeleted         RetentionPolicyOutcome = "deleted"
	RetentionPolicyOutcomePartiallyFailed RetentionPolicyOutcome = "partially_failed"
	RetentionPolicyOutcomeFailed          RetentionPolicyOutcome = "failed"
)

type RetentionPolicyRequest struct {
	CommandName            string                        `json:"commandName,omitempty"`
	RetentionDays          int                           `json:"retentionDays"`
	DerivedEndDateBoundary string                        `json:"derivedEndDateBoundary,omitempty"`
	DryRun                 bool                          `json:"dryRun,omitempty"`
	AutoConfirm            bool                          `json:"autoConfirm,omitempty"`
	Automation             bool                          `json:"automation,omitempty"`
	OutputMode             string                        `json:"outputMode,omitempty"`
	Selection              process.ProcessInstanceFilter `json:"selection,omitempty"`
	BatchSize              int32                         `json:"batchSize,omitempty"`
	Limit                  int32                         `json:"limit,omitempty"`
	Workers                int                           `json:"workers,omitempty"`
	NoWait                 bool                          `json:"noWait,omitempty"`
	NoStateCheck           bool                          `json:"noStateCheck,omitempty"`
	Force                  bool                          `json:"force,omitempty"`
	FailFast               bool                          `json:"failFast,omitempty"`
	NoWorkerLimit          bool                          `json:"noWorkerLimit,omitempty"`
	ReportFile             string                        `json:"reportFile,omitempty"`
	ReportFormat           string                        `json:"reportFormat,omitempty"`
	DiscoveredKeys         typex.Keys                    `json:"discoveredKeys,omitempty"`
	StartedAt              time.Time                     `json:"startedAt,omitempty"`
}

type RetentionDiscoveryResult struct {
	Status                 WorkflowStepStatus            `json:"status,omitempty"`
	RetentionDays          int                           `json:"retentionDays"`
	DerivedEndDateBoundary string                        `json:"derivedEndDateBoundary,omitempty"`
	Filters                process.ProcessInstanceFilter `json:"filters,omitempty"`
	SeedKeys               typex.Keys                    `json:"seedKeys,omitempty"`
	Count                  int                           `json:"count"`
	Notices                []RetentionWorkflowNotice     `json:"notices,omitempty"`
	Errors                 []string                      `json:"errors,omitempty"`
}

type RetentionDeletePlan struct {
	Status                WorkflowStepStatus        `json:"status,omitempty"`
	SeedKeys              typex.Keys                `json:"seedKeys,omitempty"`
	ResolvedRootKeys      typex.Keys                `json:"resolvedRootKeys,omitempty"`
	AffectedKeys          typex.Keys                `json:"affectedKeys,omitempty"`
	DuplicateKeys         typex.Keys                `json:"duplicateKeys,omitempty"`
	FinalStateItems       []process.ProcessInstance `json:"finalStateItems,omitempty"`
	NonFinalAffectedItems []process.ProcessInstance `json:"nonFinalAffectedItems,omitempty"`
	SkippedSeedKeys       typex.Keys                `json:"skippedSeedKeys,omitempty"`
	SkippedNonFinalRoots  []process.ProcessInstance `json:"skippedNonFinalRoots,omitempty"`
	MissingAncestors      []process.MissingAncestor `json:"missingAncestors,omitempty"`
	TraversalWarnings     []string                  `json:"traversalWarnings,omitempty"`
	RequiresConfirmation  bool                      `json:"requiresConfirmation,omitempty"`
	Errors                []string                  `json:"errors,omitempty"`
}

type RetentionDeletionResult struct {
	Status            WorkflowStepStatus     `json:"status,omitempty"`
	SubmittedRootKeys typex.Keys             `json:"submittedRootKeys,omitempty"`
	Items             []process.DeleteReport `json:"items,omitempty"`
	Submitted         bool                   `json:"submitted,omitempty"`
	Confirmed         bool                   `json:"confirmed,omitempty"`
	NoWait            bool                   `json:"noWait,omitempty"`
	Errors            []string               `json:"errors,omitempty"`
}

type RetentionAuditReport struct {
	SchemaVersion          string                        `json:"schemaVersion,omitempty"`
	CommandName            string                        `json:"commandName,omitempty"`
	StartedAt              time.Time                     `json:"startedAt,omitempty"`
	FinishedAt             time.Time                     `json:"finishedAt,omitempty"`
	Duration               string                        `json:"duration,omitempty"`
	DryRun                 bool                          `json:"dryRun,omitempty"`
	C8voltVersion          string                        `json:"c8voltVersion,omitempty"`
	CamundaVersion         string                        `json:"camundaVersion,omitempty"`
	ProfileIdentity        string                        `json:"profileIdentity,omitempty"`
	TenantID               string                        `json:"tenantId,omitempty"`
	RetentionDays          int                           `json:"retentionDays"`
	DerivedEndDateBoundary string                        `json:"derivedEndDateBoundary,omitempty"`
	SelectionFilters       process.ProcessInstanceFilter `json:"selectionFilters,omitempty"`
	Discovery              RetentionDiscoveryResult      `json:"discovery,omitempty"`
	DeletePlan             RetentionDeletePlan           `json:"deletePlan,omitempty"`
	Deletion               RetentionDeletionResult       `json:"deletion,omitempty"`
	AutoConfirm            bool                          `json:"autoConfirm,omitempty"`
	Automation             bool                          `json:"automation,omitempty"`
	NoWait                 bool                          `json:"noWait,omitempty"`
	NoStateCheck           bool                          `json:"noStateCheck,omitempty"`
	Force                  bool                          `json:"force,omitempty"`
	FailFast               bool                          `json:"failFast,omitempty"`
	NoWorkerLimit          bool                          `json:"noWorkerLimit,omitempty"`
	Errors                 []string                      `json:"errors,omitempty"`
	Outcome                RetentionPolicyOutcome        `json:"outcome,omitempty"`
}

type RetentionPolicyResult struct {
	Request    RetentionPolicyRequest   `json:"request,omitempty"`
	Discovery  RetentionDiscoveryResult `json:"discovery,omitempty"`
	DeletePlan RetentionDeletePlan      `json:"deletePlan,omitempty"`
	Deletion   RetentionDeletionResult  `json:"deletion,omitempty"`
	Report     RetentionAuditReport     `json:"report,omitempty"`
	Outcome    RetentionPolicyOutcome   `json:"outcome,omitempty"`
	Errors     []string                 `json:"errors,omitempty"`
}

type RetentionWorkflowNotice struct {
	Code     string            `json:"code,omitempty"`
	Severity string            `json:"severity,omitempty"`
	Message  string            `json:"message,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
}

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
	CommandName                            string          `json:"commandName,omitempty"`
	DryRun                                 bool            `json:"dryRun,omitempty"`
	AutoConfirm                            bool            `json:"autoConfirm,omitempty"`
	Automation                             bool            `json:"automation,omitempty"`
	OutputMode                             string          `json:"outputMode,omitempty"`
	Selection                              incident.Filter `json:"selection,omitempty"`
	BatchSize                              int32           `json:"batchSize,omitempty"`
	Limit                                  int32           `json:"limit,omitempty"`
	Workers                                int             `json:"workers,omitempty"`
	FailFast                               bool            `json:"failFast,omitempty"`
	NoWorkerLimit                          bool            `json:"noWorkerLimit,omitempty"`
	NoWait                                 bool            `json:"noWait,omitempty"`
	Force                                  bool            `json:"force,omitempty"`
	ReportFile                             string          `json:"reportFile,omitempty"`
	ReportFormat                           string          `json:"reportFormat,omitempty"`
	DiscoveredCandidateProcessInstanceKeys typex.Keys      `json:"discoveredCandidateProcessInstanceKeys,omitempty"`
	StartedAt                              time.Time       `json:"startedAt,omitempty"`
}

// IncidentPurgeSkippedIncident records a matching incident that could not produce a delete candidate.
type IncidentPurgeSkippedIncident struct {
	Incident incident.ProcessInstanceIncidentDetail `json:"incident,omitempty"`
	Reason   string                                 `json:"reason,omitempty"`
}

// IncidentDiscoveryResult captures immutable incident discovery and candidate extraction output.
type IncidentDiscoveryResult struct {
	Status                                WorkflowStepStatus                       `json:"status,omitempty"`
	Filters                               incident.Filter                          `json:"filters,omitempty"`
	CandidateIncidents                    []incident.ProcessInstanceIncidentDetail `json:"candidateIncidents,omitempty"`
	IncidentKeys                          typex.Keys                               `json:"incidentKeys,omitempty"`
	CandidateProcessInstanceKeys          typex.Keys                               `json:"candidateProcessInstanceKeys,omitempty"`
	DuplicateCandidateProcessInstanceKeys typex.Keys                               `json:"duplicateCandidateProcessInstanceKeys,omitempty"`
	SkippedIncidents                      []IncidentPurgeSkippedIncident           `json:"skippedIncidents,omitempty"`
	IncidentCount                         int                                      `json:"incidentCount"`
	CandidateProcessInstanceCount         int                                      `json:"candidateProcessInstanceCount"`
	Notices                               []IncidentPurgeWorkflowNotice            `json:"notices,omitempty"`
	Errors                                []string                                 `json:"errors,omitempty"`
}

// IncidentPurgeDeletePlan captures the validated delete plan for frozen incident candidates.
type IncidentPurgeDeletePlan struct {
	Status                                WorkflowStepStatus        `json:"status,omitempty"`
	CandidateProcessInstanceKeys          typex.Keys                `json:"candidateProcessInstanceKeys,omitempty"`
	ResolvedRootKeys                      typex.Keys                `json:"resolvedRootKeys,omitempty"`
	AffectedKeys                          typex.Keys                `json:"affectedKeys,omitempty"`
	DuplicateCandidateProcessInstanceKeys typex.Keys                `json:"duplicateCandidateProcessInstanceKeys,omitempty"`
	FinalStateItems                       []process.ProcessInstance `json:"finalStateItems,omitempty"`
	NonFinalAffectedItems                 []process.ProcessInstance `json:"nonFinalAffectedItems,omitempty"`
	MissingAncestors                      []process.MissingAncestor `json:"missingAncestors,omitempty"`
	TraversalWarnings                     []string                  `json:"traversalWarnings,omitempty"`
	RequiresConfirmation                  bool                      `json:"requiresConfirmation,omitempty"`
	Errors                                []string                  `json:"errors,omitempty"`
}

// IncidentPurgeDeletionResult captures mutation submission and confirmation output.
type IncidentPurgeDeletionResult struct {
	Status            WorkflowStepStatus     `json:"status,omitempty"`
	SubmittedRootKeys typex.Keys             `json:"submittedRootKeys,omitempty"`
	Items             []process.DeleteReport `json:"items,omitempty"`
	Submitted         bool                   `json:"submitted,omitempty"`
	Confirmed         bool                   `json:"confirmed,omitempty"`
	NoWait            bool                   `json:"noWait,omitempty"`
	Errors            []string               `json:"errors,omitempty"`
}

// IncidentPurgeReport is the stable audit model for output and report files.
type IncidentPurgeReport struct {
	SchemaVersion    string                        `json:"schemaVersion,omitempty"`
	CommandName      string                        `json:"commandName,omitempty"`
	StartedAt        time.Time                     `json:"startedAt,omitempty"`
	FinishedAt       time.Time                     `json:"finishedAt,omitempty"`
	Duration         string                        `json:"duration,omitempty"`
	DryRun           bool                          `json:"dryRun,omitempty"`
	C8voltVersion    string                        `json:"c8voltVersion,omitempty"`
	CamundaVersion   string                        `json:"camundaVersion,omitempty"`
	ProfileIdentity  string                        `json:"profileIdentity,omitempty"`
	TenantID         string                        `json:"tenantId,omitempty"`
	SelectionFilters incident.Filter               `json:"selectionFilters,omitempty"`
	Discovery        IncidentDiscoveryResult       `json:"discovery,omitempty"`
	DeletePlan       IncidentPurgeDeletePlan       `json:"deletePlan,omitempty"`
	Deletion         IncidentPurgeDeletionResult   `json:"deletion,omitempty"`
	AutoConfirm      bool                          `json:"autoConfirm,omitempty"`
	Automation       bool                          `json:"automation,omitempty"`
	NoWait           bool                          `json:"noWait,omitempty"`
	Force            bool                          `json:"force,omitempty"`
	FailFast         bool                          `json:"failFast,omitempty"`
	NoWorkerLimit    bool                          `json:"noWorkerLimit,omitempty"`
	Errors           []string                      `json:"errors,omitempty"`
	Notices          []IncidentPurgeWorkflowNotice `json:"notices,omitempty"`
	Outcome          IncidentPurgeOutcome          `json:"outcome,omitempty"`
}

// IncidentPurgeResult carries the full workflow result across the service and facade boundary.
type IncidentPurgeResult struct {
	Request    IncidentPurgeRequest          `json:"request,omitempty"`
	Discovery  IncidentDiscoveryResult       `json:"discovery,omitempty"`
	DeletePlan IncidentPurgeDeletePlan       `json:"deletePlan,omitempty"`
	Deletion   IncidentPurgeDeletionResult   `json:"deletion,omitempty"`
	Report     IncidentPurgeReport           `json:"report,omitempty"`
	Outcome    IncidentPurgeOutcome          `json:"outcome,omitempty"`
	Errors     []string                      `json:"errors,omitempty"`
	Notices    []IncidentPurgeWorkflowNotice `json:"notices,omitempty"`
}

// IncidentPurgeWorkflowNotice represents semantic workflow context for compact and structured output.
type IncidentPurgeWorkflowNotice struct {
	Code     string            `json:"code,omitempty"`
	Severity string            `json:"severity,omitempty"`
	Message  string            `json:"message,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
}
