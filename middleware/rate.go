package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"sync"
	"tgwp/log/zlog"
	"tgwp/response"
)

var limiters sync.Map

func Limiter(r rate.Limit, b int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := zlog.GetCtxFromGin(c)
		ip := c.ClientIP()
		key := fmt.Sprintf("%s|%v|%d", ip, r, b)
		limiter, ok := limiters.Load(key)
		if !ok {
			limiter = rate.NewLimiter(r, b)
			limiters.Store(key, limiter)
		}
		if !limiter.(*rate.Limiter).Allow() {
			zlog.CtxInfof(ctx, "请求过于频繁")
			response.NewResponse(c).Error(response.REQUEST_FREQUENTLY)
			c.Abort()
			return
		}
		c.Next()
	}
}
