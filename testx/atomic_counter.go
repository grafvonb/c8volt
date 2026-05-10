// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import "sync/atomic"

// AtomicCounter is a minimal synchronized counter for httptest fixtures that
// need to branch on request order while handlers may run concurrently.
type AtomicCounter struct {
	value atomic.Int64
}

// Add increments the counter by delta and returns the new value.
func (c *AtomicCounter) Add(delta int64) int64 {
	return c.value.Add(delta)
}

// Inc increments the counter by one and returns the new value.
func (c *AtomicCounter) Inc() int64 {
	return c.Add(1)
}

// Load returns the current counter value.
func (c *AtomicCounter) Load() int64 {
	return c.value.Load()
}
