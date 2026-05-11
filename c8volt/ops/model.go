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
	CommandName  string                        `json:"commandName,omitempty"`
	DryRun       bool                          `json:"dryRun,omitempty"`
	AutoConfirm  bool                          `json:"autoConfirm,omitempty"`
	Automation   bool                          `json:"automation,omitempty"`
	OutputMode   string                        `json:"outputMode,omitempty"`
	Selection    process.ProcessInstanceFilter `json:"selection,omitempty"`
	BatchSize    int32                         `json:"batchSize,omitempty"`
	Limit        int32                         `json:"limit,omitempty"`
	Workers      int                           `json:"workers,omitempty"`
	ReportFile   string                        `json:"reportFile,omitempty"`
	ReportFormat string                        `json:"reportFormat,omitempty"`
	StartedAt    time.Time                     `json:"startedAt,omitempty"`
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
