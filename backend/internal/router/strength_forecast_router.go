package router

import (
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/handler"
	"concrete-curing-maturity-monitor/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterStrengthForecastRoutes(api *gin.RouterGroup, value *handler.StrengthForecastHandler, limiter *middleware.RateLimiter) {
	group := api.Group("/strength-forecasts")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), value.List)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), value.Get)
	group.GET("/:id/compare/:other_id", middleware.RequirePermission(constants.PermissionRead), value.Compare)
	group.POST("", middleware.RequirePermission(constants.PermissionForecastRun), limiter.Middleware("forecast-run"), value.Run)
	group.POST("/:id/review", middleware.RequirePermission(constants.PermissionForecastReview), value.Review)
	group.POST("/:id/confirm", middleware.RequirePermission(constants.PermissionForecastConfirm), value.Confirm)
	group.POST("/:id/void", middleware.RequirePermission(constants.PermissionForecastReview), value.Void)
	group.POST("/:id/replay", middleware.RequirePermission(constants.PermissionForecastRun), limiter.Middleware("forecast-replay"), value.Replay)
}
