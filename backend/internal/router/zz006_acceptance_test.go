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

func buildEngine006(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	settings := config.Config{
		Port: "0", DBDriver: "sqlite",
		DBDSN: "file:acc006_" + t.Name() + "?mode=memory&cache=shared",
		JWTSecret: "acceptance-006-secret-value", JWTExpiry: time.Hour,
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

func login006(t *testing.T, engine *gin.Engine, username, password string) string {
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

func do006(t *testing.T, engine *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
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
	if method == http.MethodPost && strings.Contains(path, "/strength-forecasts") && !strings.Contains(path, "/replay") {
		req.Header.Set("Idempotency-Key", "replay-persist-key-006")
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestLoginWrongPasswordReturns401(t *testing.T) {
	engine := buildEngine006(t)
	body := `{"username":"admin","password":"definitely-wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password login = %d, want 401; body=%s", w.Code, w.Body.String())
	}
}

func TestReplayPersistsResult(t *testing.T) {
	engine := buildEngine006(t)
	token := login006(t, engine, "admin", "admin123")
	w := do006(t, engine, http.MethodPost, "/api/v1/strength-forecasts", map[string]any{
		"pour_section_id": 1, "temperature_series_id": 1,
	}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("run forecast failed: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	replay := do006(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/strength-forecasts/%d/replay", created.Data.ID), nil, token)
	if replay.Code != http.StatusOK {
		t.Fatalf("replay failed: %d %s", replay.Code, replay.Body.String())
	}
	var replayed struct {
		Data struct {
			DeterminismReplayPass *bool `json:"determinism_replay_passed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(replay.Body.Bytes(), &replayed); err != nil {
		t.Fatalf("decode replay: %v", err)
	}
	if replayed.Data.DeterminismReplayPass == nil || !*replayed.Data.DeterminismReplayPass {
		t.Fatalf("replay result was not persisted: %s", replay.Body.String())
	}
}
