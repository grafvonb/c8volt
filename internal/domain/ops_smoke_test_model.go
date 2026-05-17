// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"time"

	"github.com/grafvonb/c8volt/typex"
)

// SmokeTestOutcome is the final state of one ops smoke-test workflow run.
type SmokeTestOutcome string

const (
	// SmokeTestOutcomePlanned means the workflow produced a read-only plan.
	SmokeTestOutcomePlanned SmokeTestOutcome = "planned"
	// SmokeTestOutcomePassed means every smoke-test step and cleanup step passed.
	SmokeTestOutcomePassed SmokeTestOutcome = "passed"
	// SmokeTestOutcomePassedCleanupSkipped means proof steps passed and cleanup was intentionally skipped.
	SmokeTestOutcomePassedCleanupSkipped SmokeTestOutcome = "passed_cleanup_skipped"
	// SmokeTestOutcomePartiallyFailed means at least one mutation succeeded and a later step failed.
	SmokeTestOutcomePartiallyFailed SmokeTestOutcome = "partially_failed"
	// SmokeTestOutcomeFailed means validation, planning, or workflow execution failed before a partial success state.
	SmokeTestOutcomeFailed SmokeTestOutcome = "failed"
)

// SmokeTestRequest captures one requested ops execute smoke-test run.
type SmokeTestRequest struct {
	CommandName   string
	DryRun        bool
	Count         int
	Workers       int
	FailFast      bool
	NoWorkerLimit bool
	NoCleanup     bool
	AutoConfirm   bool
	Automation    bool
	NoWait        bool
	OutputMode    string
	ReportFile    string
	ReportFormat  string
	StartedAt     time.Time
}

// WorkflowStepResult captures compact status for one smoke-test workflow step.
type WorkflowStepResult struct {
	Name    string
	Status  OpsWorkflowStepStatus
	Message string
	Errors  []string
}

// EmbeddedSmokeTestFixture describes the selected version-specific BPMN fixture.
type EmbeddedSmokeTestFixture struct {
	CamundaVersion string
	File           string
	BpmnProcessID  string
	Available      bool
}

// SmokeTestPlan captures read-only planning output before mutation.
type SmokeTestPlan struct {
	Status           OpsWorkflowStepStatus
	CamundaVersion   string
	Fixture          EmbeddedSmokeTestFixture
	CleanupRequested bool
	PlannedSteps     []WorkflowStepResult
	Errors           []string
}

// SmokeTestDeploymentResult captures deployment step output.
type SmokeTestDeploymentResult struct {
	Status                   OpsWorkflowStepStatus
	FixtureFile              string
	BpmnProcessID            string
	ProcessDefinitionKey     string
	ProcessDefinitionVersion int32
	TenantID                 string
	Errors                   []string
}

// SmokeTestRunItem captures one process-instance creation attempt.
type SmokeTestRunItem struct {
	ProcessInstanceKey string
	Status             OpsWorkflowStepStatus
	Error              string
}

// SmokeTestRunResult captures process-instance creation output.
type SmokeTestRunResult struct {
	Status              OpsWorkflowStepStatus
	RequestedCount      int
	CreatedCount        int
	ProcessInstanceKeys typex.Keys
	Items               []SmokeTestRunItem
	Errors              []string
}

// SmokeTestTraversalSummary captures report-safe traversal details for one created instance family.
type SmokeTestTraversalSummary struct {
	ProcessInstanceKey     string
	RootProcessInstanceKey string
	FamilyKeys             typex.Keys
	MissingAncestors       []MissingAncestor
	Warning                string
	Outcome                TraversalOutcome
}

// SmokeTestWalkItem captures one traversal attempt for a created process instance.
type SmokeTestWalkItem struct {
	ProcessInstanceKey string
	Status             OpsWorkflowStepStatus
	Summary            SmokeTestTraversalSummary
	Error              string
}

// SmokeTestWalkResult captures traversal output for created process instances.
type SmokeTestWalkResult struct {
	Status OpsWorkflowStepStatus
	Items  []SmokeTestWalkItem
	Errors []string
}

// SmokeTestCleanupEligibility captures the safety decision for process-definition cleanup.
type SmokeTestCleanupEligibility struct {
	Status   OpsWorkflowStepStatus
	Eligible bool
	Blockers []string
	Errors   []string
}

// SmokeTestProcessInstanceCleanupResult captures delete-pi cleanup output.
type SmokeTestProcessInstanceCleanupResult struct {
	Status        OpsWorkflowStepStatus
	SubmittedKeys typex.Keys
	Items         []Reporter
	Submitted     bool
	Confirmed     bool
	NoWait        bool
	Errors        []string
}

// SmokeTestProcessDefinitionCleanupResult captures process-definition cleanup output.
type SmokeTestProcessDefinitionCleanupResult struct {
	Status                        OpsWorkflowStepStatus
	SubmittedProcessDefinitionKey string
	Items                         []ResourceDeleteResponse
	Submitted                     bool
	Confirmed                     bool
	NoWait                        bool
	Errors                        []string
}

// SmokeTestCleanupResult captures process-instance and process-definition cleanup output.
type SmokeTestCleanupResult struct {
	ProcessInstanceCleanup       SmokeTestProcessInstanceCleanupResult
	ProcessDefinitionEligibility SmokeTestCleanupEligibility
	ProcessDefinitionCleanup     SmokeTestProcessDefinitionCleanupResult
	NoCleanup                    bool
	Errors                       []string
}

// SmokeTestAuditReport is the stable report model for smoke-test output and audit files.
type SmokeTestAuditReport struct {
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
	Fixture          EmbeddedSmokeTestFixture
	Plan             SmokeTestPlan
	Deployment       SmokeTestDeploymentResult
	Run              SmokeTestRunResult
	Walk             SmokeTestWalkResult
	CleanupRequested bool
	NoCleanup        bool
	Cleanup          SmokeTestCleanupResult
	AutoConfirm      bool
	Automation       bool
	NoWait           bool
	Errors           []string
	Outcome          SmokeTestOutcome
}

// SmokeTestResult carries the full smoke-test workflow result across service boundaries.
type SmokeTestResult struct {
	Request    SmokeTestRequest
	Plan       SmokeTestPlan
	Fixture    EmbeddedSmokeTestFixture
	Deployment SmokeTestDeploymentResult
	Run        SmokeTestRunResult
	Walk       SmokeTestWalkResult
	Cleanup    SmokeTestCleanupResult
	Report     SmokeTestAuditReport
	Outcome    SmokeTestOutcome
	Errors     []string
}
