package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"tgwp/log/zlog"
	"tgwp/logic"
	"tgwp/response"
	"tgwp/utils/jwtUtils"
)

func WebsocketConnect(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	userID := jwtUtils.GetUserId(c)
	if userID == 0 {
		response.NewResponse(c).Error(response.USER_NOT_LOGIN)
		return
	}
	rootID := parseRootID(c)
	if err := logic.GetWsHub().Serve(ctx, c.Writer, c.Request, userID, rootID); err != nil {
		zlog.CtxErrorf(ctx, "websocket连接失败:%v", err)
	}
}

func parseRootID(c *gin.Context) int64 {
	rootIDStr := c.Query("root_id")
	if rootIDStr == "" {
		rootIDStr = c.Query("room_id")
	}
	if rootIDStr == "" {
		return 0
	}
	rootID, err := strconv.ParseInt(rootIDStr, 10, 64)
	if err != nil {
		return 0
	}
	return rootID
}
