package middleware

import (
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := util.ActorFromGin(c)
		if !ok {
			util.WriteError(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "authentication is required"))
			return
		}
		if !constants.HasPermission(actor.Role, permission) {
			util.WriteError(c, util.NewError(http.StatusForbidden, util.CodeForbidden, "the current role cannot perform this action"))
			return
		}
		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		actor, ok := util.ActorFromGin(c)
		if !ok {
			util.WriteError(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "authentication is required"))
			return
		}
		if _, ok := allowed[actor.Role]; !ok {
			util.WriteError(c, util.NewError(http.StatusForbidden, util.CodeForbidden, "the current role cannot perform this transition"))
			return
		}
		c.Next()
	}
}

type rateWindow struct {
	StartedAt time.Time
	Count     int
}

type RateLimiter struct {
	limit   int
	window  time.Duration
	mu      sync.Mutex
	entries map[string]rateWindow
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window}
}

func (limiter *RateLimiter) Middleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := scope + ":" + c.ClientIP()
		if actor, ok := util.ActorFromGin(c); ok {
			key = scope + ":user:" + actor.Username
		}
		now := time.Now()
		limiter.mu.Lock()
		entry := limiter.entries[key]
		if entry.StartedAt.IsZero() || now.Sub(entry.StartedAt) >= limiter.window {
			entry = rateWindow{StartedAt: now}
		}
		entry.Count++
		limiter.entries[key] = entry
		allowed := entry.Count <= limiter.limit
		remaining := limiter.window - now.Sub(entry.StartedAt)
		if len(limiter.entries) > 1000 {
			limiter.removeExpired(now)
		}
		limiter.mu.Unlock()
		if !allowed {
			c.Header("Retry-After", durationSeconds(remaining))
			util.WriteError(c, util.NewError(http.StatusTooManyRequests, util.CodeRateLimited, "request limit reached; retry after the current window"))
			return
		}
		c.Next()
	}
}

func (limiter *RateLimiter) removeExpired(now time.Time) {
	for key, entry := range limiter.entries {
		if now.Sub(entry.StartedAt) >= limiter.window {
			delete(limiter.entries, key)
		}
	}
}

func durationSeconds(duration time.Duration) string {
	seconds := int(duration.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return fmtInt(seconds)
}

func fmtInt(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	buffer := make([]byte, 0, 10)
	for value > 0 {
		buffer = append(buffer, digits[value%10])
		value /= 10
	}
	for left, right := 0, len(buffer)-1; left < right; left, right = left+1, right-1 {
		buffer[left], buffer[right] = buffer[right], buffer[left]
	}
	return string(buffer)
}
