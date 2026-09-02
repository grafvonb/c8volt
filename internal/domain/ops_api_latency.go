// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "time"

const (
	// APILatencySchemaVersion is the stable command/report payload schema for API latency diagnostics.
	APILatencySchemaVersion = "ops.api-latency.v1"
)

// APILatencyMode identifies whether a diagnostic is read-only or active.
type APILatencyMode string

const (
	// APILatencyModeReadOnly measures read paths without Camunda mutations.
	APILatencyModeReadOnly APILatencyMode = "read_only"
	// APILatencyModeActive measures active write, read, visibility, and cleanup behavior.
	APILatencyModeActive APILatencyMode = "active"
)

// APILatencyBackoffStrategy records the normalized visibility-polling delay family.
type APILatencyBackoffStrategy string

const (
	// APILatencyBackoffFixed keeps the same delay between visibility attempts.
	APILatencyBackoffFixed APILatencyBackoffStrategy = "fixed"
	// APILatencyBackoffExponential multiplies delays until MaxDelay is reached.
	APILatencyBackoffExponential APILatencyBackoffStrategy = "exponential"
)

// APILatencyBackoff captures the bounded visibility retry policy without importing config into domain models.
type APILatencyBackoff struct {
	Strategy     APILatencyBackoffStrategy `json:"strategy,omitempty"`
	InitialDelay time.Duration             `json:"initialDelay,omitempty"`
	MaxDelay     time.Duration             `json:"maxDelay,omitempty"`
	Multiplier   float64                   `json:"multiplier,omitempty"`
	Timeout      time.Duration             `json:"timeout,omitempty"`
	MaxRetries   int                       `json:"maxRetries,omitempty"`
}

// APILatencyRequest captures the normalized input for one API latency diagnostic.
type APILatencyRequest struct {
	CommandName string                 `json:"commandName,omitempty"`
	Mode        APILatencyMode         `json:"mode,omitempty"`
	Count       int                    `json:"count,omitempty"`
	Workers     int                    `json:"workers,omitempty"`
	DryRun      bool                   `json:"dryRun,omitempty"`
	NoCleanup   bool                   `json:"noCleanup,omitempty"`
	TenantID    string                 `json:"tenantId,omitempty"`
	HTTPTimeout time.Duration          `json:"httpTimeout,omitempty"`
	Backoff     APILatencyBackoff      `json:"backoff,omitempty"`
	OutputMode  string                 `json:"outputMode,omitempty"`
	StartedAt   time.Time              `json:"startedAt,omitempty"`
	Progress    func(OpsProgressEvent) `json:"-"`
}

// APILatencyPlan is the immutable bounded stage and derived-request plan.
type APILatencyPlan struct {
	RunID                   string                 `json:"runId,omitempty"`
	Mode                    APILatencyMode         `json:"mode,omitempty"`
	Stages                  []APILatencyStagePlan  `json:"stages,omitempty"`
	PrimarySampleLimit      int                    `json:"primarySampleLimit,omitempty"`
	PrimarySampleAllocation int                    `json:"primarySampleAllocation,omitempty"`
	DerivedRequestLimit     int                    `json:"derivedRequestLimit,omitempty"`
	VisibilityAttemptLimit  int                    `json:"visibilityAttemptLimit,omitempty"`
	SetupOperations         []string               `json:"setupOperations,omitempty"`
	Fixture                 *APILatencyFixturePlan `json:"fixture,omitempty"`
	Cleanup                 *APILatencyCleanupPlan `json:"cleanup,omitempty"`
	Notices                 []string               `json:"notices,omitempty"`
	Limitations             []string               `json:"limitations,omitempty"`
}

// APILatencyStagePlan describes one deterministic closed-loop worker stage.
type APILatencyStagePlan struct {
	Index               int `json:"index,omitempty"`
	WorkerCount         int `json:"workerCount,omitempty"`
	PrimarySamples      int `json:"primarySamples,omitempty"`
	DerivedRequestLimit int `json:"derivedRequestLimit,omitempty"`
}

// APILatencyFixturePlan identifies the active diagnostic fixture selected for a Camunda version.
type APILatencyFixturePlan struct {
	CamundaVersion string `json:"camundaVersion,omitempty"`
	File           string `json:"file,omitempty"`
	BpmnProcessID  string `json:"bpmnProcessId,omitempty"`
	Available      bool   `json:"available,omitempty"`
}

// APILatencyMeasurementCategory identifies a measured logical service-call family.
type APILatencyMeasurementCategory string

const (
	// APILatencyCategoryTopologyRead measures the cluster control path.
	APILatencyCategoryTopologyRead APILatencyMeasurementCategory = "topology_read"
	// APILatencyCategoryProcessDefinitionSearch measures process-definition queries.
	APILatencyCategoryProcessDefinitionSearch APILatencyMeasurementCategory = "process_definition_search"
	// APILatencyCategoryProcessInstanceSearch measures process-instance queries.
	APILatencyCategoryProcessInstanceSearch APILatencyMeasurementCategory = "process_instance_search"
	// APILatencyCategoryProcessDefinitionRead measures direct process-definition reads.
	APILatencyCategoryProcessDefinitionRead APILatencyMeasurementCategory = "process_definition_read"
	// APILatencyCategoryProcessInstanceRead measures direct process-instance reads.
	APILatencyCategoryProcessInstanceRead APILatencyMeasurementCategory = "process_instance_read"
	// APILatencyCategoryFixtureDeploy measures diagnostic fixture deployment.
	APILatencyCategoryFixtureDeploy APILatencyMeasurementCategory = "fixture_deploy"
	// APILatencyCategoryProcessInstanceCreate measures process-instance create responses.
	APILatencyCategoryProcessInstanceCreate APILatencyMeasurementCategory = "process_instance_create"
	// APILatencyCategoryConcurrentRead measures read probes issued while writes are active.
	APILatencyCategoryConcurrentRead APILatencyMeasurementCategory = "concurrent_read"
	// APILatencyCategorySearchVisibility measures exact-key exporter/search visibility.
	APILatencyCategorySearchVisibility APILatencyMeasurementCategory = "search_visibility"
)

// APILatencyMeasurementKind identifies whether a measurement consumes primary budget.
type APILatencyMeasurementKind string

const (
	// APILatencyMeasurementKindPrimary counts against the operator's primary sample budget.
	APILatencyMeasurementKindPrimary APILatencyMeasurementKind = "primary"
	// APILatencyMeasurementKindDerived is bounded follow-up evidence from a primary sample.
	APILatencyMeasurementKindDerived APILatencyMeasurementKind = "derived"
	// APILatencyMeasurementKindSetup is pre-sample setup or preflight evidence.
	APILatencyMeasurementKindSetup APILatencyMeasurementKind = "setup"
	// APILatencyMeasurementKindCleanup records cleanup evidence outside primary sampling.
	APILatencyMeasurementKindCleanup APILatencyMeasurementKind = "cleanup"
)

// APILatencyMeasurementOutcome records the safe terminal state of one logical measurement.
type APILatencyMeasurementOutcome string

const (
	// APILatencyMeasurementSucceeded means the logical call completed successfully.
	APILatencyMeasurementSucceeded APILatencyMeasurementOutcome = "succeeded"
	// APILatencyMeasurementUnavailable means a supported sample could not produce usable evidence.
	APILatencyMeasurementUnavailable APILatencyMeasurementOutcome = "unavailable"
	// APILatencyMeasurementFailed means the logical call failed for a non-timeout reason.
	APILatencyMeasurementFailed APILatencyMeasurementOutcome = "failed"
	// APILatencyMeasurementTimedOut means the logical call hit a timeout boundary.
	APILatencyMeasurementTimedOut APILatencyMeasurementOutcome = "timed_out"
)

// APILatencyClassification is a bounded sanitized response class used in diagnostics.
type APILatencyClassification string

const (
	// APILatencyClassificationSuccess records successful evidence.
	APILatencyClassificationSuccess APILatencyClassification = "success"
	// APILatencyClassificationUnavailable records missing or unavailable evidence.
	APILatencyClassificationUnavailable APILatencyClassification = "unavailable"
	// APILatencyClassificationUnsupported records version or API incompatibility.
	APILatencyClassificationUnsupported APILatencyClassification = "unsupported"
	// APILatencyClassificationNotFound records resource disappearance or exact lookup misses.
	APILatencyClassificationNotFound APILatencyClassification = "not_found"
	// APILatencyClassificationTimeout records request timeout evidence.
	APILatencyClassificationTimeout APILatencyClassification = "timeout"
	// APILatencyClassificationBackpressure records Camunda throttling evidence.
	APILatencyClassificationBackpressure APILatencyClassification = "backpressure"
	// APILatencyClassificationUnhealthyPartition records topology health evidence.
	APILatencyClassificationUnhealthyPartition APILatencyClassification = "unhealthy_partition"
	// APILatencyClassificationMissingLeader records topology leader absence.
	APILatencyClassificationMissingLeader APILatencyClassification = "missing_leader"
	// APILatencyClassificationAuthentication records authentication or authorization failures.
	APILatencyClassificationAuthentication APILatencyClassification = "authentication"
	// APILatencyClassificationConnectivity records connectivity or service availability failures.
	APILatencyClassificationConnectivity APILatencyClassification = "connectivity"
	// APILatencyClassificationMalformedResponse records unusable upstream payloads.
	APILatencyClassificationMalformedResponse APILatencyClassification = "malformed_response"
	// APILatencyClassificationRequestError records other sanitized request failures.
	APILatencyClassificationRequestError APILatencyClassification = "request_error"
)

// APILatencyMeasurement is one timed logical operation collected by a diagnostic stage.
type APILatencyMeasurement struct {
	StageIndex      int                           `json:"stageIndex,omitempty"`
	Category        APILatencyMeasurementCategory `json:"category,omitempty"`
	Kind            APILatencyMeasurementKind     `json:"kind,omitempty"`
	StartedAt       time.Time                     `json:"startedAt,omitempty"`
	Duration        time.Duration                 `json:"duration,omitempty"`
	Outcome         APILatencyMeasurementOutcome  `json:"outcome,omitempty"`
	Classification  APILatencyClassification      `json:"classification,omitempty"`
	OverlappedWrite bool                          `json:"overlappedWrite,omitempty"`
}

// APILatencyStageStatus identifies stage lifecycle state.
type APILatencyStageStatus string

const (
	// APILatencyStageStatusPlanned means work was previewed but not run.
	APILatencyStageStatusPlanned APILatencyStageStatus = "planned"
	// APILatencyStageStatusRunning means a stage is currently active.
	APILatencyStageStatusRunning APILatencyStageStatus = "running"
	// APILatencyStageStatusCompleted means all planned work for the stage was attempted.
	APILatencyStageStatusCompleted APILatencyStageStatus = "completed"
	// APILatencyStageStatusIncomplete means execution stopped before all planned work completed.
	APILatencyStageStatusIncomplete APILatencyStageStatus = "incomplete"
	// APILatencyStageStatusSkipped means the stage was intentionally not run.
	APILatencyStageStatusSkipped APILatencyStageStatus = "skipped"
)

// APILatencyCategorySummary aggregates measurement outcomes for one category in one stage.
type APILatencyCategorySummary struct {
	Category            APILatencyMeasurementCategory `json:"category,omitempty"`
	Attempts            int                           `json:"attempts,omitempty"`
	Successes           int                           `json:"successes,omitempty"`
	Errors              int                           `json:"errors,omitempty"`
	Timeouts            int                           `json:"timeouts,omitempty"`
	Unavailable         int                           `json:"unavailable,omitempty"`
	ThroughputPerSecond *float64                      `json:"throughputPerSecond,omitempty"`
	P50                 *time.Duration                `json:"p50,omitempty"`
	P95                 *time.Duration                `json:"p95,omitempty"`
	Max                 *time.Duration                `json:"max,omitempty"`
}

// APILatencyClassificationCount records an ordered classification frequency.
type APILatencyClassificationCount struct {
	Classification APILatencyClassification `json:"classification,omitempty"`
	Count          int                      `json:"count,omitempty"`
}

// APILatencyCategoryComparison stores prior-stage deltas where a safe baseline exists.
type APILatencyCategoryComparison struct {
	Category                 APILatencyMeasurementCategory `json:"category,omitempty"`
	P50Delta                 *time.Duration                `json:"p50Delta,omitempty"`
	P50DeltaPercent          *float64                      `json:"p50DeltaPercent,omitempty"`
	ThroughputDelta          *float64                      `json:"throughputDelta,omitempty"`
	ThroughputDeltaPercent   *float64                      `json:"throughputDeltaPercent,omitempty"`
	ComparisonSampleCount    int                           `json:"comparisonSampleCount,omitempty"`
	PreviousStageUnavailable bool                          `json:"previousStageUnavailable,omitempty"`
}

// APILatencyStageComparison contains all category deltas for one stage.
type APILatencyStageComparison struct {
	Categories []APILatencyCategoryComparison `json:"categories,omitempty"`
}

// APILatencyStageResult aggregates completed measurements for one stage.
type APILatencyStageResult struct {
	Plan                 APILatencyStagePlan             `json:"plan,omitempty"`
	Status               APILatencyStageStatus           `json:"status,omitempty"`
	StartedAt            time.Time                       `json:"startedAt,omitempty"`
	FinishedAt           time.Time                       `json:"finishedAt,omitempty"`
	ActualMaxConcurrency int                             `json:"actualMaxConcurrency,omitempty"`
	PrimaryAttempts      int                             `json:"primaryAttempts,omitempty"`
	DerivedAttempts      int                             `json:"derivedAttempts,omitempty"`
	Categories           []APILatencyCategorySummary     `json:"categories,omitempty"`
	Classifications      []APILatencyClassificationCount `json:"classifications,omitempty"`
	Comparison           *APILatencyStageComparison      `json:"comparison,omitempty"`
}

// APILatencyTopologyEvidence records safe topology facts used by findings.
type APILatencyTopologyEvidence struct {
	BrokerCount          int   `json:"brokerCount,omitempty"`
	PartitionCount       int   `json:"partitionCount,omitempty"`
	UnhealthyPartitions  []int `json:"unhealthyPartitions,omitempty"`
	LeaderlessPartitions []int `json:"leaderlessPartitions,omitempty"`
	HealthKnown          bool  `json:"healthKnown"`
}

// APILatencyFindingConfidence records confidence for a diagnostic interpretation.
type APILatencyFindingConfidence string

const (
	// APILatencyFindingConfidenceHigh means direct evidence strongly supports the finding.
	APILatencyFindingConfidenceHigh APILatencyFindingConfidence = "high"
	// APILatencyFindingConfidenceMedium means the finding is supported by comparative evidence.
	APILatencyFindingConfidenceMedium APILatencyFindingConfidence = "medium"
	// APILatencyFindingConfidenceLow means the finding is a bounded fallback.
	APILatencyFindingConfidenceLow APILatencyFindingConfidence = "low"
)

// APILatencyFinding describes one deterministic interpretation of measured evidence.
type APILatencyFinding struct {
	Code              string                      `json:"code,omitempty"`
	Evidence          []string                    `json:"evidence,omitempty"`
	LikelyArea        string                      `json:"likelyArea,omitempty"`
	Confidence        APILatencyFindingConfidence `json:"confidence,omitempty"`
	Limitation        string                      `json:"limitation,omitempty"`
	NextInvestigation string                      `json:"nextInvestigation,omitempty"`
}

// APILatencyRunContext is the safe reportable invocation context.
type APILatencyRunContext struct {
	CommandName    string    `json:"commandName,omitempty"`
	SchemaVersion  string    `json:"schemaVersion,omitempty"`
	C8voltVersion  string    `json:"c8voltVersion,omitempty"`
	CamundaVersion string    `json:"camundaVersion,omitempty"`
	Profile        string    `json:"profile,omitempty"`
	Tenant         string    `json:"tenant,omitempty"`
	StartedAt      time.Time `json:"startedAt,omitempty"`
	FinishedAt     time.Time `json:"finishedAt,omitempty"`
	Duration       string    `json:"duration,omitempty"`
}

// APILatencyOwnership records exact active-run cleanup authority.
type APILatencyOwnership struct {
	RunID                string   `json:"runId,omitempty"`
	FixtureName          string   `json:"fixtureName,omitempty"`
	BpmnProcessID        string   `json:"bpmnProcessId,omitempty"`
	DeploymentSubmitted  bool     `json:"deploymentSubmitted,omitempty"`
	ProcessDefinitionKey string   `json:"processDefinitionKey,omitempty"`
	ProcessInstanceKeys  []string `json:"processInstanceKeys,omitempty"`
}

// APILatencyVisibilityResult records exact-key search visibility evidence.
type APILatencyVisibilityResult struct {
	ProcessInstanceKey  string                   `json:"processInstanceKey,omitempty"`
	Attempts            int                      `json:"attempts,omitempty"`
	AttemptLimit        int                      `json:"attemptLimit,omitempty"`
	Visible             bool                     `json:"visible,omitempty"`
	Duration            time.Duration            `json:"duration,omitempty"`
	FinalClassification APILatencyClassification `json:"finalClassification,omitempty"`
}

// APILatencyCleanupPlan records requested cleanup capability and retention behavior.
type APILatencyCleanupPlan struct {
	Requested            bool          `json:"requested,omitempty"`
	Supported            bool          `json:"supported,omitempty"`
	IntentionalRetention bool          `json:"intentionalRetention,omitempty"`
	IndependentBudget    time.Duration `json:"independentBudget,omitempty"`
	BlockReason          string        `json:"blockReason,omitempty"`
}

// APILatencyCleanupResourceType identifies a cleanup target kind.
type APILatencyCleanupResourceType string

const (
	// APILatencyCleanupResourceProcessInstance identifies a recorded process-instance root.
	APILatencyCleanupResourceProcessInstance APILatencyCleanupResourceType = "process_instance"
	// APILatencyCleanupResourceProcessDefinition identifies a recorded process-definition key.
	APILatencyCleanupResourceProcessDefinition APILatencyCleanupResourceType = "process_definition"
)

// APILatencyCleanupStatus records terminal cleanup state for one owned resource.
type APILatencyCleanupStatus string

const (
	// APILatencyCleanupStatusPending means cleanup has not started.
	APILatencyCleanupStatusPending APILatencyCleanupStatus = "pending"
	// APILatencyCleanupStatusSubmitted means deletion was submitted but not confirmed.
	APILatencyCleanupStatusSubmitted APILatencyCleanupStatus = "submitted"
	// APILatencyCleanupStatusDeleted means the exact resource was deleted.
	APILatencyCleanupStatusDeleted APILatencyCleanupStatus = "deleted"
	// APILatencyCleanupStatusRetained means explicit no-cleanup retained the resource.
	APILatencyCleanupStatusRetained APILatencyCleanupStatus = "retained"
	// APILatencyCleanupStatusFailed means cleanup failed for the exact resource.
	APILatencyCleanupStatusFailed APILatencyCleanupStatus = "failed"
	// APILatencyCleanupStatusUnknown means terminal ownership state could not be proven.
	APILatencyCleanupStatusUnknown APILatencyCleanupStatus = "unknown"
)

// APILatencyCleanupRecord records cleanup evidence and exact-key recovery guidance.
type APILatencyCleanupRecord struct {
	ResourceType    APILatencyCleanupResourceType `json:"resourceType,omitempty"`
	Key             string                        `json:"key,omitempty"`
	Status          APILatencyCleanupStatus       `json:"status,omitempty"`
	Classification  APILatencyClassification      `json:"classification,omitempty"`
	RecoveryCommand string                        `json:"recoveryCommand,omitempty"`
}

// APILatencyOutcome identifies the final diagnostic outcome.
type APILatencyOutcome string

const (
	// APILatencyOutcomePlanned means a dry-run plan completed without mutation.
	APILatencyOutcomePlanned APILatencyOutcome = "planned"
	// APILatencyOutcomeCompleted means all planned stages completed with usable evidence.
	APILatencyOutcomeCompleted APILatencyOutcome = "completed"
	// APILatencyOutcomeCompletedRetained means an explicit no-cleanup run completed with retained resources.
	APILatencyOutcomeCompletedRetained APILatencyOutcome = "completed_retained"
	// APILatencyOutcomePartial means usable evidence exists but execution or cleanup is incomplete.
	APILatencyOutcomePartial APILatencyOutcome = "partial"
	// APILatencyOutcomeFailed means a plan, execution, report, or cleanup failure prevented completion.
	APILatencyOutcomeFailed APILatencyOutcome = "failed"
	// APILatencyOutcomeInterrupted means caller cancellation stopped measurement work.
	APILatencyOutcomeInterrupted APILatencyOutcome = "interrupted"
)

// APILatencyResult is the render-independent API latency diagnostic payload.
type APILatencyResult struct {
	SchemaVersion string                       `json:"schemaVersion,omitempty"`
	Context       APILatencyRunContext         `json:"context,omitempty"`
	Request       APILatencyRequest            `json:"request,omitempty"`
	Plan          APILatencyPlan               `json:"plan,omitempty"`
	Topology      APILatencyTopologyEvidence   `json:"topology,omitempty"`
	Stages        []APILatencyStageResult      `json:"stages,omitempty"`
	Findings      []APILatencyFinding          `json:"findings,omitempty"`
	Notices       []string                     `json:"notices,omitempty"`
	Limitations   []string                     `json:"limitations,omitempty"`
	Ownership     *APILatencyOwnership         `json:"ownership,omitempty"`
	Visibility    []APILatencyVisibilityResult `json:"visibility,omitempty"`
	Cleanup       []APILatencyCleanupRecord    `json:"cleanup,omitempty"`
	Outcome       APILatencyOutcome            `json:"outcome,omitempty"`
}
