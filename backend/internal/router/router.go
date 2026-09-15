package router

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/smartestate/smartestate/internal/config"
	"github.com/smartestate/smartestate/internal/handler"
	"github.com/smartestate/smartestate/internal/middleware"
	"github.com/smartestate/smartestate/internal/service"
	"log/slog"
)

type Services struct {
	Users         *service.UserService
	Repairs       *service.RepairService
	Payments      *service.PaymentService
	Announcements *service.AnnouncementService
	Permissions   *service.PermissionService
	Logs          *service.OperationLogService
}

func New(cfg config.Config, sv Services, logger any) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	h := &handler.Handler{Validate: validator.New()}
	r.Use(middleware.RequestLog(logger.(*slog.Logger)))
	r.GET("/healthz", func(c *gin.Context) { handler.OK(c, gin.H{"status": "healthy"}) })
	api := r.Group("/api/v1")
	auth := handler.NewAuthHandler(sv.Users, cfg.JWTSecret, h)
	api.POST("/auth/login", middleware.RateLimit(cfg.RateLimit), auth.Login)
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg.JWTSecret))
	RegisterUsers(protected, sv, h)
	RegisterRepairs(protected, sv, h)
	RegisterPayments(protected, sv, h)
	RegisterAnnouncements(protected, sv, h)
	d := handler.NewDashboardHandler(sv.Repairs, sv.Payments, sv.Announcements)
	protected.GET("/dashboard/summary", d.Summary)
	logs := handler.NewOperationLogHandler(sv.Logs)
	protected.GET("/operation-logs", middleware.RequirePermission(sv.Permissions, "log:read"), logs.List)
	return r
}
