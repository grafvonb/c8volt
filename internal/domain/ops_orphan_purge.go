// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"time"

	"github.com/grafvonb/c8volt/typex"
)

type OpsWorkflowStepStatus string

const (
	OpsWorkflowStepStatusPlanned            OpsWorkflowStepStatus = "planned"
	OpsWorkflowStepStatusSkipped            OpsWorkflowStepStatus = "skipped"
	OpsWorkflowStepStatusSubmitted          OpsWorkflowStepStatus = "submitted"
	OpsWorkflowStepStatusConfirmed          OpsWorkflowStepStatus = "confirmed"
	OpsWorkflowStepStatusConfirmationFailed OpsWorkflowStepStatus = "confirmation_failed"
	OpsWorkflowStepStatusBlocked            OpsWorkflowStepStatus = "blocked"
	OpsWorkflowStepStatusFailed             OpsWorkflowStepStatus = "failed"
)

type OrphanPurgeOutcome string

const (
	OrphanPurgeOutcomePlanned         OrphanPurgeOutcome = "planned"
	OrphanPurgeOutcomeDeleted         OrphanPurgeOutcome = "deleted"
	OrphanPurgeOutcomePartiallyFailed OrphanPurgeOutcome = "partially_failed"
	OrphanPurgeOutcomeFailed          OrphanPurgeOutcome = "failed"
)

type OrphanPurgeRequest struct {
	CommandName  string
	DryRun       bool
	AutoConfirm  bool
	Automation   bool
	OutputMode   string
	Selection    ProcessInstanceFilter
	BatchSize    int32
	Limit        int32
	Workers      int
	ReportFile   string
	ReportFormat string
	StartedAt    time.Time
}

type OrphanDiscoveryResult struct {
	Status  OpsWorkflowStepStatus
	Filters ProcessInstanceFilter
	Keys    typex.Keys
	Count   int
	Errors  []string
}

type DeletionPlan struct {
	Status               OpsWorkflowStepStatus
	RequestedKeys        typex.Keys
	AffectedKeys         typex.Keys
	RootKeys             typex.Keys
	RequiresConfirmation bool
	DryRunPreview        DryRunPIKeyExpansion
	Errors               []string
}

type DeletionResult struct {
	Status    OpsWorkflowStepStatus
	Items     []Reporter
	Errors    []string
	Submitted bool
	Confirmed bool
}

type OrphanPurgeReport struct {
	SchemaVersion    string
	CommandName      string
	StartedAt        time.Time
	FinishedAt       time.Time
	Duration         string
	DryRun           bool
	C8voltVersion    string
	CamundaVersion   string
	ProfileIdentity  string
	SelectionFilters ProcessInstanceFilter
	Discovery        OrphanDiscoveryResult
	DeletionPlan     DeletionPlan
	Deletion         DeletionResult
	DeleteRequested  bool
	AutoConfirm      bool
	Automation       bool
	Errors           []string
	Outcome          OrphanPurgeOutcome
}

type OrphanPurgeResult struct {
	Request         OrphanPurgeRequest
	Discovery       OrphanDiscoveryResult
	DeletionPlan    DeletionPlan
	Deletion        DeletionResult
	Report          OrphanPurgeReport
	DeleteRequested bool
	Outcome         OrphanPurgeOutcome
	Errors          []string
}
