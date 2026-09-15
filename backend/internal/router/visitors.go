package router

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/handler"
	"github.com/smartestate/smartestate/internal/middleware"
)

// RegisterVisitors 访客通行模块路由：
// 业主端登记/取消/查看凭证，物业审核与容量配置，门岗核对与办理进出。
// 状态变更（登记/审核/取消/进出/容量）的留痕在 service 事务内完成，
// 因此这里不再叠加 OperationLog 中间件以免重复；只读接口保留访问审计。
func RegisterVisitors(g *gin.RouterGroup, sv Services, h *handler.Handler) {
	v := handler.NewVisitorHandler(sv.Visitors, h)
	gate := handler.NewGateHandler(sv.Visitors, h)
	capH := handler.NewBuildingCapacityHandler(sv.Visitors, h)

	// 业主端：凭证登记与自助取消（仅限业主角色，归属校验在 service 内再兜底）。
	residentOnly := middleware.RequireRole(constants.UserRoleResident)
	g.GET("/visitor/passes", middleware.OperationLog(sv.Logs, "visitor.list"), v.List)
	g.POST("/visitor/passes", residentOnly, v.Create)
	g.GET("/visitor/passes/:id", v.Detail)
	g.POST("/visitor/passes/:id/cancel", residentOnly, v.Cancel)

	// 物业工作台：审核（通过/驳回）与楼栋容量配置。
	g.POST("/visitor/passes/:id/approve", middleware.RequirePermission(sv.Permissions, constants.PermissionVisitorReview), v.Approve)
	g.POST("/visitor/passes/:id/reject", middleware.RequirePermission(sv.Permissions, constants.PermissionVisitorReview), v.Reject)
	g.GET("/visitor/capacity", middleware.RequirePermission(sv.Permissions, constants.PermissionVisitorReview), capH.Overview)
	g.PUT("/visitor/capacity", middleware.RequirePermission(sv.Permissions, constants.PermissionVisitorReview), capH.Update)

	// 门岗：核对凭证、办理进出与查看门岗记录。
	g.GET("/gate/verify", middleware.RequirePermission(sv.Permissions, constants.PermissionVisitorGate), gate.Verify)
	g.POST("/gate/passes/:id/checkin", middleware.RequirePermission(sv.Permissions, constants.PermissionVisitorGate), gate.CheckIn)
	g.POST("/gate/passes/:id/checkout", middleware.RequirePermission(sv.Permissions, constants.PermissionVisitorGate), gate.CheckOut)
	g.GET("/gate/records", middleware.RequirePermission(sv.Permissions, constants.PermissionVisitorGate), middleware.OperationLog(sv.Logs, "visitor.gate.read"), gate.Records)
}
