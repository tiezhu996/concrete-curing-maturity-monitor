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

func buildEngine007(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	settings := config.Config{
		Port: "0", DBDriver: "sqlite",
		DBDSN: "file:acc007_" + t.Name() + "?mode=memory&cache=shared",
		JWTSecret: "acceptance-007-secret-value", JWTExpiry: time.Hour,
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

func login007(t *testing.T, engine *gin.Engine, username, password string) string {
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

func do007(t *testing.T, engine *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
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

func createSection007(t *testing.T, engine *gin.Engine, token, code string) int64 {
	t.Helper()
	w := do007(t, engine, http.MethodPost, "/api/v1/pour-sections", map[string]any{
		"section_code": code, "name": "Test Section " + code, "structure_part": "Foundation",
		"volume_m3": 12.5, "mix_design_id": 1, "target_strength_mpa": 28, "owner_team": "Civil works A",
	}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("create section failed: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	return created.Data.ID
}

func transition007(t *testing.T, engine *gin.Engine, token string, id int64, version int, to string) *httptest.ResponseRecorder {
	t.Helper()
	return do007(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/pour-sections/%d/transition", id),
		map[string]any{"to_state": to, "version": version, "note": "transition note for test"}, token)
}

func TestTransitionActiveToSuspendedAllowed(t *testing.T) {
	engine := buildEngine007(t)
	token := login007(t, engine, "admin", "admin123")
	w := transition007(t, engine, token, 1, 1, "suspended")
	if w.Code != http.StatusOK {
		t.Fatalf("active->suspended = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestTransitionPreparedToThresholdRejected(t *testing.T) {
	engine := buildEngine007(t)
	token := login007(t, engine, "admin", "admin123")
	id := createSection007(t, engine, token, "A-001")
	w := transition007(t, engine, token, id, 1, "threshold_reached")
	if w.Code != http.StatusConflict {
		t.Fatalf("prepared->threshold_reached = %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

func TestTransitionPouredToThresholdRejected(t *testing.T) {
	engine := buildEngine007(t)
	token := login007(t, engine, "admin", "admin123")
	id := createSection007(t, engine, token, "A-002")
	if w := transition007(t, engine, token, id, 1, "poured"); w.Code != http.StatusOK {
		t.Fatalf("prepared->poured = %d; body=%s", w.Code, w.Body.String())
	}
	w := transition007(t, engine, token, id, 2, "threshold_reached")
	if w.Code != http.StatusConflict {
		t.Fatalf("poured->threshold_reached = %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

func TestSiteEngineerCannotConfirmThreshold(t *testing.T) {
	engine := buildEngine007(t)
	token := login007(t, engine, "site", "site123")
	w := transition007(t, engine, token, 1, 1, "threshold_reached")
	if w.Code != http.StatusForbidden {
		t.Fatalf("site confirming threshold = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestTransitionToPouredSetsPouredAt(t *testing.T) {
	engine := buildEngine007(t)
	token := login007(t, engine, "admin", "admin123")
	id := createSection007(t, engine, token, "A-003")
	w := transition007(t, engine, token, id, 1, "poured")
	if w.Code != http.StatusOK {
		t.Fatalf("prepared->poured = %d; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			PouredAt *string `json:"poured_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode transition: %v", err)
	}
	if resp.Data.PouredAt == nil || *resp.Data.PouredAt == "" {
		t.Fatalf("poured_at must be set after transitioning to poured: %s", w.Body.String())
	}
}

func TestTransitionToClosedAllowed(t *testing.T) {
	engine := buildEngine007(t)
	token := login007(t, engine, "admin", "admin123")
	id := createSection007(t, engine, token, "A-004")
	if w := transition007(t, engine, token, id, 1, "poured"); w.Code != http.StatusOK {
		t.Fatalf("prepared->poured = %d; body=%s", w.Code, w.Body.String())
	}
	if w := transition007(t, engine, token, id, 2, "curing"); w.Code != http.StatusOK {
		t.Fatalf("poured->curing = %d; body=%s", w.Code, w.Body.String())
	}
	if w := transition007(t, engine, token, id, 3, "threshold_reached"); w.Code != http.StatusOK {
		t.Fatalf("curing->threshold_reached = %d; body=%s", w.Code, w.Body.String())
	}
	w := transition007(t, engine, token, id, 4, "closed")
	if w.Code != http.StatusOK {
		t.Fatalf("threshold_reached->closed = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}
