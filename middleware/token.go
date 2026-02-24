package middleware

import (
	"strings"
	"tgwp/global"
	"tgwp/log/zlog"
	"tgwp/response"
	"tgwp/utils/jwtUtils"

	"github.com/gin-gonic/gin"
)

func Authentication(role int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := zlog.GetCtxFromGin(c)
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			token := c.Query("token")
			if token != "" {
				authorization = "Bearer " + token
			}
			zlog.CtxDebugf(ctx, "token:%s", token)
		}
		if authorization == "" {
			zlog.CtxErrorf(ctx, "authorization为空")
			response.NewResponse(c).Error(response.TOKEN_IS_BLANK)
			c.Abort()
			return
		}
		list := strings.Split(authorization, " ")
		if len(list) != 2 {
			zlog.CtxErrorf(ctx, "token格式错误")
			response.NewResponse(c).Error(response.TOKEN_FORMAT_ERROR)
			c.Abort()
			return
		}
		token := list[1]
		data, err := jwtUtils.IdentifyToken(token)
		if err != nil {
			zlog.CtxErrorf(ctx, "token验证失败:%v", err)
			response.NewResponse(c).Error(response.TOKEN_IS_EXPIRED)
			c.Abort()
			return
		}
		if data.Role < role {
			zlog.CtxErrorf(ctx, "权限不足")
			response.NewResponse(c).Error(response.PERMISSION_DENIED)
			c.Abort()
			return
		}
		c.Set(global.TOKEN_USER_ID, data.UserID)
		c.Set(global.TOKEN_ROLE, data.Role)
		c.Next()
	}
}
