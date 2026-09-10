// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// opsTenantTimingOutput synchronizes command output with observations made by
// fake HTTP handlers while a real command is still executing.
type opsTenantTimingOutput struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write records one command output fragment under the shared timing lock.
func (o *opsTenantTimingOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buf.Write(p)
}

// String returns a stable output snapshot for ordering assertions.
func (o *opsTenantTimingOutput) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buf.String()
}

// opsTenantTimingObservations stores output as seen at the first backend
// request and at the first mutation request.
type opsTenantTimingObservations struct {
	mu                  sync.Mutex
	firstRequestOutput  string
	firstMutationOutput string
}

// observe snapshots output at real backend boundaries without reading a
// concurrently mutated buffer.
func (o *opsTenantTimingObservations) observe(output *opsTenantTimingOutput, mutation bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.firstRequestOutput == "" {
		o.firstRequestOutput = output.String()
	}
	if mutation && o.firstMutationOutput == "" {
		o.firstMutationOutput = output.String()
	}
}

// snapshot returns the two synchronized boundary observations.
func (o *opsTenantTimingObservations) snapshot() (string, string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.firstRequestOutput, o.firstMutationOutput
}

// newOpsTenantTimingProxy records command output before forwarding each real
// request to an existing workflow fixture.
func newOpsTenantTimingProxy(t *testing.T, upstreamURL string, output *opsTenantTimingOutput, isMutation func(*http.Request) bool) (*httptest.Server, *opsTenantTimingObservations) {
	t.Helper()

	target, err := url.Parse(upstreamURL)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(target)
	observations := &opsTenantTimingObservations{}
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observations.observe(output, isMutation(r))
		proxy.ServeHTTP(w, r)
	}))
	return server, observations
}

// executeRootForOpsTenantTiming runs a real root command with synchronized
// output and rejects any auto-confirm path that unexpectedly asks a question.
func executeRootForOpsTenantTiming(t *testing.T, output *opsTenantTimingOutput, reset func(), args ...string) (int, error) {
	t.Helper()

	previousConfirm := confirmCmdOrAbortFn
	promptCount := 0
	confirmCmdOrAbortFn = func(bool, string) error {
		promptCount++
		return fmt.Errorf("unexpected prompt during auto-confirm execution")
	}
	defer func() { confirmCmdOrAbortFn = previousConfirm }()

	root := Root()
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	if reset != nil {
		reset()
	}
	_, err := root.ExecuteC()
	return promptCount, err
}
