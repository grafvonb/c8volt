// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

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

type JobUpdateRequest struct {
	Key               string
	Retries           *int32
	TimeoutMillis     *int64
	SkipConfirmation  bool
	ConfirmRetries    bool
	RequestedTimeout  string
	RequestedDuration time.Duration
}

func (r JobUpdateRequest) HasRetries() bool {
	return r.Retries != nil
}

func (r JobUpdateRequest) HasTimeout() bool {
	return r.TimeoutMillis != nil
}

func (r JobUpdateRequest) HasUpdates() bool {
	return r.HasRetries() || r.HasTimeout()
}

type JobUpdateResult struct {
	Key                  string `json:"key,omitempty"`
	MutationAccepted     bool   `json:"mutationAccepted"`
	ConfirmationStatus   string `json:"confirmationStatus,omitempty"`
	ConfirmedRetries     *int32 `json:"confirmedRetries,omitempty"`
	SubmittedRetries     *int32 `json:"submittedRetries,omitempty"`
	SubmittedTimeoutMS   *int64 `json:"submittedTimeoutMs,omitempty"`
	ConfirmationError    string `json:"confirmationError,omitempty"`
	MutationError        string `json:"mutationError,omitempty"`
	UnsupportedOperation bool   `json:"unsupportedOperation,omitempty"`
}
