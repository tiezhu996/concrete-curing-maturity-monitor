package router_test

import (
	"bytes"
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

func buildEngine001(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	settings := config.Config{
		Port: "0", DBDriver: "sqlite",
		DBDSN: "file:acc001_" + t.Name() + "?mode=memory&cache=shared",
		JWTSecret: "acceptance-001-secret-value", JWTExpiry: time.Hour,
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

func login001(t *testing.T, engine *gin.Engine, username, password string) string {
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

func TestMissingMixGetReturns404(t *testing.T) {
	engine := buildEngine001(t)
	token := login001(t, engine, "admin", "admin123")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mix-designs/999999", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET missing mix = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestMissingMixUpdateReturns404(t *testing.T) {
	engine := buildEngine001(t)
	token := login001(t, engine, "admin", "admin123")
	raw, _ := json.Marshal(map[string]any{"lock_version": 1, "cement_type": "CEM I"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/mix-designs/999999", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("PUT missing mix = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestMissingMixTransitionReturns404(t *testing.T) {
	engine := buildEngine001(t)
	token := login001(t, engine, "admin", "admin123")
	raw, _ := json.Marshal(map[string]any{"lock_version": 1, "note": "check this design"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mix-designs/999999/validate", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("validate missing mix = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestMissingSeriesGetReturns404(t *testing.T) {
	engine := buildEngine001(t)
	token := login001(t, engine, "admin", "admin123")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/temperature-series/999999", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET missing series = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestMissingSeriesImportReturns404(t *testing.T) {
	engine := buildEngine001(t)
	token := login001(t, engine, "admin", "admin123")
	raw, _ := json.Marshal(map[string]any{
		"pour_section_id": 999999, "sensor_code": "TC-X", "sample_interval_min": 60,
		"points": []map[string]any{
			{"timestamp": "2026-08-01T00:00:00Z", "temperature_c": 20},
			{"timestamp": "2026-08-01T01:00:00Z", "temperature_c": 21},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/temperature-series", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("import series to missing section = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestMissingSeriesConfirmReturns404(t *testing.T) {
	engine := buildEngine001(t)
	token := login001(t, engine, "admin", "admin123")
	raw, _ := json.Marshal(map[string]any{"reason": "confirmed by test"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/temperature-series/999999/confirm", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("confirm missing series = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}
