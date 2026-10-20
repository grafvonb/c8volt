// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// CompletionSubmissionStatus describes only submission evidence, independently
// of any later observation of the job or its process instance.
type CompletionSubmissionStatus string

const (
	// CompletionSubmissionNotSubmitted means no completion call was made.
	CompletionSubmissionNotSubmitted CompletionSubmissionStatus = "not_submitted"
	// CompletionSubmissionAccepted means the completion endpoint accepted the call.
	CompletionSubmissionAccepted CompletionSubmissionStatus = "accepted"
	// CompletionSubmissionRejected means an explicit backend rejection was received.
	CompletionSubmissionRejected CompletionSubmissionStatus = "rejected"
	// CompletionSubmissionUncertain means dispatch may have succeeded. A later
	// missing job cannot change this evidence into a rejection.
	CompletionSubmissionUncertain CompletionSubmissionStatus = "uncertain"
)

// CompletionSubmission retains the evidence from at most one intentional
// completion attempt. HTTPStatus is absent when the adapter has no response;
// Diagnostic carries normalized error detail without replacing the returned error.
type CompletionSubmission struct {
	Status     CompletionSubmissionStatus `json:"status"`
	HTTPStatus *int                       `json:"httpStatus,omitempty"`
	Diagnostic string                     `json:"diagnostic,omitempty"`
}

// CompletionSelectionMode identifies the source of frozen membership.
type CompletionSelectionMode string

const (
	// CompletionSelectionExplicit uses authorized key-only reads.
	CompletionSelectionExplicit CompletionSelectionMode = "explicit"
	// CompletionSelectionSearch uses tenant-scoped AND predicates.
	CompletionSelectionSearch CompletionSelectionMode = "search"
)

// CompletionInputForm retains input provenance before deduplication.
type CompletionInputForm string

const (
	// CompletionInputScalar identifies one scalar key.
	CompletionInputScalar CompletionInputForm = "scalar"
	// CompletionInputCollection identifies repeated or comma-separated keys.
	CompletionInputCollection CompletionInputForm = "collection"
	// CompletionInputStdin identifies deliberate stdin input, including empty input.
	CompletionInputStdin CompletionInputForm = "stdin"
	// CompletionInputFilters identifies predicate-based discovery.
	CompletionInputFilters CompletionInputForm = "filters"
)

// CompletionFilter holds normalized AND predicates without traversal controls.
// Retries is optional; a pointer to zero is a meaningful selector.
type CompletionFilter struct {
	State              string `json:"state,omitempty"`
	Type               string `json:"type,omitempty"`
	ProcessInstanceKey string `json:"processInstanceKey,omitempty"`
	ElementInstanceKey string `json:"elementInstanceKey,omitempty"`
	ElementId          string `json:"elementId,omitempty"`
	Worker             string `json:"worker,omitempty"`
	Retries            *int32 `json:"retries,omitempty"`
	Kind               string `json:"kind,omitempty"`
	ListenerEventType  string `json:"listenerEventType,omitempty"`
}

// CompletionSelection records original input order and mutually exclusive keys
// or predicates. Zero limit is uncapped; workers/options are not provenance.
type CompletionSelection struct {
	Mode      CompletionSelectionMode `json:"mode"`
	Keys      []string                `json:"keys,omitempty"`
	Filter    CompletionFilter        `json:"filter"`
	BatchSize int32                   `json:"batchSize"`
	Limit     int32                   `json:"limit"`
	InputForm CompletionInputForm     `json:"inputForm"`
}

// CompletionRequest combines selection with an optional completion payload.
type CompletionRequest struct {
	Selection CompletionSelection `json:"selection"`
	Variables map[string]any      `json:"variables,omitempty"`
}

// CompletionPreparationStatus separates executable plans from partial evidence.
type CompletionPreparationStatus string

const (
	// CompletionPreparationComplete includes authoritative empty selections.
	CompletionPreparationComplete CompletionPreparationStatus = "complete"
	// CompletionPreparationIncomplete must never be executed.
	CompletionPreparationIncomplete CompletionPreparationStatus = "incomplete"
)

// CompletionReason records the evidence that ended preparation successfully.
type CompletionReason string

const (
	// CompletionReasonExhausted means normalized page evidence proved exhaustion.
	CompletionReasonExhausted CompletionReason = "exhausted"
	// CompletionReasonLimitReached means the unique target cap was reached.
	CompletionReasonLimitReached CompletionReason = "limit_reached"
	// CompletionReasonExplicitInputsRead means every unique explicit key was read.
	CompletionReasonExplicitInputsRead CompletionReason = "explicit_inputs_read"
)

// CompletionCandidate preserves observed facts without inventing lifecycle dates.
type CompletionCandidate struct {
	Key                string `json:"key"`
	ProcessInstanceKey string `json:"processInstanceKey,omitempty"`
	ElementInstanceKey string `json:"elementInstanceKey,omitempty"`
	ElementId          string `json:"elementId,omitempty"`
	TenantId           string `json:"tenantId,omitempty"`
	Type               string `json:"type,omitempty"`
	Kind               string `json:"kind,omitempty"`
	ListenerEventType  string `json:"listenerEventType,omitempty"`
	State              string `json:"state,omitempty"`
	Worker             string `json:"worker,omitempty"`
	Retries            int32  `json:"retries"`
}

// PreparedCompletionBatch is an immutable in-memory selection, not an authorization
// token or transactional snapshot. Incomplete candidates are diagnostic only.
type PreparedCompletionBatch struct {
	Selection        CompletionSelection         `json:"selection"`
	DiscoveryTenant  string                      `json:"discoveryTenant"`
	TenantIDs        []string                    `json:"tenantIds"`
	Candidates       []CompletionCandidate       `json:"candidates"`
	Status           CompletionPreparationStatus `json:"status"`
	CompletionReason CompletionReason            `json:"completionReason,omitempty"`
	PageCount        int64                       `json:"pageCount"`
	ObservedCount    int64                       `json:"observedCount"`
	Limited          bool                        `json:"limited"`
	Variables        map[string]any              `json:"variables,omitempty"`
}

// CompletionObservationStatus describes read evidence independently of submission.
type CompletionObservationStatus string

const (
	// CompletionObservationCompleted confirms completion without attributing it.
	CompletionObservationCompleted CompletionObservationStatus = "completed"
	// CompletionObservationNotCompleted retains a known non-completed state.
	CompletionObservationNotCompleted CompletionObservationStatus = "not_completed"
	// CompletionObservationMissing never proves successful completion.
	CompletionObservationMissing CompletionObservationStatus = "missing"
	// CompletionObservationUnreadable means a read failed.
	CompletionObservationUnreadable CompletionObservationStatus = "unreadable"
	// CompletionObservationUnknown means the observed state is not understood.
	CompletionObservationUnknown CompletionObservationStatus = "unknown"
	// CompletionObservationSkipped means independent observation was not performed.
	CompletionObservationSkipped CompletionObservationStatus = "skipped"
)

// CompletionObservationPhase identifies when evidence was collected.
type CompletionObservationPhase string

const (
	// CompletionPhaseRecheck is the mandatory read before submission.
	CompletionPhaseRecheck CompletionObservationPhase = "recheck"
	// CompletionPhaseReconciliation is a read after ambiguous or race evidence.
	CompletionPhaseReconciliation CompletionObservationPhase = "reconciliation"
)

// CompletionObservation retains last known state without changing submission evidence.
type CompletionObservation struct {
	Status     CompletionObservationStatus `json:"status"`
	State      string                      `json:"state,omitempty"`
	Phase      CompletionObservationPhase  `json:"phase,omitempty"`
	Diagnostic string                      `json:"diagnostic,omitempty"`
}

// CompletionDisposition partitions all frozen targets exactly once.
type CompletionDisposition string

const (
	// CompletionDispositionAccepted records acceptance without independent confirmation.
	CompletionDispositionAccepted CompletionDisposition = "accepted"
	// CompletionDispositionCompleted records independently observed completion.
	CompletionDispositionCompleted CompletionDisposition = "completed"
	// CompletionDispositionUnsatisfied records a target not proven satisfied.
	CompletionDispositionUnsatisfied CompletionDisposition = "unsatisfied"
	// CompletionDispositionNotStarted records a selected target never dispatched.
	CompletionDispositionNotStarted CompletionDisposition = "not_started"
)

// CompletionOutcomeReason gives stable explanations for unresolved or unstarted targets.
type CompletionOutcomeReason string

const (
	// CompletionOutcomeScopeChanged prevents completion outside the frozen predicates.
	CompletionOutcomeScopeChanged CompletionOutcomeReason = "scope_changed"
	// CompletionOutcomeOtherTerminal identifies a terminal state other than completion.
	CompletionOutcomeOtherTerminal CompletionOutcomeReason = "other_terminal"
	// CompletionOutcomeUnsupportedState blocks unknown states conservatively.
	CompletionOutcomeUnsupportedState CompletionOutcomeReason = "unsupported_state"
	// CompletionOutcomeMissing retains a missing read.
	CompletionOutcomeMissing CompletionOutcomeReason = "missing"
	// CompletionOutcomeUnreadable retains a failed read.
	CompletionOutcomeUnreadable CompletionOutcomeReason = "unreadable"
	// CompletionOutcomePayloadIncompatible blocks incompatible listener variables.
	CompletionOutcomePayloadIncompatible CompletionOutcomeReason = "payload_incompatible"
	// CompletionOutcomeRejected records unsatisfied explicit rejection.
	CompletionOutcomeRejected CompletionOutcomeReason = "rejected"
	// CompletionOutcomeUncertain preserves unresolved submission uncertainty.
	CompletionOutcomeUncertain CompletionOutcomeReason = "uncertain"
	// CompletionOutcomeFailFast records scheduling stopped after failure.
	CompletionOutcomeFailFast CompletionOutcomeReason = "fail_fast"
	// CompletionOutcomeDeclined records an operator declining the frozen batch.
	CompletionOutcomeDeclined CompletionOutcomeReason = "declined"
	// CompletionOutcomeInterrupted records caller cancellation or deadline expiry.
	CompletionOutcomeInterrupted CompletionOutcomeReason = "interrupted"
)

// CompletionOutcome keeps submission, observation and final disposition independent.
type CompletionOutcome struct {
	Candidate   CompletionCandidate     `json:"candidate"`
	Disposition CompletionDisposition   `json:"disposition"`
	Submission  CompletionSubmission    `json:"submission"`
	Observation CompletionObservation   `json:"observation"`
	Reason      CompletionOutcomeReason `json:"reason,omitempty"`
}

// CompletionBatchStatus identifies the lifecycle stage of an aggregate report.
type CompletionBatchStatus string

const (
	// CompletionBatchPreview contains candidates and no invented execution outcomes.
	CompletionBatchPreview CompletionBatchStatus = "preview"
	// CompletionBatchFinished contains all frozen execution outcomes.
	CompletionBatchFinished CompletionBatchStatus = "finished"
	// CompletionBatchAborted records declined execution.
	CompletionBatchAborted CompletionBatchStatus = "aborted"
	// CompletionBatchPreparationFailed contains diagnostic partial discovery only.
	CompletionBatchPreparationFailed CompletionBatchStatus = "preparation_failed"
	// CompletionBatchInterrupted remains an invocation failure even if returned rows succeeded.
	CompletionBatchInterrupted CompletionBatchStatus = "interrupted"
)

// CompletionCounts partitions execution targets: selected equals accepted plus
// completed plus unsatisfied plus notStarted. Every count is encoded, including zero.
type CompletionCounts struct {
	Selected    int `json:"selected"`
	Accepted    int `json:"accepted"`
	Completed   int `json:"completed"`
	Unsatisfied int `json:"unsatisfied"`
	NotStarted  int `json:"notStarted"`
}

// CompletionBatchReport carries deterministic results alongside a separate returned
// aggregate error. Preview uses candidates; execution uses outcomes, even when empty.
type CompletionBatchReport struct {
	Status           CompletionBatchStatus `json:"status"`
	Selection        CompletionSelection   `json:"selection"`
	DiscoveryTenant  string                `json:"discoveryTenant"`
	TenantIDs        []string              `json:"tenantIds"`
	CompletionReason CompletionReason      `json:"completionReason,omitempty"`
	PageCount        int64                 `json:"pageCount"`
	ObservedCount    int64                 `json:"observedCount"`
	Limited          bool                  `json:"limited"`
	Counts           CompletionCounts      `json:"counts"`
	Candidates       []CompletionCandidate `json:"candidates,omitzero"`
	Outcomes         []CompletionOutcome   `json:"outcomes,omitzero"`
}
