package middleware

import (
	"context"
	"sync"
	"testing"
	"time"

	"concrete-curing-maturity-monitor/backend/internal/config"
)

func TestConcurrentLoginsNoRace(t *testing.T) {
	db, err := config.OpenDatabase(config.Config{
		DBDriver: "sqlite", DBDSN: "file:acc005login?mode=memory&cache=shared",
		JWTSecret: "acceptance-005-secret-value", JWTExpiry: time.Hour,
		ShutdownTimeout: 10 * time.Second, LoginLimitPerMinute: 100000,
		ImportLimitPerMinute: 100000, ForecastLimitPerMinute: 100000, MaxMissingRatio: 0.1,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	auth := NewAuthenticator(db, "acceptance-005-secret-value", time.Hour)
	const workers = 12
	start := make(chan struct{})
	var done sync.WaitGroup
	done.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer done.Done()
			<-start
			for j := 0; j < 3; j++ {
				_, _ = auth.Login(context.Background(), "admin", "admin123")
			}
		}()
	}
	close(start)
	done.Wait()
}

func TestConcurrentRequestIDNoRace(t *testing.T) {
	const workers = 16
	start := make(chan struct{})
	var done sync.WaitGroup
	done.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer done.Done()
			<-start
			for j := 0; j < 500; j++ {
				_ = randomID()
			}
		}()
	}
	close(start)
	done.Wait()
}
