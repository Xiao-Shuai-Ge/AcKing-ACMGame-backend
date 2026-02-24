package api

import (
	"github.com/gin-gonic/gin"
	"tgwp/log/zlog"
	"tgwp/logic"
	"tgwp/response"
	"tgwp/types"
	"tgwp/utils/jwtUtils"
)

func CreateSinglePlayerRoom(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	userID := jwtUtils.GetUserId(c)
	resp, err := logic.NewSinglePlayerLogic().CreateRoom(ctx, userID)
	response.Response(c, resp, err)
}

func GetSinglePlayerRoomInfo(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.SinglePlayerRoomInfoReq](c)
	if err != nil {
		return
	}
	resp, err := logic.NewSinglePlayerLogic().GetRoomInfo(ctx, req)
	response.Response(c, resp, err)
}

func AbandonSinglePlayerRoom(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.SinglePlayerAbandonReq](c)
	if err != nil {
		return
	}
	userID := jwtUtils.GetUserId(c)
	resp, err := logic.NewSinglePlayerLogic().AbandonRoom(ctx, userID, req)
	response.Response(c, resp, err)
}
