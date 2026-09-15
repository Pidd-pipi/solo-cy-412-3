package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/service"
)

func OperationLog(s *service.OperationLogService, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() < 400 {
			s.Add(c.GetUint("userID"), action, c.Request.Method+" "+c.FullPath())
		}
	}
}
