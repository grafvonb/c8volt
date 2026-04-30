// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package activitysink

import "sync"

// Sink records activity starts and stops for tests that assert progress behavior.
type Sink struct {
	mu      sync.Mutex
	started int
	stopped int
	msgs    []string
}

// StartActivity records an activity start message.
func (s *Sink) StartActivity(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started++
	s.msgs = append(s.msgs, msg)
}

// StopActivity records a completed activity scope.
func (s *Sink) StopActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped++
}

// Snapshot returns a thread-safe copy of the recorded activity state.
func (s *Sink) Snapshot() (started int, stopped int, msgs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started, s.stopped, append([]string(nil), s.msgs...)
}

// Started returns the number of recorded activity starts.
func (s *Sink) Started() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

// Stopped returns the number of recorded activity stops.
func (s *Sink) Stopped() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopped
}

// Messages returns a copy of recorded activity start messages.
func (s *Sink) Messages() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.msgs...)
}
