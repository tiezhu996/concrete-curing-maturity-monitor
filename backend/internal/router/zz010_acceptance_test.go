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

func buildEngine010(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	settings := config.Config{
		Port: "0", DBDriver: "sqlite",
		DBDSN: "file:acc010_" + t.Name() + "?mode=memory&cache=shared",
		JWTSecret: "acceptance-010-secret-value", JWTExpiry: time.Hour,
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

func login010(t *testing.T, engine *gin.Engine, username, password string) string {
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

func do010(t *testing.T, engine *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func createDraftMix010(t *testing.T, engine *gin.Engine, token, code string) int64 {
	t.Helper()
	w := do010(t, engine, http.MethodPost, "/api/v1/mix-designs", map[string]any{
		"mix_code": code, "version": 1, "cement_type": "CEM II/A-L 42.5 R",
		"water_binder_ratio": 0.42, "datum_temperature_c": 0,
		"calibration_points": []map[string]any{
			{"maturity_degree_hours": 0, "strength_mpa": 0},
			{"maturity_degree_hours": 100, "strength_mpa": 8.5},
			{"maturity_degree_hours": 300, "strength_mpa": 20},
		},
	}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("create mix failed: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode mix: %v", err)
	}
	return created.Data.ID
}

func runForecast010(t *testing.T, engine *gin.Engine, token string) int64 {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/strength-forecasts", strings.NewReader(`{"pour_section_id":1,"temperature_series_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", "rbac-forecast-010")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("run forecast failed: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode forecast: %v", err)
	}
	return created.Data.ID
}

func TestAuditorCannotPublishMix(t *testing.T) {
	engine := buildEngine010(t)
	admin := login010(t, engine, "admin", "admin123")
	auditor := login010(t, engine, "auditor", "auditor123")
	id := createDraftMix010(t, engine, admin, "RBAC-01")
	w := do010(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/mix-designs/%d/publish", id), map[string]any{"lock_version": 1, "note": "publish attempt"}, auditor)
	if w.Code != http.StatusForbidden {
		t.Fatalf("auditor publish = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestAuditorCannotRetireMix(t *testing.T) {
	engine := buildEngine010(t)
	auditor := login010(t, engine, "auditor", "auditor123")
	w := do010(t, engine, http.MethodPost, "/api/v1/mix-designs/1/retire", map[string]any{"lock_version": 1, "note": "retire attempt"}, auditor)
	if w.Code != http.StatusForbidden {
		t.Fatalf("auditor retire = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestAuditorCannotTransitionSection(t *testing.T) {
	engine := buildEngine010(t)
	auditor := login010(t, engine, "auditor", "auditor123")
	w := do010(t, engine, http.MethodPost, "/api/v1/pour-sections/1/transition", map[string]any{"to_state": "suspended", "version": 1, "note": "transition attempt"}, auditor)
	if w.Code != http.StatusForbidden {
		t.Fatalf("auditor transition = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestAuditorCannotConfirmSeries(t *testing.T) {
	engine := buildEngine010(t)
	auditor := login010(t, engine, "auditor", "auditor123")
	w := do010(t, engine, http.MethodPost, "/api/v1/temperature-series/1/confirm", map[string]any{"reason": "confirm attempt"}, auditor)
	if w.Code != http.StatusForbidden {
		t.Fatalf("auditor confirm series = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestAuditorCannotInvalidateSeries(t *testing.T) {
	engine := buildEngine010(t)
	auditor := login010(t, engine, "auditor", "auditor123")
	w := do010(t, engine, http.MethodPost, "/api/v1/temperature-series/1/invalidate", map[string]any{"reason": "invalidate attempt"}, auditor)
	if w.Code != http.StatusForbidden {
		t.Fatalf("auditor invalidate series = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestAuditorCannotReviewForecast(t *testing.T) {
	engine := buildEngine010(t)
	admin := login010(t, engine, "admin", "admin123")
	auditor := login010(t, engine, "auditor", "auditor123")
	id := runForecast010(t, engine, admin)
	w := do010(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/strength-forecasts/%d/review", id), map[string]any{"note": "review attempt"}, auditor)
	if w.Code != http.StatusForbidden {
		t.Fatalf("auditor review = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestAuditorCannotConfirmForecast(t *testing.T) {
	engine := buildEngine010(t)
	admin := login010(t, engine, "admin", "admin123")
	auditor := login010(t, engine, "auditor", "auditor123")
	id := runForecast010(t, engine, admin)
	w := do010(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/strength-forecasts/%d/confirm", id), map[string]any{"note": "confirm attempt"}, auditor)
	if w.Code != http.StatusForbidden {
		t.Fatalf("auditor confirm = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestAuditorCannotVoidForecast(t *testing.T) {
	engine := buildEngine010(t)
	admin := login010(t, engine, "admin", "admin123")
	auditor := login010(t, engine, "auditor", "auditor123")
	id := runForecast010(t, engine, admin)
	w := do010(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/strength-forecasts/%d/void", id), map[string]any{"note": "void attempt"}, auditor)
	if w.Code != http.StatusForbidden {
		t.Fatalf("auditor void = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}
