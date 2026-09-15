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
