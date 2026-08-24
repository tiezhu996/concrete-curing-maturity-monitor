package router

import (
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/handler"
	"concrete-curing-maturity-monitor/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPourSectionRoutes(api *gin.RouterGroup, value *handler.PourSectionHandler) {
	group := api.Group("/pour-sections")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), value.List)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), value.Get)
	group.POST("", middleware.RequirePermission(constants.PermissionSectionWrite), value.Create)
	group.PUT("/:id", middleware.RequirePermission(constants.PermissionSectionWrite), value.Update)
	group.POST("/:id/transition", middleware.RequirePermission(constants.PermissionSectionTransit), value.Transition)
}
