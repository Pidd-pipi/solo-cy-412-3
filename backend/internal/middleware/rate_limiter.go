package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/handler"
	"golang.org/x/time/rate"
	"net/http"
)

func RateLimit(n int) gin.HandlerFunc {
	l := rate.NewLimiter(rate.Limit(float64(n)/60), n)
	return func(c *gin.Context) {
		if !l.Allow() {
			handler.Fail(c, http.StatusTooManyRequests, constants.CodeBadRequest, "请求过于频繁")
			c.Abort()
			return
		}
		c.Next()
	}
}
