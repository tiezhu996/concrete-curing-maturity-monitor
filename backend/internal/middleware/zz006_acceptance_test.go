package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"concrete-curing-maturity-monitor/backend/internal/config"
	"concrete-curing-maturity-monitor/backend/internal/util"

	"github.com/gin-gonic/gin"
)

func TestPanicReturnsJSONError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	engine := gin.New()
	engine.Use(RequestID(), Recovery(logger))
	engine.GET("/boom", func(c *gin.Context) { panic("boom") })
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic response = %d, want 500; body=%q", w.Code, w.Body.String())
	}
	if w.Body.Len() == 0 {
		t.Fatal("panic response body must contain an error envelope")
	}
}

func TestAuditRecordReturnsErrorOnCanceledCtx(t *testing.T) {
	db, err := config.OpenDatabase(config.Config{
		DBDriver: "sqlite", DBDSN: "file:acc006audit?mode=memory&cache=shared",
		JWTSecret: "acceptance-006-secret-value", JWTExpiry: 0,
		ShutdownTimeout: 10 * time.Second, LoginLimitPerMinute: 100000,
		ImportLimitPerMinute: 100000, ForecastLimitPerMinute: 100000, MaxMissingRatio: 0.1,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	recorder := NewAuditRecorder(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	actor := util.Actor{UserID: 1, Username: "admin", DisplayName: "System Administrator", Role: "admin", RequestID: "req-1"}
	if err := recorder.Record(ctx, actor, "pour_section", 1, "create", nil, map[string]any{"name": "x"}, nil); err == nil {
		t.Fatal("audit record with canceled context must return an error")
	}
}
