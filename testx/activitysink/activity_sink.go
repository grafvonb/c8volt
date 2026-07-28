// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package activitysink

import "sync"

// Sink records activity starts and stops for tests that assert progress behavior.
type Sink struct {
	mu       sync.Mutex
	started  int
	stopped  int
	msgs     []string
	updates  []string
	starts   []Start
	pupdates []Update
}

type Start struct {
	Message    string
	Importance int
}

type Update struct {
	Message    string
	Importance int
}

// StartActivity records an activity start message.
func (s *Sink) StartActivity(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started++
	s.msgs = append(s.msgs, msg)
	s.starts = append(s.starts, Start{Message: msg})
}

// StopActivity records a completed activity scope.
func (s *Sink) StopActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped++
}

// UpdateActivity records an activity progress message.
func (s *Sink) UpdateActivity(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, msg)
	s.pupdates = append(s.pupdates, Update{Message: msg})
}

// StartActivityWithImportance records a priority-aware activity start and returns an idempotent stop function.
func (s *Sink) StartActivityWithImportance(msg string, importance int) func() {
	s.mu.Lock()
	s.started++
	s.msgs = append(s.msgs, msg)
	s.starts = append(s.starts, Start{Message: msg, Importance: importance})
	s.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(s.StopActivity)
	}
}

// UpdateActivityWithImportance records a priority-aware activity progress message.
func (s *Sink) UpdateActivityWithImportance(msg string, importance int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, msg)
	s.pupdates = append(s.pupdates, Update{Message: msg, Importance: importance})
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

// Updates returns a copy of recorded activity update messages.
func (s *Sink) Updates() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.updates...)
}

// Starts returns a copy of recorded priority-aware activity starts.
func (s *Sink) Starts() []Start {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Start(nil), s.starts...)
}

// PriorityUpdates returns a copy of recorded priority-aware activity updates.
func (s *Sink) PriorityUpdates() []Update {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Update(nil), s.pupdates...)
}
