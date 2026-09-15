package router

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/handler"
	"github.com/smartestate/smartestate/internal/middleware"
)

func RegisterUsers(g *gin.RouterGroup, sv Services, h *handler.Handler) {
	x := handler.NewUserHandler(sv.Users, h)
	g.GET("/users/me", x.Me)
	g.PUT("/users/me", middleware.OperationLog(sv.Logs, "user.profile.update"), x.UpdateMe)
	g.GET("/users/staff", middleware.RequirePermission(sv.Permissions, "repair:manage"), x.Staff)
}
