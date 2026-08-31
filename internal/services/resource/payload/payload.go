// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package payload

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx/poller"
)

const (
	// DeploymentProcessDefinitionsPhase is the wording-free progress phase for deployed process definitions.
	DeploymentProcessDefinitionsPhase = "deploy process definitions"
	// DeploymentProcessDefinitionsCoreResource is the counted unit for deployment completion facts.
	DeploymentProcessDefinitionsCoreResource = "process definition(s)"
)

// RequireSingleResource verifies that a successful resource response mapped to a real domain resource.
// body is included in malformed-response errors so callers keep the raw upstream payload for diagnosis.
func RequireSingleResource(resource d.Resource, body []byte) (d.Resource, error) {
	if resource == (d.Resource{}) {
		return d.Resource{}, fmt.Errorf("%w: 200 OK but empty resource payload; body=%s", d.ErrMalformedResponse, string(body))
	}
	return resource, nil
}

// DeploymentProcessDefinitionKeys extracts non-empty process-definition keys from version-specific deployment items.
// deployments is the generated-client deployment slice, and key hides the version-specific shape of each item.
func DeploymentProcessDefinitionKeys[T any](deployments []T, key func(T) string) []string {
	keys := make([]string, 0, len(deployments))
	for _, dep := range deployments {
		k := key(dep)
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}

// ReportDeploymentProcessDefinitionCompletions emits one completion fact for every returned process definition key.
func ReportDeploymentProcessDefinitionCompletions(progress func(d.OpsProgressEvent), keys []string, disposition d.OpsCompletionDisposition) {
	if progress == nil || len(keys) == 0 {
		return
	}
	total := 0
	for _, key := range keys {
		if key != "" {
			total++
		}
	}
	if total == 0 {
		return
	}
	for _, key := range keys {
		ReportDeploymentProcessDefinitionCompletion(progress, key, total, disposition)
	}
}

// ReportDeploymentProcessDefinitionCompletion emits one deployment fact with the full deployment scope total.
func ReportDeploymentProcessDefinitionCompletion(progress func(d.OpsProgressEvent), key string, total int, disposition d.OpsCompletionDisposition) {
	if progress == nil || key == "" || total <= 0 {
		return
	}
	completion := d.OpsCompletionProgress{
		Phase:        DeploymentProcessDefinitionsPhase,
		CoreResource: DeploymentProcessDefinitionsCoreResource,
		Total:        total,
		Identity:     key,
		Disposition:  disposition,
	}
	progress(d.OpsProgressEvent{
		Kind:       d.OpsProgressEventKindCompletion,
		Completion: &completion,
	})
}

// NewProcessDefinitionVisibilityPoller builds a polling job that waits until deployed process definitions are readable.
// keys are the process-definition keys returned by deployment, and get must return the HTTP response from a single-key
// lookup. A nil response is treated as malformed; 404 and domain not-found errors mean the definition is not visible yet.
func NewProcessDefinitionVisibilityPoller(keys []string, get func(context.Context, string) (*http.Response, error)) func(context.Context) (poller.JobPollStatus, error) {
	return NewProcessDefinitionVisibilityPollerWithCallback(keys, get, nil)
}

// NewProcessDefinitionVisibilityPollerWithCallback reports each process-definition key the first time the existing
// visibility check proves it readable.
func NewProcessDefinitionVisibilityPollerWithCallback(keys []string, get func(context.Context, string) (*http.Response, error), visible func(string)) func(context.Context) (poller.JobPollStatus, error) {
	seen := make(map[string]struct{}, len(keys))
	return func(ctx context.Context) (poller.JobPollStatus, error) {
		if len(keys) == 0 {
			return poller.JobPollStatus{
				Success: true,
				Message: "no pd in deployment",
			}, nil
		}
		missing := make([]string, 0)
		for _, k := range keys {
			resp, err := get(ctx, k)
			if err != nil {
				if errors.Is(err, d.ErrNotFound) {
					missing = append(missing, k)
					continue
				}
				return poller.JobPollStatus{}, fmt.Errorf("get process definition %q: %w", k, err)
			}
			if resp == nil {
				return poller.JobPollStatus{}, fmt.Errorf("get process definition %q: empty response", k)
			}
			if resp.StatusCode == http.StatusNotFound {
				missing = append(missing, k)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				return poller.JobPollStatus{}, fmt.Errorf("get process definition %q: unexpected status %d", k, resp.StatusCode)
			}
			if visible != nil {
				if _, ok := seen[k]; !ok {
					seen[k] = struct{}{}
					visible(k)
				}
			}
		}
		if len(missing) > 0 {
			return poller.JobPollStatus{
				Success: false,
				Message: fmt.Sprintf("pd not visible; waiting %v", missing),
			}, nil
		}
		return poller.JobPollStatus{
			Success: true,
			Message: fmt.Sprintf("pd visible %v", keys),
		}, nil
	}
}
