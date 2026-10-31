// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
)

// VersionCheck owns exactly one lazy, synchronous advisory attempt per invocation.
// Its lock protects publication only, never network I/O or callback execution.
type VersionCheck struct {
	mu       sync.Mutex
	started  bool
	done     chan struct{}
	outcome  d.VersionCheckOutcome
	cfg      *config.Config
	client   *http.Client
	log      *slog.Logger
	callback func(d.VersionCheckOutcome)
}

// NewVersionCheck snapshots effective configuration without constructing a probe.
// The callback receives only bounded facts and must not reenter BeforeRequest.
func NewVersionCheck(cfg *config.Config, client *http.Client, log *slog.Logger, callback func(d.VersionCheckOutcome)) *VersionCheck {
	var snapshot *config.Config
	if cfg != nil {
		copied := *cfg
		snapshot = &copied
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &VersionCheck{cfg: snapshot, client: client, log: log, callback: callback, done: make(chan struct{})}
}

// BeforeRequest lets one winner probe and lets waiters cancel independently.
// A canceled winner consumes the attempt; later calls never retry that failure.
func (c *VersionCheck) BeforeRequest(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		select {
		case <-c.done:
		case <-ctx.Done():
		}
		return
	}
	c.started = true
	c.mu.Unlock()
	outcome := c.probe(ctx)
	if c.callback != nil {
		callbackOutcome := outcome
		callbackOutcome.BrokerVersions = slices.Clone(outcome.BrokerVersions)
		c.callback(callbackOutcome)
	}
	c.mu.Lock()
	c.outcome = outcome
	close(c.done)
	c.mu.Unlock()
}

// Outcome returns a copy only after the callback has completed and facts are final.
func (c *VersionCheck) Outcome() (d.VersionCheckOutcome, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	outcome := c.outcome
	outcome.BrokerVersions = slices.Clone(outcome.BrokerVersions)
	return outcome, outcome.Status != ""
}

// probe bounds only the advisory and retains the shared transport, jar, and controls.
func (c *VersionCheck) probe(parent context.Context) d.VersionCheckOutcome {
	outcome := d.VersionCheckOutcome{Status: d.VersionCheckFailed, FailureReason: d.VersionCheckUnavailable}
	if c.cfg == nil {
		return outcome
	}
	outcome.ConfiguredVersion = c.cfg.App.CamundaVersion.String()
	client := *c.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	budget := 2 * time.Second
	if client.Timeout > 0 && client.Timeout < budget {
		budget = client.Timeout
	}
	ctx, cancel := context.WithTimeout(parent, budget)
	defer cancel()
	ctx = httpc.BypassBeforeRequest(httpc.WithoutReadRetry(ctx))
	ctx, _ = httpc.BeginRequestRateAttempt(ctx)
	service, err := New(c.cfg, &client, c.log)
	if err != nil {
		return outcome
	}
	topology, err := service.GetClusterTopology(ctx)
	if err != nil {
		outcome.FailureReason = versionCheckFailure(ctx, err)
		return outcome
	}
	outcome.BrokerCount = len(topology.Brokers)
	outcome.ClusterSize = max(0, topology.ClusterSize)
	outcome.PartitionsCount = max(0, topology.PartitionsCount)
	outcome.ReplicationFactor = max(0, topology.ReplicationFactor)
	for _, broker := range topology.Brokers {
		version := "unknown"
		if release, err := toolx.ParseCamundaRelease(broker.Version); err == nil {
			version = release.Display
		}
		outcome.BrokerVersions = append(outcome.BrokerVersions, version)
	}
	slices.Sort(outcome.BrokerVersions)
	outcome.BrokerVersions = slices.Compact(outcome.BrokerVersions)
	observed, err := toolx.ParseCamundaRelease(topology.GatewayVersion)
	if err != nil {
		outcome.FailureReason = d.VersionCheckInvalidVersion
		return outcome
	}
	configured, err := toolx.ParseCamundaRelease(outcome.ConfiguredVersion)
	if err != nil {
		return outcome
	}
	outcome.Status = d.VersionCheckMatched
	if observed.Major != configured.Major || observed.Minor != configured.Minor {
		outcome.Status = d.VersionCheckMismatched
	}
	outcome.ObservedVersion = observed.Display
	outcome.FailureReason = ""
	return outcome
}

// versionCheckFailure classifies typed causes without retaining raw error text.
func versionCheckFailure(ctx context.Context, err error) d.VersionCheckFailure {
	var network net.Error
	var syntax *json.SyntaxError
	var shape *json.UnmarshalTypeError
	switch {
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(err, context.Canceled):
		return d.VersionCheckCanceled
	case errors.Is(ctx.Err(), context.DeadlineExceeded), errors.Is(err, context.DeadlineExceeded), errors.Is(err, d.ErrGatewayTimeout):
		return d.VersionCheckTimeout
	case errors.As(err, &network) && network.Timeout():
		return d.VersionCheckTimeout
	case errors.Is(err, d.ErrUnauthorized), errors.Is(err, d.ErrForbidden):
		return d.VersionCheckAccessDenied
	case errors.Is(err, d.ErrMalformedResponse), errors.As(err, &syntax), errors.As(err, &shape):
		return d.VersionCheckInvalidVersion
	default:
		return d.VersionCheckUnavailable
	}
}
