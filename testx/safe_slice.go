// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import "sync"

// SafeSlice is a minimal synchronized collector for values captured by
// httptest handlers that can run concurrently under worker-based commands.
type SafeSlice[T any] struct {
	mu    sync.Mutex
	items []T
}

func (s *SafeSlice[T]) Append(value T) {
	s.mu.Lock()
	s.items = append(s.items, value)
	s.mu.Unlock()
}

func (s *SafeSlice[T]) Snapshot() []T {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]T(nil), s.items...)
}
