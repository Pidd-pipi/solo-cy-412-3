package router

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/handler"
	"github.com/smartestate/smartestate/internal/middleware"
)

func RegisterRepairs(g *gin.RouterGroup, sv Services, h *handler.Handler) {
	x := handler.NewRepairHandler(sv.Repairs, h)
	g.GET("/repairs", x.List)
	g.POST("/repairs", middleware.OperationLog(sv.Logs, "repair.create"), x.Create)
	g.PATCH("/repairs/:id/assign", middleware.RequirePermission(sv.Permissions, "repair:manage"), middleware.OperationLog(sv.Logs, "repair.assign"), x.Assign)
	g.PATCH("/repairs/:id/status", middleware.RequirePermission(sv.Permissions, "repair:manage"), middleware.OperationLog(sv.Logs, "repair.status"), x.Status)
}
