package constants

import (
	"sync"
	"testing"
)

func TestConcurrentPermissionChecksNoRace(t *testing.T) {
	const workers = 16
	start := make(chan struct{})
	var done sync.WaitGroup
	done.Add(workers)
	for i := 0; i < workers; i++ {
		go func(seed int) {
			defer done.Done()
			<-start
			for j := 0; j < 500; j++ {
				if seed%2 == 0 {
					_ = HasPermission("reviewer", "forecast:confirm")
				} else {
					_ = HasPermission("auditor", "forecast:run")
				}
			}
		}(i)
	}
	close(start)
	done.Wait()
}
