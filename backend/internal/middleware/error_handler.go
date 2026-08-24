package middleware

import (
	"concrete-curing-maturity-monitor/backend/internal/util"
	"log/slog"

	"github.com/gin-gonic/gin"
)

func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		err := c.Errors.Last().Err
		logger.Error("request failed", "request_id", util.RequestID(c), "path", c.Request.URL.Path, "error", err)
		util.WriteError(c, err)
	}
}
