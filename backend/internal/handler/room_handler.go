package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"kpl-bp-simulator/internal/repository"
	"kpl-bp-simulator/internal/service"
	ws "kpl-bp-simulator/internal/websocket"

	"github.com/gin-gonic/gin"
)

// RoomHandler 房间相关接口
type RoomHandler struct {
	heroRepo         repository.HeroRepo
	gameRepo         repository.GameRepo
	teamRepo         repository.TeamRepo
	hub              *ws.Hub
	roomEngines      map[string]*service.Engine // roomID -> engine
	tournamentBanned map[int]bool
}

// NewRoomHandler 创建 RoomHandler
func NewRoomHandler(heroRepo repository.HeroRepo, gameRepo repository.GameRepo, teamRepo repository.TeamRepo, hub *ws.Hub, tournamentBanned map[int]bool) *RoomHandler {
	h := &RoomHandler{
		heroRepo:         heroRepo,
		gameRepo:         gameRepo,
		teamRepo:         teamRepo,
		hub:              hub,
		roomEngines:      make(map[string]*service.Engine),
		tournamentBanned: tournamentBanned,
	}

	// 注册 WebSocket 操作处理器
	hub.SetActionHandler(h.handleWSAction)

	return h
}

// handleWSAction 处理来自 WebSocket 的 ban/pick 操作
func (h *RoomHandler) handleWSAction(roomID string, side string, msg *ws.IncomingMessage) error {
	engine, ok := h.roomEngines[roomID]
	if !ok {
		return nil // 房间不存在，静默忽略
	}

	var actionType service.ActionType
	switch msg.Type {
	case "ban":
		actionType = service.Ban
	case "pick":
		actionType = service.Pick
	default:
		return nil
	}

	if err := engine.ExecuteAction(actionType, msg.HeroID); err != nil {
		return err
	}

	engine.Advance()

	// 广播更新
	h.broadcastUpdate(roomID, engine)
	return nil
}

// CreateRoom POST /api/rooms — 创建房间
func (h *RoomHandler) CreateRoom(c *gin.Context) {
	var req struct {
		BlueTeamID int `json:"blueTeamId"`
		RedTeamID  int `json:"redTeamId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}

	// 校验战队存在
	if _, err := h.teamRepo.GetByID(req.BlueTeamID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "蓝方战队不存在"})
		return
	}
	if _, err := h.teamRepo.GetByID(req.RedTeamID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "红方战队不存在"})
		return
	}

	// 生成房间号
	roomID := generateRoomID()

	// 创建 BP 引擎
	heroes, err := h.heroRepo.ListActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "加载英雄池失败"})
		return
	}

	perms := make(map[int]*service.HeroPerm, len(heroes))
	for _, hero := range heroes {
		if h.tournamentBanned[hero.ID] {
			perms[hero.ID] = service.NonePerm()
		} else {
			perms[hero.ID] = service.FullPerm()
		}
	}

	engine := service.NewBPEngineWithPerms(req.BlueTeamID, req.RedTeamID, perms, h.tournamentBanned, nil)
	engine.Start()

	h.roomEngines[roomID] = engine

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"data": gin.H{
			"roomId": roomID,
			"status": "waiting",
		},
	})
}

// JoinRoom POST /api/rooms/:id/join — 加入房间
func (h *RoomHandler) JoinRoom(c *gin.Context) {
	roomID := c.Param("id")

	var req struct {
		TeamID int    `json:"teamId"`
		Side   string `json:"side"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}

	if _, ok := h.roomEngines[roomID]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "房间不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"roomId": roomID,
			"side":   req.Side,
		},
	})
}

// GetRoomStatus GET /api/rooms/:id/status — 获取当前 BP 状态
func (h *RoomHandler) GetRoomStatus(c *gin.Context) {
	roomID := c.Param("id")

	engine, ok := h.roomEngines[roomID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "房间不存在"})
		return
	}

	state := h.buildRoomState(roomID, engine)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": state})
}

// Ban POST /api/rooms/:id/ban — 执行 ban 操作
func (h *RoomHandler) Ban(c *gin.Context) {
	roomID := c.Param("id")

	var req struct {
		HeroID int `json:"heroId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}

	engine, ok := h.roomEngines[roomID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "房间不存在"})
		return
	}

	if err := engine.ExecuteAction(service.Ban, req.HeroID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": err.Error()})
		return
	}

	engine.Advance()

	// 广播状态更新
	h.broadcastUpdate(roomID, engine)

	state := h.buildRoomState(roomID, engine)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": state})
}

// Pick POST /api/rooms/:id/pick — 执行 pick 操作
func (h *RoomHandler) Pick(c *gin.Context) {
	roomID := c.Param("id")

	var req struct {
		HeroID int `json:"heroId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}

	engine, ok := h.roomEngines[roomID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "房间不存在"})
		return
	}

	if err := engine.ExecuteAction(service.Pick, req.HeroID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": err.Error()})
		return
	}

	engine.Advance()

	// 广播状态更新
	h.broadcastUpdate(roomID, engine)

	state := h.buildRoomState(roomID, engine)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": state})
}

// WebSocketHandler GET /ws — WebSocket 升级入口
func (h *RoomHandler) WebSocketHandler(c *gin.Context) {
	roomID := c.Query("roomId")
	side := c.Query("side") // "blue" | "red" | "spectator"
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少 roomId"})
		return
	}
	if side == "" {
		side = "spectator"
	}

	ws.ServeWS(h.hub, c.Writer, c.Request, roomID, side)
}

// ============================================================================
// 内部辅助方法
// ============================================================================

func (h *RoomHandler) buildRoomState(roomID string, engine *service.Engine) gin.H {
	info := engine.GetTurnInfo()

	// 构建英雄状态列表
	heroes := make([]gin.H, 0)
	for heroID := range engine.HeroPool {
		heroes = append(heroes, gin.H{
			"id":     heroID,
			"status": "available",
		})
	}
	// 补充 Permissions 中的英雄
	for heroID, perm := range engine.Permissions {
		found := false
		for _, h := range heroes {
			if h["id"] == heroID {
				found = true
				break
			}
		}
		if !found {
			status := "available"
			if !perm.CanBan(service.Blue) && !perm.CanPick(service.Blue) &&
				!perm.CanBan(service.Red) && !perm.CanPick(service.Red) {
				status = "tournament_banned"
			}
			heroes = append(heroes, gin.H{
				"id":     heroID,
				"status": status,
			})
		}
	}

	status := "ongoing"
	if engine.IsFinished() {
		status = "finished"
	}

	sideStr := "blue"
	if info.Side == service.Red {
		sideStr = "red"
	}

	return gin.H{
		"roomId":       roomID,
		"status":       status,
		"blueTeamId":   engine.BlueTeamID,
		"redTeamId":    engine.RedTeamID,
		"currentTurn":  int(engine.CurrentTurn),
		"round":        int(engine.Round),
		"action":       info.Action.String(),
		"side":         sideStr,
		"blueBanned":   engine.BlueBanned,
		"redBanned":    engine.RedBanned,
		"bluePicked":   engine.BluePicked,
		"redPicked":    engine.RedPicked,
		"heroes":       heroes,
	}
}

func (h *RoomHandler) broadcastUpdate(roomID string, engine *service.Engine) {
	state := h.buildRoomState(roomID, engine)

	msgType := "bp_update"
	if engine.IsFinished() {
		msgType = "bp_finished"
	}

	h.hub.Broadcast(roomID, &ws.OutgoingMessage{
		Type: msgType,
		Data: state,
	})
}

func generateRoomID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}