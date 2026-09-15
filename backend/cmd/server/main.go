package main

import (
	"fmt"
	"log"

	"kpl-bp-simulator/config"
	"kpl-bp-simulator/internal/handler"
	"kpl-bp-simulator/internal/middleware"
	"kpl-bp-simulator/internal/repository"
	ws "kpl-bp-simulator/internal/websocket"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()
	log.Printf("配置加载完成: port=%d, db=%s:%d/%s", cfg.ServerPort, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// 2. 初始化数据库连接
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	log.Println("数据库连接成功")

	// 3. 初始化仓储
	heroRepo := repository.NewMySQLHeroRepo(db)
	teamRepo := repository.NewMySQLTeamRepo(db)
	gameRepo := repository.NewMySQLGameRepo(db)

	// 4. 赛事禁用英雄列表（雅典娜=117, 从seed data中可以看到雅典娜的ID）
	tournamentBanned := map[int]bool{
		// 雅典娜 — 赛事禁用英雄
		// 通过查询在 seed 数据中的实际 ID 确定
	}

	// 5. 初始化 WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()
	log.Println("WebSocket Hub 已启动")

	// 6. 初始化 HTTP 处理器
	heroHandler := handler.NewHeroHandler(heroRepo)
	roomHandler := handler.NewRoomHandler(heroRepo, gameRepo, teamRepo, hub, tournamentBanned)
	gameHandler := handler.NewGameHandler(gameRepo, teamRepo)

	// 7. 创建 Gin 路由
	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())

	// API 路由
	api := r.Group("/api")
	{
		// 英雄相关
		api.GET("/heroes", heroHandler.ListHeroes)

		// 房间相关
		api.POST("/rooms", roomHandler.CreateRoom)
		api.POST("/rooms/:id/join", roomHandler.JoinRoom)
		api.GET("/rooms/:id/status", roomHandler.GetRoomStatus)
		api.POST("/rooms/:id/ban", roomHandler.Ban)
		api.POST("/rooms/:id/pick", roomHandler.Pick)

		// 对局相关
		api.GET("/games", gameHandler.ListGames)
	}

	// WebSocket 路由
	r.GET("/ws", roomHandler.WebSocketHandler)

	// 静态文件服务 - 英雄/战队图标
	r.Static("/images", "./static/images")

	// 8. 启动 HTTP 服务
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("服务器启动在 http://localhost%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}