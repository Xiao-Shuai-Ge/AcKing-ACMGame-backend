package api

import (
	"tgwp/log/zlog"
	"tgwp/logic"
	"tgwp/response"
	"tgwp/types"
	"tgwp/utils/jwtUtils"

	"github.com/gin-gonic/gin"
)

func SendCode(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.SendCodeReq](c)
	if err != nil {
		return
	}
	resp, err := logic.NewLoginLogic().SendCode(ctx, req)
	response.Response(c, resp, err)
}

func Register(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.RegisterReq](c)
	if err != nil {
		return
	}
	resp, err := logic.NewLoginLogic().Register(ctx, req)
	response.Response(c, resp, err)
}

func Login(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.LoginReq](c)
	if err != nil {
		return
	}
	resp, err := logic.NewLoginLogic().Login(ctx, req)
	response.Response(c, resp, err)
}

func GetProfile(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.GetProfileReq](c)
	if err != nil {
		return
	}
	req.UserID = jwtUtils.GetUserId(c)
	resp, err := logic.NewLoginLogic().GetProfile(ctx, req)
	response.Response(c, resp, err)
}

func UpdateProfile(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.UpdateProfileReq](c)
	if err != nil {
		return
	}
	req.UserID = jwtUtils.GetUserId(c)
	resp, err := logic.NewLoginLogic().UpdateProfile(ctx, req)
	response.Response(c, resp, err)
}

func GetUserInfo(c *gin.Context) {
	ctx := zlog.GetCtxFromGin(c)
	req, err := types.BindReq[types.GetUserInfoReq](c)
	if err != nil {
		return
	}
	resp, err := logic.NewLoginLogic().GetUserInfo(ctx, req)
	response.Response(c, resp, err)
}
