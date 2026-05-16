// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import "runtime"

const (
	defaultWorkerGOMAXPROCSMultiplier = 2
	defaultWorkerHardLimit            = 32
)

func DetermineNoOfWorkers(jobsCount, wantedWorkers int, noWorkerLimit bool) int {
	workers := wantedWorkers
	if workers <= 0 {
		workers = jobsCount
		if !noWorkerLimit {
			limit := runtime.GOMAXPROCS(0) * defaultWorkerGOMAXPROCSMultiplier
			if limit > defaultWorkerHardLimit {
				limit = defaultWorkerHardLimit
			}
			if limit < 1 {
				limit = 1
			}
			if limit < workers {
				workers = limit
			}
		}
	}
	if workers > jobsCount {
		workers = jobsCount
	}
	return workers
}
