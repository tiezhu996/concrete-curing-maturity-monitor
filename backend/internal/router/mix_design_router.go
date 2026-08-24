package router

import (
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/handler"
	"concrete-curing-maturity-monitor/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterMixDesignRoutes(api *gin.RouterGroup, value *handler.MixDesignHandler) {
	group := api.Group("/mix-designs")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), value.List)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), value.Get)
	group.POST("", middleware.RequirePermission(constants.PermissionMixWrite), value.Create)
	group.PUT("/:id", middleware.RequirePermission(constants.PermissionMixWrite), value.Update)
	group.POST("/:id/validate", middleware.RequirePermission(constants.PermissionMixWrite), value.Validate)
	group.POST("/:id/publish", middleware.RequirePermission(constants.PermissionMixPublish), value.Publish)
	group.POST("/:id/retire", middleware.RequirePermission(constants.PermissionMixPublish), value.Retire)
}
