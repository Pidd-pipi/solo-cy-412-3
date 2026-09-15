package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/handler"
	"github.com/smartestate/smartestate/internal/util"
	"net/http"
	"strings"
)

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		cl, e := util.ParseJWT(secret, v)
		if e != nil {
			handler.Fail(c, http.StatusUnauthorized, constants.CodeUnauthorized, constants.MessageUnauthorized)
			c.Abort()
			return
		}
		c.Set("userID", cl.UserID)
		c.Set("role", cl.Role)
		c.Next()
	}
}
