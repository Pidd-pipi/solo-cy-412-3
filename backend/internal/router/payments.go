package router

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/handler"
	"github.com/smartestate/smartestate/internal/middleware"
)

func RegisterPayments(g *gin.RouterGroup, sv Services, h *handler.Handler) {
	x := handler.NewPaymentHandler(sv.Payments, h)
	g.GET("/payments", x.List)
	g.POST("/payments", middleware.RequirePermission(sv.Permissions, "payment:manage"), x.Create)
	g.POST("/payments/:id/pay", middleware.RateLimit(60), middleware.OperationLog(sv.Logs, "payment.pay"), x.Pay)
}
