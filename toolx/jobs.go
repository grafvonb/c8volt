package toolx

import "runtime"

func DetermineNoOfWorkers(jobsCount, wantedWorkersCount int) int {
	workers := wantedWorkersCount
	if workers <= 0 {
		workers = jobsCount
		if gp := runtime.GOMAXPROCS(0); gp < workers {
			workers = gp
		}
	}
	if workers > jobsCount {
		workers = jobsCount
	}
	return workers
}
