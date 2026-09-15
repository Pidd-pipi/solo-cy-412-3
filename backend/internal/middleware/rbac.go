package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/handler"
	"github.com/smartestate/smartestate/internal/service"
	"net/http"
)

func RequirePermission(s *service.PermissionService, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.Has(c.GetString("role"), code) {
			handler.Fail(c, http.StatusForbidden, constants.CodeForbidden, constants.MessageForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireRole 限制只有指定角色可访问（业主端登记/取消自助接口使用）。
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		if !allowed[c.GetString("role")] {
			handler.Fail(c, http.StatusForbidden, constants.CodeForbidden, constants.MessageForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}
