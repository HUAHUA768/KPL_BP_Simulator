package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// ============================================================================
// WebSocket 升级器
// ============================================================================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有来源（开发环境；生产环境需要限制）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ============================================================================
// ServeWS — WebSocket 连接入口
// ============================================================================

// ServeWS 处理 WebSocket 升级请求，创建客户端并启动读写循环
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, roomID, side string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] 升级失败: %v", err)
		return
	}

	client := hub.RegisterClient(conn, roomID, side)

	// 启动读写循环（readPump 一阻塞，writePump 在另一个 goroutine 中运行）
	go client.writePump()
	go client.readPump(hub)
}