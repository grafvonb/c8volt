// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/batchoperation"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/integration/internal/validation"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	"github.com/grafvonb/c8volt/internal/services/auth"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type liveBatchClient struct {
	api       c8volt.API
	generated *camundav810.ClientWithResponses
}

type batchScenarioEvidence struct {
	Profile           string `json:"profile"`
	ObservedVersion   string `json:"observedVersion"`
	Scenario          string `json:"scenario"`
	BatchOperationKey string `json:"batchOperationKey"`
	State             string `json:"state,omitempty"`
	Total             int32  `json:"total,omitempty"`
	Completed         int32  `json:"completed,omitempty"`
	Failed            int32  `json:"failed,omitempty"`
	Proof             string `json:"proof"`
}

// TestBatchFamily proves 8.10 cancellation and deletion batches separately
// distinguish accepted submission, terminal accounting, and resource post-state.
func TestBatchFamily(t *testing.T) {
	profiles := selectedBatchProfiles(t)
	var evidence []batchScenarioEvidence
	for _, profile := range profiles {
		profile := profile
		t.Run(profile.Name, func(t *testing.T) {
			dataset, _, err := seedIsolatedHeldDataset(t, profile, "batch-positive", 4)
			require.NoError(t, err)
			negative, _, err := seedIsolatedHeldDataset(t, profile, "batch-negative", 1)
			dataset.NegativeProcessDefinitionKeys = negative.PositiveProcessDefinitionKeys
			dataset.NegativeProcessInstanceKeys = negative.PositiveProcessInstanceKeys
			require.NoError(t, err)
			require.NotEmpty(t, dataset.PositiveProcessDefinitionKeys)
			require.NotEmpty(t, dataset.NegativeProcessDefinitionKeys)

			client, err := newLiveBatchClient(profile.Name)
			require.NoError(t, err)

			completed, err := submitAndWaitForCancellationBatch(t, client, dataset.PositiveProcessDefinitionKeys[0], int32(len(dataset.PositiveProcessInstanceKeys)))
			require.NoError(t, err)
			require.Equal(t, int32(len(dataset.PositiveProcessInstanceKeys)), completed.OperationsTotalCount)
			require.Equal(t, completed.OperationsTotalCount, completed.OperationsCompletedCount)
			require.Zero(t, completed.OperationsFailedCount)
			for _, key := range dataset.PositiveProcessInstanceKeys {
				require.NoError(t, requireProcessInstanceStateOneOf(t, profile, key, "CANCELED", "TERMINATED"))
			}
			evidence = append(evidence, newBatchScenarioEvidence(profile, "waited-cancellation", completed, "live-post-state"))

			accepted, err := client.api.CancelProcessInstancesBatch(context.Background(), process.ProcessInstanceFilter{
				ProcessDefinitionKey: dataset.NegativeProcessDefinitionKeys[0],
				State:                process.StateActive,
			})
			require.NoError(t, err)
			require.NotEmpty(t, accepted.Key)
			require.Empty(t, accepted.State, "submission must not claim observed completion")
			evidence = append(evidence, newBatchScenarioEvidence(profile, "no-wait-cancellation-accepted", accepted, "submission-only"))

			observed, err := waitForBatchAccounting(client, accepted.Key, int32(len(dataset.NegativeProcessInstanceKeys)))
			require.NoError(t, err)
			require.Equal(t, "COMPLETED", observed.State)
			require.Equal(t, observed.OperationsTotalCount, observed.OperationsCompletedCount)
			require.Zero(t, observed.OperationsFailedCount)
			for _, key := range dataset.NegativeProcessInstanceKeys {
				require.NoError(t, requireProcessInstanceStateOneOf(t, profile, key, "CANCELED", "TERMINATED"))
			}
			evidence = append(evidence, newBatchScenarioEvidence(profile, "no-wait-cancellation-observed", observed, "live-post-state"))

			for definitionKey, count := range map[string]int32{dataset.PositiveProcessDefinitionKeys[0]: int32(len(dataset.PositiveProcessInstanceKeys)), dataset.NegativeProcessDefinitionKeys[0]: int32(len(dataset.NegativeProcessInstanceKeys))} {
				deleted, err := submitAndWaitForDeletionBatch(t, client, definitionKey, count)
				require.NoError(t, err)
				require.Equal(t, deleted.OperationsTotalCount, deleted.OperationsCompletedCount)
				require.Zero(t, deleted.OperationsFailedCount)
				evidence = append(evidence, newBatchScenarioEvidence(profile, "waited-deletion", deleted, "live-post-state"))
			}
			for _, key := range dataset.allProcessInstanceKeys() {
				require.NoError(t, requireProcessInstanceAbsentEventually(t, profile, key, 2*time.Minute))
			}
		})
	}
	writeJSON(t, "batch-evidence.json", evidence)
}

// selectedBatchProfiles limits the focused slice to verified 8.10 targets;
// older minors retain their existing baseline and real-state coverage.
func selectedBatchProfiles(t *testing.T) []integrationProfile {
	t.Helper()
	profiles := requireSelectedProfiles(t)
	selected := make([]integrationProfile, 0, len(profiles))
	for _, profile := range profiles {
		minor, err := validation.CanonicalMinor(profile.ExpectedVersion)
		if err != nil || minor != "8.10" {
			continue
		}
		ready, err := checkProfileReadiness(t, profile)
		if err != nil {
			t.Fatalf("batch profile %q is not ready: %v", profile.Name, err)
		}
		selected = append(selected, ready)
	}
	if len(selected) == 0 {
		t.Skipf("no Camunda 8.10 profiles selected; set %s to a disposable 8.10 profile", envITProfiles)
	}
	return selected
}

// submitAndWaitForCancellationBatch submits a batch and returns only after
// terminal accounting is available; post-state is asserted by the caller.
func submitAndWaitForCancellationBatch(t *testing.T, client liveBatchClient, definitionKey string, count int32) (batchoperation.BatchOperation, error) {
	t.Helper()
	accepted, err := client.api.CancelProcessInstancesBatch(context.Background(), process.ProcessInstanceFilter{
		ProcessDefinitionKey: definitionKey,
		State:                process.StateActive,
	})
	if err != nil {
		return batchoperation.BatchOperation{}, err
	}
	if accepted.Key == "" || accepted.State != "" {
		return accepted, fmt.Errorf("cancellation submission returned invalid acceptance: %+v", accepted)
	}
	return waitForBatchAccounting(client, accepted.Key, count)
}

// submitAndWaitForDeletionBatch invokes the pinned 8.10 deletion endpoint and
// observes its batch through the public version-neutral waiting contract.
func submitAndWaitForDeletionBatch(t *testing.T, client liveBatchClient, definitionKey string, count int32) (batchoperation.BatchOperation, error) {
	t.Helper()
	filter, err := processDefinitionKeyBatchFilter(definitionKey)
	if err != nil {
		return batchoperation.BatchOperation{}, err
	}
	resp, err := client.generated.DeleteProcessInstancesBatchOperationWithResponse(context.Background(), camundav810.DeleteProcessInstancesBatchOperationJSONRequestBody{Filter: filter})
	if err != nil {
		return batchoperation.BatchOperation{}, err
	}
	if resp == nil {
		return batchoperation.BatchOperation{}, errors.New("delete process-instances batch returned no response")
	}
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return batchoperation.BatchOperation{}, err
	}
	if resp.JSON200 == nil || strings.TrimSpace(resp.JSON200.BatchOperationKey) == "" {
		return batchoperation.BatchOperation{}, errors.New("delete process-instances batch returned no operation key")
	}
	return waitForBatchAccounting(client, resp.JSON200.BatchOperationKey, count)
}

// processDefinitionKeyBatchFilter uses generated union decoding so the live
// request exercises the exact pinned filter representation.
func processDefinitionKeyBatchFilter(definitionKey string) (camundav810.ProcessInstanceFilter, error) {
	var filter camundav810.ProcessInstanceFilter
	raw, err := json.Marshal(map[string]string{"processDefinitionKey": definitionKey})
	if err != nil {
		return filter, err
	}
	if err := json.Unmarshal(raw, &filter); err != nil {
		return filter, err
	}
	return filter, nil
}

// newBatchScenarioEvidence converts batch facts into a credential-free report row.
func newBatchScenarioEvidence(profile integrationProfile, scenario string, op batchoperation.BatchOperation, proof string) batchScenarioEvidence {
	return batchScenarioEvidence{
		Profile: profile.Name, ObservedVersion: profile.ActualVersion,
		Scenario: scenario, BatchOperationKey: op.Key, State: op.State,
		Total: op.OperationsTotalCount, Completed: op.OperationsCompletedCount,
		Failed: op.OperationsFailedCount, Proof: proof,
	}
}

// newLiveBatchClient reproduces the command bootstrap for a named default-local
// profile without serializing configuration or credentials into evidence.
func newLiveBatchClient(profile string) (liveBatchClient, error) {
	cfg, err := loadEffectiveIntegrationConfig(profile)
	if err != nil {
		return liveBatchClient{}, err
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	httpService, err := httpc.New(cfg, log, httpc.WithCookieJar())
	if err != nil {
		return liveBatchClient{}, fmt.Errorf("create integration HTTP service: %w", err)
	}
	authenticator, err := auth.BuildAuthenticator(cfg, httpService.Client(), log)
	if err != nil {
		return liveBatchClient{}, fmt.Errorf("create integration authenticator: %w", err)
	}
	if err := authenticator.Init(context.Background()); err != nil {
		return liveBatchClient{}, fmt.Errorf("initialize integration authenticator: %w", err)
	}
	httpService.InstallAuthEditor(authenticator.Editor())
	api, err := c8volt.New(c8volt.WithConfig(cfg), c8volt.WithHTTPClient(httpService.Client()), c8volt.WithLogger(log))
	if err != nil {
		return liveBatchClient{}, fmt.Errorf("create integration client: %w", err)
	}
	generated, err := camundav810.NewClientWithResponses(cfg.APIs.Camunda.BaseURL, camundav810.WithHTTPClient(httpService.Client()))
	if err != nil {
		return liveBatchClient{}, fmt.Errorf("create generated 8.10 client: %w", err)
	}
	return liveBatchClient{api: api, generated: generated}, nil
}

// loadEffectiveIntegrationConfig resolves one named profile from the same
// default-local files and C8VOLT environment bindings used by command runs.
func loadEffectiveIntegrationConfig(profile string) (*config.Config, error) {
	var configPath string
	for _, candidate := range defaultLocalConfigPaths() {
		if _, err := os.Stat(candidate); err == nil {
			configPath = candidate
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("inspect default local config %s: %w", candidate, err)
		}
	}
	if configPath == "" {
		return nil, fmt.Errorf("default local c8volt config not found in %s", strings.Join(defaultLocalConfigPaths(), ", "))
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	config.BindConfigEnvVars(v)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read default local config %s: %w", configPath, err)
	}
	raw := config.New()
	if err := v.Unmarshal(raw); err != nil {
		return nil, fmt.Errorf("decode default local config %s: %w", configPath, err)
	}
	raw.ActiveProfile = profile
	effective, err := raw.WithProfile()
	if err != nil {
		return nil, err
	}
	if err := effective.Normalize(); err != nil {
		return nil, err
	}
	if err := effective.Validate(); err != nil {
		return nil, err
	}
	return effective, nil
}

// waitForBatchAccounting observes eventual counter export separately from the
// terminal state. Exact owned counts and zero failures remain mandatory.
func waitForBatchAccounting(client liveBatchClient, key string, count int32) (batchoperation.BatchOperation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var last batchoperation.BatchOperation
	for {
		op, err := client.api.WaitBatchOperation(ctx, key)
		if err != nil {
			return last, err
		}
		last = op
		if op.OperationsFailedCount != 0 {
			return op, fmt.Errorf("batch %s has %d failed operations", key, op.OperationsFailedCount)
		}
		if op.State == "COMPLETED" && op.OperationsTotalCount == count && op.OperationsCompletedCount == count {
			return op, nil
		}
		select {
		case <-ctx.Done():
			return last, fmt.Errorf("batch %s accounting did not converge to %d: total=%d completed=%d: %w", key, count, last.OperationsTotalCount, last.OperationsCompletedCount, ctx.Err())
		case <-time.After(time.Second):
		}
	}
}
