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
	if !noWorkerLimit && workers > jobsCount {
		workers = jobsCount
	}
	return workers
}
