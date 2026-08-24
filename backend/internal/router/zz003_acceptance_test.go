package router_test

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"concrete-curing-maturity-monitor/backend/internal/config"
	"concrete-curing-maturity-monitor/backend/internal/handler"
	"concrete-curing-maturity-monitor/backend/internal/middleware"
	"concrete-curing-maturity-monitor/backend/internal/repository"
	"concrete-curing-maturity-monitor/backend/internal/router"
	"concrete-curing-maturity-monitor/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func buildEngine003(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	settings := config.Config{
		Port: "0", DBDriver: "sqlite",
		DBDSN: "file:acc003_" + t.Name() + "?mode=memory&cache=shared",
		JWTSecret: "acceptance-003-secret-value", JWTExpiry: time.Hour,
		ShutdownTimeout: 10 * time.Second, LoginLimitPerMinute: 100000,
		ImportLimitPerMinute: 100000, ForecastLimitPerMinute: 100000, MaxMissingRatio: 0.1,
	}
	db, err := config.OpenDatabase(settings)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	audit := middleware.NewAuditRecorder(db)
	authenticator := middleware.NewAuthenticator(db, settings.JWTSecret, settings.JWTExpiry)
	mixRepo := repository.NewMixDesignRepository(db)
	sectionRepo := repository.NewPourSectionRepository(db)
	seriesRepo := repository.NewTemperatureSeriesRepository(db)
	forecastRepo := repository.NewStrengthForecastRepository(db)
	mixHandler := handler.NewMixDesignHandler(service.NewMixDesignService(mixRepo, audit))
	sectionHandler := handler.NewPourSectionHandler(service.NewPourSectionService(sectionRepo, mixRepo, audit))
	seriesHandler := handler.NewTemperatureSeriesHandler(service.NewTemperatureSeriesService(seriesRepo, sectionRepo, audit, settings.MaxMissingRatio))
	forecastHandler := handler.NewStrengthForecastHandler(service.NewStrengthForecastService(forecastRepo, sectionRepo, seriesRepo, mixRepo, audit))
	authHandler := handler.NewAuthHandler(authenticator)
	engine := gin.New()
	engine.Use(middleware.RequestID(), middleware.Recovery(logger), middleware.ErrorHandler(logger), middleware.AuditContext(audit))
	loginLimiter := middleware.NewRateLimiter(100000, time.Minute)
	apiV1 := engine.Group("/api/v1")
	apiV1.POST("/auth/login", loginLimiter.Middleware("login"), authHandler.Login)
	protected := apiV1.Group("")
	protected.Use(authenticator.Middleware())
	router.RegisterPourSectionRoutes(protected, sectionHandler)
	router.RegisterMixDesignRoutes(protected, mixHandler)
	router.RegisterTemperatureSeriesRoutes(protected, seriesHandler, middleware.NewRateLimiter(100000, time.Minute))
	router.RegisterStrengthForecastRoutes(protected, forecastHandler, middleware.NewRateLimiter(100000, time.Minute))
	return engine
}

func login003(t *testing.T, engine *gin.Engine, username, password string) string {
	t.Helper()
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login %s failed: %d %s", username, w.Code, w.Body.String())
	}
	var envelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return envelope.Data.Token
}

func TestSeedMixDesignPublished(t *testing.T) {
	engine := buildEngine003(t)
	token := login003(t, engine, "admin", "admin123")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mix-designs/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get seed mix = %d; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			DesignState string `json:"design_state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.DesignState != "published" {
		t.Fatalf("seed mix design_state = %q, want published", resp.Data.DesignState)
	}
}

func TestSeedSectionCuringActiveWithPouredAt(t *testing.T) {
	engine := buildEngine003(t)
	token := login003(t, engine, "admin", "admin123")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pour-sections/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get seed section = %d; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			CuringState string  `json:"curing_state"`
			PouredAt    *string `json:"poured_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.CuringState != "curing" {
		t.Fatalf("seed section curing_state = %q, want curing", resp.Data.CuringState)
	}
	if resp.Data.PouredAt == nil || *resp.Data.PouredAt == "" {
		t.Fatalf("seed section poured_at must be set")
	}
}

func TestFreshInstallForecastUsable(t *testing.T) {
	engine := buildEngine003(t)
	token := login003(t, engine, "admin", "admin123")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/strength-forecasts", strings.NewReader(`{"pour_section_id":1,"temperature_series_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", "fresh-install-003")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("fresh install forecast = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Explanation struct {
				DataCoveragePercent float64 `json:"data_coverage_percent"`
			} `json:"explanation"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.Explanation.DataCoveragePercent != 100 {
		t.Fatalf("seed series coverage = %v, want 100", resp.Data.Explanation.DataCoveragePercent)
	}
}

func TestSeedSeriesQualityNotePresent(t *testing.T) {
	engine := buildEngine003(t)
	token := login003(t, engine, "admin", "admin123")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/temperature-series/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get seed series = %d; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			QualityNote string `json:"quality_note"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.QualityNote == "" {
		t.Fatal("seed series quality_note must not be empty")
	}
}
