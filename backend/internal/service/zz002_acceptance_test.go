package service_test

import (
	"context"
	"testing"
	"time"

	"concrete-curing-maturity-monitor/backend/internal/config"
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/middleware"
	"concrete-curing-maturity-monitor/backend/internal/repository"
	"concrete-curing-maturity-monitor/backend/internal/service"
	"concrete-curing-maturity-monitor/backend/internal/util"
)

type deps002 struct {
	forecastSvc service.StrengthForecastService
	seriesSvc   service.TemperatureSeriesService
	sectionSvc  service.PourSectionService
}

func newDeps002(t *testing.T) deps002 {
	t.Helper()
	settings := config.Config{
		DBDriver: "sqlite", DBDSN: "file:acc002svc_" + t.Name() + "?mode=memory&cache=shared",
		JWTSecret: "acceptance-002-secret-value", JWTExpiry: time.Hour,
		ShutdownTimeout: 10 * time.Second, LoginLimitPerMinute: 100000,
		ImportLimitPerMinute: 100000, ForecastLimitPerMinute: 100000, MaxMissingRatio: 0.1,
	}
	db, err := config.OpenDatabase(settings)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	audit := middleware.NewAuditRecorder(db)
	mixRepo := repository.NewMixDesignRepository(db)
	sectionRepo := repository.NewPourSectionRepository(db)
	seriesRepo := repository.NewTemperatureSeriesRepository(db)
	forecastRepo := repository.NewStrengthForecastRepository(db)
	return deps002{
		forecastSvc: service.NewStrengthForecastService(forecastRepo, sectionRepo, seriesRepo, mixRepo, audit),
		seriesSvc:   service.NewTemperatureSeriesService(seriesRepo, sectionRepo, audit, settings.MaxMissingRatio),
		sectionSvc:  service.NewPourSectionService(sectionRepo, mixRepo, audit),
	}
}

func canceledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestForecastListHonorsCanceledCtx(t *testing.T) {
	d := newDeps002(t)
	if _, err := d.forecastSvc.List(canceledCtx(), dto.StrengthForecastQuery{Page: 1, PageSize: 20}); err == nil {
		t.Fatal("canceled forecast list did not fail")
	}
}

func TestForecastGetHonorsCanceledCtx(t *testing.T) {
	d := newDeps002(t)
	actor := util.Actor{UserID: 1, Username: "admin", DisplayName: "System Administrator", Role: "admin"}
	created, _, err := d.forecastSvc.Run(context.Background(), dto.RunStrengthForecastRequest{PourSectionID: 1, TemperatureSeriesID: 1}, "cancel-ctx-key-002", actor)
	if err != nil {
		t.Fatalf("seed run forecast: %v", err)
	}
	if _, err := d.forecastSvc.Get(canceledCtx(), created.ID); err == nil {
		t.Fatal("canceled forecast get did not fail")
	}
}

func TestForecastIdempotentReplayHonorsCanceledCtx(t *testing.T) {
	d := newDeps002(t)
	actor := util.Actor{UserID: 1, Username: "admin", DisplayName: "System Administrator", Role: "admin"}
	if _, _, err := d.forecastSvc.Run(context.Background(), dto.RunStrengthForecastRequest{PourSectionID: 1, TemperatureSeriesID: 1}, "cancel-replay-key-002", actor); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if _, _, err := d.forecastSvc.Run(canceledCtx(), dto.RunStrengthForecastRequest{PourSectionID: 1, TemperatureSeriesID: 1}, "cancel-replay-key-002", actor); err == nil {
		t.Fatal("canceled idempotent replay did not fail")
	}
}

func TestSeriesListHonorsCanceledCtx(t *testing.T) {
	d := newDeps002(t)
	if _, err := d.seriesSvc.List(canceledCtx(), dto.TemperatureSeriesQuery{Page: 1, PageSize: 20}); err == nil {
		t.Fatal("canceled series list did not fail")
	}
}

func TestSeriesGetHonorsCanceledCtx(t *testing.T) {
	d := newDeps002(t)
	if _, err := d.seriesSvc.Get(canceledCtx(), 1); err == nil {
		t.Fatal("canceled series get did not fail")
	}
}

func TestSectionListHonorsCanceledCtx(t *testing.T) {
	d := newDeps002(t)
	if _, err := d.sectionSvc.List(canceledCtx(), dto.PourSectionQuery{Page: 1, PageSize: 20}); err == nil {
		t.Fatal("canceled section list did not fail")
	}
}

func TestSectionGetHonorsCanceledCtx(t *testing.T) {
	d := newDeps002(t)
	if _, err := d.sectionSvc.Get(canceledCtx(), 1); err == nil {
		t.Fatal("canceled section get did not fail")
	}
}
