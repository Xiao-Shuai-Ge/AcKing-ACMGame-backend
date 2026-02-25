package routerg

import (
	"fmt"
	"tgwp/configs"
	"tgwp/global"
	"tgwp/internal/api"
	"tgwp/log/zlog"
	"tgwp/manager"
	"tgwp/middleware"
	"time"

	"github.com/gin-gonic/gin"

	"golang.org/x/time/rate"
)

// RunServer 启动服务器 路由层
func RunServer() {
	r, err := listen()
	if err != nil {
		zlog.Errorf("Listen error: %v", err)
		panic(err.Error())
	}
	r.Run(fmt.Sprintf("%s:%d", configs.Conf.App.Host, configs.Conf.App.Port)) // 启动 Gin 服务器
}

// listen 配置 Gin 服务器
func listen() (*gin.Engine, error) {
	r := gin.Default() // 创建默认的 Gin 引擎
	// 注册全局中间件（例如获取 Trace ID）
	manager.RequestGlobalMiddleware(r)
	//配置静态路由，用于访问上传的文件
	r.Static("/uploads", "uploads")
	// 创建 RouteManager 实例
	routeManager := manager.NewRouteManager(r)
	// 注册各业务路由组的具体路由
	registerRoutes(routeManager)
	return r, nil
}

// registerRoutes 注册各业务路由的具体处理函数
func registerRoutes(routeManager *manager.RouteManager) {

	routeManager.RegisterCommonRoutes(func(rg *gin.RouterGroup) {
		rg.GET("/test", api.Template)
		rg.GET("/profile", middleware.Limiter(rate.Every(time.Second)*5, 10), middleware.Authentication(global.ROLE_USER), api.GetProfile)
		rg.POST("/profile", middleware.Limiter(rate.Every(time.Second)*3, 6), middleware.Authentication(global.ROLE_USER), api.UpdateProfile)
		rg.GET("/user-info", middleware.Limiter(rate.Every(time.Second)*5, 10), api.GetUserInfo)
		rg.GET("/ws", middleware.Authentication(global.ROLE_USER), api.WebsocketConnect)
	})

	routeManager.RegisterSinglePlayerRoutes(func(rg *gin.RouterGroup) {
		rg.POST("/room", middleware.Authentication(global.ROLE_USER), api.CreateSinglePlayerRoom)
		rg.GET("/room", api.GetSinglePlayerRoomInfo)
		rg.POST("/room/abandon", middleware.Authentication(global.ROLE_USER), api.AbandonSinglePlayerRoom)
	})

	routeManager.RegisterTeamRoomRoutes(func(rg *gin.RouterGroup) {
		rg.POST("/room", middleware.Authentication(global.ROLE_USER), api.CreateTeamRoom)
		rg.GET("/room", api.GetTeamRoomInfo)
		rg.GET("/rooms", api.ListTeamRooms)
		rg.GET("/modes", api.ListTeamRoomModes)
	})

	routeManager.RegisterLoginRoutes(func(rg *gin.RouterGroup) {
		rg.POST("/send-code", middleware.Limiter(rate.Every(time.Minute), 4), api.SendCode)
		rg.POST("/register", middleware.Limiter(rate.Every(time.Minute), 5), api.Register)
		rg.POST("/login", middleware.Limiter(rate.Every(time.Minute), 10), api.Login)
	})
}
