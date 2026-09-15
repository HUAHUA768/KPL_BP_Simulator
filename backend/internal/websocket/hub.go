package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// ============================================================================
// Message — WebSocket 消息结构
// ============================================================================

// IncomingMessage 客户端 → 服务端
type IncomingMessage struct {
	Type   string `json:"type"`   // "ban" | "pick"
	HeroID int    `json:"heroId"` // 英雄 ID
}

// OutgoingMessage 服务端 → 客户端
type OutgoingMessage struct {
	Type string      `json:"type"` // "bp_update" | "error" | "bp_finished"
	Data interface{} `json:"data"`
}

// ============================================================================
// ActionHandler — 处理客户端 ban/pick 操作的函数类型
// ============================================================================

// ActionHandler 由上层（handler 层）注册，处理客户端的 BP 操作
type ActionHandler func(roomID string, side string, msg *IncomingMessage) error

// ============================================================================
// Client — 单个 WebSocket 连接
// ============================================================================

type Client struct {
	conn   *websocket.Conn
	RoomID string
	Side   string // "blue" | "red" | "spectator"
	send   chan []byte
}

// readPump 从连接读取消息，转发到房间的处理器
func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[WS] 读取错误: %v", err)
			}
			break
		}

		var msg IncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[WS] 消息解析失败: %v", err)
			continue
		}

		// 调用上层注册的处理器
		if hub.actionHandler != nil {
			if err := hub.actionHandler(c.RoomID, c.Side, &msg); err != nil {
				c.SendJSON(&OutgoingMessage{
					Type: "error",
					Data: map[string]interface{}{
						"code":    40000,
						"message": err.Error(),
					},
				})
			}
		}
	}
}

// writePump 将消息写入连接
func (c *Client) writePump() {
	defer c.conn.Close()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("[WS] 写入错误: %v", err)
			return
		}
	}
}

// SendJSON 向客户端发送 JSON 消息
func (c *Client) SendJSON(msg *OutgoingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] JSON 序列化失败: %v", err)
		return
	}
	select {
	case c.send <- data:
	default:
		log.Printf("[WS] 发送缓冲区满，丢弃消息")
	}
}

// ============================================================================
// Hub — WebSocket 房间管理器
// ============================================================================

type Hub struct {
	rooms         map[string]map[*Client]bool
	register      chan *Client
	unregister    chan *Client
	actionHandler ActionHandler
	mu            sync.RWMutex
}

// NewHub 创建 Hub
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// SetActionHandler 设置客户端操作的处理器
func (h *Hub) SetActionHandler(handler ActionHandler) {
	h.actionHandler = handler
}

// Run 启动 Hub 事件循环
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.RoomID] == nil {
				h.rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.rooms[client.RoomID][client] = true
			h.mu.Unlock()
			log.Printf("[WS] 客户端加入房间 %s (side=%s)", client.RoomID, client.Side)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.RoomID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					log.Printf("[WS] 客户端离开房间 %s", client.RoomID)
				}
				if len(clients) == 0 {
					delete(h.rooms, client.RoomID)
					log.Printf("[WS] 房间 %s 已清空", client.RoomID)
				}
			}
			h.mu.Unlock()
		}
	}
}

// RegisterClient 注册客户端连接
func (h *Hub) RegisterClient(conn *websocket.Conn, roomID, side string) *Client {
	client := &Client{
		conn:   conn,
		RoomID: roomID,
		Side:   side,
		send:   make(chan []byte, 256),
	}
	h.register <- client
	return client
}

// Broadcast 向房间内所有客户端广播消息
func (h *Hub) Broadcast(roomID string, msg *OutgoingMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}
	for client := range clients {
		client.SendJSON(msg)
	}
}