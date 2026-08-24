package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"strconv"
	"time"

	"concrete-curing-maturity-monitor/backend/internal/config"
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/handler"
	"concrete-curing-maturity-monitor/backend/internal/middleware"
	"concrete-curing-maturity-monitor/backend/internal/repository"
	"concrete-curing-maturity-monitor/backend/internal/service"
	"concrete-curing-maturity-monitor/backend/internal/util"

	"github.com/gin-gonic/gin"
)

type handlers008 struct {
	section  *handler.PourSectionHandler
	series   *handler.TemperatureSeriesHandler
	forecast *handler.StrengthForecastHandler
	forecastSvc service.StrengthForecastService
}

func newHandlers008(t *testing.T) handlers008 {
	t.Helper()
	gin.SetMode(gin.TestMode)
	settings := config.Config{
		DBDriver: "sqlite", DBDSN: "file:acc008_" + t.Name() + "?mode=memory&cache=shared",
		JWTSecret: "acceptance-008-secret-value", JWTExpiry: time.Hour,
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
	forecastSvc := service.NewStrengthForecastService(forecastRepo, sectionRepo, seriesRepo, mixRepo, audit)
	return handlers008{
		section:  handler.NewPourSectionHandler(service.NewPourSectionService(sectionRepo, mixRepo, audit)),
		series:   handler.NewTemperatureSeriesHandler(service.NewTemperatureSeriesService(seriesRepo, sectionRepo, audit, settings.MaxMissingRatio)),
		forecast: handler.NewStrengthForecastHandler(forecastSvc),
		forecastSvc: forecastSvc,
	}
}

func canceledCtx008() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)
	c.Request = req
	return c
}

func TestPourSectionListHonorsCanceledCtx(t *testing.T) {
	h := newHandlers008(t)
	c := canceledCtx008()
	h.section.List(c)
	if c.Writer.Status() != http.StatusInternalServerError {
		t.Fatalf("canceled section list status = %d, want 500", c.Writer.Status())
	}
}

func TestPourSectionGetHonorsCanceledCtx(t *testing.T) {
	h := newHandlers008(t)
	c := canceledCtx008()
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	h.section.Get(c)
	if c.Writer.Status() != http.StatusInternalServerError {
		t.Fatalf("canceled section get status = %d, want 500", c.Writer.Status())
	}
}

func TestSeriesListHonorsCanceledCtx(t *testing.T) {
	h := newHandlers008(t)
	c := canceledCtx008()
	h.series.List(c)
	if c.Writer.Status() != http.StatusInternalServerError {
		t.Fatalf("canceled series list status = %d, want 500", c.Writer.Status())
	}
}

func TestSeriesGetHonorsCanceledCtx(t *testing.T) {
	h := newHandlers008(t)
	c := canceledCtx008()
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	h.series.Get(c)
	if c.Writer.Status() != http.StatusInternalServerError {
		t.Fatalf("canceled series get status = %d, want 500", c.Writer.Status())
	}
}

func TestForecastListHonorsCanceledCtx(t *testing.T) {
	h := newHandlers008(t)
	c := canceledCtx008()
	h.forecast.List(c)
	if c.Writer.Status() != http.StatusInternalServerError {
		t.Fatalf("canceled forecast list status = %d, want 500", c.Writer.Status())
	}
}

func TestForecastGetHonorsCanceledCtx(t *testing.T) {
	h := newHandlers008(t)
	actor := util.Actor{UserID: 1, Username: "admin", DisplayName: "System Administrator", Role: "admin"}
	created, _, err := h.forecastSvc.Run(context.Background(), dto.RunStrengthForecastRequest{PourSectionID: 1, TemperatureSeriesID: 1}, "cancel-ctx-key-008", actor)
	if err != nil {
		t.Fatalf("seed forecast: %v", err)
	}
	c := canceledCtx008()
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(created.ID), 10)}}
	h.forecast.Get(c)
	if c.Writer.Status() != http.StatusInternalServerError {
		t.Fatalf("canceled forecast get status = %d, want 500", c.Writer.Status())
	}
}
