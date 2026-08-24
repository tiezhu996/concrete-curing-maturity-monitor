package middleware

import (
	"concrete-curing-maturity-monitor/backend/internal/util"
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered",
					"request_id", util.RequestID(c), "path", c.Request.URL.Path,
					"panic", recovered, "stack", string(debug.Stack()),
				)
			}
		}()
		c.Next()
	}
}
