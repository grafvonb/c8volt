// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"time"

	"github.com/grafvonb/c8volt/typex"
)

type RetentionPolicyOutcome string

const (
	RetentionPolicyOutcomePlanned         RetentionPolicyOutcome = "planned"
	RetentionPolicyOutcomeDeleted         RetentionPolicyOutcome = "deleted"
	RetentionPolicyOutcomePartiallyFailed RetentionPolicyOutcome = "partially_failed"
	RetentionPolicyOutcomeFailed          RetentionPolicyOutcome = "failed"
)

type RetentionPolicyRequest struct {
	CommandName            string
	RetentionDays          int
	DerivedEndDateBoundary string
	DryRun                 bool
	AutoConfirm            bool
	Automation             bool
	OutputMode             string
	Selection              ProcessInstanceFilter
	BatchSize              int32
	Limit                  int32
	Workers                int
	NoWait                 bool
	NoStateCheck           bool
	Force                  bool
	FailFast               bool
	NoWorkerLimit          bool
	ReportFile             string
	ReportFormat           string
	StartedAt              time.Time
}

type RetentionDiscoveryResult struct {
	Status                 OpsWorkflowStepStatus
	RetentionDays          int
	DerivedEndDateBoundary string
	Filters                ProcessInstanceFilter
	SeedKeys               typex.Keys
	Count                  int
	Notices                []RetentionWorkflowNotice
	Errors                 []string
}

type RetentionDeletePlan struct {
	Status                OpsWorkflowStepStatus
	SeedKeys              typex.Keys
	ResolvedRootKeys      typex.Keys
	AffectedKeys          typex.Keys
	DuplicateKeys         typex.Keys
	FinalStateItems       []ProcessInstance
	NonFinalAffectedItems []ProcessInstance
	MissingAncestors      []MissingAncestor
	TraversalWarnings     []string
	RequiresConfirmation  bool
	Errors                []string
}

type RetentionDeletionResult struct {
	Status            OpsWorkflowStepStatus
	SubmittedRootKeys typex.Keys
	Items             []Reporter
	Submitted         bool
	Confirmed         bool
	NoWait            bool
	Errors            []string
}

type RetentionAuditReport struct {
	SchemaVersion          string
	CommandName            string
	StartedAt              time.Time
	FinishedAt             time.Time
	Duration               string
	DryRun                 bool
	C8voltVersion          string
	CamundaVersion         string
	ProfileIdentity        string
	TenantID               string
	RetentionDays          int
	DerivedEndDateBoundary string
	SelectionFilters       ProcessInstanceFilter
	Discovery              RetentionDiscoveryResult
	DeletePlan             RetentionDeletePlan
	Deletion               RetentionDeletionResult
	AutoConfirm            bool
	Automation             bool
	NoWait                 bool
	NoStateCheck           bool
	Force                  bool
	FailFast               bool
	NoWorkerLimit          bool
	Errors                 []string
	Outcome                RetentionPolicyOutcome
}

type RetentionPolicyResult struct {
	Request    RetentionPolicyRequest
	Discovery  RetentionDiscoveryResult
	DeletePlan RetentionDeletePlan
	Deletion   RetentionDeletionResult
	Report     RetentionAuditReport
	Outcome    RetentionPolicyOutcome
	Errors     []string
}

type RetentionWorkflowNotice struct {
	Code     string
	Severity string
	Message  string
	Details  map[string]string
}
