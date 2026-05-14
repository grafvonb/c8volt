// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"time"

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
