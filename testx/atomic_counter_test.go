// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAtomicCounter(t *testing.T) {
	var counter AtomicCounter
	require.EqualValues(t, 1, counter.Inc())
	require.EqualValues(t, 6, counter.Add(5))
	require.EqualValues(t, 6, counter.Load())
}

func TestAtomicCounterConcurrent(t *testing.T) {
	var counter AtomicCounter
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Inc()
		}()
	}
	wg.Wait()
	require.EqualValues(t, 100, counter.Load())
}
