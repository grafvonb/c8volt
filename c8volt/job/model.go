// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import "time"

type Job struct {
	Key                string     `json:"key,omitempty"`
	State              string     `json:"state,omitempty"`
	Retries            int32      `json:"retries,omitempty"`
	Deadline           *time.Time `json:"deadline,omitempty"`
	ProcessInstanceKey string     `json:"processInstanceKey,omitempty"`
	ElementInstanceKey string     `json:"elementInstanceKey,omitempty"`
	ErrorCode          string     `json:"errorCode,omitempty"`
	ErrorMessage       string     `json:"errorMessage,omitempty"`
	TenantId           string     `json:"tenantId,omitempty"`
}

type UpdateRequest struct {
	Key              string
	Retries          *int32
	Timeout          *time.Duration
	TimeoutRaw       string
	TimeoutMillis    *int64
	NoWait           bool
	AutoConfirm      bool
	Automation       bool
	DryRun           bool
	UpdatePlan       *UpdatePlan
	ConfirmRetries   bool
	SkipConfirmation bool
}

func (r UpdateRequest) HasRetries() bool {
	return r.Retries != nil
}

func (r UpdateRequest) HasTimeout() bool {
	return r.TimeoutMillis != nil
}

func (r UpdateRequest) HasUpdates() bool {
	return r.HasRetries() || r.HasTimeout()
}

type RetryChangeStatus string

const (
	RetryChangeNotRequested RetryChangeStatus = "not_requested"
	RetryChangeChanged      RetryChangeStatus = "changed"
	RetryChangeUnchanged    RetryChangeStatus = "unchanged"
)

type UpdatePlan struct {
	Key               string            `json:"key,omitempty"`
	Current           Job               `json:"current,omitempty"`
	RequestedRetries  *int32            `json:"requestedRetries,omitempty"`
	RetryStatus       RetryChangeStatus `json:"retryStatus,omitempty"`
	RequestedTimeout  string            `json:"requestedTimeout,omitempty"`
	TimeoutMillis     *int64            `json:"timeoutMillis,omitempty"`
	MaterialChange    bool              `json:"materialChange"`
	DryRun            bool              `json:"dryRun"`
	MutationSubmitted bool              `json:"mutationSubmitted"`
	Items             []UpdatePlanItem  `json:"items,omitempty"`
}

func (p UpdatePlan) HasMaterialChange() bool {
	return p.MaterialChange
}

type UpdatePlanItem struct {
	Name   string `json:"name,omitempty"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	Status string `json:"status,omitempty"`
}

type UpdateResult struct {
	Key                string      `json:"key,omitempty"`
	Status             string      `json:"status,omitempty"`
	Plan               *UpdatePlan `json:"plan,omitempty"`
	MutationAccepted   bool        `json:"mutationAccepted"`
	ConfirmationStatus string      `json:"confirmationStatus,omitempty"`
	SubmittedRetries   *int32      `json:"submittedRetries,omitempty"`
	SubmittedTimeoutMS *int64      `json:"submittedTimeoutMs,omitempty"`
	ConfirmedRetries   *int32      `json:"confirmedRetries,omitempty"`
	Error              string      `json:"error,omitempty"`
}
