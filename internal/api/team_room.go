package api

import (
	"github.com/gin-gonic/gin"
	"tgwp/log/zlog"
	"tgwp/logic"
	"tgwp/response"
	"tgwp/types"
	"tgwp/utils/jwtUtils"
)

func CreateTeamRoom(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.TeamRoomCreateReq](c)
	if err != nil {
		return
	}
	userID := jwtUtils.GetUserId(c)
	resp, err := logic.NewTeamRoomLogic().CreateRoom(ctx, userID, req)
	response.Response(c, resp, err)
}

func GetTeamRoomInfo(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.TeamRoomInfoReq](c)
	if err != nil {
		return
	}
	resp, err := logic.NewTeamRoomLogic().GetRoomInfo(ctx, req)
	response.Response(c, resp, err)
}

func ListTeamRooms(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.TeamRoomListReq](c)
	if err != nil {
		return
	}
	resp, err := logic.NewTeamRoomLogic().ListRooms(ctx, req)
	response.Response(c, resp, err)
}

func ListTeamRoomModes(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	resp, err := logic.NewTeamRoomLogic().ListModes(ctx)
	response.Response(c, resp, err)
}
