// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"time"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/toolx"
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
	DuplicateResolvedRootKeys             typex.Keys                `json:"duplicateResolvedRootKeys,omitempty"`
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

// AllProcessDefinitionsPurgeOutcome is the final state of the all-process-definitions purge workflow.
type AllProcessDefinitionsPurgeOutcome string

const (
	AllProcessDefinitionsPurgeOutcomePlanned         AllProcessDefinitionsPurgeOutcome = "planned"
	AllProcessDefinitionsPurgeOutcomeDeleted         AllProcessDefinitionsPurgeOutcome = "deleted"
	AllProcessDefinitionsPurgeOutcomePartiallyFailed AllProcessDefinitionsPurgeOutcome = "partially_failed"
	AllProcessDefinitionsPurgeOutcomeFailed          AllProcessDefinitionsPurgeOutcome = "failed"
)

// ProcessDefinitionSelection captures supported process-definition filters for ops purge workflows.
type ProcessDefinitionSelection struct {
	Key               string `json:"key,omitempty"`
	BpmnProcessId     string `json:"bpmnProcessId,omitempty"`
	ProcessVersion    int32  `json:"processVersion,omitempty"`
	ProcessVersionTag string `json:"processVersionTag,omitempty"`
	LatestOnly        bool   `json:"latestOnly,omitempty"`
}

// String returns the active process-definition selection fields in stable CLI order.
func (s ProcessDefinitionSelection) String() string {
	parts := make([]string, 0, 5)
	parts = toolx.AppendQuotedField(parts, "key", s.Key)
	parts = toolx.AppendQuotedField(parts, "bpmnProcessId", s.BpmnProcessId)
	parts = toolx.AppendInt32Field(parts, "processVersion", s.ProcessVersion)
	parts = toolx.AppendQuotedField(parts, "processVersionTag", s.ProcessVersionTag)
	parts = toolx.AppendTrueBoolField(parts, "latestOnly", s.LatestOnly)
	return toolx.FormatActiveFields(parts)
}

// AllProcessDefinitionsPurgeRequest captures one requested all-process-definitions purge run.
type AllProcessDefinitionsPurgeRequest struct {
	CommandName                              string                     `json:"commandName,omitempty"`
	DryRun                                   bool                       `json:"dryRun,omitempty"`
	AutoConfirm                              bool                       `json:"autoConfirm,omitempty"`
	Automation                               bool                       `json:"automation,omitempty"`
	OutputMode                               string                     `json:"outputMode,omitempty"`
	Selection                                ProcessDefinitionSelection `json:"selection,omitempty"`
	Workers                                  int                        `json:"workers,omitempty"`
	FailFast                                 bool                       `json:"failFast,omitempty"`
	NoWorkerLimit                            bool                       `json:"noWorkerLimit,omitempty"`
	NoWait                                   bool                       `json:"noWait,omitempty"`
	Force                                    bool                       `json:"force,omitempty"`
	ReportFile                               string                     `json:"reportFile,omitempty"`
	ReportFormat                             string                     `json:"reportFormat,omitempty"`
	DiscoveredCandidateProcessDefinitionKeys typex.Keys                 `json:"discoveredCandidateProcessDefinitionKeys,omitempty"`
	StartedAt                                time.Time                  `json:"startedAt,omitempty"`
}

// ProcessDefinitionDiscoveryResult captures immutable process-definition discovery output.
type ProcessDefinitionDiscoveryResult struct {
	Status                                  WorkflowStepStatus                 `json:"status,omitempty"`
	Filters                                 ProcessDefinitionSelection         `json:"filters,omitempty"`
	CandidateProcessDefinitionKeys          typex.Keys                         `json:"candidateProcessDefinitionKeys,omitempty"`
	CandidateProcessDefinitions             []process.ProcessDefinition        `json:"candidateProcessDefinitions,omitempty"`
	DuplicateCandidateProcessDefinitionKeys typex.Keys                         `json:"duplicateCandidateProcessDefinitionKeys,omitempty"`
	CandidateProcessDefinitionCount         int                                `json:"candidateProcessDefinitionCount"`
	LatestOnly                              bool                               `json:"latestOnly,omitempty"`
	Notices                                 []AllProcessDefinitionsPurgeNotice `json:"notices,omitempty"`
	Errors                                  []string                           `json:"errors,omitempty"`
}

// AllProcessDefinitionsPurgeDeletePlan captures the validated delete plan for frozen candidates.
type AllProcessDefinitionsPurgeDeletePlan struct {
	Status                                  WorkflowStepStatus                         `json:"status,omitempty"`
	CandidateProcessDefinitionKeys          typex.Keys                                 `json:"candidateProcessDefinitionKeys,omitempty"`
	Items                                   []resource.DeleteProcessDefinitionPlanItem `json:"items,omitempty"`
	DuplicateCandidateProcessDefinitionKeys typex.Keys                                 `json:"duplicateCandidateProcessDefinitionKeys,omitempty"`
	AffectedProcessInstanceCount            int64                                      `json:"affectedProcessInstanceCount,omitempty"`
	ActiveProcessInstanceCount              int64                                      `json:"activeProcessInstanceCount,omitempty"`
	RequiresConfirmation                    bool                                       `json:"requiresConfirmation,omitempty"`
	RequiresForce                           bool                                       `json:"requiresForce,omitempty"`
	Errors                                  []string                                   `json:"errors,omitempty"`
}

// AllProcessDefinitionsPurgeDeletionResult captures mutation submission and confirmation output.
type AllProcessDefinitionsPurgeDeletionResult struct {
	Status                         WorkflowStepStatus      `json:"status,omitempty"`
	SubmittedProcessDefinitionKeys typex.Keys              `json:"submittedProcessDefinitionKeys,omitempty"`
	Items                          []resource.DeleteReport `json:"items,omitempty"`
	Submitted                      bool                    `json:"submitted,omitempty"`
	Confirmed                      bool                    `json:"confirmed,omitempty"`
	NoWait                         bool                    `json:"noWait,omitempty"`
	Errors                         []string                `json:"errors,omitempty"`
}

// AllProcessDefinitionsPurgeReport is the stable audit model for output and report files.
type AllProcessDefinitionsPurgeReport struct {
	SchemaVersion    string                                   `json:"schemaVersion,omitempty"`
	CommandName      string                                   `json:"commandName,omitempty"`
	StartedAt        time.Time                                `json:"startedAt,omitempty"`
	FinishedAt       time.Time                                `json:"finishedAt,omitempty"`
	Duration         string                                   `json:"duration,omitempty"`
	DryRun           bool                                     `json:"dryRun,omitempty"`
	C8voltVersion    string                                   `json:"c8voltVersion,omitempty"`
	CamundaVersion   string                                   `json:"camundaVersion,omitempty"`
	ProfileIdentity  string                                   `json:"profileIdentity,omitempty"`
	TenantID         string                                   `json:"tenantId,omitempty"`
	SelectionFilters ProcessDefinitionSelection               `json:"selectionFilters,omitempty"`
	Discovery        ProcessDefinitionDiscoveryResult         `json:"discovery,omitempty"`
	DeletePlan       AllProcessDefinitionsPurgeDeletePlan     `json:"deletePlan,omitempty"`
	Deletion         AllProcessDefinitionsPurgeDeletionResult `json:"deletion,omitempty"`
	AutoConfirm      bool                                     `json:"autoConfirm,omitempty"`
	Automation       bool                                     `json:"automation,omitempty"`
	NoWait           bool                                     `json:"noWait,omitempty"`
	Force            bool                                     `json:"force,omitempty"`
	FailFast         bool                                     `json:"failFast,omitempty"`
	NoWorkerLimit    bool                                     `json:"noWorkerLimit,omitempty"`
	Errors           []string                                 `json:"errors,omitempty"`
	Notices          []AllProcessDefinitionsPurgeNotice       `json:"notices,omitempty"`
	Outcome          AllProcessDefinitionsPurgeOutcome        `json:"outcome,omitempty"`
}

// AllProcessDefinitionsPurgeResult carries the full workflow result across the service and facade boundary.
type AllProcessDefinitionsPurgeResult struct {
	Request    AllProcessDefinitionsPurgeRequest        `json:"request,omitempty"`
	Discovery  ProcessDefinitionDiscoveryResult         `json:"discovery,omitempty"`
	DeletePlan AllProcessDefinitionsPurgeDeletePlan     `json:"deletePlan,omitempty"`
	Deletion   AllProcessDefinitionsPurgeDeletionResult `json:"deletion,omitempty"`
	Report     AllProcessDefinitionsPurgeReport         `json:"report,omitempty"`
	Outcome    AllProcessDefinitionsPurgeOutcome        `json:"outcome,omitempty"`
	Errors     []string                                 `json:"errors,omitempty"`
	Notices    []AllProcessDefinitionsPurgeNotice       `json:"notices,omitempty"`
}

// AllProcessDefinitionsPurgeNotice represents semantic workflow context for compact and structured output.
type AllProcessDefinitionsPurgeNotice struct {
	Code     string            `json:"code,omitempty"`
	Severity string            `json:"severity,omitempty"`
	Message  string            `json:"message,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
}
