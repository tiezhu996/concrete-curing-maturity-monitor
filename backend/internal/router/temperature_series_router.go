package router

import (
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/handler"
	"concrete-curing-maturity-monitor/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterTemperatureSeriesRoutes(api *gin.RouterGroup, value *handler.TemperatureSeriesHandler, limiter *middleware.RateLimiter) {
	group := api.Group("/temperature-series")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), value.List)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), value.Get)
	group.POST("", middleware.RequirePermission(constants.PermissionTemperatureWrite), limiter.Middleware("temperature-import"), value.Import)
	group.POST("/:id/confirm", middleware.RequirePermission(constants.PermissionTemperatureWrite), value.Confirm)
	group.POST("/:id/invalidate", middleware.RequirePermission(constants.PermissionTemperatureWrite), value.Invalidate)
}
