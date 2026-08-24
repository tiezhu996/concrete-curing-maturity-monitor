package middleware

import (
	"concrete-curing-maturity-monitor/backend/internal/util"
	"log/slog"
	"net/http"
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
				if c.Writer.Written() {
					return
				}
				util.WriteError(c, util.NewError(
					http.StatusInternalServerError, util.CodeInternal, "the request could not be completed",
				))
			}
		}()
		c.Next()
	}
}
