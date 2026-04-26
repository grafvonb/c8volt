// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import "runtime"

func DetermineNoOfWorkers(jobsCount, wantedWorkers int, noWorkerLimit bool) int {
	workers := wantedWorkers
	if workers <= 0 {
		workers = jobsCount
		if !noWorkerLimit {
			if gp := runtime.GOMAXPROCS(0); gp < workers {
				workers = gp
			}
		}
	}
	if workers > jobsCount {
		workers = jobsCount
	}
	return workers
}
