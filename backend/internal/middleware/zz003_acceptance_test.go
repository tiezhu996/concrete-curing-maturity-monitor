package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterEntriesInitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter(5, time.Minute)
	engine := gin.New()
	engine.Use(limiter.Middleware("login"))
	engine.GET("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("rate limited probe = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}
